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

	// Get environment variables for config
	envVars := GetEnvVars(config, opts.OS)

	// Initialize Playwright
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to start playwright: %w", err)
	}

	// Build launch options
	launchOpts := playwright.BrowserTypeLaunchOptions{
		ExecutablePath: playwright.String(opts.ExecutablePath),
		Headless:       playwright.Bool(opts.Headless),
		Env:            envVars,
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
		return nil, fmt.Errorf("failed to launch camoufox: %w", err)
	}

	return &Camoufox{
		pw:          pw,
		browser:     browser,
		fingerprint: fp,
		options:     &opts,
		envVars:     envVars,
	}, nil
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
	return nil
}
