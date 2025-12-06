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

// CreateDriver creates a new driver instance based on default config
func (f *Factory) CreateDriver() (Driver, error) {
	switch f.config.Driver {
	case "http":
		return NewHttpDriver(), nil
	case "playwright", "": // Default to playwright
		return NewPlaywrightDriver(f.config)
	case "chromedp":
		return NewChromedpDriver(f.config), nil
	default:
		return nil, fmt.Errorf("unknown driver type: %s", f.config.Driver)
	}
}

// CreateDriverFromProfile creates a driver based on browser profile configuration
// This enables dynamic driver selection per-task based on profile settings
func (f *Factory) CreateDriverFromProfile(profile *models.BrowserProfile) (Driver, error) {
	if profile == nil {
		return f.CreateDriver()
	}

	// Validate driver/browser combinations
	if err := profile.Validate(); err != nil {
		return nil, err
	}

	switch profile.DriverType {
	case "http":
		return NewHttpDriver(), nil
	case "chromedp":
		// Chromedp only supports Chromium - validation done by profile.Validate()
		return NewChromedpDriverWithProfile(f.config, profile), nil
	case "playwright", "":
		// Playwright supports all browser types
		return NewPlaywrightDriverWithProfile(f.config, profile)
	default:
		return nil, fmt.Errorf("unknown driver type: %s", profile.DriverType)
	}
}
