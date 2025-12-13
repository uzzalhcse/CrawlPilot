package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/driver"
	"go.uber.org/zap"
)

// ProbeSnapshotConfig holds configuration for snapshot capture
type ProbeSnapshotConfig struct {
	// Environment determines storage type ("local", "staging", "production")
	Environment string
	// LocalBasePath is the base directory for local storage
	LocalBasePath string
	// GCSBucket is the GCS bucket for cloud storage
	GCSBucket string
	// GCPProjectID is needed for console URLs
	GCPProjectID string
	// MaxDOMSize limits DOM capture size (default 2MB)
	MaxDOMSize int64
	// MaxContextSize limits selector context size (default 500 chars)
	MaxContextSize int
}

// DefaultProbeSnapshotConfig returns default snapshot configuration (local only)
func DefaultProbeSnapshotConfig() ProbeSnapshotConfig {
	basePath := "snapshots"

	// Try to get the working directory for an absolute path
	if cwd, err := os.Getwd(); err == nil {
		if strings.Contains(cwd, "/microservices/") {
			parts := strings.Split(cwd, "/microservices/")
			basePath = parts[0] + "/snapshots"
		} else {
			basePath = cwd + "/snapshots"
		}
	}

	return ProbeSnapshotConfig{
		Environment:    "local",
		LocalBasePath:  basePath,
		GCSBucket:      "",
		GCPProjectID:   "",
		MaxDOMSize:     2 * 1024 * 1024,
		MaxContextSize: 500,
	}
}

// ProbeSnapshotConfigFromConfig creates a snapshot config from app config
func ProbeSnapshotConfigFromConfig(cfg *config.Config) ProbeSnapshotConfig {
	baseConfig := DefaultProbeSnapshotConfig()

	if cfg == nil {
		return baseConfig
	}

	baseConfig.Environment = cfg.Environment
	baseConfig.GCSBucket = cfg.GCP.StorageBucket
	baseConfig.GCPProjectID = cfg.GCP.ProjectID

	return baseConfig
}

// IsLocal returns true if using local storage
func (c *ProbeSnapshotConfig) IsLocal() bool {
	return c.Environment == "" || c.Environment == "local"
}

// GCSConsolePath generates a GCS console URL for a given object path
func (c *ProbeSnapshotConfig) GCSConsolePath(objectPath string) string {
	return fmt.Sprintf("https://console.cloud.google.com/storage/browser/_details/%s/%s?project=%s",
		c.GCSBucket, objectPath, c.GCPProjectID)
}

