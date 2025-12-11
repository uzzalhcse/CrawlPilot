package models

import "time"

// ScrapeRequest represents a single URL scrape request
type ScrapeRequest struct {
	ID              string    `json:"id"`
	URL             string    `json:"url"`
	Driver          string    `json:"driver"`            // camoufox, playwright, http
	ProfileID       string    `json:"profile_id"`        // Browser profile ID
	OutputFormat    string    `json:"output_format"`     // html, markdown, screenshot
	Timeout         int       `json:"timeout"`           // Timeout in seconds
	WaitForSelector string    `json:"wait_for_selector"` // CSS selector to wait for
	Status          string    `json:"status"`            // pending, running, completed, failed
	CreatedAt       time.Time `json:"created_at"`
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
