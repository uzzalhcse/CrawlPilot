package discovery

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// NavigateExecutor handles navigation to URLs
type NavigateExecutor struct {
	nodes.BaseNodeExecutor
}

// NewNavigateExecutor creates a new navigate executor
func NewNavigateExecutor() *NavigateExecutor {
	return &NavigateExecutor{
		BaseNodeExecutor: nodes.BaseNodeExecutor{},
	}
}

// Type returns the node type
func (e *NavigateExecutor) Type() models.NodeType {
	return models.NodeTypeNavigate
}

// Validate validates the node parameters
func (e *NavigateExecutor) Validate(params map[string]interface{}) error {
	url := nodes.GetStringParam(params, "url")
	if url == "" {
		return fmt.Errorf("url is required for navigate node")
	}
	return nil
}

// Execute navigates to a URL
func (e *NavigateExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	targetURL := nodes.GetStringParam(input.Params, "url")

	_, err := input.BrowserContext.Navigate(targetURL)
	if err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}

	return &nodes.ExecutionOutput{
		Result: map[string]interface{}{
			"url": targetURL,
		},
	}, nil
}
