package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func (app *App) GenerateResponse(ctx context.Context, userPrompt string) (*openai.ChatCompletionMessage, error) {
	// load prompt
	dat, err := os.ReadFile("prompt.md")
	if err != nil {
		return nil, fmt.Errorf("read prompt.md: %w", err)
	}

	// create request info
	sysPrompt := string(dat)
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
		openai.SystemMessage(sysPrompt),
		openai.SystemMessage("You are being spoken to by "+information.Username+" ("+information.UserID+")"),
		openai.UserMessage(userPrompt),
	)

	// completion
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

	// history
	app.History.Append(Message{
		Role:    "user",
		Content: userPrompt,
		UserID:  information.UserID,
	})
	app.History.Append(Message{
		Role:    "assistant",
		Content: resp.Choices[0].Message.Content,
		UserID:  information.UserID,
	})

	// check for tools
	toolCalls := resp.Choices[0].Message.ToolCalls
	if len(toolCalls) != 0 {
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

		resp, err = app.OpenAI.Chat.Completions.New(
			ctx,
			openai.ChatCompletionNewParams{
				Model:    app.Config.OpenAI.Model,
				Messages: messages,
			})

		if err != nil {
			return nil, err
		}
	}

	return &resp.Choices[0].Message, nil
}

func (app *App) HandleToolCall(ctx context.Context, toolName string) (any, error) {
	tool := Filter(app.Tools, func(tool Tool) bool {
		return tool.Name == toolName
	})

	if len(tool) == 0 {
		return nil, fmt.Errorf("unknown tool %q", toolName)
	}

	return tool[0].Execute(ctx)
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
			Execute: func(ctx context.Context) (any, error) {
				args, ok := ToolArgsFromContext[map[string]any](ctx)
				if !ok {
					return nil, errors.New("tool args missing from context")
				}

				return map[string]any{
					"received_args": args,
				}, nil
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
