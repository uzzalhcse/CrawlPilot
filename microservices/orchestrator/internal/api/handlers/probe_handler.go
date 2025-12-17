package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// ProbeHandler handles probe-related HTTP requests
type ProbeHandler struct {
	probeRepo     repository.ProbeRepository
	executionRepo repository.ExecutionRepository
}

// NewProbeHandler creates a new probe handler
func NewProbeHandler(probeRepo repository.ProbeRepository, executionRepo repository.ExecutionRepository) *ProbeHandler {
	return &ProbeHandler{
		probeRepo:     probeRepo,
		executionRepo: executionRepo,
	}
}

// SaveProbeResultRequest is the request body for saving probe results
type SaveProbeResultRequest struct {
	ExecutionID string                    `json:"execution_id"`
	WorkflowID  string                    `json:"workflow_id"`
	Status      string                    `json:"status"` // healthy, degraded, broken
	DurationMs  int                       `json:"duration_ms"`
	Phases      []models.PhaseProbeResult `json:"phases"`
}

// SaveProbeResult handles POST /api/v1/internal/probes/result
// Called by worker when probe execution completes
func (h *ProbeHandler) SaveProbeResult(c *fiber.Ctx) error {
	var req SaveProbeResultRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Create probe result
	result := &models.ProbeResult{
		ExecutionID: req.ExecutionID,
		WorkflowID:  req.WorkflowID,
		Status:      req.Status,
		Duration:    int64(req.DurationMs),
		Phases:      req.Phases,
	}

	// Save to database
	if err := h.probeRepo.SaveProbeResult(c.Context(), result); err != nil {
		logger.Error("Failed to save probe result",
			zap.String("execution_id", req.ExecutionID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save probe result",
		})
	}

	// Update execution probe_status
	if err := h.probeRepo.UpdateProbeStatus(c.Context(), req.ExecutionID, req.Status); err != nil {
		logger.Warn("Failed to update probe status on execution",
			zap.String("execution_id", req.ExecutionID),
			zap.Error(err),
		)
	}

	logger.Info("Probe result saved",
		zap.String("execution_id", req.ExecutionID),
		zap.String("status", req.Status),
		zap.Int("phase_count", len(req.Phases)),
	)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":     result.ID,
		"status": result.Status,
	})
}

// SaveSampleURLRequest is the request body for saving sample URLs
type SaveSampleURLRequest struct {
	WorkflowID string            `json:"workflow_id"`
	SampleURLs map[string]string `json:"sample_urls"` // phase_id -> sample_url
}

// SaveSampleURLs handles POST /api/v1/internal/probes/sample-urls
// Called by worker to save sample URLs from successful executions
func (h *ProbeHandler) SaveSampleURLs(c *fiber.Ctx) error {
	var req SaveSampleURLRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	savedCount := 0
	for phaseID, sampleURL := range req.SampleURLs {
		if err := h.probeRepo.SaveSampleURL(c.Context(), req.WorkflowID, phaseID, sampleURL); err != nil {
			logger.Warn("Failed to save sample URL",
				zap.String("workflow_id", req.WorkflowID),
				zap.String("phase_id", phaseID),
				zap.Error(err),
			)
			continue
		}
		savedCount++
	}

	logger.Info("Sample URLs saved",
		zap.String("workflow_id", req.WorkflowID),
		zap.Int("count", savedCount),
	)

	return c.JSON(fiber.Map{
		"saved": savedCount,
	})
}

// GetProbeResults handles GET /api/v1/workflows/:id/probe/results
// Returns probe history with detailed results
func (h *ProbeHandler) GetProbeResults(c *fiber.Ctx) error {
	workflowID := c.Params("id")
	limit := c.QueryInt("limit", 10)

	results, err := h.probeRepo.GetProbeResults(c.Context(), workflowID, limit)
	if err != nil {
		logger.Error("Failed to get probe results",
			zap.String("workflow_id", workflowID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get probe results",
		})
	}

	return c.JSON(fiber.Map{
		"results": results,
		"count":   len(results),
	})
}

