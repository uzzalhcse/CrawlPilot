package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/recovery/llm"
	"go.uber.org/zap"
)

// ProbeAgent analyzes probe failures and suggests workflow fixes
type ProbeAgent struct {
	provider llm.Provider
	config   *ProbeAgentConfig
}

// ProbeAgentConfig holds configuration for the probe agent
type ProbeAgentConfig struct {
	Enabled             bool
	ConfidenceThreshold float64 // Minimum confidence to auto-apply fix (default 0.8)
	MaxTokens           int
	Temperature         float64
	Timeout             time.Duration

	// DOM Analysis Strategy
	ChunkedDOM   bool // If true, split DOM into chunks for multi-pass analysis. If false, send full DOM in single pass.
	DOMChunkSize int  // Size of each DOM chunk in bytes (default 30KB)
}

// DefaultProbeAgentConfig returns default probe agent configuration
func DefaultProbeAgentConfig() *ProbeAgentConfig {
	return &ProbeAgentConfig{
		Enabled:             true,
		ConfidenceThreshold: 0.8,
		MaxTokens:           1000,
		Temperature:         0.1,
		Timeout:             60 * time.Second,
		ChunkedDOM:          true,  // Default to chunked for reliability
		DOMChunkSize:        30000, // 30KB chunks
	}
}

// NewProbeAgent creates a new probe agent
func NewProbeAgent(provider llm.Provider, config *ProbeAgentConfig) *ProbeAgent {
	if config == nil {
		config = DefaultProbeAgentConfig()
	}
	return &ProbeAgent{
		provider: provider,
		config:   config,
	}
}

// ProbeFixPlan contains AI-suggested workflow changes
type ProbeFixPlan struct {
	WorkflowID  string                 `json:"workflow_id"`
	NodeID      string                 `json:"node_id"`
	FieldName   string                 `json:"field_name,omitempty"` // For update_field_selector
	FixType     string                 `json:"fix_type"`             // "update_selector", "update_field_selector", "skip_node", "escalate"
	Confidence  float64                `json:"confidence"`
	OldSelector string                 `json:"old_selector,omitempty"`
	NewSelector string                 `json:"new_selector,omitempty"`
	Reasoning   string                 `json:"reasoning"`
	ShouldApply bool                   `json:"should_apply"` // Confidence >= threshold
	Params      map[string]interface{} `json:"params,omitempty"`
}

// AnalyzeProbeFailure analyzes failed probe and suggests workflow fixes
// Supports two strategies based on config:
// - ChunkedDOM=true: Multi-pass analysis with DOM split into chunks
// - ChunkedDOM=false: Single pass with full DOM
func (pa *ProbeAgent) AnalyzeProbeFailure(
	ctx context.Context,
	result *models.ProbeResult,
	baseline *models.ProbeResult,
	deviations *DeviationSummary,
) (*ProbeFixPlan, error) {
	if !pa.config.Enabled || pa.provider == nil {
		return nil, fmt.Errorf("probe agent is disabled")
	}

	// Get the full DOM content
	domContent, domPath := pa.getFullDOM(result)

	// Build problem context (without DOM)
	baseContext := pa.formatProblemContext(result, deviations)

	logger.Debug("Starting DOM analysis",
		zap.String("workflow_id", result.WorkflowID),
		zap.Int("dom_size", len(domContent)),
		zap.Bool("chunked_mode", pa.config.ChunkedDOM),
		zap.Int("chunk_size", pa.config.DOMChunkSize),
		zap.String("dom_path", domPath),
	)

	// Choose strategy based on config
	if !pa.config.ChunkedDOM {
		// Full DOM single-pass strategy
		return pa.analyzeWithFullDOM(ctx, result, baseContext, domContent)
	}

	// Chunked multi-pass strategy
	return pa.analyzeWithChunkedDOM(ctx, result, baseContext, domContent)
}

