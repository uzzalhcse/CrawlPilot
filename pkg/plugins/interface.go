package plugins

import (
	"context"
	"time"

	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// PluginInfo provides metadata about the plugin
type PluginInfo struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Version     string           `json:"version"`
	Author      string           `json:"author"`
	AuthorEmail string           `json:"author_email,omitempty"`
	Description string           `json:"description"`
	PhaseType   models.PhaseType `json:"phase_type"` // discovery, extraction, or processing
	Repository  string           `json:"repository,omitempty"`
	License     string           `json:"license,omitempty"`
}

// DiscoveryInput contains all data needed for discovery phase execution
type DiscoveryInput struct {
	// Browser context for web interactions
	BrowserContext *browser.BrowserContext

	// Current URL being processed
	URL string

	// URL metadata from queue
	URLItem *models.URLQueueItem

	// Execution context with shared state
	ExecutionContext *models.ExecutionContext

	// Plugin-specific configuration
	Config map[string]interface{}

	// Execution ID for tracking
	ExecutionID string
}

// DiscoveryOutput contains results from discovery phase
type DiscoveryOutput struct {
	// Discovered URLs to be added to queue
	DiscoveredURLs []string

	// Metadata about the discovery process
	Metadata map[string]interface{}

	// Optional: Mark URLs with specific markers
	URLMarkers map[string]string // URL -> marker

	// Optional: Assign URLs to specific phases
	URLPhases map[string]string // URL -> phase_id
}

// ExtractionInput contains all data needed for extraction phase execution
type ExtractionInput struct {
	// Browser context for web interactions
	BrowserContext *browser.BrowserContext

	// Current URL being processed
	URL string

	// URL metadata from queue
	URLItem *models.URLQueueItem

	// Execution context with shared state
	ExecutionContext *models.ExecutionContext

	// Plugin-specific configuration
	Config map[string]interface{}

	// Execution ID for tracking
	ExecutionID string
}

// ExtractionOutput contains results from extraction phase
type ExtractionOutput struct {
	// Extracted structured data
	Data map[string]interface{}

	// Schema name for the extracted data
	SchemaName string

	// Metadata about the extraction process
	Metadata map[string]interface{}

	// Optional: Additional URLs discovered during extraction
	DiscoveredURLs []string
}

// DiscoveryPlugin interface for discovery phase plugins
type DiscoveryPlugin interface {
	// Info returns plugin metadata
	Info() PluginInfo

	// Discover executes discovery logic and returns discovered URLs
	Discover(ctx context.Context, input *DiscoveryInput) (*DiscoveryOutput, error)

	// Validate checks if plugin configuration is valid
	Validate(config map[string]interface{}) error

	// ConfigSchema returns JSON Schema for plugin configuration
	ConfigSchema() map[string]interface{}
}

// ExtractionPlugin interface for extraction phase plugins
type ExtractionPlugin interface {
	// Info returns plugin metadata
	Info() PluginInfo

	// Extract executes extraction logic and returns structured data
	Extract(ctx context.Context, input *ExtractionInput) (*ExtractionOutput, error)

	// Validate checks if plugin configuration is valid
	Validate(config map[string]interface{}) error

	// ConfigSchema returns JSON Schema for plugin configuration
	ConfigSchema() map[string]interface{}
}

// PluginMetrics tracks plugin execution metrics
type PluginMetrics struct {
	ExecutionCount  int64
	TotalDuration   time.Duration
	AverageDuration time.Duration
	SuccessCount    int64
	FailureCount    int64
	LastExecutedAt  time.Time
	URLsDiscovered  int64
	ItemsExtracted  int64
}

// PluginHealth represents the health status of a plugin
type PluginHealth struct {
	IsHealthy        bool
	LastError        string
	LastErrorAt      time.Time
	ConsecutiveFails int
}
