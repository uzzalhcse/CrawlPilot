package extraction

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// ValidateForHealthCheck performs health check validation for extract nodes
func (e *ExtractExecutor) ValidateForHealthCheck(ctx context.Context, input *nodes.ValidationInput) (*models.NodeValidationResult, error) {
	result := &models.NodeValidationResult{
		NodeType: string(models.NodeTypeExtract),
		Status:   models.ValidationStatusPass,
		Metrics:  make(map[string]interface{}),
		Issues:   []models.ValidationIssue{},
	}

	page := input.BrowserContext.Page

	// Validate all field selectors
	fields, ok := input.Params["fields"].(map[string]interface{})
	if !ok || len(fields) == 0 {
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity:   "warning",
			Code:       "NO_FIELDS_CONFIGURED",
			Message:    "No extraction fields configured",
			Suggestion: "Add fields to extract data from the page",
		})
		result.Status = models.ValidationStatusWarning
		return result, nil
	}

	validFields := 0
	failedFields := []string{}
	sampleData := make(map[string]interface{})

	for fieldName, fieldConfigInterface := range fields {
		fieldConfig, ok := fieldConfigInterface.(map[string]interface{})
		if !ok {
			continue
		}

		selector := nodes.GetStringParam(fieldConfig, "selector")
		if selector == "" {
			continue
		}

		locator := page.Locator(selector)
		count, _ := locator.Count()

		if count == 0 {
			failedFields = append(failedFields, fieldName)
			result.Issues = append(result.Issues, models.ValidationIssue{
				Severity:   "critical",
				Code:       "FIELD_SELECTOR_NOT_FOUND",
				Message:    fmt.Sprintf("Field '%s' selector returned no elements", fieldName),
				Selector:   selector,
				Suggestion: "Check if selector is correct for this field",
			})
		} else {
			validFields++
			// Try to extract sample data
			fieldType := nodes.GetStringParam(fieldConfig, "type", "text")
			if fieldType == "text" {
				text, _ := locator.First().TextContent()
				if len(text) > 50 {
					sampleData[fieldName] = text[:50] + "..."
				} else {
					sampleData[fieldName] = text
				}
			}
		}
	}

	// Determine status
	if validFields == 0 && len(fields) > 0 {
		result.Status = models.ValidationStatusFail
	} else if len(failedFields) > 0 {
		result.Status = models.ValidationStatusWarning
	}

	result.Metrics["total_fields"] = len(fields)
	result.Metrics["valid_fields"] = validFields
	result.Metrics["failed_fields"] = failedFields
	result.Metrics["sample_data"] = sampleData

	return result, nil
}
