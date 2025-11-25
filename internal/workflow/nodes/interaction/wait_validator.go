package interaction

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// ValidateForMonitoring performs monitoring validation for wait nodes
func (e *WaitExecutor) ValidateForMonitoring(ctx context.Context, input *nodes.ValidationInput) (*models.NodeValidationResult, error) {
	result := &models.NodeValidationResult{
		NodeType: string(models.NodeTypeWait),
		Status:   models.ValidationStatusPass,
		Metrics:  make(map[string]interface{}),
		Issues:   []models.ValidationIssue{},
	}

	selector := nodes.GetStringParam(input.Params, "selector")
	if selector == "" {
		// No selector to validate
		result.Metrics["validation_type"] = "timeout_only"
		return result, nil
	}

	state := nodes.GetStringParam(input.Params, "state", "visible")
	timeout := nodes.GetIntParam(input.Params, "timeout", 30000)

	page := input.BrowserContext.Page
	locator := page.Locator(selector)

	// Check if element exists
	count, _ := locator.Count()

	if count == 0 {
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity:   "critical",
			Code:       "WAIT_SELECTOR_NOT_FOUND",
			Message:    fmt.Sprintf("Wait selector '%s' not found on page", selector),
			Selector:   selector,
			Suggestion: fmt.Sprintf("Element may not exist or selector is incorrect. Expected state: %s", state),
		})
		result.Status = models.ValidationStatusFail
	} else {
		// Check visibility if state is visible
		if state == "visible" {
			visible, _ := locator.First().IsVisible()
			if !visible {
				result.Issues = append(result.Issues, models.ValidationIssue{
					Severity:   "warning",
					Code:       "ELEMENT_NOT_VISIBLE",
					Message:    fmt.Sprintf("Element found but not visible (state: %s)", state),
					Selector:   selector,
					Suggestion: "Element exists but may not be in the expected visible state",
				})
				result.Status = models.ValidationStatusWarning
			}
		}
	}

	result.Metrics["selector"] = selector
	result.Metrics["state"] = state
	result.Metrics["timeout"] = timeout
	result.Metrics["element_count"] = count
	result.Metrics["exists"] = count > 0

	return result, nil
}
