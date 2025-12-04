package discovery

import (
	"context"
	"fmt"
	"net/url"

	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// ValidateForMonitoring performs monitoring validation for navigate nodes
func (e *NavigateExecutor) ValidateForMonitoring(ctx context.Context, input *nodes.ValidationInput) (*models.NodeValidationResult, error) {
	result := &models.NodeValidationResult{
		NodeType: string(models.NodeTypeNavigate),
		Status:   models.ValidationStatusPass,
		Metrics:  make(map[string]interface{}),
		Issues:   []models.ValidationIssue{},
	}

	targetURL := nodes.GetStringParam(input.Params, "url")
	if targetURL == "" {
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity:   "critical",
			Code:       "NO_URL",
			Message:    "No URL provided for navigation",
			Suggestion: "Add a url parameter for navigation target",
		})
		result.Status = models.ValidationStatusFail
		return result, nil
	}

	// Validate URL format
	parsedURL, err := url.Parse(targetURL)
	if err != nil || parsedURL.Scheme == "" {
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity:   "critical",
			Code:       "INVALID_URL",
			Message:    fmt.Sprintf("Invalid URL format: %s", targetURL),
			Suggestion: "Ensure the URL is valid and includes protocol (http/https)",
		})
		result.Status = models.ValidationStatusFail
		return result, nil
	}

	// Try to navigate
	page := input.BrowserContext.Page
	response, err := page.Goto(targetURL)
	if err != nil {
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity:   "critical",
			Code:       "NAVIGATION_FAILED",
			Message:    fmt.Sprintf("Failed to navigate to %s: %v", targetURL, err),
			Suggestion: "Check if the URL is accessible and valid",
		})
		result.Status = models.ValidationStatusFail
		return result, nil
	}

	statusCode := 0
	if response != nil {
		statusCode = response.Status()
	}

	if statusCode >= 400 {
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity:   "critical",
			Code:       "HTTP_ERROR",
			Message:    fmt.Sprintf("Navigation returned HTTP %d", statusCode),
			Suggestion: "URL may be broken or requires authentication",
		})
		result.Status = models.ValidationStatusFail
	}

	result.Metrics["url"] = targetURL
	result.Metrics["status_code"] = statusCode
	result.Metrics["navigation_successful"] = err == nil

	return result, nil
}
