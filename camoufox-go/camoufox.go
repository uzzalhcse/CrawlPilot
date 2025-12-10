// Package camoufox provides a Go interface for launching Camoufox anti-detect browser
// with playwright-go. It uses BrowserForge for realistic fingerprint generation.
//
// Camoufox is a stealthy, custom build of Firefox designed for web scraping and
// automation with robust anti-bot evasion capabilities.
//
// Basic usage:
//
//	browser, err := camoufox.NewBrowser(camoufox.Options{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer browser.Close()
//
//	page, _ := browser.NewPage()
//	page.Goto("https://example.com")
package camoufox

import (
	"fmt"

	"github.com/playwright-community/playwright-go"
)

// Camoufox wraps a Playwright browser with anti-detect capabilities
type Camoufox struct {
	pw          *playwright.Playwright
	browser     playwright.Browser
	fingerprint *Fingerprint
	options     *Options
	envVars     map[string]string
	virtDisplay *VirtualDisplay
}

// NewBrowser launches a new Camoufox browser instance with anti-detect fingerprinting.
// It generates a realistic browser fingerprint using BrowserForge and configures
// the Camoufox Firefox binary with the appropriate settings.
func NewBrowser(opts Options) (*Camoufox, error) {
	// Set defaults
	if opts.OS == "" {
		opts.OS = "linux"
	}
	if opts.ExecutablePath == "" {
		path, err := GetExecutablePath()
		if err != nil {
			return nil, err
		}
		opts.ExecutablePath = path
	}

	// Generate fingerprint via Python/BrowserForge
	fp, err := GenerateFingerprint(opts.OS, opts.Screen)
	if err != nil {
		return nil, fmt.Errorf("failed to generate fingerprint: %w", err)
	}

	// Build Camoufox config from fingerprint
	config := BuildConfig(fp, &opts)

	// Apply GeoIP if specified
	if opts.GeoIP != "" {
		geo, err := GeoIPLookup(opts.GeoIP, opts.Proxy)
		if err != nil && opts.Debug {
			fmt.Printf("[Debug] GeoIP lookup failed: %v\n", err)
		} else if geo != nil {
			ApplyGeolocation(geo, config, opts.BlockWebRTC)
			if opts.Debug {
				fmt.Printf("[Debug] GeoIP: %s -> %s, %s (%s) at (%.4f, %.4f)\n",
					geo.IP, geo.City, geo.Country, geo.Timezone, geo.Latitude, geo.Longitude)
			}
		}
	}

	// Debug output
	if opts.Debug {
		fmt.Printf("[Debug] Config keys: %d\n", len(config))
		for k, v := range config {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	// Handle virtual display (Linux only)
	var virtDisplay *VirtualDisplay
	headless := opts.Headless
	if opts.VirtualHeadless && IsLinux() {
		virtDisplay = NewVirtualDisplay(opts.Debug)
		displayStr, err := virtDisplay.Start()
		if err != nil {
			return nil, fmt.Errorf("failed to start virtual display: %w", err)
		}
		// Will be added to env vars
		config["_virtdisplay"] = displayStr
		headless = false // Run as headed in virtual display
	}

	// Get environment variables for config
	envVars := GetEnvVars(config, opts.OS)

	// Add virtual display to env if needed
	if virtDisplay != nil {
		envVars["DISPLAY"] = virtDisplay.Display()
	}

	// Initialize Playwright
	pw, err := playwright.Run()
	if err != nil {
		if virtDisplay != nil {
			virtDisplay.Stop()
		}
		return nil, fmt.Errorf("failed to start playwright: %w", err)
	}

	// Build Firefox user prefs
	firefoxPrefs := make(map[string]interface{})
	if opts.FirefoxPrefs != nil {
		for k, v := range opts.FirefoxPrefs {
			firefoxPrefs[k] = v
		}
	}

	// Apply Firefox pref options
	if opts.BlockImages {
		firefoxPrefs["permissions.default.image"] = 2
	}
	if opts.BlockWebRTC {
		firefoxPrefs["media.peerconnection.enabled"] = false
	}
	if opts.BlockWebGL {
		firefoxPrefs["webgl.disabled"] = true
	} else {
		firefoxPrefs["webgl.force-enabled"] = true
	}
	if opts.DisableCOOP {
		firefoxPrefs["browser.tabs.remote.useCrossOriginOpenerPolicy"] = false
	}
	if opts.EnableCache {
		firefoxPrefs["browser.sessionhistory.max_entries"] = 10
		firefoxPrefs["browser.cache.memory.enable"] = true
		firefoxPrefs["browser.cache.disk_cache_ssl"] = true
	}
	if opts.CustomFontsOnly {
		firefoxPrefs["gfx.bundled-fonts.activate"] = 0
	}

	// Build launch options
	launchOpts := playwright.BrowserTypeLaunchOptions{
		ExecutablePath:   playwright.String(opts.ExecutablePath),
		Headless:         playwright.Bool(headless),
		Env:              envVars,
		FirefoxUserPrefs: firefoxPrefs,
	}

	// Add extra args
	if len(opts.Args) > 0 {
		launchOpts.Args = opts.Args
	}

	// Add proxy if configured
	if opts.Proxy != nil && opts.Proxy.Server != "" {
		launchOpts.Proxy = &playwright.Proxy{
			Server: opts.Proxy.Server,
		}
		if opts.Proxy.Username != "" {
			launchOpts.Proxy.Username = playwright.String(opts.Proxy.Username)
			launchOpts.Proxy.Password = playwright.String(opts.Proxy.Password)
		}
	}

	// Launch Camoufox (Firefox-based)
	browser, err := pw.Firefox.Launch(launchOpts)
	if err != nil {
		pw.Stop()
		if virtDisplay != nil {
			virtDisplay.Stop()
		}
		return nil, fmt.Errorf("failed to launch camoufox: %w", err)
	}

	cam := &Camoufox{
		pw:          pw,
		browser:     browser,
		fingerprint: fp,
		options:     &opts,
		envVars:     envVars,
	}

	// Store virtual display for cleanup
	if virtDisplay != nil {
		cam.virtDisplay = virtDisplay
	}

	return cam, nil
}

// NewPage creates a new browser page with the configured fingerprint context
func (c *Camoufox) NewPage() (playwright.Page, error) {
	contextOpts := playwright.BrowserNewContextOptions{}

	// Apply fingerprint settings to context
	if c.fingerprint != nil {
		if c.fingerprint.Navigator.UserAgent != "" {
			contextOpts.UserAgent = playwright.String(c.fingerprint.Navigator.UserAgent)
		}
		if c.fingerprint.Screen.Width > 0 && c.fingerprint.Screen.Height > 0 {
			contextOpts.Viewport = &playwright.Size{
				Width:  c.fingerprint.Screen.Width,
				Height: c.fingerprint.Screen.Height,
			}
		}
		if c.fingerprint.Navigator.Language != "" {
			contextOpts.Locale = playwright.String(c.fingerprint.Navigator.Language)
		}
	}

	// Apply proxy to context if configured
	if c.options.Proxy != nil && c.options.Proxy.Server != "" {
		contextOpts.Proxy = &playwright.Proxy{
			Server: c.options.Proxy.Server,
		}
		if c.options.Proxy.Username != "" {
			contextOpts.Proxy.Username = playwright.String(c.options.Proxy.Username)
			contextOpts.Proxy.Password = playwright.String(c.options.Proxy.Password)
		}
	}

	ctx, err := c.browser.NewContext(contextOpts)
	if err != nil {
		return nil, err
	}

	return ctx.NewPage()
}

// NewContext creates a new browser context with fingerprint settings applied
func (c *Camoufox) NewContext() (playwright.BrowserContext, error) {
	contextOpts := playwright.BrowserNewContextOptions{}

	if c.fingerprint != nil {
		if c.fingerprint.Navigator.UserAgent != "" {
			contextOpts.UserAgent = playwright.String(c.fingerprint.Navigator.UserAgent)
		}
	}

	return c.browser.NewContext(contextOpts)
}

// Browser returns the underlying Playwright browser instance
func (c *Camoufox) Browser() playwright.Browser {
	return c.browser
}

// Fingerprint returns the generated fingerprint
func (c *Camoufox) Fingerprint() *Fingerprint {
	return c.fingerprint
}

// Close closes the browser and stops Playwright
func (c *Camoufox) Close() error {
	if c.browser != nil {
		if err := c.browser.Close(); err != nil {
			return err
		}
	}
	if c.pw != nil {
		if err := c.pw.Stop(); err != nil {
			return err
		}
	}
	// Stop virtual display if running
	if c.virtDisplay != nil {
		c.virtDisplay.Stop()
	}
	return nil
}
