package models

import "time"

// Workflow represents a scraping workflow configuration
type Workflow struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Description      string         `json:"description,omitempty"`
	Config           WorkflowConfig `json:"config"`
	Status           string         `json:"status"`
	BrowserProfileID *string        `json:"browser_profile_id,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// WorkflowConfig contains the workflow configuration
type WorkflowConfig struct {
	StartURLs          []string          `json:"start_urls"`
	MaxDepth           int               `json:"max_depth,omitempty"`
	RateLimitDelay     int               `json:"rate_limit_delay,omitempty"`
	Headers            map[string]string `json:"headers,omitempty"`
	DefaultDriver      string            `json:"default_driver,omitempty"`       // playwright, chromedp, http
	DefaultBrowserName string            `json:"default_browser_name,omitempty"` // chrome, firefox, safari, edge, ios, android (for HTTP driver JA3)
	Phases             []WorkflowPhase   `json:"phases"`
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
	Status      string                 `json:"status"` // running, completed, failed, stopped
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	TriggeredBy string                 `json:"triggered_by,omitempty"` // manual, scheduled, api
	Metadata    map[string]interface{} `json:"metadata"`

	// Stats fields (aggregated)
	URLsProcessed  int `json:"urls_processed"`
	URLsDiscovered int `json:"urls_discovered"`
	ItemsExtracted int `json:"items_extracted"`
	Errors         int `json:"errors"`

	// Phase breakdown stats
	PhaseStats map[string]PhaseStatEntry `json:"phase_stats,omitempty"`
}

// PhaseStatEntry holds stats for a single phase
type PhaseStatEntry struct {
	Processed  int   `json:"processed"`
	Errors     int   `json:"errors"`
	DurationMs int64 `json:"duration_ms"`
}

// ExecutionError represents an error that occurred during execution
type ExecutionError struct {
	ID          int64     `json:"id"`
	ExecutionID string    `json:"execution_id"`
	URL         string    `json:"url"`
	ErrorType   string    `json:"error_type"` // timeout, blocked, parse_error, network, extraction
	Message     string    `json:"message"`
	PhaseID     string    `json:"phase_id,omitempty"`
	RetryCount  int       `json:"retry_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// Task represents a single URL processing task
type Task struct {
	TaskID           string                 `json:"task_id"`
	ExecutionID      string                 `json:"execution_id"`
	WorkflowID       string                 `json:"workflow_id"`
	URL              string                 `json:"url"`
	Depth            int                    `json:"depth"`
	ParentURLID      *string                `json:"parent_url_id,omitempty"`
	Marker           string                 `json:"marker,omitempty"` // URL marker (category, product, etc)
	PhaseID          string                 `json:"phase_id"`
	PhaseConfig      WorkflowPhase          `json:"phase_config"`              // Full phase config with nodes
	WorkflowConfig   *WorkflowConfig        `json:"workflow_config,omitempty"` // Workflow-level config (for defaults)
	Metadata         map[string]interface{} `json:"metadata"`
	RetryCount       int                    `json:"retry_count"`
	BrowserProfileID *string                `json:"browser_profile_id,omitempty"` // Browser profile for this task

	// Proxy settings (populated by recovery system)
	ProxyURL string `json:"proxy_url,omitempty"` // Full proxy URL with auth
	ProxyID  string `json:"proxy_id,omitempty"`  // Proxy ID for tracking
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
