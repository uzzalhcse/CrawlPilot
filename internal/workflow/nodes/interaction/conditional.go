package interaction

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// ConditionalExecutor executes nodes based on conditions
type ConditionalExecutor struct {
	nodes.BaseNodeExecutor
	registry NodeRegistry
}

// NewConditionalExecutor creates a new conditional executor
func NewConditionalExecutor(registry NodeRegistry) *ConditionalExecutor {
	return &ConditionalExecutor{
		BaseNodeExecutor: nodes.BaseNodeExecutor{},
		registry:         registry,
	}
}

// Type returns the node type
func (e *ConditionalExecutor) Type() models.NodeType {
	return models.NodeTypeConditional
}

// Validate validates the conditional configuration
func (e *ConditionalExecutor) Validate(params map[string]interface{}) error {
	condition := nodes.GetStringParam(params, "condition")
	if condition == "" {
		return fmt.Errorf("condition is required")
	}

	// Must have either 'then' or both 'then' and 'else'
	thenNode := nodes.GetMapParam(params, "then")
	if len(thenNode) == 0 {
		return fmt.Errorf("'then' node is required")
	}

	return nil
}

// Execute evaluates condition and executes appropriate branch
func (e *ConditionalExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	condition := nodes.GetStringParam(input.Params, "condition")

	// Evaluate condition
	conditionMet, err := e.evaluateCondition(condition, input)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate condition: %w", err)
	}

	// Determine which branch to execute
	var branchNode map[string]interface{}
	var branchName string

	if conditionMet {
		branchNode = nodes.GetMapParam(input.Params, "then")
		branchName = "then"
	} else {
		branchNode = nodes.GetMapParam(input.Params, "else")
		branchName = "else"
		if len(branchNode) == 0 {
			// No else branch, just return success
			return &nodes.ExecutionOutput{
				Result: map[string]interface{}{
					"condition_met": false,
					"branch":        "none",
				},
			}, nil
		}
	}

	// Execute branch
	branchType := models.NodeType(nodes.GetStringParam(branchNode, "type"))
	branchParams := nodes.GetMapParam(branchNode, "params")

	executor, err := e.registry.Get(branchType)
	if err != nil {
		return nil, fmt.Errorf("failed to get executor for %s branch type %s: %w", branchName, branchType, err)
	}

	if err := executor.Validate(branchParams); err != nil {
		return nil, fmt.Errorf("%s branch validation failed: %w", branchName, err)
	}

	branchInput := &nodes.ExecutionInput{
		BrowserContext:   input.BrowserContext,
		ExecutionContext: input.ExecutionContext,
		Params:           branchParams,
		URLItem:          input.URLItem,
		ExecutionID:      input.ExecutionID,
	}

	output, err := executor.Execute(ctx, branchInput)
	if err != nil {
		return nil, fmt.Errorf("%s branch execution failed: %w", branchName, err)
	}

	// Add metadata about which branch was taken
	if output.Metadata == nil {
		output.Metadata = make(map[string]interface{})
	}
	output.Metadata["condition_met"] = conditionMet
	output.Metadata["branch_executed"] = branchName

	return output, nil
}

// evaluateCondition evaluates the condition string
func (e *ConditionalExecutor) evaluateCondition(condition string, input *nodes.ExecutionInput) (bool, error) {
	// Simple condition evaluations
	switch {
	case condition == "element_exists":
		// Check if selector exists
		selector := nodes.GetStringParam(input.Params, "selector")
		if selector == "" {
			return false, fmt.Errorf("selector required for element_exists condition")
		}
		locator := input.BrowserContext.Page.Locator(selector)
		count, err := locator.Count()
		if err != nil {
			return false, err
		}
		return count > 0, nil

	case condition == "element_visible":
		selector := nodes.GetStringParam(input.Params, "selector")
		if selector == "" {
			return false, fmt.Errorf("selector required for element_visible condition")
		}
		visible, err := input.BrowserContext.Page.IsVisible(selector)
		if err != nil {
			return false, nil // Element not found = not visible
		}
		return visible, nil

	case condition == "url_matches":
		pattern := nodes.GetStringParam(input.Params, "pattern")
		if pattern == "" {
			return false, fmt.Errorf("pattern required for url_matches condition")
		}
		currentURL := input.BrowserContext.Page.URL()
		// Simple contains check (could be enhanced with regex)
		return containsString(currentURL, pattern), nil

	case condition == "context_value_equals":
		key := nodes.GetStringParam(input.Params, "key")
		expectedValue := input.Params["value"]
		if key == "" {
			return false, fmt.Errorf("key required for context_value_equals condition")
		}
		actualValue, _ := input.ExecutionContext.Get(key)
		return actualValue == expectedValue, nil

	default:
		return false, fmt.Errorf("unknown condition type: %s", condition)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
