package services

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/playwright-community/playwright-go"
	camoufox "github.com/uzzalhcse/camoufox-go"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
)

// ========================================
// UTILITY FUNCTIONS
// ========================================

// EnsureScheme ensures a URL has a scheme (http/https).
// If no scheme is provided, defaults to http.
func EnsureScheme(rawURL string) (string, error) {
	if rawURL == "" {
		return "", nil
	}

	// Check if the URL starts with a scheme (http://, https://, etc.)
	if !strings.Contains(rawURL, "://") {
		// If there is no scheme, default to http
		rawURL = "http://" + rawURL
	}

	// Parse and validate the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	return parsedURL.String(), nil
}

// ParseProxyURL parses a proxy URL and extracts server, username, and password.
// Handles URLs like http://user:pass@host:port or http://host:port
// Returns: server (http://host:port), username, password, error
func ParseProxyURL(proxyURL string) (server, username, password string, err error) {
	if proxyURL == "" {
		return "", "", "", nil
	}

	// Ensure URL has scheme
	if !strings.Contains(proxyURL, "://") {
		proxyURL = "http://" + proxyURL
	}

	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to parse proxy URL: %w", err)
	}

	// Extract username and password from URL
	if parsedURL.User != nil {
		username = parsedURL.User.Username()
		password, _ = parsedURL.User.Password()
	}

	// Rebuild server URL without credentials
	server = fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	return server, username, password, nil
}

// ========================================
// CAMOUFOX CONFIGURATION
// ========================================

// BuildCamoufoxOptions creates camoufox.Options from a BrowserProfile.
// This is the unified configuration function used by both orchestrator and worker.
// Returns an error if proxy URL is invalid.
func BuildCamoufoxOptions(profile *models.BrowserProfile, headless bool) (camoufox.Options, error) {
	opts := camoufox.Options{
		OS:                   "linux", // Default to linux for server environments
		Headless:             headless,
		ForceScopeAccess:     true, // Required for CAPTCHA solving
		DisableCOOP:          true, // Required for cross-origin iframe access
		Humanize:             1.0,  // Default: enable human-like mouse movement (1 second max)
		IncludeDefaultAddons: true, // Default: include uBlock Origin
	}

	if profile == nil {
		return opts, nil
	}

	// Target OS for fingerprint
	if profile.TargetOS != "" {
		opts.OS = profile.TargetOS
	}

	// GeoIP for realistic geolocation
	if profile.GeoIP != "" {
		opts.GeoIP = profile.GeoIP
	}

	// Virtual headless mode (Xvfb)
	if profile.VirtualHeadless {
		opts.VirtualHeadless = true
		opts.Headless = false // Virtual headless runs as headed in Xvfb
	}

	// Force scope access for CAPTCHA - enforce if AutoCaptchaSolve is enabled
	// ForceScopeAccess and DisableCOOP are REQUIRED for CAPTCHA solving
	if profile.AutoCaptchaSolve {
		opts.ForceScopeAccess = true
		opts.DisableCOOP = true
	} else if !profile.ForceScopeAccess {
		opts.ForceScopeAccess = false
	}

	// Block images for faster loading
	if profile.BlockImages {
		opts.BlockImages = true
	}

	// Block WebGL for fingerprint protection
	if profile.BlockWebGL {
		opts.BlockWebGL = true
	}

	// Block WebRTC for IP protection
	if profile.DisableWebRTC {
		opts.BlockWebRTC = true
	}

	// Humanize setting (0 = disabled, > 0 = max duration in seconds)
	if profile.Humanize > 0 {
		opts.Humanize = profile.Humanize
	}

	// Include default addons (uBlock Origin)
	opts.IncludeDefaultAddons = profile.IncludeDefaultAddons

	// Enable browser caching
	if profile.EnableCache {
		opts.EnableCache = true
	}

	// User data directory for persistent sessions
	if profile.UserDataDir != "" {
		opts.UserDataDir = profile.UserDataDir
	}

	// Screen dimensions with variance
	if profile.ScreenWidth > 0 && profile.ScreenHeight > 0 {
		opts.Screen = &camoufox.Screen{
			MinWidth:  profile.ScreenWidth - 100,
			MaxWidth:  profile.ScreenWidth + 100,
			MinHeight: profile.ScreenHeight - 100,
			MaxHeight: profile.ScreenHeight + 100,
		}
	}

	// Timezone
	// If GeoIP is "auto", we let Camoufox detect the timezone from the IP
	if profile.GeoIP != "auto" && profile.Timezone != "" {
		opts.Timezone = profile.Timezone
	}

	// Locale/Languages
	// If GeoIP is "auto", we let Camoufox detect the locale from the IP
	if profile.GeoIP != "auto" {
		if profile.Locale != "" {
			opts.Locale = []string{profile.Locale}
		} else if len(profile.Languages) > 0 {
			opts.Locale = profile.Languages
		}
	}

	// Proxy configuration with URL validation
	if profile.ProxyEnabled && profile.ProxyServer != "" {
		proxyURL, err := EnsureScheme(profile.ProxyServer)
		if err != nil {
			return opts, fmt.Errorf("invalid proxy server URL: %w", err)
		}
		opts.Proxy = &camoufox.ProxyConfig{
			Server:   proxyURL,
			Username: profile.ProxyUsername,
			Password: profile.ProxyPassword,
		}
	}

	return opts, nil
}

