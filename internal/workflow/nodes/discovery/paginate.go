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

// ValidateForHealthCheck performs health check validation for paginate
func (e *PaginateExecutor) ValidateForHealthCheck(ctx context.Context, input *nodes.ValidationInput) (*models.NodeValidationResult, error) {
	result := &models.NodeValidationResult{
		NodeType: string(models.NodeTypePaginate),
		Status:   models.ValidationStatusPass,
		Metrics:  make(map[string]interface{}),
		Issues:   []models.ValidationIssue{},
	}

	selector := nodes.GetStringParam(input.Params, "selector")
	linkSelector := nodes.GetStringParam(input.Params, "link_selector", "")
	maxPages := input.Config.MaxPaginationPages // Use health check config limit

	page := input.BrowserContext.Page

	// Check if pagination button exists
	locator := page.Locator(selector)
	count, _ := locator.Count()

	if count == 0 {
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity:   "warning",
			Code:       "PAGINATION_NOT_FOUND",
			Message:    fmt.Sprintf("Pagination selector '%s' not found", selector),
			Selector:   selector,
			Suggestion: "Check if pagination exists on this page or if selector is correct",
		})
		result.Status = models.ValidationStatusWarning
	}

	// Extract links from current page if link_selector is provided
	allDiscoveredURLs := []string{}
	sampleUrls := []string{}

	if linkSelector != "" {
		engine := extraction.NewExtractionEngine(page)

		// Test pagination for health check (limit to configured max pages)
		pagesChecked := 0
		for pagesChecked < maxPages && pagesChecked < 2 { // Max 2 pages for health check
			// Extract links from current page
			links, err := engine.ExtractLinks(linkSelector, 0)
			if err == nil && len(links) > 0 {
				allDiscoveredURLs = append(allDiscoveredURLs, links...)
				if len(sampleUrls) < 3 {
					for _, link := range links {
						if len(sampleUrls) < 3 {
							sampleUrls = append(sampleUrls, link)
						}
					}
				}
			}

			// Try to click next if it exists and we haven't reached max
			if count > 0 && pagesChecked < maxPages-1 {
				err = page.Click(selector, playwright.PageClickOptions{
					Timeout: playwright.Float(10000),
				})
				if err != nil {
					// Can't paginate further (expected at end)
					break
				}

				page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
					Timeout: playwright.Float(10000),
				})
				pagesChecked++
			} else {
				break
			}
		}
	}

	result.Metrics["pagination_button_found"] = count > 0
	result.Metrics["links_collected"] = len(allDiscoveredURLs)
	result.Metrics["sample_urls"] = sampleUrls
	result.Metrics["discovered_urls"] = allDiscoveredURLs // For phase chaining

	return result, nil
}
