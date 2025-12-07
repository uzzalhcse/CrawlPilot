package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/recovery/llm"
	"go.uber.org/zap"
)

// ProbeAgent analyzes probe failures and suggests workflow fixes
type ProbeAgent struct {
	provider llm.Provider
	config   *ProbeAgentConfig
}

// ProbeAgentConfig holds configuration for the probe agent
type ProbeAgentConfig struct {
	Enabled             bool
	ConfidenceThreshold float64 // Minimum confidence to auto-apply fix (default 0.8)
	MaxTokens           int
	Temperature         float64
	Timeout             time.Duration
}

// DefaultProbeAgentConfig returns default probe agent configuration
func DefaultProbeAgentConfig() *ProbeAgentConfig {
	return &ProbeAgentConfig{
		Enabled:             true,
		ConfidenceThreshold: 0.8,
		MaxTokens:           1000,
		Temperature:         0.1,
		Timeout:             60 * time.Second,
	}
}

// NewProbeAgent creates a new probe agent
func NewProbeAgent(provider llm.Provider, config *ProbeAgentConfig) *ProbeAgent {
	if config == nil {
		config = DefaultProbeAgentConfig()
	}
	return &ProbeAgent{
		provider: provider,
		config:   config,
	}
}

// ProbeFixPlan contains AI-suggested workflow changes
type ProbeFixPlan struct {
	WorkflowID  string                 `json:"workflow_id"`
	NodeID      string                 `json:"node_id"`
	FixType     string                 `json:"fix_type"` // "update_selector", "skip_node", "escalate"
	Confidence  float64                `json:"confidence"`
	OldSelector string                 `json:"old_selector,omitempty"`
	NewSelector string                 `json:"new_selector,omitempty"`
	Reasoning   string                 `json:"reasoning"`
	ShouldApply bool                   `json:"should_apply"` // Confidence >= threshold
	Params      map[string]interface{} `json:"params,omitempty"`
}

// AnalyzeProbeFailure analyzes failed probe and suggests workflow fixes
func (pa *ProbeAgent) AnalyzeProbeFailure(
	ctx context.Context,
	result *models.ProbeResult,
	baseline *models.ProbeResult,
	deviations *DeviationSummary,
) (*ProbeFixPlan, error) {
	if !pa.config.Enabled || pa.provider == nil {
		return nil, fmt.Errorf("probe agent is disabled")
	}

	// Build messages for AI
	messages := pa.buildMessages(result, deviations)

	// Get AI response with function calling
	ctx, cancel := context.WithTimeout(ctx, pa.config.Timeout)
	defer cancel()

	resp, err := pa.provider.ChatWithTools(ctx, messages, ProbeFixTools())
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	// Parse response
	plan, err := pa.parseResponse(resp, result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	plan.ShouldApply = plan.Confidence >= pa.config.ConfidenceThreshold

	logger.Info("Probe agent analysis complete",
		zap.String("workflow_id", result.WorkflowID),
		zap.String("fix_type", plan.FixType),
		zap.Float64("confidence", plan.Confidence),
		zap.Bool("should_apply", plan.ShouldApply),
	)

	return plan, nil
}

// buildMessages creates the prompt for AI analysis
func (pa *ProbeAgent) buildMessages(result *models.ProbeResult, deviations *DeviationSummary) []llm.Message {
	messages := []llm.Message{
		{
			Role:    "system",
			Content: ProbeFixSystemPrompt(),
		},
	}

	// Add probe failure context
	context := pa.formatProbeContext(result, deviations)
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: context,
	})

	return messages
}

// formatProbeContext formats probe failure details for AI
func (pa *ProbeAgent) formatProbeContext(result *models.ProbeResult, deviations *DeviationSummary) string {
	ctx := fmt.Sprintf(`PROBE FAILURE ANALYSIS

Workflow ID: %s
Overall Status: %s
Duration: %dms

`, result.WorkflowID, result.Status, result.Duration)

	// Add failed nodes
	ctx += "FAILED NODES:\n"
	for _, phase := range result.Phases {
		for _, node := range phase.Nodes {
			if node.Status == "failed" {
				ctx += fmt.Sprintf(`
Node: %s (%s)
  Type: %s
  Selector: %s
  Error: %s
`, node.NodeName, node.NodeID, node.NodeType, node.Selector, node.Error)

				// Add selector context if available
				if node.SelectorContext != "" {
					ctx += fmt.Sprintf("  HTML Context: %s\n", truncateContent(node.SelectorContext, 300))
				}

				// Add snapshot info
				if node.Snapshot != nil {
					ctx += fmt.Sprintf("  DOM Path: %s\n", node.Snapshot.DOMPath)
					if node.Snapshot.DOMPath != "" {
						// Read first 500 chars of DOM for context
						if domContent, err := os.ReadFile(node.Snapshot.DOMPath); err == nil {
							ctx += fmt.Sprintf("  DOM Excerpt: %s\n", truncateContent(string(domContent), 500))
						}
					}
				}
			}
		}
	}

	// Add deviation summary if available
	if deviations != nil && len(deviations.Deviations) > 0 {
		ctx += fmt.Sprintf(`
BASELINE COMPARISON:
Total Deviations: %d
New Failures: %d
Critical: %d, Major: %d, Minor: %d

`, deviations.TotalDeviations, deviations.NewFailures,
			deviations.CriticalCount, deviations.MajorCount, deviations.MinorCount)

		for _, d := range deviations.Deviations {
			if d.Severity == "critical" || d.Severity == "major" {
				ctx += fmt.Sprintf("- %s: %s (expected %d, got %d, %.1f%% change)\n",
					d.NodeName, d.DeviationType, d.ExpectedValue, d.ActualValue, d.PercentageChange)
			}
		}
	}

	ctx += "\nAnalyze the probe failure and call the appropriate function to suggest a fix."

	return ctx
}

