package camoufox

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/playwright-community/playwright-go"
)

// PersistentBrowser wraps a browser context with persistent storage
type PersistentBrowser struct {
	*Camoufox
	context     playwright.BrowserContext
	userDataDir string
}

// NewPersistentBrowser creates a browser with persistent context (cookies, storage, etc.)
func NewPersistentBrowser(opts Options, userDataDir string) (*PersistentBrowser, error) {
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

	// Create user data directory if it doesn't exist
	if userDataDir == "" {
		homeDir, _ := os.UserHomeDir()
		userDataDir = filepath.Join(homeDir, ".cache", "camoufox-go", "profiles", "default")
	}
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create user data directory: %w", err)
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
		if err == nil && geo != nil {
			ApplyGeolocation(geo, config, opts.BlockWebRTC)
		}
	}

	// Apply fonts (same as NewBrowser)
	if !opts.CustomFontsOnly {
		osFonts := GetFontsForOS(opts.OS)
		if len(opts.Fonts) > 0 {
			config["fonts"] = MergeFonts(opts.Fonts, osFonts)
		} else {
			config["fonts"] = osFonts
		}
	} else if len(opts.Fonts) > 0 {
		config["fonts"] = opts.Fonts
	}

	// Apply locale if specified
	if len(opts.Locale) > 0 {
		config["navigator.language"] = opts.Locale[0]
		config["navigator.languages"] = opts.Locale
	}

	// Handle default addons
	if opts.IncludeDefaultAddons {
		am, err := NewAddonManager()
		if err == nil {
			defaultAddons, err := am.GetDefaultAddons(opts.ExcludeAddons...)
			if err == nil && len(defaultAddons) > 0 {
				opts.Addons = append(opts.Addons, defaultAddons...)
			}
		}
	}
	if len(opts.Addons) > 0 {
		config["addons"] = opts.Addons
	}

	// Get fixed UserAgent from config for HTTP headers
	fixedUA := ""
	if ua, ok := config["navigator.userAgent"].(string); ok {
		fixedUA = ua
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
		config["_virtdisplay"] = displayStr
		headless = false
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

	// Build persistent context options
	contextOpts := playwright.BrowserTypeLaunchPersistentContextOptions{
		ExecutablePath:   playwright.String(opts.ExecutablePath),
		Headless:         playwright.Bool(headless),
		Env:              envVars,
		FirefoxUserPrefs: firefoxPrefs,
	}

	// Apply fixed UserAgent to context (for HTTP headers)
	if fixedUA != "" {
		contextOpts.UserAgent = playwright.String(fixedUA)
	} else if fp.Navigator.UserAgent != "" {
		contextOpts.UserAgent = playwright.String(fp.Navigator.UserAgent)
	}
	if fp.Screen.Width > 0 && fp.Screen.Height > 0 {
		contextOpts.Viewport = &playwright.Size{
			Width:  fp.Screen.Width,
			Height: fp.Screen.Height,
		}
	}
	if fp.Navigator.Language != "" {
		contextOpts.Locale = playwright.String(fp.Navigator.Language)
	}

	// Add proxy if configured
	if opts.Proxy != nil && opts.Proxy.Server != "" {
		contextOpts.Proxy = &playwright.Proxy{
			Server: opts.Proxy.Server,
		}
		if opts.Proxy.Username != "" {
			contextOpts.Proxy.Username = playwright.String(opts.Proxy.Username)
			contextOpts.Proxy.Password = playwright.String(opts.Proxy.Password)
		}
	}

	// Launch persistent context
	context, err := pw.Firefox.LaunchPersistentContext(userDataDir, contextOpts)
	if err != nil {
		pw.Stop()
		if virtDisplay != nil {
			virtDisplay.Stop()
		}
		return nil, fmt.Errorf("failed to launch persistent context: %w", err)
	}

	return &PersistentBrowser{
		Camoufox: &Camoufox{
			pw:          pw,
			fingerprint: fp,
			options:     &opts,
			envVars:     envVars,
			virtDisplay: virtDisplay,
		},
		context:     context,
		userDataDir: userDataDir,
	}, nil
}

// NewPage creates a new page in the persistent context
func (pb *PersistentBrowser) NewPage() (playwright.Page, error) {
	return pb.context.NewPage()
}

// Context returns the browser context
func (pb *PersistentBrowser) Context() playwright.BrowserContext {
	return pb.context
}

// UserDataDir returns the user data directory path
func (pb *PersistentBrowser) UserDataDir() string {
	return pb.userDataDir
}

// Close closes the persistent browser
func (pb *PersistentBrowser) Close() error {
	if pb.context != nil {
		if err := pb.context.Close(); err != nil {
			return err
		}
	}
	if pb.pw != nil {
		if err := pb.pw.Stop(); err != nil {
			return err
		}
	}
	if pb.virtDisplay != nil {
		pb.virtDisplay.Stop()
	}
	return nil
}

// ClearStorage clears all storage data (cookies, localStorage, etc.)
func (pb *PersistentBrowser) ClearStorage() error {
	return pb.context.ClearCookies()
}
