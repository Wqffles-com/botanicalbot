package main

import (
	"context"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (*App) HandleReady(_ *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Logged in as %s (%s)\n", r.User, r.User.ID)
}

func (app *App) HandleMessageCreate(_ *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	botID := app.Discord.State.User.ID
	mentioned := strings.Contains(m.Content, "<@"+botID+">") || strings.Contains(m.Content, "<@!"+botID+">")
	repliedToBot := m.ReferencedMessage != nil && m.ReferencedMessage.Author.ID == botID

	if !mentioned && !repliedToBot {
		return
	}

	ctx := context.WithValue(context.Background(), "information", struct {
		User string
	}{
		User: m.Author.Username + " (" + m.Author.ID + ")",
	})
	response, err := app.GenerateResponse(ctx, m.Content)

	if response != nil {
		app.Discord.ChannelMessageSendReply(m.ChannelID, response.Content, m.Reference())
	}

	if err != nil {
		app.Discord.ChannelMessageSendReply(m.ChannelID, "Sorry, something went wrong while generating a response.", m.Reference())
		log.Printf("Error generating response: %v\n", err)
	}
}
