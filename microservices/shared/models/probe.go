package models

import "time"

// ProbeResult stores the complete probe execution outcome
type ProbeResult struct {
	ID          string             `json:"id"`
	ExecutionID string             `json:"execution_id"`
	WorkflowID  string             `json:"workflow_id"`
	Status      string             `json:"status"` // healthy, degraded, broken
	Duration    int64              `json:"duration_ms"`
	Phases      []PhaseProbeResult `json:"phases"`
	CreatedAt   time.Time          `json:"created_at"`
	AutoFix     *AutoFixRecord     `json:"auto_fix,omitempty"`
}

// AutoFixRecord represents an auto-fix record
type AutoFixRecord struct {
	ID          string  `json:"id"`
	WorkflowID  string  `json:"workflow_id"`
	ExecutionID string  `json:"execution_id"`
	NodeID      string  `json:"node_id"`
	FieldName   string  `json:"field_name,omitempty"` // For field-level fixes
	FixType     string  `json:"fix_type"`
	OldSelector string  `json:"old_selector,omitempty"`
	NewSelector string  `json:"new_selector,omitempty"`
	Reasoning   string  `json:"reasoning"`
	Confidence  float64 `json:"confidence"`
	Status      string  `json:"status"`
	AutoApplied bool    `json:"auto_applied"`
	CreatedAt   string  `json:"created_at"`
}

// PhaseProbeResult holds probe results for one phase
type PhaseProbeResult struct {
	PhaseID   string            `json:"phase_id"`
	PhaseName string            `json:"phase_name"`
	Status    string            `json:"status"` // passed, failed
	SampleURL string            `json:"sample_url"`
	Duration  int64             `json:"duration_ms"`
	Nodes     []NodeProbeResult `json:"nodes"`
	Error     string            `json:"error,omitempty"`
}

// NodeProbeResult holds execution result for each node during probe
type NodeProbeResult struct {
	NodeID       string `json:"node_id"`
	NodeName     string `json:"node_name"`
	NodeType     string `json:"node_type"`
	Status       string `json:"status"` // passed, failed, skipped
	Duration     int64  `json:"duration_ms"`
	Selector     string `json:"selector,omitempty"`
	ElementCount int    `json:"element_count,omitempty"`
	LinksFound   int    `json:"links_found,omitempty"`
	Error        string `json:"error,omitempty"`

	// Field-level results for extract nodes
	Fields                []FieldProbeResult `json:"fields,omitempty"`
	MissingRequiredFields []string           `json:"missing_required_fields,omitempty"` // Required fields that failed to extract

	// Baseline comparison (from last successful probe)
	ExpectedElementCount int    `json:"expected_element_count,omitempty"`
	ExpectedLinksFound   int    `json:"expected_links_found,omitempty"`
	Deviation            string `json:"deviation,omitempty"` // "none", "decreased", "increased", "missing"

	// Failed node diagnostics
	SelectorContext string         `json:"selector_context,omitempty"` // HTML around the selector
	Snapshot        *ProbeSnapshot `json:"snapshot,omitempty"`         // Page state when node failed
}

// FieldProbeResult holds extraction result for each field in an extract node
type FieldProbeResult struct {
	Name     string `json:"name"`               // Field name
	Selector string `json:"selector,omitempty"` // CSS selector used
	Status   string `json:"status"`             // "ok", "empty", "error"
	Required bool   `json:"required,omitempty"` // Whether this field was marked as required
	Value    string `json:"value,omitempty"`    // Extracted value (truncated for large values)
	Error    string `json:"error,omitempty"`    // Error message if failed
}

// ProbeSnapshot captures full page state for AI analysis
type ProbeSnapshot struct {
	// Storage paths (GCS or local)
	DOMPath        string `json:"dom_path"`        // Path to full DOM HTML file
	ScreenshotPath string `json:"screenshot_path"` // Path to full page PNG

	// Page metadata
	PageURL   string `json:"page_url"`
	PageTitle string `json:"page_title"`

	// Capture info
	CapturedAt  int64 `json:"captured_at"`  // Unix timestamp
	DOMSize     int64 `json:"dom_size"`     // DOM file size in bytes
	ImageWidth  int   `json:"image_width"`  // Screenshot width
	ImageHeight int   `json:"image_height"` // Screenshot height
}
