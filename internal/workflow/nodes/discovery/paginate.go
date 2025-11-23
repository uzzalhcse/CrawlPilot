package discovery

import (
	"context"
	"fmt"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/internal/extraction"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// PaginateExecutor handles pagination
type PaginateExecutor struct {
	nodes.BaseNodeExecutor
}

// NewPaginateExecutor creates a new paginate executor
func NewPaginateExecutor() *PaginateExecutor {
	return &PaginateExecutor{
		BaseNodeExecutor: nodes.BaseNodeExecutor{},
	}
}

// Type returns the node type
func (e *PaginateExecutor) Type() models.NodeType {
	return models.NodeTypePaginate
}

// Validate validates the node parameters
func (e *PaginateExecutor) Validate(params map[string]interface{}) error {
	selector := nodes.GetStringParam(params, "selector")
	if selector == "" {
		return fmt.Errorf("selector is required for paginate node")
	}
	return nil
}

// Execute performs pagination
func (e *PaginateExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	selector := nodes.GetStringParam(input.Params, "selector")
	maxPages := nodes.GetIntParam(input.Params, "max_pages", 10)
	linkSelector := nodes.GetStringParam(input.Params, "link_selector", "")

	engine := extraction.NewExtractionEngine(input.BrowserContext.Page)
	var allLinks []string
	pagesProcessed := 0

	for pagesProcessed < maxPages {
		// Extract links from current page if link_selector is provided
		if linkSelector != "" {
			links, err := engine.ExtractLinks(linkSelector, 0)
			if err == nil {
				allLinks = append(allLinks, links...)
			}
		}

		// Check if next button exists
		locator := input.BrowserContext.Page.Locator(selector)
		count, err := locator.Count()
		if err != nil || count == 0 {
			// No more pages
			break
		}

		// Click next button
		err = input.BrowserContext.Page.Click(selector, playwright.PageClickOptions{
			Timeout: playwright.Float(30000),
		})
		if err != nil {
			// Failed to click, probably reached end
			break
		}

		// Wait for page to load (simple wait for navigation)
		err = input.BrowserContext.Page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			Timeout: playwright.Float(30000),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to wait for page load: %w", err)
		}

		pagesProcessed++
	}

	return &nodes.ExecutionOutput{
		Result:         allLinks,
		DiscoveredURLs: allLinks,
		Metadata: map[string]interface{}{
			"pages_processed": pagesProcessed,
			"links_found":     len(allLinks),
		},
	}, nil
}
