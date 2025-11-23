package discovery

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/internal/extraction"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// ExtractLinksExecutor handles link extraction
type ExtractLinksExecutor struct {
	nodes.BaseNodeExecutor
}

// NewExtractLinksExecutor creates a new extract_links executor
func NewExtractLinksExecutor() *ExtractLinksExecutor {
	return &ExtractLinksExecutor{
		BaseNodeExecutor: nodes.BaseNodeExecutor{},
	}
}

// Type returns the node type
func (e *ExtractLinksExecutor) Type() models.NodeType {
	return models.NodeTypeExtractLinks
}

// Validate validates the node parameters
func (e *ExtractLinksExecutor) Validate(params map[string]interface{}) error {
	// selector is optional, defaults to "a"
	return nil
}

// Execute extracts links from the page
func (e *ExtractLinksExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	selector := nodes.GetStringParam(input.Params, "selector", "a")
	limit := nodes.GetIntParam(input.Params, "limit", 0)

	engine := extraction.NewExtractionEngine(input.BrowserContext.Page)
	links, err := engine.ExtractLinks(selector, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to extract links: %w", err)
	}

	return &nodes.ExecutionOutput{
		Result:         links,
		DiscoveredURLs: links,
		Metadata: map[string]interface{}{
			"count": len(links),
		},
	}, nil
}
