package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/openai/openai-go"
)

type Config struct {
	Discord *DiscordConfig
	OpenAI  *OpenAIConfig
}

type DiscordConfig struct {
	Token string
}

type OpenAIConfig struct {
	ApiKey  string
	Model   string
	BaseUrl string
}

type App struct {
	Discord *discordgo.Session
	OpenAI  *openai.Client
	Config  *Config
}
