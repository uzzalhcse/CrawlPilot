package browser

import (
	"context"
	"fmt"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// BrowserLauncher handles launching different browser types and connection methods
type BrowserLauncher struct {
	playwright *playwright.Playwright
}

func NewBrowserLauncher(pw *playwright.Playwright) *BrowserLauncher {
	return &BrowserLauncher{
		playwright: pw,
	}
}

// LaunchOptions contains configuration for browser launch
type LaunchOptions struct {
	Headless bool
	Timeout  float64
	Profile  *models.BrowserProfile
}

// LaunchBrowserWithProfile launches a browser based on the profile configuration
func (bl *BrowserLauncher) LaunchBrowserWithProfile(ctx context.Context, opts *LaunchOptions) (playwright.Browser, error) {
	if opts.Profile == nil {
		// No profile specified, launch default Chromium
		return bl.launchDefaultBrowser(opts.Headless, opts.Timeout)
	}

	profile := opts.Profile

	// If CDP endpoint is specified, connect via CDP instead of launching
	if profile.CDPEndpoint != nil && *profile.CDPEndpoint != "" {
		logger.Info("Connecting to browser via CDP",
			zap.String("endpoint", *profile.CDPEndpoint),
			zap.String("profile_id", profile.ID))
		return bl.connectViaCDP(*profile.CDPEndpoint)
	}

	// Launch based on browser type
	switch profile.BrowserType {
	case "chromium":
		return bl.launchChromium(profile, opts.Headless, opts.Timeout)
	case "firefox":
		return bl.launchFirefox(profile, opts.Headless, opts.Timeout)
	case "webkit":
		return bl.launchWebKit(profile, opts.Headless, opts.Timeout)
	default:
		return nil, fmt.Errorf("unsupported browser type: %s", profile.BrowserType)
	}
}

// launchDefaultBrowser launches default Chromium browser
func (bl *BrowserLauncher) launchDefaultBrowser(headless bool, timeout float64) (playwright.Browser, error) {
	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
		Timeout:  playwright.Float(timeout),
	}

	browser, err := bl.playwright.Chromium.Launch(launchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to launch default browser: %w", err)
	}

	logger.Info("Launched default Chromium browser")
	return browser, nil
}

// launchChromium launches Chromium browser with profile configuration
func (bl *BrowserLauncher) launchChromium(profile *models.BrowserProfile, headless bool, timeout float64) (playwright.Browser, error) {
	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
		Timeout:  playwright.Float(timeout),
	}

	// Set custom executable path if provided
	if profile.ExecutablePath != nil && *profile.ExecutablePath != "" {
		launchOptions.ExecutablePath = playwright.String(*profile.ExecutablePath)
		logger.Info("Using custom Chromium executable",
			zap.String("path", *profile.ExecutablePath))
	}

	// Add launch arguments
	if len(profile.LaunchArgs) > 0 {
		launchOptions.Args = profile.LaunchArgs
		logger.Info("Using custom launch args",
			zap.Strings("args", profile.LaunchArgs))
	}

	browser, err := bl.playwright.Chromium.Launch(launchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to launch Chromium: %w", err)
	}

	logger.Info("Launched Chromium browser",
		zap.String("profile_id", profile.ID),
		zap.String("profile_name", profile.Name))
	return browser, nil
}

// launchFirefox launches Firefox browser with profile configuration
func (bl *BrowserLauncher) launchFirefox(profile *models.BrowserProfile, headless bool, timeout float64) (playwright.Browser, error) {
	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
		Timeout:  playwright.Float(timeout),
	}

	// Set custom executable path if provided
	if profile.ExecutablePath != nil && *profile.ExecutablePath != "" {
		launchOptions.ExecutablePath = playwright.String(*profile.ExecutablePath)
		logger.Info("Using custom Firefox executable",
			zap.String("path", *profile.ExecutablePath))
	}

	// Add launch arguments
	if len(profile.LaunchArgs) > 0 {
		launchOptions.Args = profile.LaunchArgs
		logger.Info("Using custom launch args",
			zap.Strings("args", profile.LaunchArgs))
	}

	browser, err := bl.playwright.Firefox.Launch(launchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to launch Firefox: %w", err)
	}

	logger.Info("Launched Firefox browser",
		zap.String("profile_id", profile.ID),
		zap.String("profile_name", profile.Name))
	return browser, nil
}

