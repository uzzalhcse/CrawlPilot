package models

import "time"

// ScrapeRequest represents a single URL scrape request
type ScrapeRequest struct {
	ID              string          `json:"id"`
	URL             string          `json:"url"`
	Driver          string          `json:"driver"`            // camoufox, playwright, http
	ProfileID       string          `json:"profile_id"`        // Browser profile ID
	Profile         *BrowserProfile `json:"profile,omitempty"` // Embedded profile data (populated by orchestrator)
	OutputFormat    string          `json:"output_format"`     // html, markdown, screenshot
	Timeout         int             `json:"timeout"`           // Timeout in seconds
	WaitForSelector string          `json:"wait_for_selector"` // CSS selector to wait for
	Headless        bool            `json:"headless"`          // Run browser in headless mode
	UseProxy        bool            `json:"use_proxy"`         // Whether to use a proxy
	ProxyTier       int             `json:"proxy_tier"`        // Proxy tier (1=datacenter, 2=residential, 3=mobile)
	Status          string          `json:"status"`            // pending, running, completed, failed
	CreatedAt       time.Time       `json:"created_at"`
}

// ScrapeResult represents the result of a scrape request
type ScrapeResult struct {
	ID          string `json:"id"`
	Status      string `json:"status"`               // pending, running, completed, failed
	Content     string `json:"content,omitempty"`    // HTML or Markdown content
	Screenshot  string `json:"screenshot,omitempty"` // Base64 encoded screenshot
	ContentType string `json:"content_type"`         // text/html, text/markdown, image/png
	StatusCode  int    `json:"status_code"`          // HTTP status code
	Duration    int64  `json:"duration_ms"`          // Execution duration in milliseconds
	Error       string `json:"error,omitempty"`      // Error message if failed
	URL         string `json:"url"`                  // Original URL
}

// ScrapeRequestInput is the API input for creating a scrape request
type ScrapeRequestInput struct {
	URL             string `json:"url" validate:"required,url"`
	Driver          string `json:"driver"`            // camoufox, playwright, http (default: http)
	ProfileID       string `json:"profile_id"`        // Optional browser profile
	OutputFormat    string `json:"output_format"`     // html, markdown, screenshot (default: html)
	Timeout         int    `json:"timeout"`           // Timeout in seconds (default: 30)
	WaitForSelector string `json:"wait_for_selector"` // Optional CSS selector to wait for
	Headless        *bool  `json:"headless"`          // Run browser in headless mode (default: true)
	UseProxy        bool   `json:"use_proxy"`         // Whether to use a proxy
	ProxyTier       int    `json:"proxy_tier"`        // Proxy tier (1=datacenter, 2=residential, 3=mobile)
}

// ValidateAndSetDefaults validates input and sets default values
func (r *ScrapeRequestInput) ValidateAndSetDefaults() {
	if r.Driver == "" {
		r.Driver = "http"
	}
	if r.OutputFormat == "" {
		r.OutputFormat = "html"
	}
	if r.Timeout == 0 {
		r.Timeout = 30
	}
	// Validate driver
	validDrivers := map[string]bool{"http": true, "playwright": true, "camoufox": true}
	if !validDrivers[r.Driver] {
		r.Driver = "http"
	}
	// Validate output format
	validFormats := map[string]bool{"html": true, "markdown": true, "screenshot": true}
	if !validFormats[r.OutputFormat] {
		r.OutputFormat = "html"
	}
}

// ScrapeStatus constants
const (
	ScrapeStatusPending   = "pending"
	ScrapeStatusRunning   = "running"
	ScrapeStatusCompleted = "completed"
	ScrapeStatusFailed    = "failed"
)

// OutputFormat constants
const (
	OutputFormatHTML       = "html"
	OutputFormatMarkdown   = "markdown"
	OutputFormatScreenshot = "screenshot"
)

// DriverType constants
const (
	DriverHTTP       = "http"
	DriverPlaywright = "playwright"
	DriverCamoufox   = "camoufox"
)

// UniversalScraperWorkflowID is the well-known UUID for synthetic scrape tasks
// This is a fixed UUID v4 to ensure DB compatibility (workflow_id column expects UUID)
const UniversalScraperWorkflowID = "00000000-0000-0000-0000-000000000001"

// ToTask converts a ScrapeRequest into a synthetic Task
// This allows Universal Scraper to reuse workflow infrastructure (recovery, proxy rotation, etc.)
func (r *ScrapeRequest) ToTask() *Task {
	// Build navigate node params
	navigateParams := map[string]interface{}{
		"url": r.URL,
	}
	if r.WaitForSelector != "" {
		navigateParams["wait_selector"] = r.WaitForSelector
	}
	if r.Timeout > 0 {
		navigateParams["timeout"] = r.Timeout * 1000 // Convert seconds to ms
	}
	// Set driver override in node params (this is how getDriverForTask selects driver)
	if r.Driver != "" {
		navigateParams["driver"] = r.Driver
	}

	// Build extract_content node params
	extractParams := map[string]interface{}{
		"output_format": r.OutputFormat,
	}

	// Create nodes
	nodes := []Node{
		{
			ID:     "navigate",
			Type:   "navigate",
			Name:   "Navigate",
			Params: navigateParams,
		},
		{
			ID:     "extract_content",
			Type:   "extract_content",
			Name:   "Extract Content",
			Params: extractParams,
		},
	}

	// Create phase with nodes embedded
	phase := WorkflowPhase{
		ID:    "scrape",
		Name:  "Scrape",
		Nodes: nodes,
	}

	// Create synthetic workflow config
	workflowConfig := &WorkflowConfig{
		StartURLs:     []string{r.URL},
		DefaultDriver: r.Driver,
		Phases:        []WorkflowPhase{phase},
	}

	// Build task metadata
	metadata := map[string]interface{}{
		"is_scrape":     true,
		"scrape_id":     r.ID,
		"output_format": r.OutputFormat,
		"driver":        r.Driver,
		"headless":      r.Headless,
		"use_proxy":     r.UseProxy,
	}

	// Set BrowserProfileID on Task struct if profile is provided
	// This is required for getDriverForTask() to use the profile
	var browserProfileID *string
	if r.Profile != nil && r.ProfileID != "" {
		browserProfileID = &r.ProfileID
		metadata["browser_profile"] = r.Profile
	}

	return &Task{
		TaskID:           r.ID,
		ExecutionID:      r.ID, // Use same ID for easy result lookup
		WorkflowID:       UniversalScraperWorkflowID,
		URL:              r.URL,
		PhaseID:          "scrape",
		PhaseConfig:      phase,                  // Current phase config
		BrowserProfileID: browserProfileID,       // Set profile ID for driver selection
		ProxyTier:        ProxyTier(r.ProxyTier), // Request specific proxy tier (0=auto, 1=datacenter, 2=residential, 3=mobile)
		Depth:            0,
		Metadata:         metadata,
		WorkflowConfig:   workflowConfig,
	}
}
