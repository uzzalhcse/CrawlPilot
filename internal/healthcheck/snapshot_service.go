package healthcheck

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// SnapshotService handles capturing diagnostic data when health checks fail
type SnapshotService struct {
	repo        *storage.SnapshotRepository
	storagePath string
	logger      *zap.Logger
}

// NewSnapshotService creates a new snapshot service
func NewSnapshotService(repo *storage.SnapshotRepository, storagePath string, logger *zap.Logger) *SnapshotService {
	return &SnapshotService{
		repo:        repo,
		storagePath: storagePath,
		logger:      logger,
	}
}

// CaptureSnapshot captures diagnostic data for a failed validation
func (s *SnapshotService) CaptureSnapshot(
	ctx context.Context,
	workflowID, reportID, nodeID, phaseName string,
	validationResult *models.NodeValidationResult,
	browserCtx *browser.BrowserContext,
	node *models.Node, // NEW: Pass node to determine field requirement
) (*models.HealthCheckSnapshot, error) {
	s.logger.Info("Capturing snapshot for failed validation",
		zap.String("node_id", nodeID),
		zap.String("phase", phaseName),
		zap.String("report_id", reportID),
		zap.String("workflow_id", workflowID),
	)

	// Create snapshot directory
	snapshotDir := filepath.Join(s.storagePath, workflowID, reportID)
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	snapshot := &models.HealthCheckSnapshot{
		ReportID:        reportID,
		NodeID:          nodeID,
		PhaseName:       phaseName,
		ElementsFound:   0,
		ConsoleLogsData: []models.ConsoleLog{},
		MetadataData:    make(map[string]interface{}),
	}

	// Get current URL
	url := browserCtx.Page.URL()
	snapshot.URL = url

	// Get page title
	title, err := browserCtx.Page.Title()
	if err == nil && title != "" {
		snapshot.PageTitle = &title
	}

	// Capture screenshot
	screenshotPath, err := s.captureScreenshot(browserCtx, snapshotDir, nodeID)
	if err != nil {
		s.logger.Warn("Failed to capture screenshot", zap.Error(err))
	} else {
		snapshot.ScreenshotPath = &screenshotPath
	}

	// Capture full-page screenshot
	fullScreenshotPath, err := s.captureFullPageScreenshot(browserCtx, snapshotDir, nodeID)
	if err != nil {
		s.logger.Warn("Failed to capture full-page screenshot", zap.Error(err))
	} else {
		// Store in metadata if we need it
		snapshot.MetadataData["full_screenshot_path"] = fullScreenshotPath
	}

	// Capture DOM
	domPath, err := s.captureDOMSnapshot(browserCtx, snapshotDir, nodeID)
	if err != nil {
		s.logger.Warn("Failed to capture DOM", zap.Error(err))
	} else {
		snapshot.DOMSnapshotPath = &domPath
	}

	// Capture console logs
	consoleLogs := s.captureConsoleLogs(browserCtx)
	snapshot.ConsoleLogsData = consoleLogs

	// Extract selector information from validation issues
	for _, issue := range validationResult.Issues {
		if issue.Selector != "" {
			selectorType := "css"
			snapshot.SelectorType = &selectorType
			snapshot.SelectorValue = &issue.Selector

			if actual, ok := issue.Actual.(int); ok {
				snapshot.ElementsFound = actual
			} else if actual, ok := issue.Actual.(float64); ok {
				snapshot.ElementsFound = int(actual)
			}

			errorMsg := issue.Message
			snapshot.ErrorMessage = &errorMsg

			// Determine if this field is required
			// Default to true for backward compatibility
			isRequired := true

			// Check if selector is for a field with required flag
			if fields, ok := node.Params["fields"].(map[string]interface{}); ok {
				for _, fieldConfig := range fields {
					if configMap, ok := fieldConfig.(map[string]interface{}); ok {
						// Check if this is the failing selector
						if sel, ok := configMap["selector"].(string); ok && sel == issue.Selector {
							// Check if 'required' is specified
							if req, exists := configMap["required"]; exists {
								if reqBool, ok := req.(bool); ok {
									isRequired = reqBool
								}
							}
							break
						}
					}
				}
			} else if topLevelSelector, ok := node.Params["selector"].(string); ok && topLevelSelector == issue.Selector {
				// Top-level selector - check for top-level required flag
				if req, exists := node.Params["required"]; exists {
					if reqBool, ok := req.(bool); ok {
						isRequired = reqBool
					}
				}
			}

			snapshot.FieldRequired = &isRequired

			s.logger.Debug("Determined field requirement status",
				zap.String("selector", issue.Selector),
				zap.Bool("required", isRequired))

			break // Take first selector issue
		}
	}

	// Add validation metrics to metadata
	snapshot.MetadataData["node_type"] = validationResult.NodeType
	snapshot.MetadataData["node_name"] = validationResult.NodeName
	snapshot.MetadataData["status"] = string(validationResult.Status)
	snapshot.MetadataData["duration_ms"] = validationResult.Duration
	if len(validationResult.Metrics) > 0 {
		snapshot.MetadataData["metrics"] = validationResult.Metrics
	}

	// Save snapshot to database
	if err := s.repo.Create(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("failed to save snapshot: %w", err)
	}

	s.logger.Info("Snapshot captured successfully",
		zap.String("snapshot_id", snapshot.ID),
		zap.String("node_id", nodeID),
	)

	return snapshot, nil
}

