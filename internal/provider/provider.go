package provider

import (
	"context"
)

// Message represents a single message in a conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ToolCall represents a function/tool call requested by the model.
type ToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Name     string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// Usage contains token usage information.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Request is the request sent to an AI provider.
type Request struct {
	Messages    []Message `json:"messages"`
	System      string    `json:"system,omitempty"`
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Tools       []Tool    `json:"tools,omitempty"`
}

// Response is the response received from an AI provider.
type Response struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Usage     Usage      `json:"usage,omitempty"`
}

// ToolParameter describes a parameter for a tool.
type ToolParameter struct {
	Type        string                   `json:"type"`
	Description string                   `json:"description"`
	Required    bool                     `json:"required"`
	Properties  map[string]ToolParameter `json:"properties,omitempty"`
	Items       *ToolParameter           `json:"items,omitempty"`
	Enum        []string                 `json:"enum,omitempty"`
}

// Tool defines a tool/function that the AI can call.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Provider is the interface that all AI providers must implement.
type Provider interface {
	// Name returns the provider name (e.g., "openai", "anthropic").
	Name() string

	// SendMessage sends a request to the AI and returns the response.
	SendMessage(ctx context.Context, req *Request) (*Response, error)

	// SendMessageStream sends a request and streams the response via the channel.
	// The channel receives partial content strings.
	SendMessageStream(ctx context.Context, req *Request) (<-chan string, <-chan error, error)
}
