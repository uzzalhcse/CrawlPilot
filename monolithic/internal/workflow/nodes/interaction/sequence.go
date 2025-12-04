package interaction

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// SequenceExecutor executes a sequence of nodes in order
type SequenceExecutor struct {
	nodes.BaseNodeExecutor
	registry NodeRegistry // Will be injected
}

// NodeRegistry interface for getting executors (to avoid circular dependency)
type NodeRegistry interface {
	Get(nodeType models.NodeType) (nodes.NodeExecutor, error)
}

// NewSequenceExecutor creates a new sequence executor
func NewSequenceExecutor(registry NodeRegistry) *SequenceExecutor {
	return &SequenceExecutor{
		BaseNodeExecutor: nodes.BaseNodeExecutor{},
		registry:         registry,
	}
}

// Type returns the node type
func (e *SequenceExecutor) Type() models.NodeType {
	return models.NodeTypeSequence
}

// Validate validates the sequence configuration
func (e *SequenceExecutor) Validate(params map[string]interface{}) error {
	steps := nodes.GetArrayParam(params, "steps")
	if len(steps) == 0 {
		return fmt.Errorf("sequence must have at least one step")
	}

	// Validate each step has required fields
	for i, stepRaw := range steps {
		stepMap, ok := stepRaw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("step %d must be an object", i)
		}
		if nodes.GetStringParam(stepMap, "type") == "" {
			return fmt.Errorf("step %d must have a type", i)
		}
	}

	return nil
}

// Execute runs the sequence of nodes
func (e *SequenceExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	steps := nodes.GetArrayParam(input.Params, "steps")

	var allResults []interface{}
	var allDiscoveredURLs []string

	for i, stepRaw := range steps {
		stepMap, ok := stepRaw.(map[string]interface{})
		if !ok {
			continue
		}

		stepType := models.NodeType(nodes.GetStringParam(stepMap, "type"))
		stepParams := nodes.GetMapParam(stepMap, "params")

		// Get executor for this step
		executor, err := e.registry.Get(stepType)
		if err != nil {
			return nil, fmt.Errorf("step %d: failed to get executor for type %s: %w", i, stepType, err)
		}

		// Validate step
		if err := executor.Validate(stepParams); err != nil {
			return nil, fmt.Errorf("step %d validation failed: %w", i, err)
		}

		// Create input for this step
		stepInput := &nodes.ExecutionInput{
			BrowserContext:   input.BrowserContext,
			ExecutionContext: input.ExecutionContext,
			Params:           stepParams,
			URLItem:          input.URLItem,
			ExecutionID:      input.ExecutionID,
		}

		// Execute step
		output, err := executor.Execute(ctx, stepInput)
		if err != nil {
			// Check if step is optional
			if nodes.GetBoolParam(stepMap, "optional", false) {
				continue
			}
			return nil, fmt.Errorf("step %d failed: %w", i, err)
		}

		// Collect results
		if output.Result != nil {
			allResults = append(allResults, output.Result)
		}
		if len(output.DiscoveredURLs) > 0 {
			allDiscoveredURLs = append(allDiscoveredURLs, output.DiscoveredURLs...)
		}
	}

	return &nodes.ExecutionOutput{
		Result:         allResults,
		DiscoveredURLs: allDiscoveredURLs,
		Metadata: map[string]interface{}{
			"steps_executed": len(steps),
		},
	}, nil
}
