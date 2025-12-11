package driver

import (
	"fmt"

	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
)

// Factory creates drivers based on configuration
type Factory struct {
	config *config.BrowserConfig
}

// NewFactory creates a new driver factory
func NewFactory(cfg *config.BrowserConfig) *Factory {
	return &Factory{
		config: cfg,
	}
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

	// Create driver based on type
	switch driverType {
	case "http":
		return NewHttpDriver(), nil
	case "chromedp":
		if profile != nil {
			return NewChromedpDriverWithProfile(cfg, profile)
		}
		return NewChromedpDriver(cfg), nil
	case "playwright", "":
		if profile != nil {
			return NewPlaywrightDriverWithProfile(cfg, profile)
		}
		return NewPlaywrightDriver(cfg)
	case "camoufox":
		if profile != nil {
			return NewCamoufoxDriverWithProfile(cfg, profile)
		}
		return NewCamoufoxDriver(cfg)
	default:
		return nil, fmt.Errorf("unknown driver type: %s", driverType)
	}
}
