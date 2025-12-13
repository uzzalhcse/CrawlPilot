package driver

import (
	"fmt"

	camoufox "github.com/uzzalhcse/camoufox-go"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/browser"
)

// Factory creates drivers based on configuration
type Factory struct {
	config         *config.BrowserConfig
	sessionChecker browser.SessionChecker // Session checker for session-aware drivers
}

// NewFactory creates a new driver factory
func NewFactory(cfg *config.BrowserConfig) *Factory {
	return &Factory{
		config: cfg,
	}
}

// SetSessionChecker sets the session checker for session-aware context selection
// All drivers created after this will have the session checker wired
func (f *Factory) SetSessionChecker(checker browser.SessionChecker) {
	f.sessionChecker = checker
}

// CreateDriver creates a new driver instance using default config settings
func (f *Factory) CreateDriver() (Driver, error) {
	return f.createDriverWithConfig(f.config, nil)
}

// CreateDriverFromProfile creates a driver using profile settings
// The config's headless setting is used by default
func (f *Factory) CreateDriverFromProfile(profile *models.BrowserProfile) (Driver, error) {
	if profile == nil {
		return f.CreateDriver()
	}
	return f.createDriverWithConfig(f.config, profile)
}

// CreateDriverFromProfileWithHeadless creates a driver with explicit headless override
// Use this for scrape requests where headless mode is specified per-request
func (f *Factory) CreateDriverFromProfileWithHeadless(profile *models.BrowserProfile, headless bool) (Driver, error) {
	// Create config copy with overridden headless
	cfgCopy := *f.config
	cfgCopy.Headless = headless

	if profile == nil {
		profile = &models.BrowserProfile{DriverType: f.config.Driver}
	}
	return f.createDriverWithConfig(&cfgCopy, profile)
}

// CreateDriverFromProfileWithProxy creates a driver with explicit headless and proxy override
// Use this for scrape requests where proxy is selected at runtime by SmartUnblocker
func (f *Factory) CreateDriverFromProfileWithProxy(profile *models.BrowserProfile, headless bool, proxyConfig *browser.ProxyConfig) (Driver, error) {
	// Create config copy with overridden headless
	cfgCopy := *f.config
	cfgCopy.Headless = headless

	if profile == nil {
		profile = &models.BrowserProfile{DriverType: f.config.Driver}
	}

	// If proxy config provided, inject into profile for browser creation
	if proxyConfig != nil && proxyConfig.Server != "" {
		// Clone profile to not modify the original
		profileCopy := *profile
		profileCopy.ProxyEnabled = true
		profileCopy.ProxyServer = proxyConfig.Server
		profileCopy.ProxyUsername = proxyConfig.Username
		profileCopy.ProxyPassword = proxyConfig.Password
		profile = &profileCopy
	}

	return f.createDriverWithConfig(&cfgCopy, profile)
}

// CreateCamoufoxWithFingerprint creates a Camoufox driver with a locked fingerprint.
// This is used for domain-locked CAPTCHA session sharing - all browsers for the same
// domain use the same fingerprint to ensure cookies remain valid.
func (f *Factory) CreateCamoufoxWithFingerprint(profile *models.BrowserProfile, fingerprint *camoufox.Fingerprint) (Driver, error) {
	d, err := NewCamoufoxDriverWithFingerprint(f.config, profile, fingerprint)
	if err != nil {
		return nil, err
	}
	// Apply session checker if set
	if f.sessionChecker != nil {
		d.SetSessionChecker(f.sessionChecker)
	}
	return d, nil
}

// createDriverWithConfig is the internal method that creates drivers
// All public methods delegate to this to avoid code duplication
func (f *Factory) createDriverWithConfig(cfg *config.BrowserConfig, profile *models.BrowserProfile) (Driver, error) {
	// Determine driver type - prefer profile, fall back to config
	driverType := cfg.Driver
	if profile != nil && profile.DriverType != "" {
		driverType = profile.DriverType
	}

	// Validate profile if provided
	if profile != nil && profile.Name != "" {
		if err := profile.Validate(); err != nil {
			return nil, err
		}
	}

	var d Driver
	var err error

	// Create driver based on type
	switch driverType {
	case "http":
		d = NewHttpDriver()
	case "chromedp":
		if profile != nil {
			d, err = NewChromedpDriverWithProfile(cfg, profile)
		} else {
			d = NewChromedpDriver(cfg)
		}
	case "playwright", "":
		if profile != nil {
			d, err = NewPlaywrightDriverWithProfile(cfg, profile)
		} else {
			d, err = NewPlaywrightDriver(cfg)
		}
	case "camoufox":
		if profile != nil {
			d, err = NewCamoufoxDriverWithProfile(cfg, profile)
		} else {
			d, err = NewCamoufoxDriver(cfg)
		}
	default:
		return nil, fmt.Errorf("unknown driver type: %s", driverType)
	}

	if err != nil {
		return nil, err
	}

	// Apply session checker if set
	if f.sessionChecker != nil {
		f.applySessionChecker(d)
	}

	return d, nil
}

// applySessionChecker applies the session checker to a driver if it supports it
func (f *Factory) applySessionChecker(d Driver) {
	switch drv := d.(type) {
	case *PlaywrightDriver:
		drv.SetSessionChecker(f.sessionChecker)
	case *CamoufoxDriver:
		drv.SetSessionChecker(f.sessionChecker)
	case *HttpDriver:
		drv.SetSessionChecker(f.sessionChecker)
	case *ChromedpDriver:
		// ChromedpDriver doesn't have SetSessionChecker yet - can add if needed
	}
}