// launchWebKit launches WebKit browser with profile configuration
func (bl *BrowserLauncher) launchWebKit(profile *models.BrowserProfile, headless bool, timeout float64) (playwright.Browser, error) {
	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
		Timeout:  playwright.Float(timeout),
	}

	// Set custom executable path if provided
	if profile.ExecutablePath != nil && *profile.ExecutablePath != "" {
		launchOptions.ExecutablePath = playwright.String(*profile.ExecutablePath)
		logger.Info("Using custom WebKit executable",
			zap.String("path", *profile.ExecutablePath))
	}

	// Add launch arguments
	if len(profile.LaunchArgs) > 0 {
		launchOptions.Args = profile.LaunchArgs
		logger.Info("Using custom launch args",
			zap.Strings("args", profile.LaunchArgs))
	}

	browser, err := bl.playwright.WebKit.Launch(launchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to launch WebKit: %w", err)
	}

	logger.Info("Launched WebKit browser",
		zap.String("profile_id", profile.ID),
		zap.String("profile_name", profile.Name))
	return browser, nil
}

// connectViaCDP connects to an existing browser via Chrome DevTools Protocol
func (bl *BrowserLauncher) connectViaCDP(endpoint string) (playwright.Browser, error) {
	logger.Info("Attempting CDP connection", zap.String("endpoint", endpoint))

	// Connect to browser via CDP WebSocket
	browser, err := bl.playwright.Chromium.ConnectOverCDP(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect via CDP: %w", err)
	}

	logger.Info("Successfully connected via CDP", zap.String("endpoint", endpoint))
	return browser, nil
}

// CreateContextWithFingerprint creates a browser context with fingerprinting applied
func (bl *BrowserLauncher) CreateContextWithFingerprint(browser playwright.Browser, profile *models.BrowserProfile) (playwright.BrowserContext, error) {
	options := playwright.BrowserNewContextOptions{
		JavaScriptEnabled: playwright.Bool(true),
		AcceptDownloads:   playwright.Bool(false),
		IgnoreHttpsErrors: playwright.Bool(true),
	}

	// Apply user agent
	if profile.UserAgent != "" {
		options.UserAgent = playwright.String(profile.UserAgent)
	}

	// Apply screen size
	if profile.ScreenWidth > 0 && profile.ScreenHeight > 0 {
		options.Viewport = &playwright.Size{
			Width:  profile.ScreenWidth,
			Height: profile.ScreenHeight,
		}
	}

	// Apply timezone
	if profile.Timezone != "" {
		options.TimezoneId = playwright.String(profile.Timezone)
	}

	// Apply locale
	if profile.Locale != "" {
		options.Locale = playwright.String(profile.Locale)
	}

	// Apply geolocation
	if profile.GeoLatitude != nil && profile.GeoLongitude != nil {
		options.Geolocation = &playwright.Geolocation{
			Latitude:  *profile.GeoLatitude,
			Longitude: *profile.GeoLongitude,
		}
		if profile.GeoAccuracy != nil {
			options.Geolocation.Accuracy = playwright.Float(float64(*profile.GeoAccuracy))
		}
		options.Permissions = []string{"geolocation"}
	}

	// Apply proxy if enabled
	if profile.ProxyEnabled && profile.ProxyServer != nil {
		proxyConfig := &playwright.Proxy{
			Server: *profile.ProxyServer,
		}
		if profile.ProxyUsername != nil && *profile.ProxyUsername != "" {
			proxyConfig.Username = profile.ProxyUsername
		}
		if profile.ProxyPassword != nil && *profile.ProxyPassword != "" {
			proxyConfig.Password = profile.ProxyPassword
		}
		options.Proxy = proxyConfig

		logger.Info("Using proxy configuration",
			zap.String("server", *profile.ProxyServer),
			zap.String("type", *profile.ProxyType))
	}

	// Create context
	ctx, err := browser.NewContext(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser context: %w", err)
	}

	// Apply additional fingerprinting via JavaScript injection
	if err := bl.applyAdvancedFingerprinting(ctx, profile); err != nil {
		logger.Warn("Failed to apply advanced fingerprinting, continuing anyway",
			zap.Error(err))
	}

	logger.Info("Created browser context with fingerprint",
		zap.String("profile_id", profile.ID),
		zap.String("user_agent", profile.UserAgent),
		zap.String("timezone", profile.Timezone))

	return ctx, nil
}

