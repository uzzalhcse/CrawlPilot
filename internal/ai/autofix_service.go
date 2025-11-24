package ai

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
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
	SuggestedSelector    string                     `json:"suggested_selector"`
	AlternativeSelectors []string                   `json:"alternative_selectors,omitempty"`
	Confidence           float64                    `json:"confidence"`
	Explanation          string                     `json:"explanation"`
	SuggestedNodeConfig  map[string]interface{}     `json:"suggested_node_config,omitempty"`
	VerificationResult   *models.VerificationResult `json:"verification_result,omitempty"`
}

// NewAutoFixService creates a new autofix service
func NewAutoFixService(aiClient AIClient, logger *zap.Logger) *AutoFixService {
	return &AutoFixService{
		aiClient: aiClient,
		logger:   logger,
	}
}

// AnalyzeSnapshot analyzes a health check snapshot and suggests fixes
func (s *AutoFixService) AnalyzeSnapshot(ctx context.Context, snapshot *models.HealthCheckSnapshot, screenshotPath string, baselinePreview string) (*FixSuggestion, error) {
	s.logger.Info("Analyzing snapshot with AI",
		zap.String("snapshot_id", snapshot.ID),
		zap.String("node_id", snapshot.NodeID))

	// Read DOM snapshot if available
	var domContent string
	if snapshot.DOMSnapshotPath != nil && fileExists(*snapshot.DOMSnapshotPath) {
		domBytes, err := readDOMSnapshot(*snapshot.DOMSnapshotPath)
		if err != nil {
			s.logger.Warn("Failed to read DOM snapshot", zap.Error(err))
		} else {
			// Send the full DOM - modern LLMs have large context windows
			domContent = string(domBytes)
			s.logger.Info("Loaded DOM snapshot for AI analysis",
				zap.Int("size_bytes", len(domBytes)))
		}
	}

	// Build AI prompt with DOM content and baseline preview
	prompt := s.buildPrompt(snapshot, domContent, baselinePreview)

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

	// Verify suggestion against DOM snapshot if available
	s.logger.Info("Checking for DOM snapshot for verification",
		zap.Bool("has_dom_path", snapshot.DOMSnapshotPath != nil),
		zap.String("dom_path", func() string {
			if snapshot.DOMSnapshotPath != nil {
				return *snapshot.DOMSnapshotPath
			}
			return "nil"
		}()))

	if snapshot.DOMSnapshotPath != nil && fileExists(*snapshot.DOMSnapshotPath) {
		s.logger.Info("Running verification", zap.String("selector", suggestion.SuggestedSelector))
		verificationResult := s.verifySuggestion(suggestion.SuggestedSelector, *snapshot.DOMSnapshotPath)
		suggestion.VerificationResult = verificationResult

		s.logger.Info("Suggestion verified",
			zap.Bool("is_valid", verificationResult.IsValid),
			zap.Int("elements_found", verificationResult.ElementsFound),
			zap.Strings("data_preview", verificationResult.DataPreview))
	} else {
		s.logger.Warn("DOM snapshot not available for verification",
			zap.Bool("path_is_nil", snapshot.DOMSnapshotPath == nil),
			zap.Bool("file_exists", snapshot.DOMSnapshotPath != nil && fileExists(*snapshot.DOMSnapshotPath)))
	}

	s.logger.Info("AI analysis complete",
		zap.String("suggested_selector", suggestion.SuggestedSelector),
		zap.Float64("confidence", suggestion.Confidence))

	return suggestion, nil
}

