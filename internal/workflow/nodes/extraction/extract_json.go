package extraction

import (
	"context"
	"fmt"

	extraction_engine "github.com/uzzalhcse/crawlify/internal/extraction"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// ExtractJSONExecutor handles JSON extraction from script tags
type ExtractJSONExecutor struct {
	nodes.BaseNodeExecutor
}

// NewExtractJSONExecutor creates a new extract_json executor
func NewExtractJSONExecutor() *ExtractJSONExecutor {
	return &ExtractJSONExecutor{
		BaseNodeExecutor: nodes.BaseNodeExecutor{},
	}
}

// Type returns the node type
func (e *ExtractJSONExecutor) Type() models.NodeType {
	return models.NodeTypeExtractJSON
}

// Validate validates the node parameters
func (e *ExtractJSONExecutor) Validate(params map[string]interface{}) error {
	selector := nodes.GetStringParam(params, "selector")
	if selector == "" {
		return fmt.Errorf("selector is required for extract_json node")
	}
	return nil
}

// Execute extracts JSON from script tags
func (e *ExtractJSONExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	selector := nodes.GetStringParam(input.Params, "selector")

	engine := extraction_engine.NewExtractionEngine(input.BrowserContext.Page)
	data, err := engine.ExtractJSON(selector)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON: %w", err)
	}

	return &nodes.ExecutionOutput{
		Result: data,
	}, nil
}