// applyAdvancedFingerprinting applies WebGL, canvas, and other fingerprinting via page injection
func (bl *BrowserLauncher) applyAdvancedFingerprinting(ctx playwright.BrowserContext, profile *models.BrowserProfile) error {
	// This would be called on each new page to inject fingerprint spoofing
	// We'll add init scripts that run before page load

	script := ""

	// Override navigator properties
	if profile.Platform != "" {
		script += fmt.Sprintf(`
			Object.defineProperty(navigator, 'platform', {
				get: () => '%s'
			});
		`, profile.Platform)
	}

	// Override hardware concurrency
	if profile.HardwareConcurrency > 0 {
		script += fmt.Sprintf(`
			Object.defineProperty(navigator, 'hardwareConcurrency', {
				get: () => %d
			});
		`, profile.HardwareConcurrency)
	}

	// Override device memory
	if profile.DeviceMemory > 0 {
		script += fmt.Sprintf(`
			Object.defineProperty(navigator, 'deviceMemory', {
				get: () => %d
			});
		`, profile.DeviceMemory)
	}

	// Override WebGL vendor and renderer
	if profile.WebGLVendor != nil || profile.WebGLRenderer != nil {
		vendor := "Intel Inc."
		renderer := "Intel Iris OpenGL Engine"
		if profile.WebGLVendor != nil {
			vendor = *profile.WebGLVendor
		}
		if profile.WebGLRenderer != nil {
			renderer = *profile.WebGLRenderer
		}

		script += fmt.Sprintf(`
			const getParameter = WebGLRenderingContext.prototype.getParameter;
			WebGLRenderingContext.prototype.getParameter = function(parameter) {
				if (parameter === 37445) return '%s';
				if (parameter === 37446) return '%s';
				return getParameter.apply(this, arguments);
			};
		`, vendor, renderer)
	}

	// Add canvas noise if enabled
	if profile.CanvasNoise {
		script += `
			const toDataURL = HTMLCanvasElement.prototype.toDataURL;
			HTMLCanvasElement.prototype.toDataURL = function() {
				const context = this.getContext('2d');
				if (context) {
					const imageData = context.getImageData(0, 0, this.width, this.height);
					for (let i = 0; i < imageData.data.length; i++) {
						imageData.data[i] += Math.floor(Math.random() * 3) - 1;
					}
					context.putImageData(imageData, 0, 0);
				}
				return toDataURL.apply(this, arguments);
			};
		`
	}

	// Disable WebRTC if requested
	if profile.DisableWebRTC {
		script += `
			window.RTCPeerConnection = undefined;
			window.RTCSessionDescription = undefined;
			window.RTCIceCandidate = undefined;
		`
	}

	if script != "" {
		if err := ctx.AddInitScript(playwright.Script{Content: playwright.String(script)}); err != nil {
			return fmt.Errorf("failed to add fingerprint script: %w", err)
		}
	}

	return nil
}
