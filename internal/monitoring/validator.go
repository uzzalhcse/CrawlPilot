package monitoring

import (
	"context"
	"fmt"
	"strings"

	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// GenericValidator provides generic validation for any node type
type GenericValidator struct {
	nodeType models.NodeType
}

// NewGenericValidator creates a new generic validator
func NewGenericValidator(nodeType models.NodeType) *GenericValidator {
	return &GenericValidator{nodeType: nodeType}
}

// ValidateForMonitoring performs generic validation
func (v *GenericValidator) ValidateForMonitoring(ctx context.Context, input *nodes.ValidationInput) (*models.NodeValidationResult, error) {
	result := &models.NodeValidationResult{
		NodeType: string(v.nodeType),
		Status:   models.ValidationStatusPass,
		Metrics:  make(map[string]interface{}),
		Issues:   []models.ValidationIssue{},
	}

	// Extract and validate all selector parameters
	selectors := v.extractSelectorParams(input.Params)
	for paramName, selector := range selectors {
		issue := v.validateSelector(input.BrowserContext, paramName, selector)
		if issue != nil {
			result.Issues = append(result.Issues, *issue)
		}
	}

	// Add metrics
	result.Metrics["selectors_checked"] = len(selectors)

	// Determine status based on issues
	result.Status = v.determineStatus(result.Issues)

	return result, nil
}

// extractSelectorParams finds all selector-like parameters recursively
func (v *GenericValidator) extractSelectorParams(params map[string]interface{}) map[string]string {
	selectors := make(map[string]string)

	for key, value := range params {
		// Check if parameter name contains "selector"
		if strings.Contains(strings.ToLower(key), "selector") {
			if str, ok := value.(string); ok && str != "" {
				selectors[key] = str
			}
		}

		// Recursively check nested maps (for field configs in extract nodes)
		if nested, ok := value.(map[string]interface{}); ok {
			for nestedKey, nestedVal := range v.extractSelectorParams(nested) {
				selectors[key+"."+nestedKey] = nestedVal
			}
		}
	}

	return selectors
}

// validateSelector validates a single CSS selector
func (v *GenericValidator) validateSelector(browserCtx *browser.BrowserContext, paramName, selector string) *models.ValidationIssue {
	page := browserCtx.Page
	locator := page.Locator(selector)

	count, err := locator.Count()
	if err != nil {
		return &models.ValidationIssue{
			Severity:   "critical",
			Code:       "SELECTOR_ERROR",
			Message:    fmt.Sprintf("Selector '%s' caused error: %v", selector, err),
			Selector:   selector,
			Suggestion: "Check selector syntax",
		}
	}

	if count == 0 {
		return &models.ValidationIssue{
			Severity:   "critical",
			Code:       "SELECTOR_NOT_FOUND",
			Message:    fmt.Sprintf("Parameter '%s': selector returned no elements", paramName),
			Selector:   selector,
			Expected:   "> 0 elements",
			Actual:     0,
			Suggestion: "Verify selector is correct, website structure may have changed",
		}
	}

	return nil
}

// determineStatus determines validation status from issues
func (v *GenericValidator) determineStatus(issues []models.ValidationIssue) models.ValidationStatus {
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
