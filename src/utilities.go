package main

import (
	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
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

	return &App{
		Discord: dg,
		Config:  &config,
	}
}
