package main

import "github.com/bwmarrin/discordgo"

type Config struct {
	Discord *DiscordConfig
}

type DiscordConfig struct {
	Token string
}

type App struct {
	Discord *discordgo.Session
	Config  *Config
}
