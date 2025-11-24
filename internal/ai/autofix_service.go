package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// AIClient interface for AI providers
type AIClient interface {
	GenerateText(ctx context.Context, prompt string) (string, error)
	GenerateWithImage(ctx context.Context, prompt string, imageData []byte, imageMimeType string) (string, error)
	Close() error
}

// AutoFixService analyzes health check failures and suggests fixes
type AutoFixService struct {
	aiClient AIClient
	logger   *zap.Logger
}

// FixSuggestion represents an AI-generated fix suggestion
type FixSuggestion struct {
	SuggestedSelector    string                 `json:"suggested_selector"`
	AlternativeSelectors []string               `json:"alternative_selectors,omitempty"`
	Confidence           float64                `json:"confidence"`
	Explanation          string                 `json:"explanation"`
	SuggestedNodeConfig  map[string]interface{} `json:"suggested_node_config,omitempty"`
}

// NewAutoFixService creates a new autofix service
func NewAutoFixService(aiClient AIClient, logger *zap.Logger) *AutoFixService {
	return &AutoFixService{
		aiClient: aiClient,
		logger:   logger,
	}
}

// AnalyzeSnapshot analyzes a health check snapshot and suggests fixes
func (s *AutoFixService) AnalyzeSnapshot(ctx context.Context, snapshot *models.HealthCheckSnapshot, screenshotPath string) (*FixSuggestion, error) {
	s.logger.Info("Analyzing snapshot with AI",
		zap.String("snapshot_id", snapshot.ID),
		zap.String("node_id", snapshot.NodeID))

	// Build AI prompt
	prompt := s.buildPrompt(snapshot)

	var aiResponse string
	var err error

	// If we have a screenshot, use multimodal generation
	if screenshotPath != "" && fileExists(screenshotPath) {
		imageData, readErr := os.ReadFile(screenshotPath)
		if readErr == nil {
			s.logger.Debug("Using screenshot for multimodal analysis")
			aiResponse, err = s.aiClient.GenerateWithImage(ctx, prompt, imageData, "image/png")
		} else {
			s.logger.Warn("Failed to read screenshot, falling back to text-only analysis",
				zap.Error(readErr))
			aiResponse, err = s.aiClient.GenerateText(ctx, prompt)
		}
	} else {
		// Text-only analysis
		s.logger.Debug("Using text-only analysis (no screenshot)")
		aiResponse, err = s.aiClient.GenerateText(ctx, prompt)
	}

	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	// Parse AI response
	suggestion, err := s.parseAIResponse(aiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	s.logger.Info("AI analysis complete",
		zap.String("suggested_selector", suggestion.SuggestedSelector),
		zap.Float64("confidence", suggestion.Confidence))

	return suggestion, nil
}

// buildPrompt constructs the AI prompt from snapshot data
func (s *AutoFixService) buildPrompt(snapshot *models.HealthCheckSnapshot) string {
	var prompt strings.Builder

	prompt.WriteString("You are an expert web scraping engineer. Analyze this failed CSS selector and suggest a fix.\n\n")

	// Error context
	prompt.WriteString("## Failure Details\n")
	selectorValue := ""
	if snapshot.SelectorValue != nil {
		selectorValue = *snapshot.SelectorValue
	}
	prompt.WriteString(fmt.Sprintf("**Failed Selector**: `%s`\n", selectorValue))
	prompt.WriteString(fmt.Sprintf("**Elements Found**: %d (expected > 0)\n", snapshot.ElementsFound))
	if snapshot.ErrorMessage != nil && *snapshot.ErrorMessage != "" {
		prompt.WriteString(fmt.Sprintf("**Error**: %s\n", *snapshot.ErrorMessage))
	}
	prompt.WriteString("\n")

	// Page context
	prompt.WriteString("## Page Context\n")
	prompt.WriteString(fmt.Sprintf("**URL**: %s\n", snapshot.URL))
	if snapshot.PageTitle != nil && *snapshot.PageTitle != "" {
		prompt.WriteString(fmt.Sprintf("**Title**: %s\n", *snapshot.PageTitle))
	}
	if snapshot.StatusCode != nil {
		prompt.WriteString(fmt.Sprintf("**Status**: %d\n", *snapshot.StatusCode))
	}
	prompt.WriteString("\n")

	// Console logs (errors/warnings only)
	if len(snapshot.ConsoleLogsData) > 0 {
		prompt.WriteString("## Console Errors/Warnings\n")
		count := 0
		for _, log := range snapshot.ConsoleLogsData {
			if log.Type == "error" || log.Type == "warn" {
				prompt.WriteString(fmt.Sprintf("- [%s] %s\n", log.Type, log.Message))
				count++
				if count >= 5 {
					break // Limit to 5 most recent
				}
			}
		}
		prompt.WriteString("\n")
	}

	// Instructions
	prompt.WriteString("## Task\n")
	prompt.WriteString("Analyze the screenshot (if provided) and suggest a working CSS selector.\n\n")

	prompt.WriteString("## Requirements\n")
	prompt.WriteString("1. Suggest a CSS selector that will find the intended elements\n")
	prompt.WriteString("2. Provide 1-2 alternative selectors as backup options\n")
	prompt.WriteString("3. Explain WHY the original selector failed and WHY your suggestion will work\n")
	prompt.WriteString("4. Provide a confidence score (0.0-1.0)\n\n")

	prompt.WriteString("## Response Format (JSON)\n")
	prompt.WriteString("```json\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"suggested_selector\": \"your suggested CSS selector\",\n")
	prompt.WriteString("  \"alternative_selectors\": [\"backup option 1\", \"backup option 2\"],\n")
	prompt.WriteString("  \"confidence\": 0.85,\n")
	prompt.WriteString("  \"explanation\": \"Detailed explanation of why the original failed and why this will work\"\n")
	prompt.WriteString("}\n")
	prompt.WriteString("```\n\n")

	prompt.WriteString("**Respond ONLY with the JSON object. No additional text.**")

	return prompt.String()
}

// parseAIResponse extracts the fix suggestion from AI response
func (s *AutoFixService) parseAIResponse(response string) (*FixSuggestion, error) {
	// Extract JSON from response (it might be wrapped in markdown code blocks)
	jsonStr := extractJSON(response)

	var suggestion FixSuggestion
	if err := json.Unmarshal([]byte(jsonStr), &suggestion); err != nil {
		// Log the raw response for debugging
		s.logger.Error("Failed to parse AI response",
			zap.String("response", response),
			zap.Error(err))
		return nil, fmt.Errorf("invalid JSON response from AI: %w", err)
	}

	// Validate suggestion
	if suggestion.SuggestedSelector == "" {
		return nil, fmt.Errorf("AI did not provide a suggested selector")
	}

	// Clamp confidence to [0, 1]
	if suggestion.Confidence < 0 {
		suggestion.Confidence = 0
	} else if suggestion.Confidence > 1 {
		suggestion.Confidence = 1
	}

	return &suggestion, nil
}

// extractJSON extracts JSON from a response that might be wrapped in markdown
func extractJSON(response string) string {
	// Remove markdown code blocks if present
	response = strings.TrimSpace(response)

	// Check for ```json ... ``` blocks
	if strings.Contains(response, "```json") {
		start := strings.Index(response, "```json") + 7
		end := strings.Index(response[start:], "```")
		if end != -1 {
			return strings.TrimSpace(response[start : start+end])
		}
	}

	// Check for ``` ... ``` blocks
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		var jsonLines []string
		inBlock := false
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				inBlock = !inBlock
				continue
			}
			if inBlock || (strings.HasPrefix(strings.TrimSpace(line), "{") || strings.HasPrefix(strings.TrimSpace(line), "}") || strings.Contains(line, "\":")) {
				jsonLines = append(jsonLines, line)
			}
		}
		if len(jsonLines) > 0 {
			return strings.Join(jsonLines, "\n")
		}
	}

	// Assume the whole response is JSON
	return response
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
