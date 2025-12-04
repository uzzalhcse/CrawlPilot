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
	// URL is optional - can be provided via params or will use URL from execution context
	return nil
}

// Execute navigates to a URL
func (e *NavigateExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	targetURL := nodes.GetStringParam(input.Params, "url")

	// If no URL in params, get it from execution context (from URLQueueItem)
	if targetURL == "" {
		if input.URLItem != nil {
			targetURL = input.URLItem.URL
		}
	}

	if targetURL == "" {
		return nil, fmt.Errorf("no URL provided: specify 'url' param or use URL from queue")
	}

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
