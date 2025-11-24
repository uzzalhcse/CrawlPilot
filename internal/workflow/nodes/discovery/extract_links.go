package discovery

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/internal/extraction"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// ExtractLinksExecutor handles link extraction
type ExtractLinksExecutor struct {
	nodes.BaseNodeExecutor
}

// NewExtractLinksExecutor creates a new extract_links executor
func NewExtractLinksExecutor() *ExtractLinksExecutor {
	return &ExtractLinksExecutor{
		BaseNodeExecutor: nodes.BaseNodeExecutor{},
	}
}

// Type returns the node type
func (e *ExtractLinksExecutor) Type() models.NodeType {
	return models.NodeTypeExtractLinks
}

// Validate validates the node parameters
func (e *ExtractLinksExecutor) Validate(params map[string]interface{}) error {
	// selector is optional, defaults to "a"
	return nil
}

// Execute extracts links from the page
func (e *ExtractLinksExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	selector := nodes.GetStringParam(input.Params, "selector", "a")
	limit := nodes.GetIntParam(input.Params, "limit", 0)

	engine := extraction.NewExtractionEngine(input.BrowserContext.Page)
	links, err := engine.ExtractLinks(selector, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to extract links: %w", err)
	}

	return &nodes.ExecutionOutput{
		Result:         links,
		DiscoveredURLs: links,
		Metadata: map[string]interface{}{
			"count": len(links),
		},
	}, nil
}

// ValidateForHealthCheck performs health check validation for extract_links
func (e *ExtractLinksExecutor) ValidateForHealthCheck(ctx context.Context, input *nodes.ValidationInput) (*models.NodeValidationResult, error) {
	result := &models.NodeValidationResult{
		NodeType: string(models.NodeTypeExtractLinks),
		Status:   models.ValidationStatusPass,
		Metrics:  make(map[string]interface{}),
		Issues:   []models.ValidationIssue{},
	}

	// Get selector (defaults to "a")
	selector := nodes.GetStringParam(input.Params, "selector", "a")

	page := input.BrowserContext.Page
	locator := page.Locator(selector)
	count, _ := locator.Count()

	// Check if elements have valid href attributes
	validHrefs := 0
	sampleUrls := []string{}
	allDiscoveredURLs := []string{}

	// Check all elements (up to a reasonable limit)
	maxToCheck := min(count, 50)
	for i := 0; i < maxToCheck; i++ {
		href, _ := locator.Nth(i).GetAttribute("href")
		if href != "" && href != "#" {
			validHrefs++
			allDiscoveredURLs = append(allDiscoveredURLs, href)
			if len(sampleUrls) < 3 {
				sampleUrls = append(sampleUrls, href)
			}
		}
	}

	// Validate results
	if count == 0 {
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity:   "critical",
			Code:       "SELECTOR_NOT_FOUND",
			Message:    fmt.Sprintf("Selector '%s' returned no elements", selector),
			Selector:   selector,
			Suggestion: "Check if the selector is correct and elements exist on the page",
		})
		result.Status = models.ValidationStatusFail
	} else if validHrefs == 0 {
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity:   "critical",
			Code:       "NO_VALID_HREFS",
			Message:    "Elements found but none have valid href attributes",
			Selector:   selector,
			Suggestion: "Check if selector targets the correct link elements",
		})
		result.Status = models.ValidationStatusFail
	}

	// Add metrics
	result.Metrics["element_count"] = count
	result.Metrics["valid_hrefs"] = validHrefs
	result.Metrics["sample_urls"] = sampleUrls
	result.Metrics["discovered_urls"] = allDiscoveredURLs // For phase chaining

	return result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
