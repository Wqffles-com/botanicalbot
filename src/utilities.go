package main

import (
	"context"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func LoadConfig() *Config {
	var config Config
	_, err := toml.DecodeFile("data/config.toml", &config)
	if err != nil {
		panic(err)
	}

	return &config
}

func CreateApp(config Config) *App {
	dg, err := discordgo.New("Bot " + config.Discord.Token)
	if err != nil {
		panic(err)
	}

	ai := openai.NewClient(
		option.WithAPIKey(config.OpenAI.ApiKey),
		option.WithBaseURL(config.OpenAI.BaseUrl),
	)

	return &App{
		Discord: dg,
		OpenAI:  &ai,
		Config:  &config,
		History: &ConversationHistory{Messages: make([]Message, 0)},
	}
}

func (app *App) GenerateResponse(ctx context.Context, userPrompt string) (*openai.ChatCompletionMessage, error) {
	dat, err := os.ReadFile("prompt.md")
	if err != nil {
		panic(err)
	}

	sysPrompt := string(dat)
	information := ctx.Value("information").(struct {
		Username string
		UserID   string
	})

	history := Filter(app.History.Messages, func(m Message) bool {
		return m.UserID == information.UserID
	})

	messages := Map(history, func(m Message) openai.ChatCompletionMessageParamUnion {
		if m.Role == "user" {
			return openai.UserMessage(m.Content)
		}

		return openai.AssistantMessage(m.Content)
	})
	messages = append(messages,
		openai.SystemMessage(sysPrompt),
		openai.SystemMessage(information.Username+" ("+information.UserID+")"),
		openai.UserMessage(userPrompt),
	)

	resp, err := app.OpenAI.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model:    app.Config.OpenAI.Model,
			Messages: messages,
		})

	if err != nil {
		return nil, err
	}

	return &resp.Choices[0].Message, nil
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
