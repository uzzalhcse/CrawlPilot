// Package llm provides an abstraction layer for LLM providers.
// Supports Ollama (Qwen2.5) and OpenAI with function calling.
package llm

import (
	"context"
	"encoding/json"
)

// Provider is the interface for LLM providers
type Provider interface {
	// Chat sends a chat completion request
	Chat(ctx context.Context, messages []Message) (*Response, error)

	// ChatWithTools sends a chat completion request with function calling
	ChatWithTools(ctx context.Context, messages []Message, tools []Tool) (*Response, error)

	// Name returns the provider name
	Name() string

	// Model returns the model being used
	Model() string

	// Close cleans up resources
	Close() error
}

// Message represents a chat message
type Message struct {
	Role       string     `json:"role"` // system, user, assistant, tool
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall represents a function call made by the model
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // function
	Function FunctionCall `json:"function"`
}

// FunctionCall represents the function being called
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// Tool represents a function that can be called
type Tool struct {
	Type     string   `json:"type"` // function
	Function Function `json:"function"`
}

// Function describes a callable function
type Function struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// Response represents a chat completion response
type Response struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string     `json:"finish_reason"`
	TotalTokens  int        `json:"total_tokens"`
}

// Config holds configuration for LLM providers
type Config struct {
	Provider string `yaml:"provider"` // ollama, openai
	Model    string `yaml:"model"`    // qwen2.5, gpt-4o-mini
	Endpoint string `yaml:"endpoint"` // http://localhost:11434
	APIKey   string `yaml:"api_key"`  // For OpenAI
	Timeout  int    `yaml:"timeout"`  // Seconds
}

// ParseFunctionArgs parses JSON arguments from a tool call
func ParseFunctionArgs(args string, v interface{}) error {
	return json.Unmarshal([]byte(args), v)
}

// DefaultTools returns the recovery action tools for function calling
func DefaultTools() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: Function{
				Name:        "switch_proxy",
				Description: "Switch to a different proxy server when current one is blocked or rate limited",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"reason": {
							"type": "string",
							"description": "Why switching proxy is needed"
						},
						"prefer_country": {
							"type": "string",
							"description": "Preferred proxy country code (e.g., US, JP)"
						}
					},
					"required": ["reason"]
				}`),
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "add_delay",
				Description: "Add a delay before retrying when rate limited",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"seconds": {
							"type": "integer",
							"description": "Number of seconds to wait before retry"
						},
						"reason": {
							"type": "string",
							"description": "Why delay is needed"
						}
					},
					"required": ["seconds", "reason"]
				}`),
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "skip_domain",
				Description: "Skip this domain temporarily when consistently failing",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"duration_minutes": {
							"type": "integer",
							"description": "Minutes to skip this domain"
						},
						"reason": {
							"type": "string",
							"description": "Why domain should be skipped"
						}
					},
					"required": ["duration_minutes", "reason"]
				}`),
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "send_to_dlq",
				Description: "Send task to dead letter queue for manual review (permanent failures)",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"reason": {
							"type": "string",
							"description": "Why task cannot be recovered automatically"
						},
						"category": {
							"type": "string",
							"enum": ["captcha", "auth_required", "layout_changed", "permanent_block", "other"],
							"description": "Category of permanent failure"
						}
					},
					"required": ["reason", "category"]
				}`),
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "retry",
				Description: "Simply retry the task with optional modifications",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"clear_cookies": {
							"type": "boolean",
							"description": "Clear browser cookies before retry"
						},
						"reason": {
							"type": "string",
							"description": "Why retry is appropriate"
						}
					},
					"required": ["reason"]
				}`),
			},
		},
	}
}

// RecoverySystemPrompt returns the system prompt for recovery analysis
func RecoverySystemPrompt() string {
	return `You are an error recovery agent for a web scraping system.

Your job is to analyze errors and decide the best recovery action.

CONTEXT:
- You receive error details including pattern, status code, domain, and page content
- You must choose ONE of the available functions to handle the error
- Your decisions affect system performance and reliability

DECISION GUIDELINES:

1. BLOCKED or ACCESS_DENIED (403, "forbidden", "blocked"):
   → Call switch_proxy (IP is likely blocked)

2. RATE_LIMITED (429, "too many requests"):
   → Call add_delay with appropriate seconds (start with 30, increase if repeated)

3. CAPTCHA detected:
   → Call send_to_dlq with category="captcha" (cannot solve automatically)

4. AUTH_REQUIRED (401, "login"):
   → Call send_to_dlq with category="auth_required"

5. TIMEOUT or CONNECTION errors:
   → Call retry (transient error) or switch_proxy if persistent

6. LAYOUT_CHANGED (selector not found):
   → Call send_to_dlq with category="layout_changed" (needs human review)

7. SERVER_ERROR (500, 502, 503):
   → Call retry with clear_cookies=false (server issue, will likely resolve)

8. REPEATED_FAILURES on same domain:
   → Call skip_domain with duration_minutes based on severity

IMPORTANT:
- Always provide clear reasoning in the "reason" parameter
- Consider error patterns and history when making decisions
- Prefer less disruptive actions when possible (retry < switch_proxy < skip_domain < send_to_dlq)`
}