// GetLatestProbeResult handles GET /api/v1/workflows/:id/probe/latest
// Returns the most recent probe result
func (h *ProbeHandler) GetLatestProbeResult(c *fiber.Ctx) error {
	workflowID := c.Params("id")

	result, err := h.probeRepo.GetLatestProbeResult(c.Context(), workflowID)
	if err != nil {
		logger.Warn("No probe results found",
			zap.String("workflow_id", workflowID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "no probe results found",
		})
	}

	return c.JSON(result)
}

// GetSampleURLs handles GET /api/v1/workflows/:id/probe/sample-urls
// Returns stored sample URLs for a workflow
func (h *ProbeHandler) GetSampleURLs(c *fiber.Ctx) error {
	workflowID := c.Params("id")

	urls, err := h.probeRepo.GetAllSampleURLs(c.Context(), workflowID)
	if err != nil {
		logger.Error("Failed to get sample URLs",
			zap.String("workflow_id", workflowID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get sample URLs",
		})
	}

	return c.JSON(fiber.Map{
		"workflow_id": workflowID,
		"sample_urls": urls,
	})
}

// GetBaseline handles GET /api/v1/internal/probes/baseline/:id
// Returns the last successful probe result for baseline comparison
func (h *ProbeHandler) GetBaseline(c *fiber.Ctx) error {
	workflowID := c.Params("id")

	// Get last healthy probe result
	result, err := h.probeRepo.GetBaselineProbeResult(c.Context(), workflowID)
	if err != nil {
		logger.Warn("No baseline probe result found",
			zap.String("workflow_id", workflowID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "no baseline found",
		})
	}

	return c.JSON(result)
}

// WorkflowFixRequest is the request body for applying workflow fixes
type WorkflowFixRequest struct {
	WorkflowID  string  `json:"workflow_id"`
	ExecutionID string  `json:"execution_id"`
	NodeID      string  `json:"node_id"`
	FieldName   string  `json:"field_name,omitempty"` // For update_field_selector
	FixType     string  `json:"fix_type"`             // "update_selector", "update_field_selector", "skip_node"
	OldSelector string  `json:"old_selector,omitempty"`
	NewSelector string  `json:"new_selector,omitempty"`
	Reasoning   string  `json:"reasoning"`
	Confidence  float64 `json:"confidence"`
	AutoApplied bool    `json:"auto_applied"`
}

// applyFixToWorkflow applies the fix logic to the workflow configuration
func (h *ProbeHandler) applyFixToWorkflow(ctx context.Context, fix *models.AutoFixRecord) error {
	switch fix.FixType {
	case "update_selector":
		// Apply selector update to workflow node
		if err := h.probeRepo.UpdateNodeSelector(ctx, fix.WorkflowID, fix.NodeID, fix.NewSelector, fix.Reasoning); err != nil {
			return fmt.Errorf("failed to apply selector update: %w", err)
		}
		logger.Info("Selector update applied",
			zap.String("workflow_id", fix.WorkflowID),
			zap.String("node_id", fix.NodeID),
			zap.String("new_selector", fix.NewSelector),
		)

	case "update_field_selector":
		if fix.FieldName == "" {
			return fmt.Errorf("missing field_name for update_field_selector fix")
		}
		if err := h.probeRepo.UpdateFieldSelector(ctx, fix.WorkflowID, fix.NodeID, fix.FieldName, fix.NewSelector, fix.Reasoning); err != nil {
			return fmt.Errorf("failed to apply field selector update: %w", err)
		}
		logger.Info("Field selector update applied",
			zap.String("workflow_id", fix.WorkflowID),
			zap.String("node_id", fix.NodeID),
			zap.String("field_name", fix.FieldName),
			zap.String("new_selector", fix.NewSelector),
		)

	case "skip_node":
		// Mark node as skipped/disabled
		if err := h.probeRepo.DisableNode(ctx, fix.WorkflowID, fix.NodeID, fix.Reasoning); err != nil {
			return fmt.Errorf("failed to skip node: %w", err)
		}
		logger.Info("Node disabled",
			zap.String("workflow_id", fix.WorkflowID),
			zap.String("node_id", fix.NodeID),
		)

	default:
		return fmt.Errorf("unknown fix type: %s", fix.FixType)
	}
	return nil
}

