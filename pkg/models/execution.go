package models

import (
	"encoding/json"
	"time"
)

// WorkflowExecution represents a single execution instance of a workflow
type WorkflowExecution struct {
	ID           string           `json:"id" db:"id"`
	WorkflowID   string           `json:"workflow_id" db:"workflow_id"`
	WorkflowName string           `json:"workflow_name,omitempty" db:"-"`
	Status       ExecutionStatus  `json:"status" db:"status"`
	StartedAt    time.Time        `json:"started_at" db:"started_at"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty" db:"completed_at"`
	Error        string           `json:"error,omitempty" db:"error"`
	Stats        ExecutionStats   `json:"stats" db:"stats"`
	Context      ExecutionContext `json:"context" db:"context"`
}

// ExecutionStatus represents the status of a workflow execution
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

// ExecutionStats contains statistics about the execution
type ExecutionStats struct {
	URLsDiscovered  int       `json:"urls_discovered"`
	URLsProcessed   int       `json:"urls_processed"`
	URLsFailed      int       `json:"urls_failed"`
	ItemsExtracted  int       `json:"items_extracted"`
	BytesDownloaded int64     `json:"bytes_downloaded"`
	Duration        int64     `json:"duration"` // milliseconds
	NodesExecuted   int       `json:"nodes_executed"`
	NodesFailed     int       `json:"nodes_failed"`
	LastUpdate      time.Time `json:"last_update"`
}

// ExecutionContext stores runtime context data passed between nodes
type ExecutionContext struct {
	Data      map[string]interface{} `json:"data"`
	Variables map[string]string      `json:"variables"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// NodeExecution represents the execution of a single node
type NodeExecution struct {
	ID                    string          `json:"id" db:"id"`
	ExecutionID           string          `json:"execution_id" db:"execution_id"`
	NodeID                string          `json:"node_id" db:"node_id"`
	Status                ExecutionStatus `json:"status" db:"status"`
	URLID                 *string         `json:"url_id,omitempty" db:"url_id"`
	ParentNodeExecutionID *string         `json:"parent_node_execution_id,omitempty" db:"parent_node_execution_id"`
	NodeType              *string         `json:"node_type,omitempty" db:"node_type"`
	URLsDiscovered        int             `json:"urls_discovered" db:"urls_discovered"`
	ItemsExtracted        int             `json:"items_extracted" db:"items_extracted"`
	ErrorMessage          *string         `json:"error_message,omitempty" db:"error_message"`
	DurationMs            *int            `json:"duration_ms,omitempty" db:"duration_ms"`
	StartedAt             time.Time       `json:"started_at" db:"started_at"`
	CompletedAt           *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	Input                 json.RawMessage `json:"input,omitempty" db:"input"`
	Output                json.RawMessage `json:"output,omitempty" db:"output"`
	Error                 string          `json:"error,omitempty" db:"error"`
	RetryCount            int             `json:"retry_count" db:"retry_count"`
}

// Scan implements sql.Scanner for ExecutionStats
func (es *ExecutionStats) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, es)
}

// Value implements driver.Valuer for ExecutionStats
func (es ExecutionStats) Value() (interface{}, error) {
	return json.Marshal(es)
}

// Scan implements sql.Scanner for ExecutionContext
func (ec *ExecutionContext) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, ec)
}

// Value implements driver.Valuer for ExecutionContext
func (ec ExecutionContext) Value() (interface{}, error) {
	return json.Marshal(ec)
}

// NewExecutionContext creates a new ExecutionContext
func NewExecutionContext() ExecutionContext {
	return ExecutionContext{
		Data:      make(map[string]interface{}),
		Variables: make(map[string]string),
		Metadata:  make(map[string]interface{}),
	}
}

// Set sets a value in the context
func (ec *ExecutionContext) Set(key string, value interface{}) {
	ec.Data[key] = value
}

// Get retrieves a value from the context
func (ec *ExecutionContext) Get(key string) (interface{}, bool) {
	val, ok := ec.Data[key]
	return val, ok
}

// SetVariable sets a variable in the context
func (ec *ExecutionContext) SetVariable(key, value string) {
	ec.Variables[key] = value
}

// GetVariable retrieves a variable from the context
func (ec *ExecutionContext) GetVariable(key string) (string, bool) {
	val, ok := ec.Variables[key]
	return val, ok
}

// GetAll returns all data from the context
func (ec *ExecutionContext) GetAll() map[string]interface{} {
	return ec.Data
}
