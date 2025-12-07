package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/driver"
	"go.uber.org/zap"
)

// ProbeSnapshotConfig holds configuration for snapshot capture
type ProbeSnapshotConfig struct {
	// LocalBasePath is the base directory for local storage
	LocalBasePath string
	// GCSBucket is the GCS bucket for cloud storage (optional)
	GCSBucket string
	// MaxDOMSize limits DOM capture size (default 2MB)
	MaxDOMSize int64
	// MaxContextSize limits selector context size (default 500 chars)
	MaxContextSize int
}

// DefaultProbeSnapshotConfig returns default snapshot configuration
func DefaultProbeSnapshotConfig() ProbeSnapshotConfig {
	return ProbeSnapshotConfig{
		LocalBasePath:  "snapshots",
		GCSBucket:      "", // Empty means local only
		MaxDOMSize:     2 * 1024 * 1024,
		MaxContextSize: 500,
	}
}

// captureNodeSnapshot captures full page DOM and screenshot when a node fails
func (e *TaskExecutor) captureNodeSnapshot(ctx context.Context, page driver.Page, task *models.Task, nodeID string) (*models.ProbeSnapshot, error) {
	if page == nil {
		return nil, fmt.Errorf("page is nil, cannot capture snapshot")
	}

	config := DefaultProbeSnapshotConfig()
	snapshot := &models.ProbeSnapshot{
		CapturedAt: time.Now().Unix(),
	}

	// Create directory structure: base/workflow_id/execution_id/phase_id/node_id/
	snapshotDir := filepath.Join(
		config.LocalBasePath,
		task.WorkflowID,
		task.ExecutionID,
		task.PhaseID,
		nodeID,
	)

	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	// Capture page URL and title
	snapshot.PageURL = task.URL
	title, err := page.Title()
	if err == nil {
		snapshot.PageTitle = title
	}

	// Capture DOM
	domPath := filepath.Join(snapshotDir, "dom.html")
	if err := e.captureDOM(ctx, page, domPath, config.MaxDOMSize); err != nil {
		logger.Warn("Failed to capture DOM",
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
	} else {
		snapshot.DOMPath = domPath
		if fi, err := os.Stat(domPath); err == nil {
			snapshot.DOMSize = fi.Size()
		}
	}

	// Capture screenshot
	screenshotPath := filepath.Join(snapshotDir, "screenshot.png")
	width, height, err := e.captureScreenshot(ctx, page, screenshotPath)
	if err != nil {
		logger.Warn("Failed to capture screenshot",
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
	} else {
		snapshot.ScreenshotPath = screenshotPath
		snapshot.ImageWidth = width
		snapshot.ImageHeight = height
	}

	logger.Debug("Node snapshot captured",
		zap.String("node_id", nodeID),
		zap.String("dom_path", snapshot.DOMPath),
		zap.String("screenshot_path", snapshot.ScreenshotPath),
		zap.Int64("dom_size", snapshot.DOMSize),
	)

	return snapshot, nil
}

// extractSelectorContext reads DOM from file and extracts context around a selector
func (e *TaskExecutor) extractSelectorContext(domPath string, selector string) string {
	domContent, err := os.ReadFile(domPath)
	if err != nil {
		return ""
	}
	config := DefaultProbeSnapshotConfig()
	return captureSelectorContext(string(domContent), selector, config.MaxContextSize)
}

// captureDOM captures the full page DOM HTML
func (e *TaskExecutor) captureDOM(ctx context.Context, page driver.Page, path string, maxSize int64) error {
	html, err := page.Content()
	if err != nil {
		return fmt.Errorf("failed to get page content: %w", err)
	}

	// Truncate if too large
	if int64(len(html)) > maxSize {
		html = html[:maxSize] + "\n<!-- TRUNCATED -->"
	}

	if err := os.WriteFile(path, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write DOM file: %w", err)
	}

	return nil
}

// captureScreenshot captures full page screenshot
func (e *TaskExecutor) captureScreenshot(ctx context.Context, page driver.Page, path string) (width, height int, err error) {
	// Use page's screenshot capability with full page option
	data, err := page.Screenshot(driver.WithFullPage(true))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to capture screenshot: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return 0, 0, fmt.Errorf("failed to write screenshot: %w", err)
	}

	// Default viewport size (actual image dimensions may differ)
	width, height = 1920, 1080

	return width, height, nil
}

// uploadSnapshotToGCS uploads snapshot files to GCS
func (e *TaskExecutor) uploadSnapshotToGCS(ctx context.Context, snapshot *models.ProbeSnapshot, task *models.Task, bucket string) error {
	if e.gcsClient == nil {
		return fmt.Errorf("GCS client not configured")
	}

	gcsPrefix := fmt.Sprintf("probes/%s/%s/%s", task.WorkflowID, task.ExecutionID, task.PhaseID)

	// Upload DOM
	if snapshot.DOMPath != "" {
		domData, err := os.ReadFile(snapshot.DOMPath)
		if err == nil {
			gcsPath := fmt.Sprintf("gs://%s/%s/dom.html", bucket, gcsPrefix)
			// Note: Actual GCS upload would use e.gcsClient.Upload()
			// For now, keep local path but log the intended GCS path
			logger.Debug("Would upload DOM to GCS",
				zap.String("gcs_path", gcsPath),
				zap.Int("size", len(domData)),
			)
		}
	}

	// Upload screenshot
	if snapshot.ScreenshotPath != "" {
		screenshotData, err := os.ReadFile(snapshot.ScreenshotPath)
		if err == nil {
			gcsPath := fmt.Sprintf("gs://%s/%s/screenshot.png", bucket, gcsPrefix)
			logger.Debug("Would upload screenshot to GCS",
				zap.String("gcs_path", gcsPath),
				zap.Int("size", len(screenshotData)),
			)
		}
	}

	return nil
}

// captureSelectorContext extracts HTML around a failed selector for debugging
func captureSelectorContext(html string, selector string, maxSize int) string {
	if html == "" || selector == "" {
		return ""
	}

	// Try to find elements that might match the selector pattern
	// This is a heuristic - we look for class names or IDs from the selector
	patterns := extractSelectorPatterns(selector)

	for _, pattern := range patterns {
		idx := strings.Index(html, pattern)
		if idx != -1 {
			// Extract context around the match
			start := idx - maxSize/2
			if start < 0 {
				start = 0
			}
			end := idx + len(pattern) + maxSize/2
			if end > len(html) {
				end = len(html)
			}

			context := html[start:end]
			// Clean up for readability
			context = strings.ReplaceAll(context, "\n", " ")
			context = strings.ReplaceAll(context, "\t", " ")
			// Collapse multiple spaces
			context = regexp.MustCompile(`\s+`).ReplaceAllString(context, " ")

			return "..." + strings.TrimSpace(context) + "..."
		}
	}

	// If no match found, return beginning of body or document
	bodyIdx := strings.Index(strings.ToLower(html), "<body")
	if bodyIdx != -1 {
		end := bodyIdx + maxSize
		if end > len(html) {
			end = len(html)
		}
		return html[bodyIdx:end] + "..."
	}

	// Last resort: return start of document
	if len(html) > maxSize {
		return html[:maxSize] + "..."
	}
	return html
}

// extractSelectorPatterns extracts searchable patterns from a CSS selector
func extractSelectorPatterns(selector string) []string {
	var patterns []string

	// Extract class names (e.g., ".my-class" -> "my-class")
	classRegex := regexp.MustCompile(`\.([a-zA-Z_-][a-zA-Z0-9_-]*)`)
	for _, match := range classRegex.FindAllStringSubmatch(selector, -1) {
		if len(match) > 1 {
			patterns = append(patterns, match[1])
		}
	}

	// Extract IDs (e.g., "#my-id" -> "my-id")
	idRegex := regexp.MustCompile(`#([a-zA-Z_-][a-zA-Z0-9_-]*)`)
	for _, match := range idRegex.FindAllStringSubmatch(selector, -1) {
		if len(match) > 1 {
			patterns = append(patterns, match[1])
		}
	}

	// Extract tag names with classes (e.g., "div.container" -> "container")
	tagClassRegex := regexp.MustCompile(`[a-z]+\.([a-zA-Z_-][a-zA-Z0-9_-]*)`)
	for _, match := range tagClassRegex.FindAllStringSubmatch(selector, -1) {
		if len(match) > 1 {
			patterns = append(patterns, match[1])
		}
	}

	return patterns
}

// enrichFailedNodesWithContext adds HTML context around failed selectors
func (e *TaskExecutor) enrichFailedNodesWithContext(result *TaskResult, domPath string) {
	// Read DOM file
	domContent, err := os.ReadFile(domPath)
	if err != nil {
		logger.Warn("Failed to read DOM for context enrichment", zap.Error(err))
		return
	}

	html := string(domContent)
	config := DefaultProbeSnapshotConfig()

	// Enrich failed and degraded nodes with selector context
	for i := range result.NodeResults {
		node := &result.NodeResults[i]
		// Include failed nodes OR degraded nodes (0 results)
		isProblematic := node.Status == "failed" ||
			(node.NodeType == "extract_links" && node.LinksFound == 0) ||
			(node.NodeType == "extract" && node.ElementCount == 0)
		if isProblematic && node.Selector != "" {
			node.SelectorContext = captureSelectorContext(html, node.Selector, config.MaxContextSize)
		}
	}
}
