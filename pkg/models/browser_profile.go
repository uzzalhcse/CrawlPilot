package models

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

// BrowserProfile represents a browser profile with fingerprinting configuration
type BrowserProfile struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status"` // active, inactive, archived
	Folder      *string  `json:"folder,omitempty"`
	Tags        []string `json:"tags,omitempty"`

	// Browser Configuration
	BrowserType    string   `json:"browser_type"` // chromium, firefox, webkit
	ExecutablePath *string  `json:"executable_path,omitempty"`
	CDPEndpoint    *string  `json:"cdp_endpoint,omitempty"`
	LaunchArgs     []string `json:"launch_args,omitempty"`

	// Fingerprint Configuration
	UserAgent    string   `json:"user_agent,omitempty"`
	Platform     string   `json:"platform,omitempty"`
	ScreenWidth  int      `json:"screen_width"`
	ScreenHeight int      `json:"screen_height"`
	Timezone     string   `json:"timezone,omitempty"`
	Locale       string   `json:"locale"`
	Languages    []string `json:"languages,omitempty"`

	// Advanced Fingerprinting
	WebGLVendor         *string  `json:"webgl_vendor,omitempty"`
	WebGLRenderer       *string  `json:"webgl_renderer,omitempty"`
	CanvasNoise         bool     `json:"canvas_noise"`
	HardwareConcurrency int      `json:"hardware_concurrency"`
	DeviceMemory        int      `json:"device_memory"`
	Fonts               []string `json:"fonts,omitempty"`

	// Privacy & Security
	DoNotTrack    bool     `json:"do_not_track"`
	DisableWebRTC bool     `json:"disable_webrtc"`
	GeoLatitude   *float64 `json:"geo_latitude,omitempty"`
	GeoLongitude  *float64 `json:"geo_longitude,omitempty"`
	GeoAccuracy   *int     `json:"geo_accuracy,omitempty"`

	// Proxy Configuration (basic)
	ProxyEnabled  bool    `json:"proxy_enabled"`
	ProxyType     *string `json:"proxy_type,omitempty"`   // http, https, socks5
	ProxyServer   *string `json:"proxy_server,omitempty"` // host:port
	ProxyUsername *string `json:"proxy_username,omitempty"`
	ProxyPassword *string `json:"proxy_password,omitempty"` // Consider encrypting

	// Cookies & Storage
	Cookies        *JSONMap `json:"cookies,omitempty"`
	LocalStorage   *JSONMap `json:"local_storage,omitempty"`
	SessionStorage *JSONMap `json:"session_storage,omitempty"`
	IndexedDB      *JSONMap `json:"indexed_db,omitempty"`
	ClearOnClose   bool     `json:"clear_on_close"`

	// Team & Sharing
	OwnerID     *string  `json:"owner_id,omitempty"`
	SharedWith  []string `json:"shared_with,omitempty"`
	Permissions *JSONMap `json:"permissions,omitempty"`

	// Statistics
	UsageCount int        `json:"usage_count"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BrowserProfileProxy represents a proxy configuration for rotation
type BrowserProfileProxy struct {
	ID        string `json:"id"`
	ProfileID string `json:"profile_id"`

	// Proxy Configuration
	ProxyType     string  `json:"proxy_type"` // http, https, socks5
	ProxyServer   string  `json:"proxy_server"`
	ProxyUsername *string `json:"proxy_username,omitempty"`
	ProxyPassword *string `json:"proxy_password,omitempty"`

	// Rotation Configuration
	Priority         int    `json:"priority"`
	RotationStrategy string `json:"rotation_strategy"` // round-robin, random, sticky
	IsActive         bool   `json:"is_active"`

	// Health Check
	LastHealthCheck     *time.Time `json:"last_health_check,omitempty"`
	HealthStatus        string     `json:"health_status"` // healthy, unhealthy, unknown
	FailureCount        int        `json:"failure_count"`
	SuccessCount        int        `json:"success_count"`
	AverageResponseTime *int       `json:"average_response_time,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate validates the browser profile configuration
