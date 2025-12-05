package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIClient implements Provider for OpenAI API
type OpenAIClient struct {
	endpoint string
	apiKey   string
	model    string
	client   *http.Client
}

// OpenAIConfig holds configuration for OpenAI
type OpenAIConfig struct {
	Endpoint string // https://api.openai.com/v1 (default)
	APIKey   string // sk-...
	Model    string // gpt-4o-mini, gpt-4o
	Timeout  time.Duration
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(cfg OpenAIConfig) *OpenAIClient {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://api.openai.com/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-4o-mini"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}

	return &OpenAIClient{
		endpoint: cfg.Endpoint,
		apiKey:   cfg.APIKey,
		model:    cfg.Model,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Name returns the provider name
func (c *OpenAIClient) Name() string {
	return "openai"
}

// Model returns the model being used
func (c *OpenAIClient) Model() string {
	return c.model
}

// Close cleans up resources
func (c *OpenAIClient) Close() error {
	c.client.CloseIdleConnections()
	return nil
}

// OpenAI API types
type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Tools       []Tool          `json:"tools,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type openAIMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type openAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      openAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openAIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Chat sends a chat completion request
func (c *OpenAIClient) Chat(ctx context.Context, messages []Message) (*Response, error) {
	return c.ChatWithTools(ctx, messages, nil)
}

// ChatWithTools sends a chat completion request with function calling
func (c *OpenAIClient) ChatWithTools(ctx context.Context, messages []Message, tools []Tool) (*Response, error) {
	// Convert messages to OpenAI format
	openAIMessages := make([]openAIMessage, len(messages))
	for i, m := range messages {
		openAIMessages[i] = openAIMessage{
			Role:       m.Role,
			Content:    m.Content,
			Name:       m.Name,
			ToolCalls:  m.ToolCalls,
			ToolCallID: m.ToolCallID,
		}
	}

	reqBody := openAIChatRequest{
		Model:       c.model,
		Messages:    openAIMessages,
		Tools:       tools,
		Temperature: 0.1, // Low temperature for consistent decisions
		MaxTokens:   500,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.endpoint + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp openAIError
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("openai error (%s): %s", errResp.Error.Type, errResp.Error.Message)
		}
		return nil, fmt.Errorf("openai error (status %d): %s", resp.StatusCode, string(body))
	}

	var openAIResp openAIChatResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned")
	}

	choice := openAIResp.Choices[0]

	return &Response{
		Content:      choice.Message.Content,
		ToolCalls:    choice.Message.ToolCalls,
		FinishReason: choice.FinishReason,
		TotalTokens:  openAIResp.Usage.TotalTokens,
	}, nil
}

// IsAvailable checks if OpenAI API is accessible
func (c *OpenAIClient) IsAvailable(ctx context.Context) bool {
	url := c.endpoint + "/models"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
