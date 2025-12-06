package driver

import (
	"fmt"

	"github.com/uzzalhcse/crawlify/microservices/shared/config"
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

// CreateDriver creates a new driver instance
func (f *Factory) CreateDriver() (Driver, error) {
	switch f.config.Driver {
	case "http":
		return NewHttpDriver(), nil
	case "playwright", "": // Default to playwright
		return NewPlaywrightDriver(f.config)
	default:
		return nil, fmt.Errorf("unknown driver type: %s", f.config.Driver)
	}
}
