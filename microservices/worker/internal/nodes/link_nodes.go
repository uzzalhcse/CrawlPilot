package nodes

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// ExtractLinksNode discovers and extracts URLs from the page
type ExtractLinksNode struct{}

func NewExtractLinksNode() NodeExecutor {
	return &ExtractLinksNode{}
}

func (n *ExtractLinksNode) Type() string {
	return "extract_links"
}

func (n *ExtractLinksNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	selector := "a[href]" // Default to all links
	if s, ok := node.Params["selector"].(string); ok && s != "" {
		selector = s
	}

	// Get marker to tag discovered URLs
	marker := ""
	if m, ok := node.Params["marker"].(string); ok {
		marker = m
	}

	// Get limit
	limit := 0
	if l, ok := node.Params["limit"].(float64); ok {
		limit = int(l)
	}

	logger.Info("Extracting links",
		zap.String("selector", selector),
		zap.String("marker", marker),
		zap.Int("limit", limit),
	)

	// Get current page URL for resolving relative URLs
	currentURL, err := execCtx.Page.URL()
	if err != nil {
		// If URL retrieval fails, we might still proceed if we don't need to resolve relative URLs
		// But usually we need it. Let's log warning.
		logger.Warn("Failed to get current URL", zap.Error(err))
		currentURL = ""
	}

	var baseURL *url.URL
	if currentURL != "" {
		baseURL, err = url.Parse(currentURL)
		if err != nil {
			logger.Warn("Failed to parse current URL", zap.String("url", currentURL), zap.Error(err))
			baseURL = nil
		}
	}

	// Find all links
	elements, err := execCtx.Page.QuerySelectorAll(selector)
	if err != nil {
		return fmt.Errorf("failed to find links: %w", err)
	}

	count := len(elements)

	// Apply limit if specified
	if limit > 0 && count > limit {
		count = limit
		elements = elements[:limit]
	}

	discoveredURLs := make([]map[string]interface{}, 0)

	for _, element := range elements {
		href, err := element.Attribute("href")
		if err != nil {
			continue
		}

		if href == "" {
			continue
		}

		// Skip javascript:, mailto:, tel:, # anchors
		if strings.HasPrefix(href, "javascript:") ||
			strings.HasPrefix(href, "mailto:") ||
			strings.HasPrefix(href, "tel:") ||
			href == "#" {
			continue
		}

		// Resolve relative URLs to absolute
		absoluteURL := href
		if baseURL != nil && !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
			parsedHref, err := url.Parse(href)
			if err == nil {
				absoluteURL = baseURL.ResolveReference(parsedHref).String()
			}
		}

		urlData := map[string]interface{}{
			"url": absoluteURL,
		}
		if marker != "" {
			urlData["marker"] = marker
		}
		discoveredURLs = append(discoveredURLs, urlData)
	}

	// Store discovered URLs in execution context
	if execCtx.Variables == nil {
		execCtx.Variables = make(map[string]interface{})
	}
	execCtx.Variables["discovered_urls"] = discoveredURLs

	logger.Info("Links extracted",
		zap.Int("count", len(discoveredURLs)),
		zap.String("marker", marker),
	)

	return nil
}