func (bp *BrowserProfile) Validate() error {
	if strings.TrimSpace(bp.Name) == "" {
		return errors.New("profile name is required")
	}

	// Validate browser type
	validBrowserTypes := map[string]bool{"chromium": true, "firefox": true, "webkit": true}
	if !validBrowserTypes[bp.BrowserType] {
		return fmt.Errorf("invalid browser type: %s (must be chromium, firefox, or webkit)", bp.BrowserType)
	}

	// Validate CDP endpoint if provided
	if bp.CDPEndpoint != nil && *bp.CDPEndpoint != "" {
		if err := validateCDPEndpoint(*bp.CDPEndpoint); err != nil {
			return fmt.Errorf("invalid CDP endpoint: %w", err)
		}
	}

	// Validate executable path if provided
	if bp.ExecutablePath != nil && *bp.ExecutablePath != "" {
		if err := validateExecutablePath(*bp.ExecutablePath); err != nil {
			return fmt.Errorf("invalid executable path: %w", err)
		}
	}

	// Validate proxy configuration
	if bp.ProxyEnabled {
		if bp.ProxyType == nil || *bp.ProxyType == "" {
			return errors.New("proxy type is required when proxy is enabled")
		}
		if bp.ProxyServer == nil || *bp.ProxyServer == "" {
			return errors.New("proxy server is required when proxy is enabled")
		}
		validProxyTypes := map[string]bool{"http": true, "https": true, "socks5": true}
		if !validProxyTypes[*bp.ProxyType] {
			return fmt.Errorf("invalid proxy type: %s", *bp.ProxyType)
		}
	}

	// Validate screen resolution
	if bp.ScreenWidth <= 0 || bp.ScreenHeight <= 0 {
		return errors.New("screen width and height must be positive")
	}

	// Validate status
	validStatuses := map[string]bool{"active": true, "inactive": true, "archived": true}
	if !validStatuses[bp.Status] {
		return fmt.Errorf("invalid status: %s", bp.Status)
	}

	return nil
}

// validateCDPEndpoint validates a Chrome DevTools Protocol WebSocket endpoint
func validateCDPEndpoint(endpoint string) error {
	if !strings.HasPrefix(endpoint, "ws://") && !strings.HasPrefix(endpoint, "wss://") {
		return errors.New("CDP endpoint must start with ws:// or wss://")
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if u.Host == "" {
		return errors.New("CDP endpoint must include a host")
	}

	return nil
}

// validateExecutablePath validates that an executable path exists and is executable
func validateExecutablePath(path string) error {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("executable file does not exist")
		}
		return fmt.Errorf("cannot access executable: %w", err)
	}

	// Check if it's a regular file
	if info.IsDir() {
		return errors.New("path is a directory, not an executable")
	}

	// On Unix-like systems, check if executable bit is set
	if info.Mode()&0111 == 0 {
		return errors.New("file is not executable (missing execute permission)")
	}

	return nil
}

// DefaultBrowserProfile creates a browser profile with default settings
func DefaultBrowserProfile(name string) *BrowserProfile {
	return &BrowserProfile{
		Name:                name,
		Status:              "active",
		BrowserType:         "chromium",
		ScreenWidth:         1920,
		ScreenHeight:        1080,
		Locale:              "en-US",
		Languages:           []string{"en-US", "en"},
		CanvasNoise:         false,
		HardwareConcurrency: 4,
		DeviceMemory:        8,
		DoNotTrack:          false,
		DisableWebRTC:       false,
		ProxyEnabled:        false,
		ClearOnClose:        true,
		UsageCount:          0,
	}
}

