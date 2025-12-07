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
