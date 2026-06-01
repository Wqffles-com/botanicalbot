package main

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func (*App) HandleReady(_ *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Logged in as %s (%s)\n", r.User, r.User.ID)
}
