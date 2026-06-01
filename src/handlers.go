package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (*App) HandleReady(_ *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Logged in as %s (%s)\n", r.User, r.User.ID)
}

func (app *App) HandleMessageCreate(_ *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	// check if bot was mentioned
	botID := app.Discord.State.User.ID
	mentioned := strings.Contains(m.Content, "<@"+botID+">") || strings.Contains(m.Content, "<@!"+botID+">")
	repliedToBot := m.ReferencedMessage != nil && m.ReferencedMessage.Author.ID == botID

	if !mentioned && !repliedToBot {
		return
	}

	// get information
	ctx := context.WithValue(context.Background(), "information", struct {
		Username string
		UserID   string
	}{
		Username: m.Author.Username,
		UserID:   m.Author.ID,
	})

	// typing indicator
	stopTyping := make(chan struct{})
	go func() {
		ticker := time.NewTicker(7 * time.Second)
		defer ticker.Stop()
		// Send typing immediately, then refresh every 7 seconds
		app.Discord.ChannelTyping(m.ChannelID)
		for {
			select {
			case <-ticker.C:
				app.Discord.ChannelTyping(m.ChannelID)
			case <-stopTyping:
				return
			}
		}
	}()

	// response
	prompt := strings.ReplaceAll(m.Content, "<@"+botID+">", "")
	prompt = strings.ReplaceAll(prompt, "<@!"+botID+">", "")
	prompt = strings.TrimSpace(prompt)
	response, err := app.GenerateResponse(ctx, prompt)

	// stop typing indicator
	close(stopTyping)

	// log to history
	app.History.Messages = append(app.History.Messages, Message{
		Role:    "user",
		Content: m.Content,
		UserID:  m.Author.ID,
	})
	if response != nil {
		app.History.Messages = append(app.History.Messages, Message{
			Role:    "assistant",
			Content: response.Content,
			UserID:  m.Author.Username,
		})
	}

	// send response
	if response != nil {
		app.Discord.ChannelMessageSendReply(m.ChannelID, response.Content, m.Reference())
	}

	if err != nil {
		app.Discord.ChannelMessageSendReply(m.ChannelID, "Sorry, something went wrong while generating a response.", m.Reference())
		log.Printf("Error generating response: %v\n", err)
	}
}
