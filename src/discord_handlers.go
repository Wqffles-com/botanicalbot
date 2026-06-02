package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (*App) HandleReady(_ *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Logged in as %s (%s)\n", r.User, r.User.ID)
}

func (app *App) HandleMessageCreate(_ *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil {
		return
	}
	if m.Author.Bot {
		return
	}

	// check if bot was mentioned
	if app.Discord == nil || app.Discord.State == nil || app.Discord.State.User == nil {
		log.Printf("discord state is not ready; ignoring message %s", m.ID)
		return
	}
	botID := app.Discord.State.User.ID
	mentioned := strings.Contains(m.Content, "<@"+botID+">") || strings.Contains(m.Content, "<@!"+botID+">")
	repliedToBot := m.ReferencedMessage != nil && m.ReferencedMessage.Author != nil && m.ReferencedMessage.Author.ID == botID

	if !mentioned && !repliedToBot {
		return
	}

	// get information
	ctx := WithRequestInfo(context.Background(), RequestInfo{
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

	// send response
	if response != nil {
		message := response.Choices[0].Message
		usage := response.Usage
		if _, sendErr := app.Discord.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("%s\n-# Input/Output/Total: %v/%v/%v", message.Content, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens), m.Reference()); sendErr != nil {
			log.Printf("Error sending response reply: %v\n", sendErr)
		}
	}

	if err != nil {
		log.Printf("error generating response: %v\n", err)
		_, sendErr := app.Discord.ChannelMessageSendReply(m.ChannelID, "Sorry, there was an error generating a response.", m.Reference())
		if sendErr != nil {
			log.Printf("Error sending failure reply: %v\n", sendErr)
		}
	}
}
