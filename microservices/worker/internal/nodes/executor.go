package nodes

import (
	"context"

	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/driver"
)

// ExecutionContext holds the current execution state
type ExecutionContext struct {
	Page                    driver.Page
	Task                    *models.Task
	Variables               map[string]interface{}
	ExtractedItems          []map[string]interface{}                   // Items extracted during execution
	DiscoveredURLs          []string                                   // URLs discovered during execution
	BranchNodes             []models.Node                              // Nodes to execute from conditional branches
	SwitchDriver            func(string) error                         // Callback to switch driver (legacy)
	SwitchDriverWithProfile func(driverType, profileID string) error   // Switch driver with optional profile
	SwitchDriverWithBrowser func(driverType, browserName string) error // Switch HTTP driver with browser name for JA3
	OnWarning               func(field, message string)                // Callback for logging warnings
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