// parseResponse extracts the fix plan from AI response
func (pa *ProbeAgent) parseResponse(resp *llm.Response, result *models.ProbeResult) (*ProbeFixPlan, error) {
	plan := &ProbeFixPlan{
		WorkflowID: result.WorkflowID,
		Params:     make(map[string]interface{}),
	}

	// Check for tool calls
	if len(resp.ToolCalls) > 0 {
		tc := resp.ToolCalls[0]
		return pa.parseToolCall(tc, plan)
	}

	// Fallback: try to parse content
	if resp.Content != "" {
		plan.FixType = "escalate"
		plan.Confidence = 0.3
		plan.Reasoning = resp.Content
		return plan, nil
	}

	return nil, fmt.Errorf("no action in AI response")
}

// parseToolCall extracts fix plan from tool call
func (pa *ProbeAgent) parseToolCall(tc llm.ToolCall, plan *ProbeFixPlan) (*ProbeFixPlan, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	plan.Params = args

	switch tc.Function.Name {
	case "update_selector":
		plan.FixType = "update_selector"
		plan.NodeID = getString(args, "node_id")
		plan.OldSelector = getString(args, "old_selector")
		plan.NewSelector = getString(args, "new_selector")
		plan.Confidence = getFloat(args, "confidence")
		plan.Reasoning = getString(args, "reason")

	case "skip_node":
		plan.FixType = "skip_node"
		plan.NodeID = getString(args, "node_id")
		plan.Confidence = 0.9 // Skipping is a safe action
		plan.Reasoning = getString(args, "reason")

	case "escalate_to_human":
		plan.FixType = "escalate"
		plan.Confidence = 0.0 // Always escalate
		plan.Reasoning = getString(args, "reason")

	default:
		return nil, fmt.Errorf("unknown action: %s", tc.Function.Name)
	}

	return plan, nil
}

// IsAvailable checks if the probe agent is available
func (pa *ProbeAgent) IsAvailable(ctx context.Context) bool {
	if pa.provider == nil {
		return false
	}

	type availabilityChecker interface {
		IsAvailable(ctx context.Context) bool
	}

	if checker, ok := pa.provider.(availabilityChecker); ok {
		return checker.IsAvailable(ctx)
	}

	return true
}

// Close cleans up resources
func (pa *ProbeAgent) Close() error {
	if pa.provider != nil {
		return pa.provider.Close()
	}
	return nil
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0.0
}

// ProbeFixTools returns the AI tools for workflow fixes
func ProbeFixTools() []llm.Tool {
	return []llm.Tool{
		{
			Type: "function",
			Function: llm.Function{
				Name:        "update_selector",
				Description: "Update a CSS selector in the workflow when the old one no longer matches",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"node_id": {
							"type": "string",
							"description": "ID of the node to update"
						},
						"old_selector": {
							"type": "string",
							"description": "The current selector that is failing"
						},
						"new_selector": {
							"type": "string",
							"description": "The new CSS selector to use"
						},
						"confidence": {
							"type": "number",
							"description": "Confidence level 0.0-1.0 that this fix will work"
						},
						"reason": {
							"type": "string",
							"description": "Explanation of why this selector change should work"
						}
					},
					"required": ["node_id", "old_selector", "new_selector", "confidence", "reason"]
				}`),
			},
		},
		{
			Type: "function",
			Function: llm.Function{
				Name:        "skip_node",
				Description: "Temporarily disable a broken node that cannot be fixed automatically",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"node_id": {
							"type": "string",
							"description": "ID of the node to skip"
						},
						"reason": {
							"type": "string",
							"description": "Why this node should be skipped"
						}
					},
					"required": ["node_id", "reason"]
				}`),
			},
		},
		{
			Type: "function",
			Function: llm.Function{
				Name:        "escalate_to_human",
				Description: "Cannot determine a fix automatically, requires human review",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"reason": {
							"type": "string",
							"description": "Detailed explanation of why human review is needed"
						}
					},
					"required": ["reason"]
				}`),
			},
		},
	}
}

// ProbeFixSystemPrompt returns the system prompt for probe fix analysis
func ProbeFixSystemPrompt() string {
	return `You are a web scraping workflow repair agent.

Your job is to analyze probe failures and suggest fixes for broken selectors.

CONTEXT:
- A probe execution tested a workflow against a website
- Some nodes failed, likely due to website structure changes
- You receive the failed node details, selector, error, and DOM context

DECISION GUIDELINES:

1. SELECTOR NOT FOUND:
   - Analyze the DOM context to find a working alternative selector
   - Look for similar class names, IDs, or structural patterns
   - Call update_selector with the new selector and your confidence level

2. ELEMENT COUNT DROP:
   - If elements are missing, the page structure may have changed
   - Suggest a new selector or escalate if too complex

3. CANNOT DETERMINE FIX:
   - If the DOM is too different or you're unsure, call escalate_to_human
   - Provide detailed reasoning for human review

4. TEMPORARY WORKAROUND:
   - If a node is blocking the workflow but not critical, call skip_node
   - Only use this if the node failure won't affect downstream nodes

CONFIDENCE LEVELS:
- 0.9-1.0: Very confident (selector is clearly identifiable)
- 0.7-0.9: Confident (good match but not certain)
- 0.5-0.7: Moderate (educated guess)
- 0.0-0.5: Low confidence (escalate to human)

IMPORTANT:
- Always analyze the DOM context carefully
- Prefer specific selectors over generic ones
- Consider if parent/sibling elements changed
- If unsure, escalate rather than apply a wrong fix`
}