// ApplyWorkflowFix handles POST /api/v1/internal/probes/fix
// Applies AI-suggested workflow fixes (selector updates, skip nodes)
func (h *ProbeHandler) ApplyWorkflowFix(c *fiber.Ctx) error {
	var req WorkflowFixRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	logger.Info("Received workflow fix request",
		zap.String("workflow_id", req.WorkflowID),
		zap.String("node_id", req.NodeID),
		zap.String("fix_type", req.FixType),
		zap.Float64("confidence", req.Confidence),
		zap.Bool("auto_applied", req.AutoApplied),
	)

	// Create auto-fix record
	autoFix := &models.AutoFixRecord{
		ID:          uuid.New().String(),
		WorkflowID:  req.WorkflowID,
		ExecutionID: req.ExecutionID,
		NodeID:      req.NodeID,
		FieldName:   req.FieldName,
		FixType:     req.FixType,
		OldSelector: req.OldSelector,
		NewSelector: req.NewSelector,
		Reasoning:   req.Reasoning,
		Confidence:  req.Confidence,
		Status:      "pending", // Default to pending
		AutoApplied: req.AutoApplied,
	}

	// If auto-applied is true, try to apply immediately
	if req.AutoApplied {
		// Special handling for field selector since we have the field name in request
		if req.FixType == "update_field_selector" {
			if err := h.probeRepo.UpdateFieldSelector(c.Context(), req.WorkflowID, req.NodeID, req.FieldName, req.NewSelector, req.Reasoning); err != nil {
				logger.Error("Failed to update field selector", zap.Error(err))
				autoFix.Status = "failed"
			} else {
				autoFix.Status = "applied"
				logger.Info("Field selector update applied immediately")
			}
		} else {
			// Use helper for other types
			if err := h.applyFixToWorkflow(c.Context(), autoFix); err != nil {
				logger.Error("Failed to auto-apply fix", zap.Error(err))
				autoFix.Status = "failed"
			} else {
				autoFix.Status = "applied"
			}
		}
	}

	// Record the fix (whether pending, applied, or failed)
	if err := h.probeRepo.CreateAutoFix(c.Context(), autoFix); err != nil {
		logger.Warn("Failed to record auto-fix", zap.Error(err))
		// If it was applied but failed to record, we still return success but log error
	} else {
		logger.Info("Auto-fix recorded",
			zap.String("fix_id", autoFix.ID),
			zap.String("status", autoFix.Status),
		)

		// Update probe status if applied
		if autoFix.Status == "applied" && req.ExecutionID != "" {
			h.probeRepo.UpdateProbeStatus(c.Context(), req.ExecutionID, "fixed")
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"applied": autoFix.Status == "applied",
		"status":  autoFix.Status,
		"fix_id":  autoFix.ID,
	})
}

// GetRecentProbes handles GET /api/v1/probes/recent
// Returns recent probe results across all workflows
func (h *ProbeHandler) GetRecentProbes(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 20)

	results, total, err := h.probeRepo.GetRecentProbes(c.Context(), limit)
	if err != nil {
		logger.Error("Failed to get recent probes", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get recent probes",
		})
	}

	return c.JSON(fiber.Map{
		"probes": results,
		"total":  total,
	})
}

// ProbeStats holds aggregate probe statistics
type ProbeStats struct {
	Total      int            `json:"total"`
	Healthy    int            `json:"healthy"`
	Degraded   int            `json:"degraded"`
	Broken     int            `json:"broken"`
	ByWorkflow map[string]int `json:"by_workflow"`
}

// GetProbeStats handles GET /api/v1/probes/stats
// Returns aggregate statistics across all probes
func (h *ProbeHandler) GetProbeStats(c *fiber.Ctx) error {
	stats, err := h.probeRepo.GetProbeStats(c.Context())
	if err != nil {
		logger.Error("Failed to get probe stats", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get probe stats",
		})
	}

	return c.JSON(stats)
}

