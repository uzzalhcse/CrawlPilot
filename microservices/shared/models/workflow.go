package models

import "time"

// Workflow represents a scraping workflow configuration
type Workflow struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Config      WorkflowConfig `json:"config"`
	Status      string         `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// WorkflowConfig contains the workflow configuration
type WorkflowConfig struct {
	StartURLs      []string          `json:"start_urls"`
	MaxDepth       int               `json:"max_depth,omitempty"`
	RateLimitDelay int               `json:"rate_limit_delay,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	Phases         []WorkflowPhase   `json:"phases"`
}

// WorkflowPhase represents a phase in the workflow
type WorkflowPhase struct {
	ID         string           `json:"id"`
	Type       string           `json:"type"` // discovery, extraction
	Name       string           `json:"name"`
	Nodes      []Node           `json:"nodes"` // Nodes are inside the phase
	URLFilter  *URLFilter       `json:"url_filter,omitempty"`
	Transition *PhaseTransition `json:"transition,omitempty"`
}

// URLFilter defines URL filtering rules for a phase
type URLFilter struct {
	Depth   int      `json:"depth,omitempty"`
	Markers []string `json:"markers,omitempty"`
}

// PhaseTransition defines transition rules between phases
type PhaseTransition struct {
	Condition string `json:"condition"` // all_nodes_complete, etc.
	NextPhase string `json:"next_phase"`
}

// Node represents a workflow node
type Node struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Name   string                 `json:"name,omitempty"`
	Params map[string]interface{} `json:"params"` // Changed from config to params
}

// Execution represents a workflow execution
type Execution struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflow_id"`
	Status      string                 `json:"status"` // running, completed, failed, paused
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Task represents a single URL processing task
type Task struct {
	TaskID      string                 `json:"task_id"`
	ExecutionID string                 `json:"execution_id"`
	WorkflowID  string                 `json:"workflow_id"`
	URL         string                 `json:"url"`
	Depth       int                    `json:"depth"`
	ParentURLID *string                `json:"parent_url_id,omitempty"`
	Marker      string                 `json:"marker,omitempty"` // URL marker (category, product, etc)
	PhaseID     string                 `json:"phase_id"`
	PhaseConfig WorkflowPhase          `json:"phase_config"` // Full phase config with nodes
	Metadata    map[string]interface{} `json:"metadata"`
	RetryCount  int                    `json:"retry_count"`
}

// ExtractedItem represents extracted data
type ExtractedItem struct {
	ID          string                 `json:"id"`
	ExecutionID string                 `json:"execution_id"`
	URL         string                 `json:"url"`
	SchemaName  string                 `json:"schema_name"`
	Data        map[string]interface{} `json:"data"`
	ExtractedAt time.Time              `json:"extracted_at"`
}

// ExtractedItemMetadata stores metadata about extracted items
type ExtractedItemMetadata struct {
	ID          string    `json:"id"`
	ExecutionID string    `json:"execution_id"`
	URL         string    `json:"url"`
	ItemCount   int       `json:"item_count"`
	GCSPath     string    `json:"gcs_path"`
	ExtractedAt time.Time `json:"extracted_at"`
}

// ExecutionContext holds runtime execution data
type ExecutionContext struct {
	Variables map[string]interface{} `json:"variables"`
}
