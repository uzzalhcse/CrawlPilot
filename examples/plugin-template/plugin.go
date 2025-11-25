package main

import (
	"context"

	"github.com/uzzalhcse/crawlify/pkg/models"
	"github.com/uzzalhcse/crawlify/pkg/plugins"
	"go.uber.org/zap"
)

// MyDiscoveryPlugin implements the DiscoveryPlugin interface
type MyDiscoveryPlugin struct {
	sdk *plugins.SDK
}

// NewDiscoveryPlugin creates a new instance of the plugin
// This function MUST be exported for the plugin loader to find it
func NewDiscoveryPlugin() plugins.DiscoveryPlugin {
	return &MyDiscoveryPlugin{
		sdk: plugins.NewSDK(nil), // SDK will be initialized by the loader
	}
}

// Info returns plugin metadata
func (p *MyDiscoveryPlugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		ID:          "my-discovery-plugin",
		Name:        "My Discovery Plugin",
		Version:     "1.0.0",
		Author:      "Your Name",
		AuthorEmail: "you@example.com",
		Description: "Template for discovery phase plugins",
		PhaseType:   models.PhaseTypeDiscovery,
		Repository:  "https://github.com/yourusername/my-plugin",
		License:     "MIT",
	}
}

// Discover executes the discovery logic
func (p *MyDiscoveryPlugin) Discover(ctx context.Context, input *plugins.DiscoveryInput) (*plugins.DiscoveryOutput, error) {
	// Initialize helpers
	browserHelpers := p.sdk.NewBrowserHelpers(input.BrowserContext)
	configHelpers := p.sdk.NewConfigHelpers()
	urlHelpers := p.sdk.NewURLHelpers()
	logger := p.sdk.NewLogger(p.Info().ID)

	// Get configuration
	linkSelector := configHelpers.GetString(input.Config, "link_selector", "a")
	maxLinks := configHelpers.GetInt(input.Config, "max_links", 100)

	logger.Info("Starting discovery",
		zap.String("url", input.URL),
		zap.String("selector", linkSelector))

	// Wait for page to load
	if err := browserHelpers.WaitForSelector("body", 10000); err != nil {
		return nil, err
	}

	// Extract links
	links, err := browserHelpers.ExtractLinks(linkSelector)
	if err != nil {
		logger.Error("Failed to extract links", err)
		return nil, err
	}

	// Normalize and filter URLs
	var discoveredURLs []string
	for _, link := range links {
		// Convert relative URLs to absolute
		if !urlHelpers.IsAbsoluteURL(link) {
			absoluteURL, err := urlHelpers.JoinURL(input.URL, link)
			if err != nil {
				continue
			}
			link = absoluteURL
		}

		// Normalize URL
		normalizedURL, err := urlHelpers.NormalizeURL(link)
		if err != nil {
			continue
		}

		discoveredURLs = append(discoveredURLs, normalizedURL)

		// Limit results
		if len(discoveredURLs) >= maxLinks {
			break
		}
	}

	logger.Info("Discovery complete",
		zap.Int("links_found", len(discoveredURLs)))

	return &plugins.DiscoveryOutput{
		DiscoveredURLs: discoveredURLs,
		Metadata: map[string]interface{}{
			"total_links": len(discoveredURLs),
			"selector":    linkSelector,
		},
	}, nil
}

// Validate checks if the configuration is valid
func (p *MyDiscoveryPlugin) Validate(config map[string]interface{}) error {
	// Add validation logic here
	// For example, check if required configuration fields are present
	return nil
}

// ConfigSchema returns the JSON Schema for plugin configuration
func (p *MyDiscoveryPlugin) ConfigSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"link_selector": map[string]interface{}{
				"type":        "string",
				"description": "CSS selector for links to discover",
				"default":     "a",
			},
			"max_links": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of links to discover",
				"default":     100,
				"minimum":     1,
				"maximum":     1000,
			},
		},
	}
}
