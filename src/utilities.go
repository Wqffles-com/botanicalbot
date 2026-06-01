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
	}
}

func (app *App) GenerateResponse(ctx context.Context, userPrompt string) (*openai.ChatCompletionMessage, error) {
	dat, err := os.ReadFile("prompt.md")
	if err != nil {
		panic(err)
	}

	sysPrompt := string(dat)
	information := ctx.Value("information").(struct {
		User string
	})

	resp, err := app.OpenAI.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model: app.Config.OpenAI.Model,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(sysPrompt),
				openai.SystemMessage(information.User),
				openai.UserMessage(userPrompt),
			},
		})

	if err != nil {
		return nil, err
	}

	return &resp.Choices[0].Message, nil
}