// captureScreenshot takes a screenshot of the current viewport
func (s *SnapshotService) captureScreenshot(browserCtx *browser.BrowserContext, dir, nodeID string) (string, error) {
	buf, err := browserCtx.Page.Screenshot(playwright.PageScreenshotOptions{
		Type: playwright.ScreenshotTypePng,
	})
	if err != nil {
		return "", fmt.Errorf("failed to capture screenshot: %w", err)
	}

	filename := fmt.Sprintf("%s_screenshot_%d.png", nodeID, time.Now().Unix())
	fullPath := filepath.Join(dir, filename)

	if err := os.WriteFile(fullPath, buf, 0644); err != nil {
		return "", fmt.Errorf("failed to write screenshot: %w", err)
	}

	// Return relative path from storage root
	relativePath := filepath.Join(filepath.Base(filepath.Dir(dir)), filepath.Base(dir), filename)
	return relativePath, nil
}

// captureFullPageScreenshot captures the full scrollable page
func (s *SnapshotService) captureFullPageScreenshot(browserCtx *browser.BrowserContext, dir, nodeID string) (string, error) {
	buf, err := browserCtx.Page.Screenshot(playwright.PageScreenshotOptions{
		Type:     playwright.ScreenshotTypePng,
		FullPage: playwright.Bool(true),
	})
	if err != nil {
		return "", fmt.Errorf("failed to capture full-page screenshot: %w", err)
	}

	filename := fmt.Sprintf("%s_fullscreen_%d.png", nodeID, time.Now().Unix())
	fullPath := filepath.Join(dir, filename)

	if err := os.WriteFile(fullPath, buf, 0644); err != nil {
		return "", fmt.Errorf("failed to write full-page screenshot: %w", err)
	}

	// Return relative path
	relativePath := filepath.Join(filepath.Base(filepath.Dir(dir)), filepath.Base(dir), filename)
	return relativePath, nil
}

// captureDOMSnapshot captures and compresses the full HTML DOM
func (s *SnapshotService) captureDOMSnapshot(browserCtx *browser.BrowserContext, dir, nodeID string) (string, error) {
	html, err := browserCtx.Page.Content()
	if err != nil {
		return "", fmt.Errorf("failed to get DOM: %w", err)
	}

	filename := fmt.Sprintf("%s_dom_%d.html.gz", nodeID, time.Now().Unix())
	fullPath := filepath.Join(dir, filename)

	// Compress and save
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create DOM file: %w", err)
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	if _, err := gzWriter.Write([]byte(html)); err != nil {
		return "", fmt.Errorf("failed to write compressed DOM: %w", err)
	}

	// Return relative path
	relativePath := filepath.Join(filepath.Base(filepath.Dir(dir)), filepath.Base(dir), filename)
	return relativePath, nil
}

// captureConsoleLogs captures browser console logs
// Note: Playwright requires console event listeners to be set up before page load
// This is a simplified version - for production, you'd need to set up listeners earlier
func (s *SnapshotService) captureConsoleLogs(browserCtx *browser.BrowserContext) []models.ConsoleLog {
	var logs []models.ConsoleLog

	// Try to get console logs from page evaluation
	// This is best-effort - ideally console listeners should be set up when page loads
	var consoleData interface{}
	_, err := browserCtx.Page.Evaluate(`() => {
		if (window.__crawlifyConsoleLogs) {
			return window.__crawlifyConsoleLogs;
		}
		return [];
	}`, &consoleData)

	if err != nil {
		s.logger.Debug("No console logs captured", zap.Error(err))
		return logs
	}

	// Convert to structured logs
	if logArray, ok := consoleData.([]interface{}); ok {
		for _, logEntry := range logArray {
			if logMap, ok := logEntry.(map[string]interface{}); ok {
				log := models.ConsoleLog{
					Type:      getStringValue(logMap, "type"),
					Message:   getStringValue(logMap, "message"),
					Timestamp: time.Now(),
				}
				logs = append(logs, log)
			}
		}
	}

	return logs
}

// GetSnapshot retrieves a snapshot by ID
func (s *SnapshotService) GetSnapshot(ctx context.Context, id string) (*models.HealthCheckSnapshot, error) {
	return s.repo.GetByID(ctx, id)
}

// GetSnapshotsByReport retrieves all snapshots for a report
func (s *SnapshotService) GetSnapshotsByReport(ctx context.Context, reportID string) ([]*models.HealthCheckSnapshot, error) {
	return s.repo.GetByReportID(ctx, reportID)
}

// GetScreenshotPath returns the full filesystem path for a screenshot
func (s *SnapshotService) GetScreenshotPath(relativePath string) string {
	return filepath.Join(s.storagePath, relativePath)
}

// GetDOMPath returns the full filesystem path for a DOM snapshot
func (s *SnapshotService) GetDOMPath(relativePath string) string {
	return filepath.Join(s.storagePath, relativePath)
}

// Helper function to get string value from map
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
