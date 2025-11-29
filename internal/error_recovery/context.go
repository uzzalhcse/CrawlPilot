package error_recovery

import (
	"net/http"
)

// ExecutionContext contains information about the current execution
type ExecutionContext struct {
	URL           string
	Domain        string
	Error         ErrorInfo
	Response      ResponseInfo
	CurrentConfig map[string]interface{}
	FailedRules   []string
	WorkflowID    string
	ExecutionID   string
}

// ErrorInfo contains details about an error
type ErrorInfo struct {
	Type    string
	Message string
	Code    int
}

// ResponseInfo contains HTTP response details
type ResponseInfo struct {
	StatusCode int
	Header     http.Header
	Body       string
}

// NewExecutionContext creates a new execution context
func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{
		CurrentConfig: make(map[string]interface{}),
		FailedRules:   []string{},
		Response: ResponseInfo{
			Header: make(http.Header),
		},
	}
}
