package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const anthropicMessagesEndpoint = "/messages"

// AnthropicProvider implements Provider for Anthropic's Claude API.
type AnthropicProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewAnthropicProvider creates a new Anthropic provider.
func NewAnthropicProvider(apiKey, baseURL, model string) *AnthropicProvider {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &AnthropicProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

// Name returns "anthropic".
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// anthropicRequest is the request body for Anthropic's messages API.
type anthropicRequest struct {
	Model         string              `json:"model"`
	Messages      []anthropicMessage  `json:"messages"`
	System        string              `json:"system,omitempty"`
	MaxTokens     int                 `json:"max_tokens"`
	Temperature   float64             `json:"temperature,omitempty"`
	Stream        bool                `json:"stream,omitempty"`
	Tools         []anthropicTool     `json:"tools,omitempty"`
}

// anthropicMessage is a message in Anthropic format.
type anthropicMessage struct {
	Role    string        `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

// anthropicContentBlock is a content block in Anthropic format.
type anthropicContentBlock struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	Name   string `json:"name,omitempty"`
	Input  interface{} `json:"input,omitempty"`
	ID     string `json:"id,omitempty"`
}

// anthropicTool is a tool definition in Anthropic format.
type anthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// anthropicResponse is the response from Anthropic's messages API.
type anthropicResponse struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Role       string             `json:"role"`
	Content    []anthropicContentBlock `json:"content"`
	Model      string             `json:"model"`
	StopReason string             `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Convert internal messages to Anthropic format.
func (p *AnthropicProvider) toAnthropicMessages(messages []Message) []anthropicMessage {
	var result []anthropicMessage
	for _, m := range messages {
		block := anthropicContentBlock{
			Type: "text",
			Text: m.Content,
		}
		result = append(result, anthropicMessage{
			Role:    m.Role,
			Content: []anthropicContentBlock{block},
		})
	}
	return result
}

// Convert internal tool definitions to Anthropic format.
func (p *AnthropicProvider) toAnthropicTools(tools []Tool) []anthropicTool {
	result := make([]anthropicTool, len(tools))
	for i, t := range tools {
		result[i] = anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		}
	}
	return result
}

// SendMessage sends a request and returns the full response.
func (p *AnthropicProvider) SendMessage(ctx context.Context, req *Request) (*Response, error) {
	anthropicReq := anthropicRequest{
		Model:       p.model,
		Messages:    p.toAnthropicMessages(req.Messages),
		System:      req.System,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}

	if len(req.Tools) > 0 {
		anthropicReq.Tools = p.toAnthropicTools(req.Tools)
	}

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+anthropicMessagesEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	result := &Response{
		Usage: Usage{
			InputTokens:  anthropicResp.Usage.InputTokens,
			OutputTokens: anthropicResp.Usage.OutputTokens,
		},
	}

	// Extract content and tool calls from content blocks
	for _, block := range anthropicResp.Content {
		switch block.Type {
		case "text":
			result.Content += block.Text
		case "tool_use":
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:   block.ID,
				Type: "tool_use",
				Name: block.Name,
				Arguments: func() map[string]interface{} {
					if m, ok := block.Input.(map[string]interface{}); ok {
						return m
					}
					return map[string]interface{}{}
				}(),
			})
		}
	}

	return result, nil
}

// SendMessageStream sends a request and streams the response.
func (p *AnthropicProvider) SendMessageStream(ctx context.Context, req *Request) (<-chan string, <-chan error, error) {
	anthropicReq := anthropicRequest{
		Model:       p.model,
		Messages:    p.toAnthropicMessages(req.Messages),
		System:      req.System,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}

	if len(req.Tools) > 0 {
		anthropicReq.Tools = p.toAnthropicTools(req.Tools)
	}

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+anthropicMessagesEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("sending request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	contentCh := make(chan string, 100)
	errCh := make(chan error, 1)

	go func() {
		defer resp.Body.Close()
		defer close(contentCh)
		defer close(errCh)

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					return
				}

				var event struct {
					Type string `json:"type"`
					Delta *struct {
						Text string `json:"text"`
					} `json:"delta,omitempty"`
					ContentBlock *struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content_block,omitempty"`
				}
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					continue
				}

				switch event.Type {
				case "content_block_delta":
					if event.Delta != nil && event.Delta.Text != "" {
						contentCh <- event.Delta.Text
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errCh <- fmt.Errorf("reading stream: %w", err)
		}
	}()

	return contentCh, errCh, nil
}
