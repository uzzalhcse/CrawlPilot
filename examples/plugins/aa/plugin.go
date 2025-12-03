package main

import (
	"context"

	"github.com/uzzalhcse/crawlify/pkg/plugins"
	"go.uber.org/zap"
)

// Plugin metadata
const (
	PluginName        = "aa"
	PluginVersion     = "1.0.0"
	PluginDescription = "aaa"
)

// Aa implements the DiscoveryPlugin interface
type Aa struct {
	logger *zap.Logger
}

// Info returns plugin metadata
func (p *Aa) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        PluginName,
		Version:     PluginVersion,
		Description: PluginDescription,
		Author:      "aa",
		PhaseType:   "discovery",
	}
}

// Discover performs URL discovery
func (p *Aa) Discover(ctx context.Context, input *plugins.DiscoveryInput) (*plugins.DiscoveryOutput, error) {
	p.logger.Info("Starting discovery", zap.String("url", input.URL))

	// TODO: Implement your discovery logic here
	// Example: Find product links
	var discoveredURLs []string

	// Use input.BrowserContext to interact with the page
	// page := input.BrowserContext.Page

	return &plugins.DiscoveryOutput{
		DiscoveredURLs: discoveredURLs,
		URLMarkers:     make(map[string]string),
		Metadata:       make(map[string]interface{}),
	}, nil
}

// Validate checks if the configuration is valid
func (p *Aa) Validate(config map[string]interface{}) error {
	return nil
}

// ConfigSchema returns the JSON schema for plugin configuration
func (p *Aa) ConfigSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// NewDiscoveryPlugin is the required exported function for plugin loading
func NewDiscoveryPlugin(logger *zap.Logger) (plugins.DiscoveryPlugin, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Aa{
		logger: logger,
	}, nil
}

// Ensure the plugin implements the interface at compile time
var _ plugins.DiscoveryPlugin = (*Aa)(nil)
