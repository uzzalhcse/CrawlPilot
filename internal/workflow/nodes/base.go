package nodes

import (
	"context"

	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// NodeExecutor defines the interface all node types must implement
type NodeExecutor interface {
	// Execute runs the node logic
	Execute(ctx context.Context, input *ExecutionInput) (*ExecutionOutput, error)

	// Validate checks if node configuration is valid
	Validate(params map[string]interface{}) error

	// Type returns the node type this executor handles
	Type() models.NodeType
}

// IHealthCheckValidator interface for nodes that support monitoring validation
type IHealthCheckValidator interface {
	ValidateForMonitoring(ctx context.Context, input *ValidationInput) (*models.NodeValidationResult, error)
}

// ValidationInput contains input for monitoring validation
type ValidationInput struct {
	BrowserContext   *browser.BrowserContext
	ExecutionContext *models.ExecutionContext
	Params           map[string]interface{}
	Config           *models.MonitoringConfig
}

// ExecutionInput contains everything a node needs to execute
type ExecutionInput struct {
	BrowserContext   *browser.BrowserContext
	ExecutionContext *models.ExecutionContext
	Params           map[string]interface{}
	URLItem          *models.URLQueueItem
	ExecutionID      string
}

// ExecutionOutput contains the results of node execution
type ExecutionOutput struct {
	Result         interface{}
	Metadata       map[string]interface{}
	DiscoveredURLs []string
}

// BaseNodeExecutor provides common functionality for node executors
type BaseNodeExecutor struct {
	nodeType models.NodeType
}

// Type returns the node type
func (b *BaseNodeExecutor) Type() models.NodeType {
	return b.nodeType
}

// GetStringParam safely retrieves a string parameter
func GetStringParam(params map[string]interface{}, key string, defaultValue ...string) string {
	if val, ok := params[key].(string); ok {
		return val
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// GetIntParam safely retrieves an int parameter
func GetIntParam(params map[string]interface{}, key string, defaultValue ...int) int {
	if val, ok := params[key].(float64); ok {
		return int(val)
	}
	if val, ok := params[key].(int); ok {
		return val
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

// GetBoolParam safely retrieves a bool parameter
func GetBoolParam(params map[string]interface{}, key string, defaultValue ...bool) bool {
	if val, ok := params[key].(bool); ok {
		return val
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return false
}

// GetMapParam safely retrieves a map parameter
func GetMapParam(params map[string]interface{}, key string) map[string]interface{} {
	if val, ok := params[key].(map[string]interface{}); ok {
		return val
	}
	return make(map[string]interface{})
}

// GetArrayParam safely retrieves an array parameter
func GetArrayParam(params map[string]interface{}, key string) []interface{} {
	if val, ok := params[key].([]interface{}); ok {
		return val
	}
	return []interface{}{}
}
