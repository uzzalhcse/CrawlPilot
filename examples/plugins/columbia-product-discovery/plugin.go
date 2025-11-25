package main

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/pkg/plugins"
	"go.uber.org/zap"
)

// ColumbiaDiscoveryPlugin discovers product links from Columbia Sportswear category pages
type ColumbiaDiscoveryPlugin struct {
	logger *zap.Logger
}

// Info returns plugin metadata
func (p *ColumbiaDiscoveryPlugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "Columbia Product Discovery",
		Version:     "1.0.0",
		Description: "Discovers product links from Columbia Sportswear category pages with pagination support",
		Author:      "Crawlify",
		PhaseType:   "discovery",
	}
}

// Discover extracts product links from Columbia category pages
func (p *ColumbiaDiscoveryPlugin) Discover(ctx context.Context, input *plugins.DiscoveryInput) (*plugins.DiscoveryOutput, error) {
	page := input.BrowserContext.Page

	p.logger.Info("Columbia discovery plugin starting",
		zap.String("url", input.URL),
	)

	var discoveredURLs []string

	// Extract product links from the current page
	// Using selector from tested columbia_crawler_phase.json workflow
	p.logger.Info("Columbia discovery plugin - attempting to find product links",
		zap.String("url", input.URL),
		zap.String("selector", "ul.block-thumbnail-t--items a"),
	)

	productLinks, err := page.Locator("ul.block-thumbnail-t--items a").All()
	if err != nil {
		p.logger.Warn("Failed to find product links",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find product links: %w", err)
	}

	p.logger.Info("Product links found",
		zap.Int("count", len(productLinks)),
	)

	// Get href attributes
	for _, link := range productLinks {
		href, err := link.GetAttribute("href")
		if err != nil {
			continue
		}

		// Convert relative URLs to absolute
		absoluteURL := href
		if len(href) > 0 && href[0] == '/' {
			absoluteURL = "https://www.columbia.com" + href
		}

		discoveredURLs = append(discoveredURLs, absoluteURL)

		// Respect the limit if set
		if config, ok := input.Config["limit"].(float64); ok {
			if len(discoveredURLs) >= int(config) {
				break
			}
		}
	}

	p.logger.Info("Columbia discovery completed",
		zap.Int("urls_discovered", len(discoveredURLs)),
	)

	return &plugins.DiscoveryOutput{
		DiscoveredURLs: discoveredURLs,
		Metadata: map[string]interface{}{
			"plugin":         "columbia-product-discovery",
			"page_url":       input.URL,
			"products_found": len(discoveredURLs),
		},
	}, nil
}

// ConfigSchema returns the configuration schema for the plugin
func (p *ColumbiaDiscoveryPlugin) ConfigSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "Maximum number of product links to extract",
				"default":     10,
			},
		},
	}
}

// Validate validates the plugin configuration
func (p *ColumbiaDiscoveryPlugin) Validate(config map[string]interface{}) error {
	// Config is optional for this plugin
	// If limit is provided, ensure it's a positive number
	if limit, ok := config["limit"]; ok {
		if limitNum, ok := limit.(float64); ok {
			if limitNum <= 0 {
				return fmt.Errorf("limit must be a positive number")
			}
		}
	}
	return nil
}

// NewDiscoveryPlugin is the required constructor function for discovery plugins
func NewDiscoveryPlugin(logger *zap.Logger) (plugins.DiscoveryPlugin, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ColumbiaDiscoveryPlugin{
		logger: logger,
	}, nil
}