// NewCamoufoxBrowser creates a configured Camoufox browser instance.
// Use this for both orchestrator manual launch and worker workflow execution.
func NewCamoufoxBrowser(profile *models.BrowserProfile, headless bool) (*camoufox.Camoufox, error) {
	opts, err := BuildCamoufoxOptions(profile, headless)
	if err != nil {
		return nil, err
	}

	cam, err := camoufox.NewBrowser(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create camoufox browser: %w", err)
	}
	return cam, nil
}

// ========================================
// PLAYWRIGHT CONFIGURATION
// ========================================

// BuildPlaywrightContextOptions creates Playwright context options from a BrowserProfile.
// This provides consistent fingerprinting across orchestrator and worker.
// Returns an error if proxy URL is invalid.
func BuildPlaywrightContextOptions(profile *models.BrowserProfile, fpService *FingerprintService) (playwright.BrowserNewContextOptions, error) {
	contextOpts := playwright.BrowserNewContextOptions{
		IgnoreHttpsErrors: playwright.Bool(true),
		JavaScriptEnabled: playwright.Bool(true),
	}

	if profile == nil {
		return contextOpts, nil
	}

	// Generate fingerprint with correct browser type
	if fpService != nil {
		targetOS := "linux"
		if profile.TargetOS != "" {
			targetOS = profile.TargetOS
		}

		browserType := profile.BrowserType
		if browserType == "" {
			browserType = "chromium"
		}

		maxWidth := profile.ScreenWidth
		if maxWidth == 0 {
			maxWidth = 1920
		}
		maxHeight := profile.ScreenHeight
		if maxHeight == 0 {
			maxHeight = 1080
		}

		fp, err := fpService.GenerateFingerprintWithBrowser(targetOS, browserType, maxWidth, maxHeight)
		if err == nil && fp != nil {
			// Apply user agent
			contextOpts.UserAgent = playwright.String(fp.Navigator.UserAgent)

			// Apply screen dimensions
			if fp.Screen.Width > 0 && fp.Screen.Height > 0 {
				contextOpts.Screen = &playwright.Size{
					Width:  fp.Screen.Width,
					Height: fp.Screen.Height,
				}
				contextOpts.Viewport = &playwright.Size{
					Width:  fp.Screen.InnerWidth,
					Height: fp.Screen.InnerHeight,
				}
			}

			// Apply locale
			if fp.Navigator.Language != "" {
				contextOpts.Locale = playwright.String(fp.Navigator.Language)
			}
		}
	}

	// Override with explicit profile settings
	if profile.ScreenWidth > 0 && profile.ScreenHeight > 0 {
		contextOpts.Screen = &playwright.Size{
			Width:  profile.ScreenWidth,
			Height: profile.ScreenHeight,
		}
		contextOpts.Viewport = &playwright.Size{
			Width:  profile.ScreenWidth,
			Height: profile.ScreenHeight,
		}
	}

	// Locale
	if profile.Locale != "" {
		contextOpts.Locale = playwright.String(profile.Locale)
	}

	// Timezone
	if profile.Timezone != "" {
		contextOpts.TimezoneId = playwright.String(profile.Timezone)
	}

	// Geolocation
	if profile.GeolocationLatitude != nil || profile.GeolocationLongitude != nil {
		lat := 0.0
		lng := 0.0
		if profile.GeolocationLatitude != nil {
			lat = *profile.GeolocationLatitude
		}
		if profile.GeolocationLongitude != nil {
			lng = *profile.GeolocationLongitude
		}
		contextOpts.Geolocation = &playwright.Geolocation{
			Latitude:  lat,
			Longitude: lng,
		}
		contextOpts.Permissions = []string{"geolocation"}
	}

	// Proxy configuration with URL validation
	if profile.ProxyEnabled && profile.ProxyServer != "" {
		proxyURL, err := EnsureScheme(profile.ProxyServer)
		if err != nil {
			return contextOpts, fmt.Errorf("invalid proxy server URL: %w", err)
		}
		contextOpts.Proxy = &playwright.Proxy{
			Server:   proxyURL,
			Username: playwright.String(profile.ProxyUsername),
			Password: playwright.String(profile.ProxyPassword),
		}
	}

	return contextOpts, nil
}

