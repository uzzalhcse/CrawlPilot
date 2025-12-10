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

	// VirtualHeadless uses Xvfb virtual display (Linux only).
	// This is useful for running headed browsers in headless environments.
	// Requires Xvfb to be installed: apt-get install xvfb
	VirtualHeadless bool

	// Window sets a fixed window size (width, height)
	Window *WindowSize

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

	// WebGLConfig forces a specific WebGL vendor/renderer pair
	// Use SampleWebGL() to get realistic values
	WebGLConfig *WebGLConfig

	// Addons is a list of Firefox addon paths to load
	Addons []string

	// IncludeDefaultAddons includes default addons (uBlock Origin)
	// Set to true to automatically download and include uBlock Origin
	IncludeDefaultAddons bool

	// ExcludeAddons excludes specific default addons
	ExcludeAddons []DefaultAddon

	// Fonts is a list of additional font family names to load
	Fonts []string

	// CustomFontsOnly disables OS-specific system fonts
	// Only the fonts in the Fonts field will be used
	CustomFontsOnly bool

	// EnableCache enables browser caching (uses more memory)
	EnableCache bool

	// DisableCOOP disables Cross-Origin-Opener-Policy
	// Useful for clicking elements in cross-origin iframes (e.g., Turnstile)
	DisableCOOP bool

	// MainWorldEval enables running scripts in the main world
	// To use, prepend "mw:" to scripts: page.evaluate("mw:" + script)
	MainWorldEval bool

	// UserDataDir is the path for persistent browser data (cookies, localStorage)
	// If set, browser will use persistent context mode
	UserDataDir string

	// ForceScopeAccess enables access to closed Shadow DOM roots via shadowRootUnl.
	// Required for CAPTCHA solving (e.g., Cloudflare Turnstile).
	// When enabled, closed shadow roots become accessible via element.shadowRootUnl
	ForceScopeAccess bool

	// Debug prints the config being sent to Camoufox
	Debug bool
}

// WindowSize specifies fixed window dimensions
type WindowSize struct {
	Width  int
	Height int
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
