package interaction

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// HoverExecutor handles hover interactions
type HoverExecutor struct {
	nodes.BaseNodeExecutor
}

// NewHoverExecutor creates a new hover executor
func NewHoverExecutor() *HoverExecutor {
	return &HoverExecutor{
		BaseNodeExecutor: nodes.BaseNodeExecutor{},
	}
}

// Type returns the node type
func (e *HoverExecutor) Type() models.NodeType {
	return models.NodeTypeHover
}

// Validate validates the node parameters
func (e *HoverExecutor) Validate(params map[string]interface{}) error {
	selector := nodes.GetStringParam(params, "selector")
	if selector == "" {
		return fmt.Errorf("selector is required for hover node")
	}
	return nil
}

// Execute performs the hover action
func (e *HoverExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	selector := nodes.GetStringParam(input.Params, "selector")

	interactionEngine := browser.NewInteractionEngine(input.BrowserContext)
	if err := interactionEngine.Hover(selector); err != nil {
		return nil, fmt.Errorf("hover failed: %w", err)
	}

	return &nodes.ExecutionOutput{
		Result: map[string]interface{}{
			"hovered": selector,
		},
	}, nil
}
