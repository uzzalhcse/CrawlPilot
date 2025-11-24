package validators

import (
	"context"

	"github.com/uzzalhcse/crawlify/internal/healthcheck"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// ExtractLinksValidator validates extract_links nodes
type ExtractLinksValidator struct {
	*healthcheck.GenericValidator
}

// NewExtractLinksValidator creates a new extract links validator
func NewExtractLinksValidator() *ExtractLinksValidator {
	return &ExtractLinksValidator{
		GenericValidator: healthcheck.NewGenericValidator(models.NodeTypeExtractLinks),
	}
}

// ValidateForHealthCheck performs validation specific to extract_links nodes
func (v *ExtractLinksValidator) ValidateForHealthCheck(ctx context.Context, input *nodes.ValidationInput) (*models.NodeValidationResult, error) {
	// Start with generic validation
	result, err := v.GenericValidator.ValidateForHealthCheck(ctx, input)
	if err != nil {
		return result, err
	}

	// Add link-specific validation
	selector := nodes.GetStringParam(input.Params, "selector")
	if selector == "" {
		// No selector parameter, skip link validation
		return result, nil
	}

	page := input.BrowserContext.Page
	locator := page.Locator(selector)
	count, _ := locator.Count()

	logger.Debug("extract_links validation",
		zap.String("selector", selector),
		zap.Int("element_count", count))

	// Check if elements have valid href attributes
	validHrefs := 0
	sampleUrls := []string{}
	allDiscoveredURLs := []string{}

	// Check all elements (up to a reasonable limit)
	maxToCheck := min(count, 50) // Check up to 50 links
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

	logger.Debug("extract_links results",
		zap.Int("valid_hrefs", validHrefs),
		zap.Int("total_discovered", len(allDiscoveredURLs)),
		zap.Strings("sample_urls", sampleUrls))

	if validHrefs == 0 && count > 0 {
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity:   "critical",
			Code:       "NO_VALID_HREFS",
			Message:    "Elements found but none have valid href attributes",
			Selector:   selector,
			Suggestion: "Check if selector targets the correct link elements",
		})
	}

	// Add metrics
	result.Metrics["element_count"] = count
	result.Metrics["valid_hrefs"] = validHrefs
	result.Metrics["sample_urls"] = sampleUrls
	result.Metrics["discovered_urls"] = allDiscoveredURLs // For phase chaining

	// Re-determine status after link validation
	result.Status = v.DetermineStatus(result.Issues)

	return result, nil
}

// DetermineStatus is exported for reuse
func (v *ExtractLinksValidator) DetermineStatus(issues []models.ValidationIssue) models.ValidationStatus {
	hasCritical := false
	hasWarning := false

	for _, issue := range issues {
		if issue.Severity == "critical" {
			hasCritical = true
		} else if issue.Severity == "warning" {
			hasWarning = true
		}
	}

	if hasCritical {
		return models.ValidationStatusFail
	}
	if hasWarning {
		return models.ValidationStatusWarning
	}
	return models.ValidationStatusPass
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
