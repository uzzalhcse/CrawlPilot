package models

import "time"

// MonitoringReport represents a workflow monitoring report
type MonitoringReport struct {
	ID           string                            `json:"id" db:"id"`
	WorkflowID   string                            `json:"workflow_id" db:"workflow_id"`
	WorkflowName string                            `json:"workflow_name" db:"-"`
	ExecutionID  *string                           `json:"execution_id,omitempty" db:"execution_id"`
	Status       MonitoringStatus                  `json:"status" db:"status"`
	StartedAt    time.Time                         `json:"started_at" db:"started_at"`
	CompletedAt  *time.Time                        `json:"completed_at,omitempty" db:"completed_at"`
	Duration     int64                             `json:"duration_ms" db:"duration_ms"`
	Results      map[string]*PhaseValidationResult `json:"phase_results" db:"-"`
	ResultsJSON  []byte                            `json:"-" db:"results"`
	Summary      *MonitoringSummary                `json:"summary" db:"-"`
	SummaryJSON  []byte                            `json:"-" db:"summary"`
	Config       *MonitoringConfig                 `json:"config" db:"-"`
	ConfigJSON   []byte                            `json:"-" db:"config"`
}

// MonitoringStatus represents the overall health status
type MonitoringStatus string

const (
	MonitoringStatusRunning  MonitoringStatus = "running"
	MonitoringStatusHealthy  MonitoringStatus = "healthy"
	MonitoringStatusDegraded MonitoringStatus = "degraded"
	MonitoringStatusFailed   MonitoringStatus = "failed"
)

// MonitoringConfig contains configuration for monitoring execution
type MonitoringConfig struct {
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

// MonitoringSummary provides an aggregate summary of monitoring results
type MonitoringSummary struct {
	TotalPhases    int               `json:"total_phases"`
	TotalNodes     int               `json:"total_nodes"`
	PassedNodes    int               `json:"passed_nodes"`
	FailedNodes    int               `json:"failed_nodes"`
	WarningNodes   int               `json:"warning_nodes"`
	CriticalIssues []ValidationIssue `json:"critical_issues"`
}

// MonitoringSchedule represents a scheduled monitoring configuration
type MonitoringSchedule struct {
	ID                 string              `json:"id" db:"id"`
	WorkflowID         string              `json:"workflow_id" db:"workflow_id"`
	Schedule           string              `json:"schedule" db:"schedule"` // cron format
	Enabled            bool                `json:"enabled" db:"enabled"`
	LastRunAt          *time.Time          `json:"last_run_at,omitempty" db:"last_run_at"`
	NextRunAt          *time.Time          `json:"next_run_at,omitempty" db:"next_run_at"`
	NotificationConfig *NotificationConfig `json:"notification_config,omitempty" db:"-"`
	NotificationJSON   []byte              `json:"-" db:"notification_config"`
	CreatedAt          time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at" db:"updated_at"`
}

// NotificationConfig defines how to notify on monitoring results
type NotificationConfig struct {
	Slack         *SlackConfig `json:"slack,omitempty"`
	OnlyOnFailure bool         `json:"only_on_failure"`
	OnlyOnChange  bool         `json:"only_on_change"` // Only notify if status changed
}

// SlackConfig for Slack webhook notifications
type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel,omitempty"`
}

// BaselineComparison represents a comparison between current and baseline metrics
type BaselineComparison struct {
	Metric        string           `json:"metric"`
	Baseline      interface{}      `json:"baseline"`
	Current       interface{}      `json:"current"`
	ChangePercent float64          `json:"change_percent,omitempty"`
	Status        ComparisonStatus `json:"status"`
}

type ComparisonStatus string

const (
	ComparisonImproved  ComparisonStatus = "improved"
	ComparisonDegraded  ComparisonStatus = "degraded"
	ComparisonUnchanged ComparisonStatus = "unchanged"
)

// MonitoringSnapshot stores diagnostic data captured when monitoring fails
type MonitoringSnapshot struct {
	ID              string                 `json:"id" db:"id"`
	ReportID        string                 `json:"report_id" db:"report_id"`
	NodeID          string                 `json:"node_id" db:"node_id"`
	PhaseName       string                 `json:"phase_name" db:"phase_name"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	URL             string                 `json:"url" db:"url"`
	PageTitle       *string                `json:"page_title,omitempty" db:"page_title"`
	StatusCode      *int                   `json:"status_code,omitempty" db:"status_code"`
	ScreenshotPath  *string                `json:"screenshot_path,omitempty" db:"screenshot_path"`
	DOMSnapshotPath *string                `json:"dom_snapshot_path,omitempty" db:"dom_snapshot_path"`
	ConsoleLogs     []byte                 `json:"-" db:"console_logs"` // JSONB
	ConsoleLogsData []ConsoleLog           `json:"console_logs,omitempty" db:"-"`
	SelectorType    *string                `json:"selector_type,omitempty" db:"selector_type"`
	SelectorValue   *string                `json:"selector_value,omitempty" db:"selector_value"`
	ElementsFound   int                    `json:"elements_found" db:"elements_found"`
	ErrorMessage    *string                `json:"error_message,omitempty" db:"error_message"`
	FieldRequired   *bool                  `json:"field_required,omitempty" db:"field_required"` // NEW: Indicates if this field is required
	Metadata        []byte                 `json:"-" db:"metadata"`                              // JSONB
	MetadataData    map[string]interface{} `json:"metadata,omitempty" db:"-"`
}

// ConsoleLog represents a browser console log entry
type ConsoleLog struct {
	Type      string    `json:"type"` // log, warn, error, info
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source,omitempty"`
}

// FixSuggestion represents an AI-generated fix for a failed monitoring
type FixSuggestion struct {
	ID                   string                 `json:"id"`
	SnapshotID           string                 `json:"snapshot_id"`
	WorkflowID           string                 `json:"workflow_id"`
	NodeID               string                 `json:"node_id"`
	SuggestedSelector    string                 `json:"suggested_selector"`
	AlternativeSelectors []string               `json:"alternative_selectors,omitempty"`
	SuggestedNodeConfig  map[string]interface{} `json:"suggested_node_config,omitempty"`
	FixExplanation       string                 `json:"fix_explanation"`
	ConfidenceScore      float64                `json:"confidence_score"`
	Status               string                 `json:"status"` // pending, approved, rejected, applied, reverted
	ReviewedBy           *string                `json:"reviewed_by,omitempty"`
	ReviewedAt           *time.Time             `json:"reviewed_at,omitempty"`
	AppliedAt            *time.Time             `json:"applied_at,omitempty"`
	RevertedAt           *time.Time             `json:"reverted_at,omitempty"`
	AIModel              string                 `json:"ai_model"`
	AIPromptTokens       *int                   `json:"ai_prompt_tokens,omitempty"`
	AIResponseTokens     *int                   `json:"ai_response_tokens,omitempty"`
	AIResponseRaw        *string                `json:"ai_response_raw,omitempty"`
	VerificationResult   *VerificationResult    `json:"verification_result,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

// VerificationResult contains the result of verifying a suggested selector
type VerificationResult struct {
	IsValid       bool     `json:"is_valid"`
	ElementsFound int      `json:"elements_found"`
	DataPreview   []string `json:"data_preview"`
	ErrorMessage  string   `json:"error_message,omitempty"`
}
