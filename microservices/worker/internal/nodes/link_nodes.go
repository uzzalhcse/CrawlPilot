package nodes

import (
	"context"
	"fmt"

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

	// Find all links
	links := execCtx.Page.Locator(selector)
	count, err := links.Count()
	if err != nil {
		return fmt.Errorf("failed to count links: %w", err)
	}

	// Apply limit if specified
	if limit > 0 && count > limit {
		count = limit
	}

	discoveredURLs := make([]map[string]interface{}, 0)

	for i := 0; i < count; i++ {
		link := links.Nth(i)
		href, err := link.GetAttribute("href")
		if err != nil {
			continue
		}

		if href != "" {
			urlData := map[string]interface{}{
				"url": href,
			}
			if marker != "" {
				urlData["marker"] = marker
			}
			discoveredURLs = append(discoveredURLs, urlData)
		}
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
