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

// Ollama API types (OpenAI-compatible endpoint)
type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Tools    []Tool              `json:"tools,omitempty"`
	Stream   bool                `json:"stream"`
	Options  *ollamaOptions      `json:"options,omitempty"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type ollamaChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ollamaChatResponse struct {
	Model         string            `json:"model"`
	Message       ollamaChatMessage `json:"message"`
	Done          bool              `json:"done"`
	TotalDuration int64             `json:"total_duration"`
	EvalCount     int               `json:"eval_count"`
}

// Chat sends a chat completion request
func (c *OllamaClient) Chat(ctx context.Context, messages []Message) (*Response, error) {
	return c.ChatWithTools(ctx, messages, nil)
}

// ChatWithTools sends a chat completion request with function calling
func (c *OllamaClient) ChatWithTools(ctx context.Context, messages []Message, tools []Tool) (*Response, error) {
	// Convert messages to Ollama format
	ollamaMessages := make([]ollamaChatMessage, len(messages))
	for i, m := range messages {
		ollamaMessages[i] = ollamaChatMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCalls:  m.ToolCalls,
			ToolCallID: m.ToolCallID,
		}
	}

	reqBody := ollamaChatRequest{
		Model:    c.model,
		Messages: ollamaMessages,
		Tools:    tools,
		Stream:   false,
		Options: &ollamaOptions{
			Temperature: 0.1, // Low temperature for consistent decisions
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.endpoint + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(body))
	}

	var ollamaResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &Response{
		Content:      ollamaResp.Message.Content,
		ToolCalls:    ollamaResp.Message.ToolCalls,
		FinishReason: "stop",
		TotalTokens:  ollamaResp.EvalCount,
	}, nil
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
