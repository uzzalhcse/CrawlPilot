package error_recovery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/uzzalhcse/crawlify/internal/logger"
	"go.uber.org/zap"
)

// AIClient interface for AI providers
type AIClient interface {
	GenerateText(ctx context.Context, prompt string) (string, error)
	Close() error
}

// AIReasoningEngine uses AI to solve complex problems when rules fail
type AIReasoningEngine struct {
	client         AIClient
	learningEngine *LearningEngine
	enabled        bool
}

// NewAIReasoningEngine creates a new AI reasoning engine
func NewAIReasoningEngine(client AIClient, learningEngine *LearningEngine, enabled bool) *AIReasoningEngine {
	return &AIReasoningEngine{
		client:         client,
		learningEngine: learningEngine,
		enabled:        enabled,
	}
}

// ReasonAndSolve uses AI to reason about the error and provide a solution
func (a *AIReasoningEngine) ReasonAndSolve(ctx context.Context, execCtx *ExecutionContext, err error) (*Solution, error) {
	if !a.enabled || a.client == nil {
		return nil, fmt.Errorf("AI reasoning is disabled or no client available")
	}

	logger.Warn("🤖 Activating AI Reasoning (expensive)",
		zap.String("url", execCtx.URL),
		zap.String("error", err.Error()))

	// Build reasoning prompt
	prompt := a.buildReasoningPrompt(execCtx, err)

	// Call AI
	response, aiErr := a.client.GenerateText(ctx, prompt)
	if aiErr != nil {
		logger.Error("AI reasoning failed", zap.Error(aiErr))
		return nil, fmt.Errorf("AI reasoning failed: %w", aiErr)
	}

	// Parse AI response
	solution, parseErr := a.parseAIResponse(response)
	if parseErr != nil {
		logger.Error("Failed to parse AI response", zap.Error(parseErr))
		return nil, fmt.Errorf("failed to parse AI response: %w", parseErr)
	}

	solution.Type = "ai"

	// Try to learn from this solution
	if a.learningEngine != nil {
		if rule := a.learningEngine.ConvertToContextAwareRule(solution, execCtx); rule != nil {
			logger.Info("✅ Created context-aware rule from AI solution",
				zap.String("rule", rule.Name))
		}
	}

	return solution, nil
}

// buildReasoningPrompt builds the prompt for AI reasoning
func (a *AIReasoningEngine) buildReasoningPrompt(ctx *ExecutionContext, err error) string {
	return fmt.Sprintf(`You are a web scraping expert analyzing a failure.

Context:
- URL: %s
- Domain: %s
- Error: %s
- Status Code: %d
- Response Body (first 500 chars): %s
- Failed Rules Tried: %v

Available Actions:
- enable_stealth: Enable stealth mode (parameters: level [standard, moderate, aggressive])
- rotate_proxy: Rotate to a different proxy
- adjust_timeout: Increase timeout (parameters: multiplier [1.5, 2.0, 3.0])
- reduce_workers: Reduce concurrent workers (parameters: count)
- add_delay: Add delay between requests (parameters: duration in ms)
- wait: Wait before retrying (parameters: duration in seconds)

Think step by step:
1. What type of protection or issue is this?
2. Why did the predefined rules fail?
3. What specific strategy should work?
4. What domain-specific parameters should be used?

Respond with a JSON solution in this exact format:
{
  "actions": [
    {
      "type": "action_name",
      "parameters": {
        "param_name": "param_value"
      }
    }
  ],
  "confidence": 0.85,
  "reasoning": "brief explanation"
}`,
		ctx.URL,
		ctx.Domain,
		err.Error(),
		ctx.Response.StatusCode,
		truncate(ctx.Response.Body, 500),
		ctx.FailedRules,
	)
}

// parseAIResponse parses the AI response into a Solution
func (a *AIReasoningEngine) parseAIResponse(response string) (*Solution, error) {
	var parsed struct {
		Actions    []Action `json:"actions"`
		Confidence float64  `json:"confidence"`
		Reasoning  string   `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return nil, err
	}

	return &Solution{
		RuleName:   "ai_generated",
		Actions:    parsed.Actions,
		Confidence: parsed.Confidence,
		Context: map[string]interface{}{
			"reasoning": parsed.Reasoning,
		},
	}, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
