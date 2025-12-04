package interaction

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// ValidateForMonitoring performs monitoring validation for scroll nodes
func (e *ScrollExecutor) ValidateForMonitoring(ctx context.Context, input *nodes.ValidationInput) (*models.NodeValidationResult, error) {
	result := &models.NodeValidationResult{
		NodeType: string(models.NodeTypeScroll),
		Status:   models.ValidationStatusPass,
		Metrics:  make(map[string]interface{}),
		Issues:   []models.ValidationIssue{},
	}

	selector := nodes.GetStringParam(input.Params, "selector", "")
	direction := nodes.GetStringParam(input.Params, "direction", "down")
	amount := nodes.GetIntParam(input.Params, "amount", 500)

	page := input.BrowserContext.Page

	// If selector is provided, validate it exists
	if selector != "" {
		locator := page.Locator(selector)
		count, _ := locator.Count()

		if count == 0 {
			result.Issues = append(result.Issues, models.ValidationIssue{
				Severity:   "warning",
				Code:       "SCROLL_TARGET_NOT_FOUND",
				Message:    fmt.Sprintf("Scroll target selector '%s' not found", selector),
				Selector:   selector,
				Suggestion: "Element to scroll to does not exist. Scroll will use page body instead.",
			})
			result.Status = models.ValidationStatusWarning
		} else {
			result.Metrics["target_found"] = true
		}
	} else {
		result.Metrics["target_found"] = "page_scroll"
	}

	result.Metrics["direction"] = direction
	result.Metrics["amount"] = amount
	result.Metrics["has_selector"] = selector != ""

	return result, nil
}