// AutoFix represents an AI-suggested workflow fix
type AutoFix struct {
	ID          string  `json:"id"`
	WorkflowID  string  `json:"workflow_id"`
	ExecutionID string  `json:"execution_id"`
	NodeID      string  `json:"node_id"`
	FixType     string  `json:"fix_type"`
	OldSelector string  `json:"old_selector,omitempty"`
	NewSelector string  `json:"new_selector,omitempty"`
	Reasoning   string  `json:"reasoning"`
	Confidence  float64 `json:"confidence"`
	Status      string  `json:"status"` // applied, pending, rejected
	AutoApplied bool    `json:"auto_applied"`
	CreatedAt   string  `json:"created_at"`
}

// GetAutoFixes handles GET /api/v1/probes/auto-fixes
// Returns list of auto-fixes with optional status filter
func (h *ProbeHandler) GetAutoFixes(c *fiber.Ctx) error {
	status := c.Query("status", "")
	limit := c.QueryInt("limit", 50)

	fixes, total, err := h.probeRepo.GetAutoFixes(c.Context(), status, limit)
	if err != nil {
		logger.Error("Failed to get auto-fixes", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get auto-fixes",
		})
	}

	return c.JSON(fiber.Map{
		"fixes": fixes,
		"total": total,
	})
}

// ApproveAutoFixRequest is the request body for approving an auto-fix
type ApproveAutoFixRequest struct {
	NewSelector string `json:"new_selector,omitempty"`
}

// PreviewSelectorRequest is the request body for previewing a selector
type PreviewSelectorRequest struct {
	ExecutionID string `json:"execution_id"`
	NodeID      string `json:"node_id"`
	Selector    string `json:"selector"`
}

// PreviewSelectorResponse is the response for selector preview
type PreviewSelectorResponse struct {
	MatchCount int      `json:"match_count"`
	Matches    []string `json:"matches"`
	Error      string   `json:"error,omitempty"`
}

// PreviewSelector handles POST /api/v1/probes/preview-selector
// Previews what a selector would extract from the stored DOM snapshot
func (h *ProbeHandler) PreviewSelector(c *fiber.Ctx) error {
	var req PreviewSelectorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.ExecutionID == "" || req.NodeID == "" || req.Selector == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "execution_id, node_id, and selector are required",
		})
	}

	// 1. Get snapshot path
	domPath, err := h.probeRepo.GetSnapshotPath(c.Context(), req.ExecutionID, req.NodeID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "snapshot not found: " + err.Error(),
		})
	}

	// 2. Read DOM file
	// Note: This assumes local file access. For GCS, we'd need to download it.
	// Since we are in dev/local mode, this works.
	// In prod, we might need to handle GCS paths (download via storage client).
	// For now, we check if it's a local file.
	f, err := os.Open(domPath)
	if err != nil {
		logger.Error("Failed to open DOM file", zap.String("path", domPath), zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to read snapshot file at %s: %v", domPath, err),
		})
	}
	defer f.Close()

	// 3. Parse HTML
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		logger.Error("Failed to parse DOM", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to parse snapshot HTML",
		})
	}

	// 4. Run selector
	var matches []string
	doc.Find(req.Selector).Each(func(i int, s *goquery.Selection) {
		if len(matches) < 5 { // Limit to first 5 matches
			text := strings.TrimSpace(s.Text())
			if text != "" {
				// Truncate long text
				if len(text) > 100 {
					text = text[:97] + "..."
				}
				matches = append(matches, text)
			}
		}
	})

	return c.JSON(PreviewSelectorResponse{
		MatchCount: doc.Find(req.Selector).Length(),
		Matches:    matches,
	})
}

