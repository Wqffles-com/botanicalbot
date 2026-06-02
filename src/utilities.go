package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func LoadConfig() (*Config, error) {
	var config Config
	_, err := toml.DecodeFile("data/config.toml", &config)
	if err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	return &config, nil
}

func CreateApp(config Config) (*App, error) {
	if config.Discord == nil {
		return nil, errors.New("discord config is required")
	}
	if config.Discord.Token == "" {
		return nil, errors.New("discord token is required")
	}
	if config.OpenAI == nil {
		return nil, errors.New("openai config is required")
	}
	if config.OpenAI.ApiKey == "" {
		return nil, errors.New("openai api key is required")
	}
	if config.OpenAI.Model == "" {
		return nil, errors.New("openai model is required")
	}

	dg, err := discordgo.New("Bot " + config.Discord.Token)
	if err != nil {
		return nil, fmt.Errorf("create discord session: %w", err)
	}

	clientOptions := []option.RequestOption{option.WithAPIKey(config.OpenAI.ApiKey)}
	if config.OpenAI.BaseUrl != "" {
		clientOptions = append(clientOptions, option.WithBaseURL(config.OpenAI.BaseUrl))
	}
	ai := openai.NewClient(clientOptions...)

	return &App{
		Discord: dg,
		OpenAI:  &ai,
		Config:  &config,
		History: NewConversationHistory(),
		Tools:   LoadTools(),
	}, nil
}

func (app *App) GenerateResponse(ctx context.Context, userPrompt string) (*openai.ChatCompletion, error) {
	const maxToolCallRounds = 8

	// load prompt.md
	dat, err := os.ReadFile("prompt.md")
	if err != nil {
		return nil, fmt.Errorf("read prompt.md: %w", err)
	}
	sysPrompt := string(dat)

	// load lang.md
	dat, err = os.ReadFile("lang.md")
	if err != nil {
		return nil, fmt.Errorf("read lang.md: %w", err)
	}
	lang := string(dat)

	// create request info

	information, ok := RequestInfoFromContext(ctx)
	if !ok {
		return nil, errors.New("request info missing from context")
	}

	// load history
	history := Filter(app.History.Snapshot(), func(m Message) bool {
		return m.UserID == information.UserID
	})

	messages := Map(history, func(m Message) openai.ChatCompletionMessageParamUnion {
		if m.Role == "user" {
			return openai.UserMessage(m.Content)
		}

		return openai.AssistantMessage(m.Content)
	})
	// construct messages
	messages = append(messages,
		openai.SystemMessage(lang),
		openai.SystemMessage(sysPrompt),
		openai.SystemMessage("You are being spoken to by "+information.Username+" ("+information.UserID+")"),
		openai.UserMessage(userPrompt),
	)

	// history - store user message immediately
	app.History.Append(Message{
		Role:    "user",
		Content: userPrompt,
		UserID:  information.UserID,
	})

	// keep asking the model until it stops requesting tools.
	for round := 0; round < maxToolCallRounds; round++ {
		resp, err := app.OpenAI.Chat.Completions.New(
			ctx,
			openai.ChatCompletionNewParams{
				Model:    app.Config.OpenAI.Model,
				Messages: messages,
				Tools:    Map(app.Tools, ToolToCompletionTool),
			})

		if resp == nil || len(resp.Choices) == 0 || err != nil {
			return nil, err
		}

		toolCalls := resp.Choices[0].Message.ToolCalls
		if len(toolCalls) == 0 {
			// history - store assistant response (final, after tool calls if any)
			app.History.Append(Message{
				Role:    "assistant",
				Content: resp.Choices[0].Message.Content,
				UserID:  information.UserID,
			})

			log.Printf("message: %s", resp.Choices[0].Message.Content)
			return resp, nil
		}

		// include the assistant's tool_calls message so tool results can follow it
		messages = append(messages, resp.Choices[0].Message.ToParam())

		// loop through tool calls
		for i := range toolCalls {
			// parse args
			toolCall := toolCalls[i]
			var args map[string]any
			err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
			if err != nil {
				return nil, err
			}

			// handle tool
			toolCtx := WithToolArgs(ctx, args)
			toolCtx = WithRequestInfo(toolCtx, information)
			toolResponse, err := app.HandleToolCall(toolCtx, toolCall.Function.Name)
			if err != nil {
				return nil, err
			}

			// parse response
			marshalled, err := json.Marshal(toolResponse)
			if err != nil {
				return nil, err
			}

			// add to messages
			messages = append(messages, openai.ToolMessage(string(marshalled), toolCall.ID))
		}
	}

	return nil, fmt.Errorf("tool call recursion limit reached after %d rounds", maxToolCallRounds)
}

func (app *App) HandleToolCall(ctx context.Context, toolName string) (any, error) {
	tool := Filter(app.Tools, func(tool Tool) bool {
		return tool.Name == toolName
	})

	if len(tool) == 0 {
		return nil, fmt.Errorf("unknown tool %q", toolName)
	}

	return tool[0].Execute(ctx, app)
}

