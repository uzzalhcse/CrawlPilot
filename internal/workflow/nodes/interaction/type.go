package interaction

import (
	"context"
	"fmt"
	"time"

	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// TypeExecutor handles text input interactions
type TypeExecutor struct {
	nodes.BaseNodeExecutor
}

// NewTypeExecutor creates a new type executor
func NewTypeExecutor() *TypeExecutor {
	return &TypeExecutor{
		BaseNodeExecutor: nodes.BaseNodeExecutor{},
	}
}

// Type returns the node type
func (e *TypeExecutor) Type() models.NodeType {
	return models.NodeTypeType
}

// Validate validates the node parameters
func (e *TypeExecutor) Validate(params map[string]interface{}) error {
	selector := nodes.GetStringParam(params, "selector")
	if selector == "" {
		return fmt.Errorf("selector is required for type node")
	}
	text := nodes.GetStringParam(params, "text")
	if text == "" {
		return fmt.Errorf("text is required for type node")
	}
	return nil
}

// Execute performs the type action
func (e *TypeExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	selector := nodes.GetStringParam(input.Params, "selector")
	text := nodes.GetStringParam(input.Params, "text")
	delay := time.Duration(nodes.GetIntParam(input.Params, "delay", 0)) * time.Millisecond

	interactionEngine := browser.NewInteractionEngine(input.BrowserContext)
	if err := interactionEngine.Type(selector, text, delay); err != nil {
		return nil, fmt.Errorf("type failed: %w", err)
	}

	return &nodes.ExecutionOutput{
		Result: map[string]interface{}{
			"typed":    true,
			"selector": selector,
		},
	}, nil
}