// verifySuggestion verifies a selector against a DOM snapshot
func (s *AutoFixService) verifySuggestion(selector string, domPath string) *models.VerificationResult {
	result := &models.VerificationResult{
		IsValid:       false,
		ElementsFound: 0,
		DataPreview:   []string{},
	}

	// Open DOM file
	f, err := os.Open(domPath)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to open DOM file: %v", err)
		return result
	}
	defer f.Close()

	// Parse HTML
	var doc *goquery.Document
	if strings.HasSuffix(domPath, ".gz") {
		// Handle gzipped file if needed (though current implementation might not gzip yet)
		// For now assuming plain HTML or handling decompression if we add it
		// If we strictly use .html.gz, we need gzip reader.
		// Let's check if we are using gzip in snapshot service.
		// SnapshotService uses gzip.NewWriter. So we need gzip reader.

		gz, err := gzip.NewReader(f)
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("Failed to create gzip reader: %v", err)
			return result
		}
		defer gz.Close()
		doc, err = goquery.NewDocumentFromReader(gz)
	} else {
		doc, err = goquery.NewDocumentFromReader(f)
	}

	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to parse HTML: %v", err)
		return result
	}

	// Find elements
	selection := doc.Find(selector)
	result.ElementsFound = selection.Length()

	if result.ElementsFound > 0 {
		result.IsValid = true
		// Extract preview data (limit to 5 items)
		selection.EachWithBreak(func(i int, s *goquery.Selection) bool {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				// Truncate long text
				if len(text) > 100 {
					text = text[:97] + "..."
				}
				result.DataPreview = append(result.DataPreview, text)
			}
			return i < 4 // Stop after 5 items (0-4)
		})
	} else {
		result.ErrorMessage = "Selector found no elements in the DOM snapshot"
	}

	return result
}

// buildPrompt constructs the AI prompt from snapshot data
func (s *AutoFixService) buildPrompt(snapshot *models.HealthCheckSnapshot, domContent string, baselinePreview string) string {
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

	// Baseline preview - show AI what data SHOULD be extracted
	if baselinePreview != "" {
		prompt.WriteString("## Baseline (Expected Output)\n")
		prompt.WriteString("**This is what the selector SHOULD extract** (from a working baseline):\n")
		prompt.WriteString("```\n")
		prompt.WriteString(baselinePreview)
		prompt.WriteString("\n```\n")
		prompt.WriteString("**IMPORTANT**: Your suggested selector must extract similar data as shown above.\n\n")
	}

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

	// DOM Content - CRITICAL for accurate suggestions
	if domContent != "" {
		prompt.WriteString("## HTML Structure (Actual DOM)\n")
		prompt.WriteString("```html\n")
		prompt.WriteString(domContent)
		prompt.WriteString("\n```\n\n")
		prompt.WriteString("**IMPORTANT**: The above HTML is the ACTUAL page content. You MUST suggest selectors that exist in this HTML.\n\n")
	}

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
	if domContent != "" {
		prompt.WriteString("Analyze the provided HTML structure and suggest a working CSS selector that EXISTS in the DOM.\n\n")
	} else {
		prompt.WriteString("Analyze the screenshot (if provided) and suggest a working CSS selector.\n\n")
	}

	prompt.WriteString("## Requirements\n")
	prompt.WriteString("1. **VERIFY** the selector exists in the HTML above before suggesting it\n")
	prompt.WriteString("2. Suggest a CSS selector that will find the intended elements\n")
	prompt.WriteString("3. Provide 1-2 alternative selectors as backup options\n")
	prompt.WriteString("4. Explain WHY the original selector failed and WHY your suggestion will work\n")
	prompt.WriteString("5. **Keep explanation BRIEF** (1-2 sentences max, readable in 3-5 seconds)\n")
	prompt.WriteString("6. Provide a confidence score (0.0-1.0)\n\n")

	prompt.WriteString("## Response Format (JSON)\n")
	prompt.WriteString("```json\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"suggested_selector\": \"your suggested CSS selector\",\n")
	prompt.WriteString("  \"alternative_selectors\": [\"backup option 1\", \"backup option 2\"],\n")
	prompt.WriteString("  \"confidence\": 0.85,\n")
	prompt.WriteString("  \"explanation\": \"Brief 1-2 sentence explanation\"\n")
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
			response = strings.TrimSpace(response[start : start+end])
		}
	} else if strings.HasPrefix(response, "```") {
		// Check for ``` ... ``` blocks
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
			response = strings.Join(jsonLines, "\n")
		}
	}

	// Try to parse and re-marshal to ensure proper JSON formatting
	// This handles cases where the JSON has unescaped newlines in string values
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(response), &raw); err != nil {
		// If it fails, the response likely has formatting issues
		// Return as-is and let the caller handle the error
		return response
	}

	// Re-marshal to get properly formatted JSON
	cleaned, err := json.Marshal(raw)
	if err != nil {
		return response
	}

	return string(cleaned)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// readDOMSnapshot reads and decompresses a DOM snapshot file
func readDOMSnapshot(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Check if file is gzipped
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		return io.ReadAll(gz)
	}

	// Read plain file
	return io.ReadAll(f)
}