func LoadTools() []Tool {
	return []Tool{
		{
			Name:        "get_time",
			Description: "Get the current time in a specific timezone",
			Parameters: map[string]ToolParameter{
				"timezone": {
					Type:        "string",
					Description: "The timezone to get the current time for (e.g. 'America/New_York')",
				},
			},
			Execute: func(ctx context.Context, app *App) (any, error) {
				args, ok := ToolArgsFromContext[map[string]any](ctx)
				if !ok {
					return nil, errors.New("tool args missing from context")
				}

				return map[string]any{
					"received_args": args,
				}, nil
			},
		},
		{
			Name:        "take_note",
			Description: "Take a note on the current user and save it to a Markdown file.",
			Parameters: map[string]ToolParameter{
				"content": {
					Type:        "string",
					Description: "The content to add to the Markdown file.",
				},
				"title": {
					Type:        "string",
					Description: "The title of the note.",
				},
			},
			Execute: func(ctx context.Context, app *App) (any, error) {
				args, ok := ToolArgsFromContext[map[string]any](ctx)
				if !ok {
					return nil, errors.New("tool args missing from context")
				}

				requestInfo, ok := RequestInfoFromContext(ctx)
				if !ok {
					return nil, errors.New("request info missing from context")
				}

				filename := fmt.Sprintf("data/notes/%s/%s.md", requestInfo.UserID, args["title"].(string))
				os.MkdirAll(fmt.Sprintf("data/notes/%s", requestInfo.UserID), os.ModePerm)
				err := os.WriteFile(filename, []byte(args["content"].(string)), os.ModeAppend|os.ModePerm)
				if os.IsNotExist(err) {
					err = os.WriteFile(filename, []byte(args["content"].(string)), os.ModePerm)
				}
				if err != nil {
					return nil, fmt.Errorf("write notes file: %w", err)
				}

				return map[string]any{
					"status": "note saved",
				}, nil
			},
		},
		{
			Name:        "get_note",
			Description: "Get a specific note for the current user.",
			Parameters: map[string]ToolParameter{
				"title": {
					Type:        "string",
					Description: "The title of the note to retrieve.",
				},
			},
			Execute: func(ctx context.Context, app *App) (any, error) {
				requestInfo, ok := RequestInfoFromContext(ctx)
				if !ok {
					return nil, errors.New("request info missing from context")
				}

				args, ok := ToolArgsFromContext[map[string]any](ctx)
				if !ok {
					return nil, errors.New("tool args missing from context")
				}

				filename := fmt.Sprintf("data/notes/%s/%s.md", requestInfo.UserID, args["title"].(string))
				dat, err := os.ReadFile(filename)
				if os.IsNotExist(err) {
					return map[string]any{
						"notes": "user has no notes",
					}, nil
				}
				if err != nil {
					return nil, fmt.Errorf("read notes file: %w", err)
				}

				return map[string]any{
					"notes": string(dat),
				}, nil
			},
		},
		{
			Name:        "get_notes",
			Description: "Get the titles of all of the notes for the current user",
			Parameters:  make(map[string]ToolParameter),
			Execute: func(ctx context.Context, app *App) (any, error) {
				requestInfo, ok := RequestInfoFromContext(ctx)
				if !ok {
					return nil, errors.New("erquest info missing from context")
				}

				filename := fmt.Sprintf("data/notes/%s", requestInfo.UserID)
				files, err := os.ReadDir(filename)
				if os.IsNotExist(err) {
					return map[string][]string{
						"files": {},
					}, nil
				}
				if err != nil {
					return nil, err
				}

				fileNames := Map(files, func(e os.DirEntry) string {
					return e.Name()
				})

				return map[string][]string{
					"files": fileNames,
				}, nil
			},
		},
		{
			Name:        "run_code",
			Description: "Runs code in the custom programming language.",
			Parameters: map[string]ToolParameter{
				"code": {
					Type:        "string",
					Description: "The code to run.",
				},
			},
			Execute: func(ctx context.Context, app *App) (any, error) {
				args, ok := ToolArgsFromContext[map[string]any](ctx)
				if !ok {
					return nil, errors.New("tool args missing from context")
				}

				code, ok := args["code"].(string)
				if !ok {
					return nil, errors.New("code argument is not a string")
				}

				call, err := InterpretLine(code)
				if err != nil {
					return nil, err
				}

				res, err := CallFunction(*call, app)
				if err != nil {
					return nil, err
				}

				return res, nil
			},
		},
	}
}

func Filter[T any](items []T, predicate func(T) bool) []T {
	result := make([]T, 0, len(items))
	for _, item := range items {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

func Map[T, U any](items []T, transform func(T) U) []U {
	result := make([]U, len(items))
	for i, item := range items {
		result[i] = transform(item)
	}
	return result
}

func ToolToCompletionTool(tool Tool) openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        tool.Name,
			Description: openai.String(tool.Description),
			Parameters: map[string]any{
				"type":       "object",
				"properties": tool.Parameters,
			},
		},
	}
}