// ApproveAutoFix handles POST /api/v1/probes/auto-fixes/:id/approve
// Approves a pending auto-fix, optionally overriding the selector
func (h *ProbeHandler) ApproveAutoFix(c *fiber.Ctx) error {
	fixID := c.Params("id")

	// Parse optional body for overrides
	var req ApproveAutoFixRequest
	if err := c.BodyParser(&req); err != nil {
		// Ignore error as body is optional
	}

	// 1. Get the fix record
	fix, err := h.probeRepo.GetAutoFixByID(c.Context(), fixID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "auto-fix not found",
		})
	}

	if fix.Status == "applied" {
		return c.JSON(fiber.Map{"success": true, "status": "applied", "message": "already applied"})
	}

	// 2. Apply overrides if provided
	if req.NewSelector != "" && req.NewSelector != fix.NewSelector {
		logger.Info("Applying manual override to auto-fix",
			zap.String("fix_id", fixID),
			zap.String("old_new_selector", fix.NewSelector),
			zap.String("manual_selector", req.NewSelector),
		)
		fix.NewSelector = req.NewSelector
		fix.Reasoning += " (Manually modified by user)"
		fix.Confidence = 1.0
		fix.AutoApplied = false // It was manually approved/modified
	}

	// 3. Apply the fix
	if err := h.applyFixToWorkflow(c.Context(), fix); err != nil {
		logger.Error("Failed to apply approved fix", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to apply fix: " + err.Error(),
		})
	}

	// 4. Update status and record changes in DB
	// We use UpdateAutoFix to persist any manual overrides (new selector, reasoning) and set status to applied
	fix.Status = "applied"
	if err := h.probeRepo.UpdateAutoFix(c.Context(), fix); err != nil {
		logger.Error("Failed to update auto-fix record", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update fix record",
		})
	}

	// 5. Update probe status
	if fix.ExecutionID != "" {
		h.probeRepo.UpdateProbeStatus(c.Context(), fix.ExecutionID, "fixed")
	}

	logger.Info("Auto-fix approved and applied", zap.String("fix_id", fixID))

	return c.JSON(fiber.Map{
		"success": true,
		"id":      fixID,
		"status":  "applied",
	})
}

// RejectAutoFix handles POST /api/v1/probes/auto-fixes/:id/reject
// Rejects a pending auto-fix
func (h *ProbeHandler) RejectAutoFix(c *fiber.Ctx) error {
	fixID := c.Params("id")

	if err := h.probeRepo.UpdateAutoFixStatus(c.Context(), fixID, "rejected"); err != nil {
		logger.Error("Failed to reject auto-fix",
			zap.String("fix_id", fixID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to reject auto-fix",
		})
	}

	logger.Info("Auto-fix rejected", zap.String("fix_id", fixID))

	return c.JSON(fiber.Map{
		"success": true,
		"id":      fixID,
		"status":  "rejected",
	})
}

// GetSnapshotFile handles GET /api/v1/snapshots/file
// Serves local snapshot files (screenshots, DOM) for viewing in the UI
// Query params: path (the local file path from probe snapshot)
func (h *ProbeHandler) GetSnapshotFile(c *fiber.Ctx) error {
	filePath := c.Query("path")
	if filePath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "path query parameter is required",
		})
	}

	// Security: Ensure path contains "snapshots" directory to prevent path traversal
	// This works for both absolute paths (like /home/.../snapshots/...) and relative paths
	if !strings.Contains(filePath, "snapshots") {
		logger.Warn("Attempt to access file outside snapshots directory",
			zap.String("path", filePath),
		)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "access denied - only snapshot files can be served",
		})
	}

	// Clean the path to prevent directory traversal
	cleanPath := filepath.Clean(filePath)
	if strings.Contains(cleanPath, "..") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "invalid path",
		})
	}

	// If path is not absolute, try to make it absolute based on project structure
	if !filepath.IsAbs(cleanPath) {
		// Try current working directory
		if cwd, err := os.Getwd(); err == nil {
			// Check if we're in a microservices subdirectory
			if strings.Contains(cwd, "/microservices/") {
				parts := strings.Split(cwd, "/microservices/")
				cleanPath = filepath.Join(parts[0], cleanPath)
			} else {
				cleanPath = filepath.Join(cwd, cleanPath)
			}
		}
	}

	// Check if file exists
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		logger.Debug("Snapshot file not found",
			zap.String("path", cleanPath),
		)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "snapshot file not found",
			"path":  cleanPath,
		})
	}

	// Determine content type based on file extension
	ext := strings.ToLower(filepath.Ext(cleanPath))
	switch ext {
	case ".png":
		c.Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		c.Set("Content-Type", "image/jpeg")
	case ".html":
		c.Set("Content-Type", "text/html; charset=utf-8")
	default:
		c.Set("Content-Type", "application/octet-stream")
	}

	return c.SendFile(cleanPath)
}
