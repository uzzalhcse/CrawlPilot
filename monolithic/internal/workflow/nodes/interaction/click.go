package interaction

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// ClickExecutor handles click interactions
type ClickExecutor struct {
	nodes.BaseNodeExecutor
}

// NewClickExecutor creates a new click executor
func NewClickExecutor() *ClickExecutor {
	return &ClickExecutor{
		BaseNodeExecutor: nodes.BaseNodeExecutor{},
	}
}

// Type returns the node type
func (e *ClickExecutor) Type() models.NodeType {
	return models.NodeTypeClick
}

// Validate validates the node parameters
func (e *ClickExecutor) Validate(params map[string]interface{}) error {
	selector := nodes.GetStringParam(params, "selector")
	if selector == "" {
		return fmt.Errorf("selector is required for click node")
	}
	return nil
}

// Execute performs the click action
func (e *ClickExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	selector := nodes.GetStringParam(input.Params, "selector")

	interactionEngine := browser.NewInteractionEngine(input.BrowserContext)
	if err := interactionEngine.Click(selector); err != nil {
		return nil, fmt.Errorf("click failed: %w", err)
	}

	return &nodes.ExecutionOutput{
		Result: map[string]interface{}{
			"clicked": selector,
		},
	}, nil
}
