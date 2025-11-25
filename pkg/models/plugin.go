package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Plugin represents a plugin in the marketplace
type Plugin struct {
	ID               string     `json:"id" db:"id"`
	Name             string     `json:"name" db:"name"`
	Slug             string     `json:"slug" db:"slug"` // URL-friendly identifier
	Description      string     `json:"description" db:"description"`
	AuthorName       string     `json:"author_name" db:"author_name"`
	AuthorEmail      string     `json:"author_email" db:"author_email"`
	RepositoryURL    string     `json:"repository_url" db:"repository_url"`
	DocumentationURL string     `json:"documentation_url" db:"documentation_url"`
	PhaseType        PhaseType  `json:"phase_type" db:"phase_type"`
	PluginType       PluginType `json:"plugin_type" db:"plugin_type"`
	Category         string     `json:"category" db:"category"`
	Tags             JSONArray  `json:"tags" db:"tags"`
	IsVerified       bool       `json:"is_verified" db:"is_verified"`
	TotalDownloads   int        `json:"total_downloads" db:"total_downloads"`
	TotalInstalls    int        `json:"total_installs" db:"total_installs"`
	AverageRating    float64    `json:"average_rating" db:"average_rating"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// PluginType defines the type of plugin
type PluginType string

const (
	PluginTypeBuiltIn   PluginType = "builtin"   // Core plugins shipped with Crawlify
	PluginTypeOfficial  PluginType = "official"  // Verified plugins from Crawlify team
	PluginTypeCommunity PluginType = "community" // Third-party plugins
	PluginTypePrivate   PluginType = "private"   // Organization-specific plugins
)

// PluginVersion represents a specific version of a plugin
type PluginVersion struct {
	ID                 string `json:"id" db:"id"`
	PluginID           string `json:"plugin_id" db:"plugin_id"`
	Version            string `json:"version" db:"version"` // Semantic version: 1.2.3
	Changelog          string `json:"changelog" db:"changelog"`
	IsStable           bool   `json:"is_stable" db:"is_stable"`
	MinCrawlifyVersion string `json:"min_crawlify_version" db:"min_crawlify_version"`

	// Platform-specific binaries
	LinuxAmd64BinaryPath  string `json:"linux_amd64_binary_path" db:"linux_amd64_binary_path"`
	LinuxArm64BinaryPath  string `json:"linux_arm64_binary_path" db:"linux_arm64_binary_path"`
	DarwinAmd64BinaryPath string `json:"darwin_amd64_binary_path" db:"darwin_amd64_binary_path"`
	DarwinArm64BinaryPath string `json:"darwin_arm64_binary_path" db:"darwin_arm64_binary_path"`

	// Binary metadata
	BinaryHash      string `json:"binary_hash" db:"binary_hash"` // SHA-256 hash
	BinarySizeBytes int64  `json:"binary_size_bytes" db:"binary_size_bytes"`

	// Configuration schema (JSON Schema for plugin config)
	ConfigSchema JSONObject `json:"config_schema" db:"config_schema"`

	Downloads   int       `json:"downloads" db:"downloads"`
	PublishedAt time.Time `json:"published_at" db:"published_at"`
}

// PluginInstallation tracks plugin installations
type PluginInstallation struct {
	ID              string     `json:"id" db:"id"`
	PluginID        string     `json:"plugin_id" db:"plugin_id"`
	PluginVersionID string     `json:"plugin_version_id" db:"plugin_version_id"`
	WorkspaceID     string     `json:"workspace_id" db:"workspace_id"`
	InstalledAt     time.Time  `json:"installed_at" db:"installed_at"`
	LastUsedAt      *time.Time `json:"last_used_at" db:"last_used_at"`
	UsageCount      int        `json:"usage_count" db:"usage_count"`
}

// PluginReview represents a user review for a plugin
type PluginReview struct {
	ID         string    `json:"id" db:"id"`
	PluginID   string    `json:"plugin_id" db:"plugin_id"`
	UserID     string    `json:"user_id" db:"user_id"`
	Rating     int       `json:"rating" db:"rating"` // 1-5
	ReviewText string    `json:"review_text" db:"review_text"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// PluginFilters for searching/filtering plugins
type PluginFilters struct {
	Query        string     // Search query (deprecated, use SearchQuery)
	SearchQuery  string     // Search query for name/description
	Category     string     // Filter by category
	PhaseType    PhaseType  // Filter by phase type
	PluginType   PluginType // Filter by plugin type
	Tags         []string   // Filter by tags
	IsVerified   *bool      // Filter verified plugins (pointer for optional)
	VerifiedOnly bool       // Filter verified plugins only (boolean)
	SortBy       string     // Sort field: popular, recent, rating, name
	SortOrder    string     // asc, desc
	Limit        int
	Offset       int
}

// JSONArray is a custom type for JSON array in database
type JSONArray []string

// Scan implements sql.Scanner for JSONArray
func (ja *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*ja = []string{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		var err error
		bytes, err = json.Marshal(v)
		if err != nil {
			return err
		}
	}

	return json.Unmarshal(bytes, ja)
}

// Value implements driver.Valuer for JSONArray
func (ja JSONArray) Value() (driver.Value, error) {
	if len(ja) == 0 {
		return "[]", nil
	}
	return json.Marshal(ja)
}

// JSONObject is a custom type for JSON object in database
type JSONObject map[string]interface{}

// Scan implements sql.Scanner for JSONObject
func (jo *JSONObject) Scan(value interface{}) error {
	if value == nil {
		*jo = make(map[string]interface{})
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		bytes = b
	}

	return json.Unmarshal(bytes, jo)
}

// Value implements driver.Valuer for JSONObject
func (jo JSONObject) Value() (driver.Value, error) {
	if len(jo) == 0 {
		return "{}", nil
	}
	return json.Marshal(jo)
}

// PlatformInfo represents platform and architecture information
type PlatformInfo struct {
	OS   string // linux, darwin
	Arch string // amd64, arm64
}

// GetBinaryPath returns the appropriate binary path for platform
func (pv *PluginVersion) GetBinaryPath(platform PlatformInfo) string {
	switch platform.OS {
	case "linux":
		if platform.Arch == "arm64" {
			return pv.LinuxArm64BinaryPath
		}
		return pv.LinuxAmd64BinaryPath
	case "darwin":
		if platform.Arch == "arm64" {
			return pv.DarwinArm64BinaryPath
		}
		return pv.DarwinAmd64BinaryPath
	default:
		return ""
	}
}
