package handlers

import (
	"github.com/gofiber/fiber/v2"
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
	NodeID      string  `json:"node_id"`
	FieldName   string  `json:"field_name,omitempty"` // For update_field_selector
	FixType     string  `json:"fix_type"`             // "update_selector", "update_field_selector", "skip_node"
	OldSelector string  `json:"old_selector,omitempty"`
	NewSelector string  `json:"new_selector,omitempty"`
	Reasoning   string  `json:"reasoning"`
	Confidence  float64 `json:"confidence"`
	AutoApplied bool    `json:"auto_applied"`
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
	)

	// Create auto-fix record
	autoFix := &repository.AutoFixRecord{
		WorkflowID:  req.WorkflowID,
		NodeID:      req.NodeID,
		FixType:     req.FixType,
		OldSelector: req.OldSelector,
		NewSelector: req.NewSelector,
		Reasoning:   req.Reasoning,
		Confidence:  req.Confidence,
		Status:      "applied",
		AutoApplied: true,
	}

	switch req.FixType {
	case "update_selector":
		// Apply selector update to workflow node
		if err := h.probeRepo.UpdateNodeSelector(c.Context(), req.WorkflowID, req.NodeID, req.NewSelector, req.Reasoning); err != nil {
			logger.Error("Failed to update node selector",
				zap.String("workflow_id", req.WorkflowID),
				zap.String("node_id", req.NodeID),
				zap.Error(err),
			)
			autoFix.Status = "failed"
			h.probeRepo.CreateAutoFix(c.Context(), autoFix) // Record failure
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to apply selector update",
			})
		}

		logger.Info("Selector update applied",
			zap.String("workflow_id", req.WorkflowID),
			zap.String("node_id", req.NodeID),
			zap.String("new_selector", req.NewSelector),
		)

	case "update_field_selector":
		// Apply field selector update within an extract node
		if err := h.probeRepo.UpdateFieldSelector(c.Context(), req.WorkflowID, req.NodeID, req.FieldName, req.NewSelector, req.Reasoning); err != nil {
			logger.Error("Failed to update field selector",
				zap.String("workflow_id", req.WorkflowID),
				zap.String("node_id", req.NodeID),
				zap.String("field_name", req.FieldName),
				zap.Error(err),
			)
			autoFix.Status = "failed"
			h.probeRepo.CreateAutoFix(c.Context(), autoFix) // Record failure
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to apply field selector update",
			})
		}

		logger.Info("Field selector update applied",
			zap.String("workflow_id", req.WorkflowID),
			zap.String("node_id", req.NodeID),
			zap.String("field_name", req.FieldName),
			zap.String("new_selector", req.NewSelector),
		)

	case "skip_node":
		// Mark node as skipped/disabled
		if err := h.probeRepo.DisableNode(c.Context(), req.WorkflowID, req.NodeID, req.Reasoning); err != nil {
			logger.Error("Failed to skip node",
				zap.String("workflow_id", req.WorkflowID),
				zap.String("node_id", req.NodeID),
				zap.Error(err),
			)
			autoFix.Status = "failed"
			h.probeRepo.CreateAutoFix(c.Context(), autoFix) // Record failure
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to skip node",
			})
		}

		logger.Info("Node disabled",
			zap.String("workflow_id", req.WorkflowID),
			zap.String("node_id", req.NodeID),
		)

	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown fix type",
		})
	}

	// Record successful auto-fix
	if err := h.probeRepo.CreateAutoFix(c.Context(), autoFix); err != nil {
		logger.Warn("Failed to record auto-fix (fix was still applied)",
			zap.String("workflow_id", req.WorkflowID),
			zap.Error(err),
		)
	} else {
		logger.Info("Auto-fix recorded",
			zap.String("fix_id", autoFix.ID),
			zap.String("workflow_id", req.WorkflowID),
			zap.String("fix_type", req.FixType),
		)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"applied": req.FixType,
		"node_id": req.NodeID,
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

// ApproveAutoFix handles POST /api/v1/probes/auto-fixes/:id/approve
// Approves a pending auto-fix
func (h *ProbeHandler) ApproveAutoFix(c *fiber.Ctx) error {
	fixID := c.Params("id")

	if err := h.probeRepo.UpdateAutoFixStatus(c.Context(), fixID, "applied"); err != nil {
		logger.Error("Failed to approve auto-fix",
			zap.String("fix_id", fixID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to approve auto-fix",
		})
	}

	logger.Info("Auto-fix approved", zap.String("fix_id", fixID))

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
