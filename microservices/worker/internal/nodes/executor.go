package nodes

import (
	"context"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
)

// ExecutionContext holds the current execution state
type ExecutionContext struct {
	Page      playwright.Page
	Task      *models.Task
	Variables map[string]interface{}
}

// NodeExecutor defines the interface for node execution
type NodeExecutor interface {
	// Execute runs the node logic
	Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error

	// Type returns the node type this executor handles
	Type() string
}

// Result holds the result of a node execution
type Result struct {
	Success bool
	Data    map[string]interface{}
	Error   error
}