// captureNodeSnapshot captures full page DOM and screenshot when a node fails
func (e *TaskExecutor) captureNodeSnapshot(ctx context.Context, page driver.Page, task *models.Task, nodeID string) (*models.ProbeSnapshot, error) {
	if page == nil {
		return nil, fmt.Errorf("page is nil, cannot capture snapshot")
	}

	// Build snapshot config from app config (passed via TaskExecutor)
	snapshotConfig := DefaultProbeSnapshotConfig()
	if e.snapshotConfig != nil {
		// Use storage type from config
		if e.snapshotConfig.StorageType != "" {
			if e.snapshotConfig.StorageType == "gcs" {
				snapshotConfig.Environment = "production" // Force GCS mode
			} else {
				snapshotConfig.Environment = "local"
			}
		}
		// Override local base path if set
		if e.snapshotConfig.LocalBasePath != "" {
			localPath := e.snapshotConfig.LocalBasePath
			// Convert relative path to absolute
			if !filepath.IsAbs(localPath) {
				if cwd, err := os.Getwd(); err == nil {
					// Check if we're in a microservices subdirectory
					if strings.Contains(cwd, "/microservices/") {
						parts := strings.Split(cwd, "/microservices/")
						localPath = filepath.Join(parts[0], strings.TrimPrefix(localPath, "./"))
					} else {
						localPath = filepath.Join(cwd, strings.TrimPrefix(localPath, "./"))
					}
				}
			}
			snapshotConfig.LocalBasePath = localPath
		}
	}
	// Use GCP config for bucket and project ID
	if e.gcpConfig != nil {
		snapshotConfig.GCSBucket = e.gcpConfig.StorageBucket
		snapshotConfig.GCPProjectID = e.gcpConfig.ProjectID
	}

	snapshot := &models.ProbeSnapshot{
		CapturedAt: time.Now().Unix(),
	}

	// Define the path structure for both local and GCS
	snapshotRelPath := filepath.Join(
		"probes",
		task.WorkflowID,
		task.ExecutionID,
		task.PhaseID,
		nodeID,
	)

	// Capture page URL and title
	snapshot.PageURL = task.URL
	title, err := page.Title()
	if err == nil {
		snapshot.PageTitle = title
	}

	// For local environment: store locally
	if snapshotConfig.IsLocal() || e.gcsClient == nil {
		localDir := filepath.Join(snapshotConfig.LocalBasePath, snapshotRelPath)
		if err := os.MkdirAll(localDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
		}

		// Capture DOM locally
		domPath := filepath.Join(localDir, "dom.html")
		if err := e.captureDOM(ctx, page, domPath, snapshotConfig.MaxDOMSize); err != nil {
			logger.Warn("Failed to capture DOM", zap.String("node_id", nodeID), zap.Error(err))
		} else {
			snapshot.DOMPath = domPath
			if fi, err := os.Stat(domPath); err == nil {
				snapshot.DOMSize = fi.Size()
			}
		}

		// Capture screenshot locally
		screenshotPath := filepath.Join(localDir, "screenshot.png")
		width, height, err := e.captureScreenshot(ctx, page, screenshotPath)
		if err != nil {
			logger.Warn("Failed to capture screenshot", zap.String("node_id", nodeID), zap.Error(err))
		} else {
			snapshot.ScreenshotPath = screenshotPath
			snapshot.ImageWidth = width
			snapshot.ImageHeight = height
		}
	} else {
		// For staging/production: upload to GCS and store console URLs
		// Capture DOM to temp file then upload
		domData, err := page.Content()
		if err == nil {
			if int64(len(domData)) > snapshotConfig.MaxDOMSize {
				domData = domData[:snapshotConfig.MaxDOMSize] + "\n<!-- TRUNCATED -->"
			}
			gcsPath := snapshotRelPath + "/dom.html"
			if _, err := e.gcsClient.UploadBytes(ctx, gcsPath, []byte(domData), "text/html"); err != nil {
				logger.Warn("Failed to upload DOM to GCS", zap.String("path", gcsPath), zap.Error(err))
			} else {
				snapshot.DOMPath = snapshotConfig.GCSConsolePath(gcsPath)
				snapshot.DOMSize = int64(len(domData))
			}
		}

		// Capture screenshot and upload
		imgData, err := page.Screenshot(driver.WithFullPage(true))
		if err == nil {
			gcsPath := snapshotRelPath + "/screenshot.png"
			if _, err := e.gcsClient.UploadBytes(ctx, gcsPath, imgData, "image/png"); err != nil {
				logger.Warn("Failed to upload screenshot to GCS", zap.String("path", gcsPath), zap.Error(err))
			} else {
				snapshot.ScreenshotPath = snapshotConfig.GCSConsolePath(gcsPath)
				snapshot.ImageWidth = 1920
				snapshot.ImageHeight = 1080
			}
		} else {
			logger.Warn("Failed to capture screenshot", zap.String("node_id", nodeID), zap.Error(err))
		}
	}

	logger.Debug("Node snapshot captured",
		zap.String("node_id", nodeID),
		zap.String("dom_path", snapshot.DOMPath),
		zap.String("screenshot_path", snapshot.ScreenshotPath),
		zap.Int64("dom_size", snapshot.DOMSize),
		zap.Bool("is_local", snapshotConfig.IsLocal()),
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
