package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/uzzalhcse/crawlify/pkg/models"
	"github.com/uzzalhcse/crawlify/pkg/plugins"
	"go.uber.org/zap"
)

// EcommerceDiscoveryPlugin discovers product URLs from e-commerce websites
type EcommerceDiscoveryPlugin struct {
	sdk *plugins.SDK
}

// NewDiscoveryPlugin creates a new instance of the plugin
func NewDiscoveryPlugin() plugins.DiscoveryPlugin {
	return &EcommerceDiscoveryPlugin{
		sdk: plugins.NewSDK(nil),
	}
}

// Info returns plugin metadata
func (p *EcommerceDiscoveryPlugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		ID:          "ecommerce-product-discovery",
		Name:        "E-commerce Product Discovery",
		Version:     "1.0.0",
		Author:      "Crawlify Team",
		AuthorEmail: "team@crawlify.io",
		Description: "Discovers product URLs from e-commerce websites with pagination support",
		PhaseType:   models.PhaseTypeDiscovery,
		Repository:  "https://github.com/crawlify/plugins/ecommerce-discovery",
		License:     "MIT",
	}
}

// Discover executes the discovery logic
func (p *EcommerceDiscoveryPlugin) Discover(ctx context.Context, input *plugins.DiscoveryInput) (*plugins.DiscoveryOutput, error) {
	// Initialize helpers
	browserHelpers := p.sdk.NewBrowserHelpers(input.BrowserContext)
	configHelpers := p.sdk.NewConfigHelpers()
	urlHelpers := p.sdk.NewURLHelpers()
	logger := p.sdk.NewLogger(p.Info().ID)

	// Get configuration with defaults
	productSelector := configHelpers.GetString(input.Config, "product_selector", "a.product")
	nextPageSelector := configHelpers.GetString(input.Config, "next_page_selector", "a.next-page")
	maxProducts := configHelpers.GetInt(input.Config, "max_products", 100)
	maxPages := configHelpers.GetInt(input.Config, "max_pages", 5)
	waitAfterClick := configHelpers.GetInt(input.Config, "wait_after_click", 2000)
	urlPattern := configHelpers.GetString(input.Config, "url_pattern", "")

	logger.Info("Starting e-commerce product discovery",
		zap.String("url", input.URL),
		zap.String("product_selector", productSelector),
		zap.Int("max_products", maxProducts))

	var allProductURLs []string
	currentPage := 1

	for currentPage <= maxPages {
		logger.Info("Processing page",
			zap.Int("page", currentPage),
			zap.Int("products_found_so_far", len(allProductURLs)))

		// Wait for products to load
		err := browserHelpers.WaitForSelector(productSelector, 10000)
		if err != nil {
			logger.Warn("Product selector not found, stopping pagination",
				zap.Error(err),
				zap.Int("page", currentPage))
			break
		}

		// Extract product links
		links, err := browserHelpers.ExtractLinks(productSelector)
		if err != nil {
			return nil, fmt.Errorf("failed to extract product links: %w", err)
		}

		// Process and filter links
		for _, link := range links {
			// Convert to absolute URL
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

			// Apply URL pattern filter if specified
			if urlPattern != "" {
				matches, err := urlHelpers.MatchesPattern(normalizedURL, urlPattern)
				if err != nil || !matches {
					continue
				}
			}

			// Deduplicate
			if !contains(allProductURLs, normalizedURL) {
				allProductURLs = append(allProductURLs, normalizedURL)
			}

			// Check if we've reached the limit
			if len(allProductURLs) >= maxProducts {
				logger.Info("Reached maximum product limit", zap.Int("count", len(allProductURLs)))
				goto done
			}
		}

		// Try to click next page button
		if currentPage < maxPages {
			hasNextPage, err := p.clickNextPage(browserHelpers, nextPageSelector, waitAfterClick, logger)
			if err != nil {
				logger.Warn("Error checking for next page",
					zap.Error(err),
					zap.Int("page", currentPage))
				break
			}

			if !hasNextPage {
				logger.Info("No more pages available", zap.Int("last_page", currentPage))
				break
			}
		}

		currentPage++
	}

done:
	logger.Info("E-commerce discovery complete",
		zap.Int("total_products", len(allProductURLs)),
		zap.Int("pages_processed", currentPage))

	// Assign markers for discovered URLs
	urlMarkers := make(map[string]string)
	for _, url := range allProductURLs {
		urlMarkers[url] = "product"
	}

	return &plugins.DiscoveryOutput{
		DiscoveredURLs: allProductURLs,
		URLMarkers:     urlMarkers,
		Metadata: map[string]interface{}{
			"total_products":   len(allProductURLs),
			"pages_processed":  currentPage,
			"product_selector": productSelector,
		},
	}, nil
}

// clickNextPage attempts to click the next page button
func (p *EcommerceDiscoveryPlugin) clickNextPage(browserHelpers *plugins.BrowserHelpers, selector string, waitMs int, logger *plugins.Logger) (bool, error) {
	// Try to find next page button
	err := browserHelpers.WaitForSelector(selector, 3000)
	if err != nil {
		// No next page button found
		return false, nil
	}

	// Click the next page button
	if err := browserHelpers.Click(selector); err != nil {
		return false, fmt.Errorf("failed to click next page: %w", err)
	}

	// Wait for page to load
	if waitMs > 0 {
		browserHelpers.ScrollToBottom(0) // Just trigger navigation
		// In real implementation, would wait for network idle
	}

	logger.Debug("Clicked next page button")
	return true, nil
}

// Validate checks if the configuration is valid
func (p *EcommerceDiscoveryPlugin) Validate(config map[string]interface{}) error {
	configHelpers := p.sdk.NewConfigHelpers()

	// Product selector is required
	productSelector := configHelpers.GetString(config, "product_selector", "")
	if productSelector == "" {
		return fmt.Errorf("product_selector is required")
	}

	// Validate max_products range
	maxProducts := configHelpers.GetInt(config, "max_products", 100)
	if maxProducts < 1 || maxProducts > 10000 {
		return fmt.Errorf("max_products must be between 1 and 10000")
	}

	// Validate max_pages range
	maxPages := configHelpers.GetInt(config, "max_pages", 5)
	if maxPages < 1 || maxPages > 100 {
		return fmt.Errorf("max_pages must be between 1 and 100")
	}

	return nil
}

// ConfigSchema returns the JSON Schema for plugin configuration
func (p *EcommerceDiscoveryPlugin) ConfigSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":     "object",
		"required": []string{"product_selector"},
		"properties": map[string]interface{}{
			"product_selector": map[string]interface{}{
				"type":        "string",
				"description": "CSS selector for product links (required)",
				"examples":    []string{"a.product", ".product-item a", "div.product a"},
			},
			"next_page_selector": map[string]interface{}{
				"type":        "string",
				"description": "CSS selector for next page button",
				"default":     "a.next-page",
				"examples":    []string{"a.next", "button.pagination-next", ".paging-next"},
			},
			"max_products": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of products to discover",
				"default":     100,
				"minimum":     1,
				"maximum":     10000,
			},
			"max_pages": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of pages to process",
				"default":     5,
				"minimum":     1,
				"maximum":     100,
			},
			"wait_after_click": map[string]interface{}{
				"type":        "integer",
				"description": "Milliseconds to wait after clicking next page",
				"default":     2000,
				"minimum":     0,
				"maximum":     10000,
			},
			"url_pattern": map[string]interface{}{
				"type":        "string",
				"description": "Regex pattern to filter product URLs (optional)",
				"examples":    []string{`/product/\d+`, `/item/[a-z0-9-]+`},
			},
		},
	}
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
