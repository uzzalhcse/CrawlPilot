package interaction

import (
	"context"
	"fmt"
	"time"

	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// WaitExecutor handles wait/sleep interactions
type WaitExecutor struct {
	nodes.BaseNodeExecutor
}

// NewWaitExecutor creates a new wait executor
func NewWaitExecutor() *WaitExecutor {
	return &WaitExecutor{
		BaseNodeExecutor: nodes.BaseNodeExecutor{},
	}
}

// Type returns the node type
func (e *WaitExecutor) Type() models.NodeType {
	return models.NodeTypeWait
}

// Validate validates the node parameters
func (e *WaitExecutor) Validate(params map[string]interface{}) error {
	// Check if this is a selector-based wait or a simple duration wait
	selector := nodes.GetStringParam(params, "selector", "")
	duration := nodes.GetIntParam(params, "duration", 0)
	timeout := nodes.GetIntParam(params, "timeout", 0)

	if selector != "" {
		// Selector-based wait: needs selector and timeout
		if timeout <= 0 {
			return fmt.Errorf("timeout must be positive for wait node with selector")
		}
	} else {
		// Simple duration wait: needs duration
		if duration <= 0 {
			return fmt.Errorf("duration must be positive for wait node without selector")
		}
	}
	return nil
}

// Execute performs the wait action
func (e *WaitExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	selector := nodes.GetStringParam(input.Params, "selector", "")

	interactionEngine := browser.NewInteractionEngine(input.BrowserContext)

	if selector != "" {
		// Wait for selector to be in specific state
		timeout := time.Duration(nodes.GetIntParam(input.Params, "timeout", 30000)) * time.Millisecond
		state := nodes.GetStringParam(input.Params, "state", "visible")

		if err := interactionEngine.WaitForSelector(selector, timeout, state); err != nil {
			return nil, fmt.Errorf("wait for selector '%s' failed: %w", selector, err)
		}

		return &nodes.ExecutionOutput{
			Result: map[string]interface{}{
				"_selector": selector,
				"_state":    state,
				"_timeout":  timeout.Milliseconds(),
			},
		}, nil
	} else {
		// Simple wait/sleep
		duration := time.Duration(nodes.GetIntParam(input.Params, "duration", 0)) * time.Millisecond

		if err := interactionEngine.Wait(duration); err != nil {
			return nil, fmt.Errorf("wait failed: %w", err)
		}

		return &nodes.ExecutionOutput{
			Result: map[string]interface{}{
				"_waited": duration.Milliseconds(),
			},
		}, nil
	}
}