// BuildPlaywrightLaunchOptions creates Playwright launch options from a BrowserProfile.
func BuildPlaywrightLaunchOptions(profile *models.BrowserProfile, headless bool) playwright.BrowserTypeLaunchOptions {
	launchOpts := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
		Args: []string{
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-dev-shm-usage",
			"--disable-gpu",
		},
	}

	if profile == nil {
		return launchOpts
	}

	// Custom executable path
	if profile.ExecutablePath != "" {
		launchOpts.ExecutablePath = playwright.String(profile.ExecutablePath)
	}

	// Custom launch arguments
	if len(profile.LaunchArgs) > 0 {
		launchOpts.Args = append(launchOpts.Args, profile.LaunchArgs...)
	}

	return launchOpts
}

// GetPlaywrightBrowserType returns the browser type string for Playwright.
func GetPlaywrightBrowserType(profile *models.BrowserProfile) string {
	if profile == nil || profile.BrowserType == "" {
		return "chromium"
	}
	return profile.BrowserType
}

// ========================================
// CHROMEDP CONFIGURATION
// ========================================

// ChromedpOptions contains options for ChromeDP driver
type ChromedpOptions struct {
	Headless      bool
	UserAgent     string
	ProxyServer   string
	ExecPath      string
	WindowWidth   int
	WindowHeight  int
	ExtraArgs     []string
	DisableImages bool
}

// BuildChromedpOptions creates ChromeDP options from a BrowserProfile.
// Returns an error if proxy URL is invalid.
func BuildChromedpOptions(profile *models.BrowserProfile, headless bool) (ChromedpOptions, error) {
	opts := ChromedpOptions{
		Headless:     headless,
		WindowWidth:  1920,
		WindowHeight: 1080,
	}

	if profile == nil {
		return opts, nil
	}

	// Screen dimensions
	if profile.ScreenWidth > 0 {
		opts.WindowWidth = profile.ScreenWidth
	}
	if profile.ScreenHeight > 0 {
		opts.WindowHeight = profile.ScreenHeight
	}

	// Proxy with URL validation
	if profile.ProxyEnabled && profile.ProxyServer != "" {
		proxyURL, err := EnsureScheme(profile.ProxyServer)
		if err != nil {
			return opts, fmt.Errorf("invalid proxy server URL: %w", err)
		}
		opts.ProxyServer = proxyURL
	}

	// Executable path
	if profile.ExecutablePath != "" {
		opts.ExecPath = profile.ExecutablePath
	}

	// Launch args
	if len(profile.LaunchArgs) > 0 {
		opts.ExtraArgs = profile.LaunchArgs
	}

	// Block images
	if profile.BlockImages {
		opts.DisableImages = true
	}

	return opts, nil
}
