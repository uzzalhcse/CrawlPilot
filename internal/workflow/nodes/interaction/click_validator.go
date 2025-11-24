package interaction

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// ValidateForHealthCheck performs health check validation for click nodes
func (e *ClickExecutor) ValidateForHealthCheck(ctx context.Context, input *nodes.ValidationInput) (*models.NodeValidationResult, error) {
	result := &models.NodeValidationResult{
		NodeType: string(models.NodeTypeClick),
		Status:   models.ValidationStatusPass,
		Metrics:  make(map[string]interface{}),
		Issues:   []models.ValidationIssue{},
	}

	selector := nodes.GetStringParam(input.Params, "selector")
	if selector == "" {
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity:   "critical",
			Code:       "NO_SELECTOR",
			Message:    "No selector provided for click action",
			Suggestion: "Add a selector parameter to target the element to click",
		})
		result.Status = models.ValidationStatusFail
		return result, nil
	}

	page := input.BrowserContext.Page
	locator := page.Locator(selector)
	count, _ := locator.Count()

	if count == 0 {
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity:   "critical",
			Code:       "CLICK_TARGET_NOT_FOUND",
			Message:    fmt.Sprintf("Click target '%s' not found", selector),
			Selector:   selector,
			Suggestion: "Check if the element exists and selector is correct",
		})
		result.Status = models.ValidationStatusFail
	} else {
		// Check if element is clickable (visible and enabled)
		visible, _ := locator.First().IsVisible()
		enabled, _ := locator.First().IsEnabled()

		if !visible {
			result.Issues = append(result.Issues, models.ValidationIssue{
				Severity:   "warning",
				Code:       "ELEMENT_NOT_VISIBLE",
				Message:    "Click target exists but is not visible",
				Selector:   selector,
				Suggestion: "Element may be hidden or off-screen",
			})
			result.Status = models.ValidationStatusWarning
		}

		if !enabled {
			result.Issues = append(result.Issues, models.ValidationIssue{
				Severity:   "warning",
				Code:       "ELEMENT_DISABLED",
				Message:    "Click target exists but is disabled",
				Selector:   selector,
				Suggestion: "Element may be in a disabled state",
			})
			result.Status = models.ValidationStatusWarning
		}

		result.Metrics["visible"] = visible
		result.Metrics["enabled"] = enabled
	}

	result.Metrics["element_count"] = count
	result.Metrics["exists"] = count > 0

	return result, nil
}
