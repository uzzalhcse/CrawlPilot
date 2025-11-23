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
	duration := nodes.GetIntParam(params, "duration", 0)
	if duration <= 0 {
		return fmt.Errorf("duration must be positive for wait node")
	}
	return nil
}

// Execute performs the wait action
func (e *WaitExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	duration := time.Duration(nodes.GetIntParam(input.Params, "duration")) * time.Millisecond

	interactionEngine := browser.NewInteractionEngine(input.BrowserContext)
	if err := interactionEngine.Wait(duration); err != nil {
		return nil, fmt.Errorf("wait failed: %w", err)
	}

	return &nodes.ExecutionOutput{
		Result: map[string]interface{}{
			"waited": duration.Milliseconds(),
		},
	}, nil
}
