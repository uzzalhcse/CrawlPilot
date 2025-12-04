package interaction

import (
	"context"
	"fmt"
	"time"

	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// WaitForExecutor handles waiting for selectors
type WaitForExecutor struct {
	nodes.BaseNodeExecutor
}

// NewWaitForExecutor creates a new wait_for executor
func NewWaitForExecutor() *WaitForExecutor {
	return &WaitForExecutor{
		BaseNodeExecutor: nodes.BaseNodeExecutor{},
	}
}

// Type returns the node type
func (e *WaitForExecutor) Type() models.NodeType {
	return models.NodeTypeWaitFor
}

// Validate validates the node parameters
func (e *WaitForExecutor) Validate(params map[string]interface{}) error {
	selector := nodes.GetStringParam(params, "selector")
	if selector == "" {
		return fmt.Errorf("selector is required for wait_for node")
	}
	return nil
}

// Execute performs the wait for selector action
func (e *WaitForExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	selector := nodes.GetStringParam(input.Params, "selector")
	timeout := time.Duration(nodes.GetIntParam(input.Params, "timeout", 30000)) * time.Millisecond
	state := nodes.GetStringParam(input.Params, "state", "visible")

	interactionEngine := browser.NewInteractionEngine(input.BrowserContext)
	if err := interactionEngine.WaitForSelector(selector, timeout, state); err != nil {
		return nil, fmt.Errorf("wait_for failed: %w", err)
	}

	return &nodes.ExecutionOutput{
		Result: map[string]interface{}{
			"selector": selector,
			"state":    state,
		},
	}, nil
}
