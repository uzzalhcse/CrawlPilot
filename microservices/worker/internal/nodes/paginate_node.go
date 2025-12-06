package nodes

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/driver"
	"go.uber.org/zap"
)

// PaginateNode handles multi-page crawling with automatic pagination
type PaginateNode struct{}

func NewPaginateNode() NodeExecutor {
	return &PaginateNode{}
}

func (n *PaginateNode) Type() string {
	return "paginate"
}

func (n *PaginateNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	// Required: selector for next button
	nextSelector, ok := node.Params["selector"].(string)
	if !ok || nextSelector == "" {
		return fmt.Errorf("selector is required for paginate node")
	}

	// Optional parameters
	maxPages := getIntParam(node.Params, "max_pages", 10)
	linkSelector := getStringParam(node.Params, "link_selector", "")
	waitBetweenPages := getIntParam(node.Params, "wait_between_pages", 1000)
	timeout := getIntParam(node.Params, "timeout", 30000)

	logger.Info("Starting pagination",
		zap.String("next_selector", nextSelector),
		zap.Int("max_pages", maxPages),
		zap.String("link_selector", linkSelector),
	)

	var allLinks []string
	pagesProcessed := 0
	baseURL := execCtx.Task.URL

	// Extract links from the FIRST page before pagination loop
	if linkSelector != "" {
		links, err := n.extractLinks(execCtx.Page, linkSelector, baseURL)
		if err != nil {
			logger.Warn("Failed to extract links from first page",
				zap.Error(err),
			)
		} else {
			allLinks = append(allLinks, links...)
			logger.Info("Extracted links from first page",
				zap.Int("links", len(links)),
			)
		}
		pagesProcessed = 1 // First page counts as processed
	}

	for pagesProcessed < maxPages {
		// Check if next button exists
		elements, err := execCtx.Page.QuerySelectorAll(nextSelector)
		if err != nil || len(elements) == 0 {
			logger.Info("No more pagination buttons found, stopping")
			break
		}

		// Check if button is visible
		// We can check visibility by trying to wait for it with a short timeout
		// or just assume if it's found it might be clickable.
		// Better: use WaitForSelector with Visible option
		err = execCtx.Page.WaitForSelector(nextSelector,
			driver.WithWaitTimeout(100*time.Millisecond),
			driver.WithState("visible"),
		)
		if err != nil {
			logger.Info("Pagination button not visible, stopping")
			break
		}

		// Click next button
		err = execCtx.Page.Click(nextSelector, driver.WithElementTimeout(time.Duration(timeout)*time.Millisecond))
		if err != nil {
			logger.Warn("Failed to click next button",
				zap.Int("page", pagesProcessed+1),
				zap.Error(err),
			)
			break
		}

		// Wait for page to load
		err = execCtx.Page.WaitForState("networkidle", driver.WithWaitTimeout(time.Duration(timeout)*time.Millisecond))
		if err != nil {
			// Fall back to domcontentloaded
			execCtx.Page.WaitForState("domcontentloaded", driver.WithWaitTimeout(time.Duration(timeout)*time.Millisecond))
		}

		// Wait between pages
		if waitBetweenPages > 0 {
			time.Sleep(time.Duration(waitBetweenPages) * time.Millisecond)
		}

		// Extract links from current page AFTER navigation
		if linkSelector != "" {
			links, err := n.extractLinks(execCtx.Page, linkSelector, baseURL)
			if err != nil {
				logger.Warn("Failed to extract links from page",
					zap.Int("page", pagesProcessed+1),
					zap.Error(err),
				)
			} else {
				allLinks = append(allLinks, links...)
				logger.Info("Extracted links from page",
					zap.Int("page", pagesProcessed+1),
					zap.Int("links", len(links)),
				)
			}
		}

		pagesProcessed++
	}

	// Get marker from params
	marker := getStringParam(node.Params, "marker", "")

	// Store discovered URLs with markers in Variables (same format as extract_links)
	discoveredURLs := make([]map[string]interface{}, 0, len(allLinks))
	for _, link := range allLinks {
		urlData := map[string]interface{}{
			"url": link,
		}
		if marker != "" {
			urlData["marker"] = marker
		}
		discoveredURLs = append(discoveredURLs, urlData)
	}

	// Store in Variables for compatibility with phase transitions
	if execCtx.Variables == nil {
		execCtx.Variables = make(map[string]interface{})
	}
	execCtx.Variables["discovered_urls"] = discoveredURLs

	// Also store in DiscoveredURLs for backward compatibility
	execCtx.DiscoveredURLs = append(execCtx.DiscoveredURLs, allLinks...)

	logger.Info("Pagination complete",
		zap.Int("pages_processed", pagesProcessed),
		zap.Int("total_links", len(allLinks)),
		zap.String("marker", marker),
	)

	return nil
}

// extractLinks extracts links from the page using the given selector
func (n *PaginateNode) extractLinks(page driver.Page, selector, baseURL string) ([]string, error) {
	elements, err := page.QuerySelectorAll(selector)
	if err != nil {
		return nil, err
	}

	// Parse base URL for resolving relative links
	base, err := url.Parse(baseURL)
	if err != nil {
		base = nil
	}

	var links []string
	seen := make(map[string]bool)

	for _, element := range elements {
		href, err := element.Attribute("href")
		if err != nil || href == "" {
			continue
		}

		// Clean and resolve URL
		href = strings.TrimSpace(href)

		// Skip javascript: and mailto: links
		if strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "mailto:") {
			continue
		}

		// Resolve relative URLs
		if base != nil && !strings.HasPrefix(href, "http") {
			resolved, err := base.Parse(href)
			if err != nil {
				continue
			}
			href = resolved.String()
		}

		// Deduplicate
		if !seen[href] {
			seen[href] = true
			links = append(links, href)
		}
	}

	return links, nil
}
