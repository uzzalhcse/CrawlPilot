package interaction

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// ScrollExecutor handles scroll interactions
type ScrollExecutor struct {
	nodes.BaseNodeExecutor
}

// NewScrollExecutor creates a new scroll executor
func NewScrollExecutor() *ScrollExecutor {
	return &ScrollExecutor{
		BaseNodeExecutor: nodes.BaseNodeExecutor{},
	}
}

// Type returns the node type
func (e *ScrollExecutor) Type() models.NodeType {
	return models.NodeTypeScroll
}

// Validate validates the node parameters
func (e *ScrollExecutor) Validate(params map[string]interface{}) error {
	// x and y are optional, default to 0
	return nil
}

// Execute performs the scroll action
func (e *ScrollExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	x := nodes.GetIntParam(input.Params, "x", 0)
	y := nodes.GetIntParam(input.Params, "y", 0)

	interactionEngine := browser.NewInteractionEngine(input.BrowserContext)
	if err := interactionEngine.Scroll(x, y); err != nil {
		return nil, fmt.Errorf("scroll failed: %w", err)
	}

	return &nodes.ExecutionOutput{
		Result: map[string]interface{}{
			"scrolled": true,
			"x":        x,
			"y":        y,
		},
	}, nil
}