// analyzeWithFullDOM sends the entire DOM in a single request
func (pa *ProbeAgent) analyzeWithFullDOM(
	ctx context.Context,
	result *models.ProbeResult,
	baseContext string,
	domContent string,
) (*ProbeFixPlan, error) {

	fullContext := fmt.Sprintf(`%s

FULL PAGE DOM:
%s

Analyze the DOM and find the correct CSS selector for the failed field. Call the appropriate fix function.
`, baseContext, domContent)

	messages := []llm.Message{
		{Role: "system", Content: ProbeFixSystemPrompt()},
		{Role: "user", Content: fullContext},
	}

	ctx, cancel := context.WithTimeout(ctx, pa.config.Timeout)
	defer cancel()

	resp, err := pa.provider.ChatWithTools(ctx, messages, ProbeFixTools())
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	logger.Debug("AI response received (full DOM mode)",
		zap.Int("tool_calls", len(resp.ToolCalls)),
		zap.Int("content_length", len(resp.Content)),
	)

	plan, err := pa.parseResponse(resp, result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	plan.ShouldApply = plan.Confidence >= pa.config.ConfidenceThreshold

	logger.Info("Probe agent analysis complete (full DOM mode)",
		zap.String("workflow_id", result.WorkflowID),
		zap.String("fix_type", plan.FixType),
		zap.Float64("confidence", plan.Confidence),
		zap.Bool("should_apply", plan.ShouldApply),
	)

	return plan, nil
}

// analyzeWithChunkedDOM splits DOM into chunks and analyzes each until fix is found
func (pa *ProbeAgent) analyzeWithChunkedDOM(
	ctx context.Context,
	result *models.ProbeResult,
	baseContext string,
	domContent string,
) (*ProbeFixPlan, error) {
	// Use configured chunk size
	chunkSize := pa.config.DOMChunkSize
	if chunkSize <= 0 {
		chunkSize = 30000 // Default 30KB
	}

	domChunks := pa.splitDOMIntoChunks(domContent, chunkSize)

	logger.Debug("Starting chunked DOM analysis",
		zap.Int("chunk_count", len(domChunks)),
		zap.Int("chunk_size", chunkSize),
	)

	ctx, cancel := context.WithTimeout(ctx, pa.config.Timeout*time.Duration(len(domChunks)+1))
	defer cancel()

	// Try each chunk until AI finds the relevant selector
	for i, chunk := range domChunks {
		chunkContext := fmt.Sprintf(`%s

DOM CHUNK %d of %d:
%s

Analyze this DOM chunk. If you find a selector that matches the failed field (look for classes containing "ProductName", "name", "title", etc.), call the appropriate fix function. If you don't find it in this chunk, call escalate_to_human with reason "need_more_context".
`, baseContext, i+1, len(domChunks), chunk)

		messages := []llm.Message{
			{Role: "system", Content: ProbeFixSystemPrompt()},
			{Role: "user", Content: chunkContext},
		}

		resp, err := pa.provider.ChatWithTools(ctx, messages, ProbeFixTools())
		if err != nil {
			logger.Warn("AI analysis failed for chunk",
				zap.Int("chunk", i+1),
				zap.Error(err),
			)
			continue
		}

		logger.Debug("AI response for chunk",
			zap.Int("chunk", i+1),
			zap.Int("tool_calls", len(resp.ToolCalls)),
			zap.Int("content_length", len(resp.Content)),
		)

		plan, err := pa.parseResponse(resp, result)
		if err != nil {
			logger.Debug("Failed to parse chunk response",
				zap.Int("chunk", i+1),
				zap.Error(err),
			)
			continue
		}

		// If AI found a fix (not escalate with need_more_context), return it
		if plan.FixType != "escalate" || !strings.Contains(plan.Reasoning, "need_more_context") {
			plan.ShouldApply = plan.Confidence >= pa.config.ConfidenceThreshold

			logger.Info("Probe agent analysis complete (chunked mode)",
				zap.String("workflow_id", result.WorkflowID),
				zap.String("fix_type", plan.FixType),
				zap.Float64("confidence", plan.Confidence),
				zap.Bool("should_apply", plan.ShouldApply),
				zap.Int("found_in_chunk", i+1),
			)

			return plan, nil
		}

		logger.Debug("AI needs more context, trying next chunk",
			zap.Int("current_chunk", i+1),
			zap.Int("total_chunks", len(domChunks)),
		)
	}

	// Exhausted all chunks without finding a fix
	return &ProbeFixPlan{
		WorkflowID:  result.WorkflowID,
		FixType:     "escalate",
		Confidence:  0.0,
		Reasoning:   "AI analyzed all DOM chunks but could not find a suitable selector fix",
		ShouldApply: false,
	}, nil
}

// getFullDOM extracts the full DOM content from probe result
func (pa *ProbeAgent) getFullDOM(result *models.ProbeResult) (string, string) {
	for _, phase := range result.Phases {
		for _, node := range phase.Nodes {
			if node.Snapshot != nil && node.Snapshot.DOMPath != "" {
				if content, err := os.ReadFile(node.Snapshot.DOMPath); err == nil {
					return string(content), node.Snapshot.DOMPath
				}
			}
		}
	}
	return "", ""
}

// splitDOMIntoChunks splits DOM into chunks of specified size
// Tries to split at line boundaries to preserve HTML structure
func (pa *ProbeAgent) splitDOMIntoChunks(dom string, chunkSize int) []string {
	if len(dom) == 0 {
		return []string{}
	}

	if len(dom) <= chunkSize {
		return []string{dom}
	}

	var chunks []string
	lines := strings.Split(dom, "\n")
	currentChunk := ""

	for _, line := range lines {
		// If adding this line would exceed chunk size, save current chunk and start new one
		if len(currentChunk)+len(line)+1 > chunkSize && len(currentChunk) > 0 {
			chunks = append(chunks, currentChunk)
			currentChunk = line + "\n"
		} else {
			currentChunk += line + "\n"
		}
	}

	// Don't forget the last chunk
	if len(currentChunk) > 0 {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

// formatProblemContext creates the problem description without DOM
func (pa *ProbeAgent) formatProblemContext(result *models.ProbeResult, deviations *DeviationSummary) string {
	ctx := fmt.Sprintf(`PROBE FAILURE ANALYSIS

Workflow ID: %s
Execution ID: %s
Status: %s

FAILED/DEGRADED NODES:
`, result.WorkflowID, result.ExecutionID, result.Status)

	for _, phase := range result.Phases {
		for _, node := range phase.Nodes {
			// Include failed nodes OR nodes with missing required fields
			isProblematic := node.Status == "failed" ||
				(node.NodeType == "extract_links" && node.LinksFound == 0) ||
				(node.NodeType == "extract" && node.ElementCount == 0) ||
				(node.NodeType == "extract" && len(node.MissingRequiredFields) > 0)

			if isProblematic {
				status := node.Status
				if node.Status == "passed" && len(node.MissingRequiredFields) > 0 {
					status = "degraded (missing required fields)"
				}

				ctx += fmt.Sprintf(`
Node: %s (%s)
  Type: %s
  Status: %s
  Selector: %s
`, node.NodeName, node.NodeID, node.NodeType, status, node.Selector)

				// Add missing required fields with their selectors
				if len(node.MissingRequiredFields) > 0 {
					ctx += fmt.Sprintf("  Missing Required Fields: %v\n", node.MissingRequiredFields)
					for _, fieldName := range node.MissingRequiredFields {
						for _, field := range node.Fields {
							if field.Name == fieldName {
								ctx += fmt.Sprintf("    - Field '%s' uses selector: '%s'\n", field.Name, field.Selector)
								if field.Error != "" {
									ctx += fmt.Sprintf("      Error: %s\n", field.Error)
								}
								break
							}
						}
					}
				}

				if node.Error != "" {
					ctx += fmt.Sprintf("  Error: %s\n", node.Error)
				}
			}
		}
	}

	ctx += "\nYour task: Find the correct CSS selector in the DOM chunk below and call update_field_selector or update_selector."

	return ctx
}

// buildMessages creates the prompt for AI analysis
func (pa *ProbeAgent) buildMessages(result *models.ProbeResult, deviations *DeviationSummary) []llm.Message {
	messages := []llm.Message{
		{
			Role:    "system",
			Content: ProbeFixSystemPrompt(),
		},
	}

	// Add probe failure context
	context := pa.formatProbeContext(result, deviations)
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: context,
	})

	return messages
}

// formatProbeContext formats probe failure details for AI
func (pa *ProbeAgent) formatProbeContext(result *models.ProbeResult, deviations *DeviationSummary) string {
	ctx := fmt.Sprintf(`PROBE FAILURE ANALYSIS

Workflow ID: %s
Overall Status: %s
Duration: %dms

`, result.WorkflowID, result.Status, result.Duration)

	// Add failed and degraded nodes
	ctx += "AFFECTED NODES:\n"
	for _, phase := range result.Phases {
		for _, node := range phase.Nodes {
			// Include failed nodes OR nodes with 0 results OR missing required fields (degraded)
			isProblematic := node.Status == "failed" ||
				(node.NodeType == "extract_links" && node.LinksFound == 0) ||
				(node.NodeType == "extract" && node.ElementCount == 0) ||
				(node.NodeType == "extract" && len(node.MissingRequiredFields) > 0)

			if isProblematic {
				status := node.Status
				if node.Status == "passed" {
					if len(node.MissingRequiredFields) > 0 {
						status = "degraded (missing required fields)"
					} else {
						status = "degraded (0 results)"
					}
				}

				ctx += fmt.Sprintf(`
Phase: %s
Node: %s (%s)
  Type: %s
  Status: %s
  Selector: %s
`, phase.PhaseName, node.NodeName, node.NodeID, node.NodeType, status, node.Selector)

				// Add element/link counts
				if node.NodeType == "extract_links" {
					ctx += fmt.Sprintf("  Links Found: %d\n", node.LinksFound)
				} else if node.NodeType == "extract" {
					ctx += fmt.Sprintf("  Elements Found: %d\n", node.ElementCount)
				}

				// Add missing required fields if any
				if len(node.MissingRequiredFields) > 0 {
					ctx += fmt.Sprintf("  Missing Required Fields: %v\n", node.MissingRequiredFields)
					// Add field selector details for AI to analyze
					for _, fieldName := range node.MissingRequiredFields {
						for _, field := range node.Fields {
							if field.Name == fieldName {
								ctx += fmt.Sprintf("    - Field '%s' uses selector: '%s'\n", field.Name, field.Selector)
								if field.Error != "" {
									ctx += fmt.Sprintf("      Error: %s\n", field.Error)
								}
								break
							}
						}
					}
				}

				// Add error if any
				if node.Error != "" {
					ctx += fmt.Sprintf("  Error: %s\n", node.Error)
				}

				// Add selector context if available
				if node.SelectorContext != "" {
					ctx += fmt.Sprintf("  HTML Context: %s\n", truncateContent(node.SelectorContext, 500))
				}

				// Add snapshot info and DOM excerpt
				if node.Snapshot != nil {
					ctx += fmt.Sprintf("  Page Title: %s\n", node.Snapshot.PageTitle)

					if node.Snapshot.DOMPath != "" {
						// Read DOM and extract relevant section around the selector
						if domContent, err := os.ReadFile(node.Snapshot.DOMPath); err == nil {
							domStr := string(domContent)
							// Extract relevant portion around ProductInfo or similar patterns
							relevantDOM := extractRelevantDOM(domStr, node.Selector, node.MissingRequiredFields)
							if relevantDOM != "" {
								ctx += fmt.Sprintf("  RELEVANT DOM SECTION:\n%s\n", relevantDOM)
							}
						}
					}
				}
			}
		}
	}

	// Add deviation summary if available
	if deviations != nil && len(deviations.Deviations) > 0 {
		ctx += fmt.Sprintf(`
BASELINE COMPARISON:
Total Deviations: %d
New Failures: %d
Critical: %d, Major: %d, Minor: %d

`, deviations.TotalDeviations, deviations.NewFailures,
			deviations.CriticalCount, deviations.MajorCount, deviations.MinorCount)

		for _, d := range deviations.Deviations {
			if d.Severity == "critical" || d.Severity == "major" {
				ctx += fmt.Sprintf("- %s: %s (expected %d, got %d, %.1f%% change)\n",
					d.NodeName, d.DeviationType, d.ExpectedValue, d.ActualValue, d.PercentageChange)
			}
		}
	}

	ctx += "\nAnalyze the probe failure and call the appropriate function to suggest a fix."

	return ctx
}

// parseResponse extracts the fix plan from AI response
func (pa *ProbeAgent) parseResponse(resp *llm.Response, result *models.ProbeResult) (*ProbeFixPlan, error) {
	plan := &ProbeFixPlan{
		WorkflowID: result.WorkflowID,
		Params:     make(map[string]interface{}),
	}

	// Log what we received for debugging
	logger.Debug("AI response received",
		zap.Int("tool_calls", len(resp.ToolCalls)),
		zap.Int("content_length", len(resp.Content)),
		zap.String("content_preview", truncateContent(resp.Content, 200)),
	)

	// Check for tool calls
	if len(resp.ToolCalls) > 0 {
		tc := resp.ToolCalls[0]
		return pa.parseToolCall(tc, plan)
	}

	// Fallback: try to parse text response as function call
	// Some models return function call syntax as text instead of proper tool calls
	if resp.Content != "" {
		if parsedPlan, ok := pa.parseTextFunctionCall(resp.Content, plan); ok {
			logger.Info("Parsed function call from text response",
				zap.String("workflow_id", result.WorkflowID),
				zap.String("fix_type", parsedPlan.FixType),
			)
			return parsedPlan, nil
		}

		// If parsing failed, treat as escalation
		plan.FixType = "escalate"
		plan.Confidence = 0.3
		plan.Reasoning = resp.Content

		logger.Info("AI returned text response instead of tool call, treating as escalation",
			zap.String("workflow_id", result.WorkflowID),
			zap.String("content", truncateContent(resp.Content, 300)),
		)
		return plan, nil
	}

	return nil, fmt.Errorf("no action in AI response: tool_calls=%d, content_length=%d",
		len(resp.ToolCalls), len(resp.Content))
}

// parseTextFunctionCall attempts to extract a function call from text response
// Some models return function call syntax as text instead of proper tool calls
func (pa *ProbeAgent) parseTextFunctionCall(content string, plan *ProbeFixPlan) (*ProbeFixPlan, bool) {
	content = strings.TrimSpace(content)

	// Pattern 1: update_field_selector("node_id", "field_name", ".old", ".new")
	// Pattern 2: update_selector("node_id", ".old", ".new")
	// Pattern 3: escalate_to_human("reason")

	// Check for update_field_selector
	if strings.Contains(content, "update_field_selector") {
		// Try to extract selector from various formats
		// Look for class selector pattern: .ClassName or "ClassName"
		selectors := pa.extractSelectorsFromText(content)
		if len(selectors) > 0 {
			plan.FixType = "update_field_selector"
			plan.NewSelector = selectors[0]
			plan.Confidence = 0.75 // Medium-high confidence for parsed text
			plan.Reasoning = "Parsed from text response: " + content
			return plan, true
		}
	}

	// Check for update_selector
	if strings.Contains(content, "update_selector") {
		selectors := pa.extractSelectorsFromText(content)
		if len(selectors) > 0 {
			plan.FixType = "update_selector"
			plan.NewSelector = selectors[0]
			plan.Confidence = 0.75
			plan.Reasoning = "Parsed from text response: " + content
			return plan, true
		}
	}

	// Check for any selector suggestion in the text
	// Look for patterns like ".ProductInfo_Head_Main_ProductName" or "class='ProductName'"
	if strings.Contains(strings.ToLower(content), "selector") ||
		strings.Contains(strings.ToLower(content), "class") {
		selectors := pa.extractSelectorsFromText(content)
		if len(selectors) > 0 {
			// Determine if this is for a field or node based on context
			if strings.Contains(strings.ToLower(content), "field") ||
				strings.Contains(strings.ToLower(content), "product_name") {
				plan.FixType = "update_field_selector"
			} else {
				plan.FixType = "update_selector"
			}
			plan.NewSelector = selectors[0]
			plan.Confidence = 0.7
			plan.Reasoning = "Extracted selector from text: " + content
			return plan, true
		}
	}

	// Check for escalate_to_human with need_more_context
	if strings.Contains(content, "escalate_to_human") && strings.Contains(content, "need_more_context") {
		plan.FixType = "escalate"
		plan.Confidence = 0.3
		plan.Reasoning = "need_more_context"
		return plan, true
	}

	return nil, false
}

// extractSelectorsFromText finds CSS selectors in text content
func (pa *ProbeAgent) extractSelectorsFromText(content string) []string {
	var selectors []string

	// Pattern 1: Find .ClassName patterns (class selectors)
	classPattern := regexp.MustCompile(`\.([A-Za-z][A-Za-z0-9_-]+)`)
	matches := classPattern.FindAllString(content, -1)
	for _, m := range matches {
		// Filter out common non-selector matches
		if !strings.HasPrefix(m, ".com") && !strings.HasPrefix(m, ".html") &&
			!strings.HasPrefix(m, ".js") && !strings.HasPrefix(m, ".css") &&
			len(m) > 3 {
			selectors = append(selectors, m)
		}
	}

	// Pattern 2: Find class="ClassName" or class='ClassName'
	classAttrPattern := regexp.MustCompile(`class\s*=\s*["']([^"']+)["']`)
	attrMatches := classAttrPattern.FindAllStringSubmatch(content, -1)
	for _, m := range attrMatches {
		if len(m) > 1 {
			// Convert to CSS selector
			selectors = append(selectors, "."+strings.Replace(m[1], " ", ".", -1))
		}
	}

	// Pattern 3: Find explicit selector mentions like "selector: .ClassName"
	selectorPattern := regexp.MustCompile(`selector[:\s]+["']?([.#][A-Za-z][A-Za-z0-9_-]+)["']?`)
	selectorMatches := selectorPattern.FindAllStringSubmatch(content, -1)
	for _, m := range selectorMatches {
		if len(m) > 1 {
			// Prepend these as they're explicitly mentioned
			selectors = append([]string{m[1]}, selectors...)
		}
	}

	return selectors
}

// parseToolCall extracts fix plan from tool call
func (pa *ProbeAgent) parseToolCall(tc llm.ToolCall, plan *ProbeFixPlan) (*ProbeFixPlan, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.ArgumentsString()), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	plan.Params = args

	switch tc.Function.Name {
	case "update_selector":
		plan.FixType = "update_selector"
		plan.NodeID = getString(args, "node_id")
		plan.OldSelector = getString(args, "old_selector")
		plan.NewSelector = getString(args, "new_selector")
		plan.Confidence = getFloat(args, "confidence")
		plan.Reasoning = getString(args, "reason")

	case "update_field_selector":
		plan.FixType = "update_field_selector"
		plan.NodeID = getString(args, "node_id")
		plan.FieldName = getString(args, "field_name")
		plan.OldSelector = getString(args, "old_selector")
		plan.NewSelector = getString(args, "new_selector")
		plan.Confidence = getFloat(args, "confidence")
		plan.Reasoning = getString(args, "reason")

	case "skip_node":
		plan.FixType = "skip_node"
		plan.NodeID = getString(args, "node_id")
		plan.Confidence = 0.9 // Skipping is a safe action
		plan.Reasoning = getString(args, "reason")

	case "escalate_to_human":
		plan.FixType = "escalate"
		plan.Confidence = 0.0 // Always escalate
		plan.Reasoning = getString(args, "reason")

	default:
		return nil, fmt.Errorf("unknown action: %s", tc.Function.Name)
	}

	return plan, nil
}

