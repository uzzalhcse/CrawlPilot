package camoufox

// Options for launching Camoufox browser
type Options struct {
	// OS is the target operating system for fingerprint generation.
	// Valid values: "windows", "macos", "linux"
	// Default: "linux"
	OS string

	// Screen constraints for fingerprint generation
	Screen *Screen

	// Headless runs the browser in headless mode.
	// Default: false
	Headless bool

	// Humanize enables human-like mouse movement.
	// Set to true or a float representing max duration in seconds.
	// Default: false (0)
	Humanize float64

	// Proxy configuration for the browser
	Proxy *ProxyConfig

	// GeoIP calculates location based on IP address.
	// Pass an IP address string, or "auto" to detect automatically.
	GeoIP string

	// Locale(s) to use in Camoufox.
	// The first listed locale will be used for the Intl API.
	Locale []string

	// Timezone to use (e.g., "America/New_York")
	Timezone string

	// ExecutablePath is the path to the Camoufox executable.
	// If empty, it will attempt to find the installed Camoufox.
	ExecutablePath string

	// FirefoxPrefs are Firefox user preferences to set
	FirefoxPrefs map[string]interface{}

	// Args are additional command-line arguments for the browser
	Args []string

	// BlockImages blocks all images from loading
	BlockImages bool

	// BlockWebRTC blocks WebRTC entirely
	BlockWebRTC bool

	// BlockWebGL blocks WebGL (use sparingly, may cause detection)
	BlockWebGL bool

	// Addons is a list of Firefox addon paths to load
	Addons []string

	// Fonts is a list of additional font family names to load
	Fonts []string

	// Debug prints the config being sent to Camoufox
	Debug bool
}

// Screen constraints for fingerprint generation
type Screen struct {
	// MinWidth is the minimum screen width
	MinWidth int
	// MaxWidth is the maximum screen width
	MaxWidth int
	// MinHeight is the minimum screen height
	MinHeight int
	// MaxHeight is the maximum screen height
	MaxHeight int
}

// ProxyConfig holds proxy settings for the browser
type ProxyConfig struct {
	// Server is the proxy server URL (e.g., "http://host:port" or "socks5://host:port")
	Server string
	// Username for proxy authentication
	Username string
	// Password for proxy authentication
	Password string
}
