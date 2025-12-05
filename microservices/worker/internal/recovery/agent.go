package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/recovery/llm"
	"go.uber.org/zap"
)

// Agent uses AI to analyze errors and suggest recovery actions
// This is used as a fallback when no rules match
type Agent struct {
	provider llm.Provider
	config   *AgentConfig
}

// AgentConfig holds configuration for the AI agent
type AgentConfig struct {
	Enabled     bool
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
}

// NewAgent creates a new AI recovery agent
func NewAgent(provider llm.Provider, config *AgentConfig) *Agent {
	if config == nil {
		config = &AgentConfig{
			Enabled:     true,
			MaxTokens:   500,
			Temperature: 0.1,
			Timeout:     30 * time.Second,
		}
	}

	return &Agent{
		provider: provider,
		config:   config,
	}
}

// Analyze uses AI to analyze an error and suggest a recovery action
func (a *Agent) Analyze(ctx context.Context, err *DetectedError, history []*RecoveryAttempt) (*RecoveryPlan, string, error) {
	if !a.config.Enabled || a.provider == nil {
		return nil, "", fmt.Errorf("AI agent is disabled")
	}

	// Build the prompt
	messages := a.buildMessages(err, history)

	// Get AI response with function calling
	ctx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	resp, err2 := a.provider.ChatWithTools(ctx, messages, llm.DefaultTools())
	if err2 != nil {
		return nil, "", fmt.Errorf("AI analysis failed: %w", err2)
	}

	// Parse the response
	plan, reasoning, err2 := a.parseResponse(resp)
	if err2 != nil {
		return nil, "", fmt.Errorf("failed to parse AI response: %w", err2)
	}

	plan.Source = "ai"

	logger.Info("AI agent suggested recovery action",
		zap.String("action", string(plan.Action)),
		zap.String("reason", plan.Reason),
		zap.String("provider", a.provider.Name()),
		zap.String("model", a.provider.Model()),
	)

	return plan, reasoning, nil
}

// buildMessages constructs the message sequence for the AI
func (a *Agent) buildMessages(err *DetectedError, history []*RecoveryAttempt) []llm.Message {
	messages := []llm.Message{
		{
			Role:    "system",
			Content: llm.RecoverySystemPrompt(),
		},
	}

	// Add history context if available
	if len(history) > 0 {
		historyContext := a.formatHistory(history)
		messages = append(messages, llm.Message{
			Role:    "user",
			Content: "PREVIOUS RECOVERY ATTEMPTS:\n" + historyContext,
		})
	}

	// Add current error context
	errorContext := a.formatError(err)
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: errorContext,
	})

	return messages
}

// formatError formats the error for AI analysis
func (a *Agent) formatError(err *DetectedError) string {
	context := fmt.Sprintf(`CURRENT ERROR:
Pattern: %s (confidence: %.2f)
Domain: %s
URL: %s
Status Code: %d
Error Message: %s
`, err.Pattern, err.Confidence, err.Domain, err.URL, err.StatusCode, truncateContent(err.RawError, 200))

	if err.PageContent != "" {
		context += fmt.Sprintf("\nPage Content (excerpt): %s", truncateContent(err.PageContent, 500))
	}

	context += "\n\nAnalyze this error and call the appropriate recovery function."

	return context
}

// formatHistory formats previous recovery attempts
func (a *Agent) formatHistory(history []*RecoveryAttempt) string {
	if len(history) == 0 {
		return "No previous attempts"
	}

	result := ""
	for i, h := range history {
		if i >= 3 {
			break // Only show last 3 attempts
		}
		result += fmt.Sprintf("- Attempt %d: %s action=%s, success=%t\n",
			i+1, h.DetectedError.Pattern, h.Plan.Action, h.Success)
	}
	return result
}

// parseResponse extracts the recovery plan from AI response
func (a *Agent) parseResponse(resp *llm.Response) (*RecoveryPlan, string, error) {
	// Check for tool calls first (preferred method)
	if len(resp.ToolCalls) > 0 {
		return a.parseToolCall(resp.ToolCalls[0])
	}

	// Fallback: try to parse content as JSON
	if resp.Content != "" {
		return a.parseContentAsAction(resp.Content)
	}

	return nil, "", fmt.Errorf("no action in AI response")
}

// parseToolCall extracts action from a tool call
func (a *Agent) parseToolCall(tc llm.ToolCall) (*RecoveryPlan, string, error) {
	plan := &RecoveryPlan{
		Params:      make(map[string]interface{}),
		ShouldRetry: true,
	}

	// Parse action type from function name
	switch tc.Function.Name {
	case "switch_proxy":
		plan.Action = ActionSwitchProxy
	case "add_delay":
		plan.Action = ActionAddDelay
	case "skip_domain":
		plan.Action = ActionSkipDomain
	case "send_to_dlq":
		plan.Action = ActionSendToDLQ
		plan.ShouldRetry = false
	case "retry":
		plan.Action = ActionRetry
	default:
		return nil, "", fmt.Errorf("unknown action: %s", tc.Function.Name)
	}

	// Parse arguments
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return nil, "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	plan.Params = args
	plan.Reason = fmt.Sprintf("%v", args["reason"])

	// Handle action-specific parameters
	switch plan.Action {
	case ActionAddDelay:
		if seconds, ok := args["seconds"].(float64); ok {
			plan.RetryDelay = time.Duration(seconds) * time.Second
		} else {
			plan.RetryDelay = 30 * time.Second
		}
	case ActionSkipDomain:
		if minutes, ok := args["duration_minutes"].(float64); ok {
			plan.Params["duration"] = time.Duration(minutes) * time.Minute
		}
	}

	// Extract reasoning for learning
	reasoning := ""
	if r, ok := args["reason"].(string); ok {
		reasoning = r
	}

	return plan, reasoning, nil
}

// parseContentAsAction tries to parse plain text response as an action
func (a *Agent) parseContentAsAction(content string) (*RecoveryPlan, string, error) {
	// Try JSON parsing
	var result struct {
		Action string                 `json:"action"`
		Params map[string]interface{} `json:"params"`
		Reason string                 `json:"reason"`
	}

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// If not JSON, default to retry
		return &RecoveryPlan{
			Action:      ActionRetry,
			Params:      map[string]interface{}{},
			Reason:      "AI suggested retry (unparseable response)",
			ShouldRetry: true,
			RetryDelay:  5 * time.Second,
		}, content, nil
	}

	plan := &RecoveryPlan{
		Action:      ActionType(result.Action),
		Params:      result.Params,
		Reason:      result.Reason,
		ShouldRetry: result.Action != "send_to_dlq",
	}

	return plan, result.Reason, nil
}

// Close cleans up resources
func (a *Agent) Close() error {
	if a.provider != nil {
		return a.provider.Close()
	}
	return nil
}

// IsAvailable checks if the AI provider is available
func (a *Agent) IsAvailable(ctx context.Context) bool {
	if a.provider == nil {
		return false
	}

	// Type assertion for availability check
	type availabilityChecker interface {
		IsAvailable(ctx context.Context) bool
	}

	if checker, ok := a.provider.(availabilityChecker); ok {
		return checker.IsAvailable(ctx)
	}

	return true // Assume available if no check method
}