// IsAvailable checks if the probe agent is available
func (pa *ProbeAgent) IsAvailable(ctx context.Context) bool {
	if pa.provider == nil {
		return false
	}

	type availabilityChecker interface {
		IsAvailable(ctx context.Context) bool
	}

	if checker, ok := pa.provider.(availabilityChecker); ok {
		return checker.IsAvailable(ctx)
	}

	return true
}

// Close cleans up resources
func (pa *ProbeAgent) Close() error {
	if pa.provider != nil {
		return pa.provider.Close()
	}
	return nil
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0.0
}

// extractRelevantDOM finds relevant sections of DOM around failing selectors
// This keeps context small for the AI while providing enough information to suggest fixes
func extractRelevantDOM(dom string, selector string, missingFields []string) string {
	// Keywords to search for based on failing selector and page structure
	searchTerms := []string{}

	// Extract class/id from selector for search
	if selector != "" {
		// Extract class name like .ProductName -> ProductName
		if len(selector) > 1 && selector[0] == '.' {
			searchTerms = append(searchTerms, selector[1:])
		}
	}

	// Add common product-related terms
	commonTerms := []string{
		"ProductName", "ProductInfo", "product_name", "product-name",
		"heading", "title", "h1", "h2",
	}
	searchTerms = append(searchTerms, commonTerms...)

	// Find sections containing these terms
	var relevantSections []string
	lines := strings.Split(dom, "\n")

	for _, term := range searchTerms {
		for i, line := range lines {
			if strings.Contains(line, term) {
				// Get context: 3 lines before and 5 lines after
				start := i - 3
				if start < 0 {
					start = 0
				}
				end := i + 6
				if end > len(lines) {
					end = len(lines)
				}

				section := strings.Join(lines[start:end], "\n")
				// Avoid duplicates and limit size
				if len(section) < 1000 && !containsSection(relevantSections, section) {
					relevantSections = append(relevantSections, section)
				}

				// Limit number of sections
				if len(relevantSections) >= 3 {
					break
				}
			}
		}
		if len(relevantSections) >= 3 {
			break
		}
	}

	if len(relevantSections) == 0 {
		// Fallback: just return first 1000 chars if no relevant sections found
		if len(dom) > 1000 {
			return dom[:1000] + "..."
		}
		return dom
	}

	result := strings.Join(relevantSections, "\n\n---\n\n")
	if len(result) > 3000 {
		return result[:3000] + "..."
	}
	return result
}

