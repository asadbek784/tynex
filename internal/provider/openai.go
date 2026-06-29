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

const openaiChatEndpoint = "/chat/completions"

// OpenAIProvider implements Provider for OpenAI-compatible APIs.
// This works with: OpenAI, DeepSeek, Groq, Together AI, and any OpenAI-format API.
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(apiKey, baseURL, model string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

// Name returns "openai".
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// openAIRequest is the request body for OpenAI-compatible chat completions.
type openAIRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIMessage     `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
	Tools       []openAITool        `json:"tools,omitempty"`
}

// openAIMessage is a message in OpenAI format.
type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAITool is a tool definition in OpenAI format.
type openAITool struct {
	Type     string                 `json:"type"`
	Function openAIFunction         `json:"function"`
}

// openAIFunction is a function definition in OpenAI format.
type openAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// openAIResponse is the response body from OpenAI-compatible chat completions.
type openAIResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      openAIChoiceMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// openAIChoiceMessage is the message part of an OpenAI response choice.
type openAIChoiceMessage struct {
	Role      string        `json:"role"`
	Content   *string       `json:"content"`
	ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
}

// openAIToolCall is a tool call in OpenAI format.
type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// Convert internal messages to OpenAI format.
func (p *OpenAIProvider) toOpenAIMessages(messages []Message) []openAIMessage {
	result := make([]openAIMessage, len(messages))
	for i, m := range messages {
		result[i] = openAIMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}
	return result
}

// Convert internal tool definitions to OpenAI format.
func (p *OpenAIProvider) toOpenAITools(tools []Tool) []openAITool {
	result := make([]openAITool, len(tools))
	for i, t := range tools {
		result[i] = openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		}
	}
	return result
}

// SendMessage sends a request and returns the full response.
func (p *OpenAIProvider) SendMessage(ctx context.Context, req *Request) (*Response, error) {
	openAIReq := openAIRequest{
		Model:       p.model,
		Messages:    p.toOpenAIMessages(req.Messages),
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}

	// Add system message if provided
	if req.System != "" {
		systemMsg := openAIMessage{Role: "system", Content: req.System}
		openAIReq.Messages = append([]openAIMessage{systemMsg}, openAIReq.Messages...)
	}

	// Add tools if provided
	if len(req.Tools) > 0 {
		openAIReq.Tools = p.toOpenAITools(req.Tools)
	}

	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+openaiChatEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

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

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := openAIResp.Choices[0]
	result := &Response{
		Usage: Usage{
			InputTokens:  openAIResp.Usage.PromptTokens,
			OutputTokens: openAIResp.Usage.CompletionTokens,
		},
	}

	// Extract content
	if choice.Message.Content != nil {
		result.Content = *choice.Message.Content
	}

	// Extract tool calls
	for _, tc := range choice.Message.ToolCalls {
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			args = map[string]interface{}{"raw": tc.Function.Arguments}
		}
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:        tc.ID,
			Type:      tc.Type,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}

	return result, nil
}

// SendMessageStream sends a request and streams the response.
func (p *OpenAIProvider) SendMessageStream(ctx context.Context, req *Request) (<-chan string, <-chan error, error) {
	openAIReq := openAIRequest{
		Model:       p.model,
		Messages:    p.toOpenAIMessages(req.Messages),
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}

	if req.System != "" {
		systemMsg := openAIMessage{Role: "system", Content: req.System}
		openAIReq.Messages = append([]openAIMessage{systemMsg}, openAIReq.Messages...)
	}

	if len(req.Tools) > 0 {
		openAIReq.Tools = p.toOpenAITools(req.Tools)
	}

	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+openaiChatEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
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

				var chunk struct {
					Choices []struct {
						Delta struct {
							Content string `json:"content"`
						} `json:"delta"`
						FinishReason *string `json:"finish_reason"`
					} `json:"choices"`
				}
				if err := json.Unmarshal([]byte(data), &chunk); err != nil {
					continue // Skip malformed chunks
				}
				for _, choice := range chunk.Choices {
					if choice.Delta.Content != "" {
						contentCh <- choice.Delta.Content
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
