package models

import "time"

// HealthCheckReport represents a workflow health check report
type HealthCheckReport struct {
	ID           string                            `json:"id" db:"id"`
	WorkflowID   string                            `json:"workflow_id" db:"workflow_id"`
	WorkflowName string                            `json:"workflow_name" db:"-"`
	ExecutionID  *string                           `json:"execution_id,omitempty" db:"execution_id"`
	Status       HealthCheckStatus                 `json:"status" db:"status"`
	StartedAt    time.Time                         `json:"started_at" db:"started_at"`
	CompletedAt  *time.Time                        `json:"completed_at,omitempty" db:"completed_at"`
	Duration     int64                             `json:"duration_ms" db:"duration_ms"`
	Results      map[string]*PhaseValidationResult `json:"phase_results" db:"-"`
	ResultsJSON  []byte                            `json:"-" db:"results"`
	Summary      *HealthCheckSummary               `json:"summary" db:"-"`
	SummaryJSON  []byte                            `json:"-" db:"summary"`
	Config       *HealthCheckConfig                `json:"config" db:"-"`
	ConfigJSON   []byte                            `json:"-" db:"config"`
}

// HealthCheckStatus represents the overall health status
type HealthCheckStatus string

const (
	HealthCheckStatusRunning  HealthCheckStatus = "running"
	HealthCheckStatusHealthy  HealthCheckStatus = "healthy"
	HealthCheckStatusDegraded HealthCheckStatus = "degraded"
	HealthCheckStatusFailed   HealthCheckStatus = "failed"
)

// HealthCheckConfig contains configuration for health check execution
type HealthCheckConfig struct {
	MaxURLsPerPhase    int  `json:"max_urls_per_phase"`
	MaxPaginationPages int  `json:"max_pagination_pages"`
	MaxDepth           int  `json:"max_depth"`
	TimeoutSeconds     int  `json:"timeout_seconds"`
	SkipDataStorage    bool `json:"skip_data_storage"`
}

// PhaseValidationResult contains validation results for a workflow phase
type PhaseValidationResult struct {
	PhaseID           string                 `json:"phase_id"`
	PhaseName         string                 `json:"phase_name"`
	NodeResults       []NodeValidationResult `json:"node_results"`
	NavigationError   string                 `json:"navigation_error,omitempty"`
	HasCriticalIssues bool                   `json:"has_critical_issues"`
}

// NodeValidationResult contains validation results for a single node
type NodeValidationResult struct {
	NodeID   string                 `json:"node_id"`
	NodeName string                 `json:"node_name"`
	NodeType string                 `json:"node_type"`
	Status   ValidationStatus       `json:"status"`
	Metrics  map[string]interface{} `json:"metrics"`
	Issues   []ValidationIssue      `json:"issues"`
	Duration int64                  `json:"duration_ms"`
}

// ValidationStatus represents the validation result status
type ValidationStatus string

const (
	ValidationStatusPass    ValidationStatus = "pass"
	ValidationStatusFail    ValidationStatus = "fail"
	ValidationStatusWarning ValidationStatus = "warning"
	ValidationStatusSkip    ValidationStatus = "skip"
)

// ValidationIssue represents a specific validation issue found
type ValidationIssue struct {
	Severity   string      `json:"severity"`
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	Selector   string      `json:"selector,omitempty"`
	Expected   interface{} `json:"expected,omitempty"`
	Actual     interface{} `json:"actual,omitempty"`
	Suggestion string      `json:"suggestion,omitempty"`
}

// HealthCheckSummary provides an aggregate summary of health check results
type HealthCheckSummary struct {
	TotalPhases    int               `json:"total_phases"`
	TotalNodes     int               `json:"total_nodes"`
	PassedNodes    int               `json:"passed_nodes"`
	FailedNodes    int               `json:"failed_nodes"`
	WarningNodes   int               `json:"warning_nodes"`
	CriticalIssues []ValidationIssue `json:"critical_issues"`
}