// containsSection checks if a section is already in the list (to avoid duplicates)
func containsSection(sections []string, section string) bool {
	for _, s := range sections {
		if strings.Contains(s, section) || strings.Contains(section, s) {
			return true
		}
	}
	return false
}

// ProbeFixTools returns the AI tools for workflow fixes
func ProbeFixTools() []llm.Tool {
	return []llm.Tool{
		{
			Type: "function",
			Function: llm.Function{
				Name:        "update_selector",
				Description: "Update a CSS selector in the workflow when the old one no longer matches",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"node_id": {
							"type": "string",
							"description": "ID of the node to update"
						},
						"old_selector": {
							"type": "string",
							"description": "The current selector that is failing"
						},
						"new_selector": {
							"type": "string",
							"description": "The new CSS selector to use"
						},
						"confidence": {
							"type": "number",
							"description": "Confidence level 0.0-1.0 that this fix will work"
						},
						"reason": {
							"type": "string",
							"description": "Explanation of why this selector change should work"
						}
					},
					"required": ["node_id", "old_selector", "new_selector", "confidence", "reason"]
				}`),
			},
		},
		{
			Type: "function",
			Function: llm.Function{
				Name:        "update_field_selector",
				Description: "Update a CSS selector for a specific field within an extract node when the field's selector no longer matches",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"node_id": {
							"type": "string",
							"description": "ID of the extract node containing the field"
						},
						"field_name": {
							"type": "string",
							"description": "Name of the field to update (e.g., 'product_name', 'price')"
						},
						"old_selector": {
							"type": "string",
							"description": "The current selector that is failing"
						},
						"new_selector": {
							"type": "string",
							"description": "The new CSS selector to use"
						},
						"confidence": {
							"type": "number",
							"description": "Confidence level 0.0-1.0 that this fix will work"
						},
						"reason": {
							"type": "string",
							"description": "Explanation of why this selector change should work"
						}
					},
					"required": ["node_id", "field_name", "old_selector", "new_selector", "confidence", "reason"]
				}`),
			},
		},
		{
			Type: "function",
			Function: llm.Function{
				Name:        "skip_node",
				Description: "Temporarily disable a broken node that cannot be fixed automatically",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"node_id": {
							"type": "string",
							"description": "ID of the node to skip"
						},
						"reason": {
							"type": "string",
							"description": "Why this node should be skipped"
						}
					},
					"required": ["node_id", "reason"]
				}`),
			},
		},
		{
			Type: "function",
			Function: llm.Function{
				Name:        "escalate_to_human",
				Description: "Cannot determine a fix automatically, requires human review",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"reason": {
							"type": "string",
							"description": "Detailed explanation of why human review is needed"
						}
					},
					"required": ["reason"]
				}`),
			},
		},
	}
}

