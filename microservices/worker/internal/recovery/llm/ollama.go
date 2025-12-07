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

// OllamaClient implements Provider for Ollama (Qwen2.5, etc.)
type OllamaClient struct {
	endpoint string
	model    string
	client   *http.Client
}

// OllamaConfig holds configuration for Ollama
type OllamaConfig struct {
	Endpoint string // http://localhost:11434
	Model    string // qwen2.5
	Timeout  time.Duration
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(cfg OllamaConfig) *OllamaClient {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "http://localhost:11434"
	}
	if cfg.Model == "" {
		cfg.Model = "qwen2.5"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}

	return &OllamaClient{
		endpoint: cfg.Endpoint,
		model:    cfg.Model,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Name returns the provider name
func (c *OllamaClient) Name() string {
	return "ollama"
}

// Model returns the model being used
func (c *OllamaClient) Model() string {
	return c.model
}

// Close cleans up resources
func (c *OllamaClient) Close() error {
	c.client.CloseIdleConnections()
	return nil
}

// Ollama OpenAI-compatible API types (renamed to avoid conflicts with openai.go)
type ollamaOpenAIRequest struct {
	Model       string                `json:"model"`
	Messages    []ollamaOpenAIMessage `json:"messages"`
	Tools       []Tool                `json:"tools,omitempty"`
	Stream      bool                  `json:"stream"`
	Temperature float64               `json:"temperature,omitempty"`
	MaxTokens   int                   `json:"max_tokens,omitempty"`
}

type ollamaOpenAIMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ollamaOpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int                 `json:"index"`
		Message      ollamaOpenAIMessage `json:"message"`
		FinishReason string              `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Chat sends a chat completion request
func (c *OllamaClient) Chat(ctx context.Context, messages []Message) (*Response, error) {
	return c.ChatWithTools(ctx, messages, nil)
}

// ChatWithTools sends a chat completion request using OpenAI-compatible endpoint
func (c *OllamaClient) ChatWithTools(ctx context.Context, messages []Message, tools []Tool) (*Response, error) {
	// Convert messages to OpenAI format
	ollamaMessages := make([]ollamaOpenAIMessage, len(messages))
	for i, m := range messages {
		ollamaMessages[i] = ollamaOpenAIMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCalls:  m.ToolCalls,
			ToolCallID: m.ToolCallID,
		}
	}

	reqBody := ollamaOpenAIRequest{
		Model:       c.model,
		Messages:    ollamaMessages,
		Tools:       tools,
		Stream:      false,
		Temperature: 0.1, // Low temperature for consistent decisions
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Debug: Log the request being sent
	fmt.Printf("[OLLAMA DEBUG] Request (OpenAI-compat): model=%s, tools=%d, messages=%d\n", c.model, len(tools), len(messages))
	if len(tools) > 0 {
		fmt.Printf("[OLLAMA DEBUG] First tool: %s\n", tools[0].Function.Name)
	}

	// Use OpenAI-compatible endpoint
	url := c.endpoint + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ollama") // Ollama ignores this but some proxies need it

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(body))
	}

	// Read body for debugging
	bodyBytes, _ := io.ReadAll(resp.Body)
	fmt.Printf("[OLLAMA DEBUG] Raw response: %s\n", string(bodyBytes)[:min(500, len(bodyBytes))])

	var ollamaResp ollamaOpenAIResponse
	if err := json.Unmarshal(bodyBytes, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract response from choices
	if len(ollamaResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := ollamaResp.Choices[0]

	// Debug: Log parsed response
	fmt.Printf("[OLLAMA DEBUG] Parsed: content_len=%d, tool_calls=%d, finish_reason=%s\n",
		len(choice.Message.Content), len(choice.Message.ToolCalls), choice.FinishReason)

	return &Response{
		Content:      choice.Message.Content,
		ToolCalls:    choice.Message.ToolCalls,
		FinishReason: choice.FinishReason,
		TotalTokens:  ollamaResp.Usage.TotalTokens,
	}, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// IsAvailable checks if Ollama is running and the model is available
func (c *OllamaClient) IsAvailable(ctx context.Context) bool {
	url := c.endpoint + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
