package models

import "time"

// BrowserProfile represents a browser configuration with fingerprint settings
type BrowserProfile struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status"` // active, inactive, archived, running
	Folder      string   `json:"folder,omitempty"`
	Tags        []string `json:"tags,omitempty"`

	// Driver Configuration
	DriverType     string   `json:"driver_type"`  // playwright, chromedp, http
	BrowserType    string   `json:"browser_type"` // chromium, firefox, webkit
	ExecutablePath string   `json:"executable_path,omitempty"`
	CDPEndpoint    string   `json:"cdp_endpoint,omitempty"`
	LaunchArgs     []string `json:"launch_args,omitempty"`

	// Fingerprint Settings
	UserAgent           string   `json:"user_agent"`
	Platform            string   `json:"platform"`
	ScreenWidth         int      `json:"screen_width"`
	ScreenHeight        int      `json:"screen_height"`
	Timezone            string   `json:"timezone,omitempty"`
	Locale              string   `json:"locale,omitempty"`
	Languages           []string `json:"languages,omitempty"`
	WebGLVendor         string   `json:"webgl_vendor,omitempty"`
	WebGLRenderer       string   `json:"webgl_renderer,omitempty"`
	CanvasNoise         bool     `json:"canvas_noise"`
	HardwareConcurrency int      `json:"hardware_concurrency"`
	DeviceMemory        int      `json:"device_memory"`
	Fonts               []string `json:"fonts,omitempty"`
	DoNotTrack          bool     `json:"do_not_track"`
	DisableWebRTC       bool     `json:"disable_webrtc"`

	// Geolocation
	GeolocationLatitude  *float64 `json:"geolocation_latitude,omitempty"`
	GeolocationLongitude *float64 `json:"geolocation_longitude,omitempty"`
	GeolocationAccuracy  *int     `json:"geolocation_accuracy,omitempty"`

	// Proxy Configuration
	ProxyEnabled  bool   `json:"proxy_enabled"`
	ProxyType     string `json:"proxy_type,omitempty"`
	ProxyServer   string `json:"proxy_server,omitempty"`
	ProxyUsername string `json:"proxy_username,omitempty"`
	ProxyPassword string `json:"proxy_password,omitempty"`

	// Metadata
	UsageCount int        `json:"usage_count"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Validate checks if the browser profile configuration is valid
func (p *BrowserProfile) Validate() error {
	if p.Name == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}

	// Validate driver type
	switch p.DriverType {
	case "playwright", "chromedp", "http", "":
		// Valid
	default:
		return &ValidationError{Field: "driver_type", Message: "invalid driver type: " + p.DriverType}
	}

	// Validate browser type
	switch p.BrowserType {
	case "chromium", "firefox", "webkit", "":
		// Valid
	default:
		return &ValidationError{Field: "browser_type", Message: "invalid browser type: " + p.BrowserType}
	}

	// Chromedp only works with chromium
	if p.DriverType == "chromedp" && p.BrowserType != "" && p.BrowserType != "chromium" {
		return &ValidationError{
			Field:   "driver_type",
			Message: "chromedp driver only supports chromium browser",
		}
	}

	return nil
}

// SetDefaults sets default values for empty fields
func (p *BrowserProfile) SetDefaults() {
	if p.Status == "" {
		p.Status = "active"
	}
	if p.DriverType == "" {
		p.DriverType = "playwright"
	}
	if p.BrowserType == "" {
		p.BrowserType = "chromium"
	}
	if p.Platform == "" {
		p.Platform = "Win32"
	}
	if p.ScreenWidth == 0 {
		p.ScreenWidth = 1920
	}
	if p.ScreenHeight == 0 {
		p.ScreenHeight = 1080
	}
	if p.HardwareConcurrency == 0 {
		p.HardwareConcurrency = 4
	}
	if p.DeviceMemory == 0 {
		p.DeviceMemory = 8
	}
	if len(p.Languages) == 0 {
		p.Languages = []string{"en-US", "en"}
	}
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