// ProbeFixSystemPrompt returns the system prompt for probe fix analysis
func ProbeFixSystemPrompt() string {
	return `You are a web scraping workflow repair agent that MUST respond using function calls.

CRITICAL: You MUST call one of the available functions. DO NOT respond with text. ALWAYS use a function call.

Your job is to analyze probe failures and suggest fixes for broken selectors.

CONTEXT:
- A probe execution tested a workflow against a website
- Some nodes failed or degraded, likely due to website structure changes
- You receive the failed node details, selector, error, and DOM context
- For extract nodes, you may see missing required fields with their selectors

MANDATORY ACTIONS (pick ONE and call the function):

1. FIELD SELECTOR FIX (most common):
   - If "Missing Required Fields" shows a field like "product_name" with selector ".ProductName"
   - Search the DOM for classes containing "ProductName", "product", "name", "title", "heading"
   - When you find a match (e.g., "ProductInfo_Head_Main_ProductName"), call:
     update_field_selector(node_id, field_name, old_selector, new_selector, confidence, reason)

2. NODE SELECTOR FIX:
   - For failed extract_links, navigate, or click nodes
   - Call: update_selector(node_id, old_selector, new_selector, confidence, reason)

3. ESCALATE:
   - If selector not found in this DOM chunk, call:
     escalate_to_human(reason="need_more_context", affected_nodes=[...])
   - If genuinely cannot fix, call:
     escalate_to_human(reason="complex_change", affected_nodes=[...])

CONFIDENCE LEVELS:
- 0.9-1.0: Exact class match found
- 0.7-0.9: Similar pattern found
- 0.5-0.7: Educated guess
- Below 0.5: Should escalate

REMEMBER: 
- DO NOT write text responses. ONLY call functions.
- If you find "ProductInfo_Head_Main_ProductName" in DOM and field selector is ".ProductName", call update_field_selector immediately.
- For field fixes, use update_field_selector NOT update_selector.`
}