// BrowserProfileCreateRequest represents a request to create a browser profile
type BrowserProfileCreateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Folder      *string  `json:"folder,omitempty"`
	Tags        []string `json:"tags,omitempty"`

	// Browser Configuration
	BrowserType    string   `json:"browser_type"`
	ExecutablePath *string  `json:"executable_path,omitempty"`
	CDPEndpoint    *string  `json:"cdp_endpoint,omitempty"`
	LaunchArgs     []string `json:"launch_args,omitempty"`

	// Fingerprint
	UserAgent           string   `json:"user_agent,omitempty"`
	Platform            string   `json:"platform,omitempty"`
	ScreenWidth         int      `json:"screen_width"`
	ScreenHeight        int      `json:"screen_height"`
	Timezone            string   `json:"timezone,omitempty"`
	Locale              string   `json:"locale"`
	Languages           []string `json:"languages,omitempty"`
	WebGLVendor         *string  `json:"webgl_vendor,omitempty"`
	WebGLRenderer       *string  `json:"webgl_renderer,omitempty"`
	CanvasNoise         bool     `json:"canvas_noise"`
	HardwareConcurrency int      `json:"hardware_concurrency"`
	DeviceMemory        int      `json:"device_memory"`
	Fonts               []string `json:"fonts,omitempty"`

	// Privacy
	DoNotTrack    bool     `json:"do_not_track"`
	DisableWebRTC bool     `json:"disable_webrtc"`
	GeoLatitude   *float64 `json:"geo_latitude,omitempty"`
	GeoLongitude  *float64 `json:"geo_longitude,omitempty"`
	GeoAccuracy   *int     `json:"geo_accuracy,omitempty"`

	// Proxy
	ProxyEnabled  bool    `json:"proxy_enabled"`
	ProxyType     *string `json:"proxy_type,omitempty"`
	ProxyServer   *string `json:"proxy_server,omitempty"`
	ProxyUsername *string `json:"proxy_username,omitempty"`
	ProxyPassword *string `json:"proxy_password,omitempty"`

	// Storage
	ClearOnClose bool `json:"clear_on_close"`
}

// BrowserProfileUpdateRequest represents a request to update a browser profile
type BrowserProfileUpdateRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Status      *string  `json:"status,omitempty"`
	Folder      *string  `json:"folder,omitempty"`
	Tags        []string `json:"tags,omitempty"`

	// Browser Configuration
	BrowserType    *string  `json:"browser_type,omitempty"`
	ExecutablePath *string  `json:"executable_path,omitempty"`
	CDPEndpoint    *string  `json:"cdp_endpoint,omitempty"`
	LaunchArgs     []string `json:"launch_args,omitempty"`

	// Fingerprint fields...
	UserAgent           *string  `json:"user_agent,omitempty"`
	Platform            *string  `json:"platform,omitempty"`
	ScreenWidth         *int     `json:"screen_width,omitempty"`
	ScreenHeight        *int     `json:"screen_height,omitempty"`
	Timezone            *string  `json:"timezone,omitempty"`
	Locale              *string  `json:"locale,omitempty"`
	Languages           []string `json:"languages,omitempty"`
	WebGLVendor         *string  `json:"webgl_vendor,omitempty"`
	WebGLRenderer       *string  `json:"webgl_renderer,omitempty"`
	CanvasNoise         *bool    `json:"canvas_noise,omitempty"`
	HardwareConcurrency *int     `json:"hardware_concurrency,omitempty"`
	DeviceMemory        *int     `json:"device_memory,omitempty"`
	Fonts               []string `json:"fonts,omitempty"`

	// Privacy
	DoNotTrack    *bool    `json:"do_not_track,omitempty"`
	DisableWebRTC *bool    `json:"disable_webrtc,omitempty"`
	GeoLatitude   *float64 `json:"geo_latitude,omitempty"`
	GeoLongitude  *float64 `json:"geo_longitude,omitempty"`
	GeoAccuracy   *int     `json:"geo_accuracy,omitempty"`

	// Proxy
	ProxyEnabled  *bool   `json:"proxy_enabled,omitempty"`
	ProxyType     *string `json:"proxy_type,omitempty"`
	ProxyServer   *string `json:"proxy_server,omitempty"`
	ProxyUsername *string `json:"proxy_username,omitempty"`
	ProxyPassword *string `json:"proxy_password,omitempty"`

	// Storage
	ClearOnClose *bool `json:"clear_on_close,omitempty"`
}
