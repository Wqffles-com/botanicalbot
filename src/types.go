package main

import (
	"context"
	"sync"

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

type RequestInfo struct {
	Username string
	UserID   string
}

type requestInfoKey struct{}

type toolArgsKey struct{}

func WithRequestInfo(ctx context.Context, info RequestInfo) context.Context {
	return context.WithValue(ctx, requestInfoKey{}, info)
}

func RequestInfoFromContext(ctx context.Context) (RequestInfo, bool) {
	info, ok := ctx.Value(requestInfoKey{}).(RequestInfo)
	return info, ok
}

func WithToolArgs[T any](ctx context.Context, args T) context.Context {
	return context.WithValue(ctx, toolArgsKey{}, args)
}

func ToolArgsFromContext[T any](ctx context.Context) (T, bool) {
	args, ok := ctx.Value(toolArgsKey{}).(T)
	if !ok {
		var zero T
		return zero, false
	}

	return args, true
}

type ConversationHistory struct {
	mu       sync.RWMutex
	Messages []Message
}

func NewConversationHistory() *ConversationHistory {
	return &ConversationHistory{Messages: make([]Message, 0)}
}

func (h *ConversationHistory) Snapshot() []Message {
	if h == nil {
		return nil
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	snapshot := make([]Message, len(h.Messages))
	copy(snapshot, h.Messages)
	return snapshot
}

func (h *ConversationHistory) Append(messages ...Message) {
	if h == nil || len(messages) == 0 {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.Messages = append(h.Messages, messages...)
}

type Message struct {
	Role    string
	Content string
	UserID  string
}

type Tool struct {
	Name        string
	Description string
	Parameters  map[string]ToolParameter
	Execute     func(ctx context.Context) (any, error)
}

type ToolParameter struct {
	Type        string
	Description string
}

type App struct {
	Discord *discordgo.Session
	OpenAI  *openai.Client
	History *ConversationHistory
	Config  *Config
	Tools   []Tool
}
