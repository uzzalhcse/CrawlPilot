package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/healthcheck"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/internal/workflow"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// HealthCheckHandler handles health check API requests
type HealthCheckHandler struct {
	workflowRepo    *storage.WorkflowRepository
	healthCheckRepo *storage.HealthCheckRepository
	browserPool     *browser.BrowserPool
	nodeRegistry    *workflow.NodeRegistry
	snapshotService *healthcheck.SnapshotService
}

// NewHealthCheckHandler creates a new health check handler
func NewHealthCheckHandler(
	workflowRepo *storage.WorkflowRepository,
	healthCheckRepo *storage.HealthCheckRepository,
	browserPool *browser.BrowserPool,
	nodeRegistry *workflow.NodeRegistry,
	snapshotService *healthcheck.SnapshotService,
) *HealthCheckHandler {
	return &HealthCheckHandler{
		workflowRepo:    workflowRepo,
		healthCheckRepo: healthCheckRepo,
		browserPool:     browserPool,
		nodeRegistry:    nodeRegistry,
		snapshotService: snapshotService,
	}
}

// RunHealthCheck triggers a health check for a workflow
func (h *HealthCheckHandler) RunHealthCheck(c *fiber.Ctx) error {
	ctx := context.Background()
	workflowID := c.Params("id")

	// Parse config from request body (optional)
	config := &models.HealthCheckConfig{
		MaxURLsPerPhase:    1,
		MaxPaginationPages: 2,
		MaxDepth:           2,
		TimeoutSeconds:     300,
		SkipDataStorage:    true,
	}

	// Try to parse custom config if provided
	if err := c.BodyParser(config); err != nil {
		// Use defaults if parse fails
		logger.Debug("Using default health check config")
	}

	// Get workflow
	wf, err := h.workflowRepo.GetByID(ctx, workflowID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Workflow not found",
		})
	}

	// Run health check in background
	go func() {
		bgCtx := context.Background()

		// Create orchestrator with snapshot service
		orchestrator := healthcheck.NewOrchestrator(h.browserPool, h.nodeRegistry, config, h.snapshotService)

		// Create initial report to get ID
		report := &models.HealthCheckReport{
			ID:         uuid.New().String(), // Generate UUID
			WorkflowID: workflowID,
			Status:     models.HealthCheckStatusRunning,
			StartedAt:  time.Now(),
		}
		logger.Info("Creating initial health check report",
			zap.String("report_id", report.ID),
			zap.String("workflow_id", workflowID))

		if err := h.healthCheckRepo.Create(bgCtx, report); err != nil {
			logger.Error("Failed to create health check report", zap.Error(err))
			return
		}

		// Add workflow_id and report_id to context for snapshot capture
		ctxWithIDs := context.WithValue(bgCtx, "workflowID", workflowID)
		ctxWithIDs = context.WithValue(ctxWithIDs, "reportID", report.ID)

		logger.Info("Starting health check with context IDs",
			zap.String("workflow_id", workflowID),
			zap.String("report_id", report.ID))

		// Run health check with context
		updatedReport, err := orchestrator.RunHealthCheck(ctxWithIDs, wf)

		if err != nil {
			logger.Error("Health check failed", zap.Error(err), zap.String("workflow_id", workflowID))
			// Update report status to failed
			now := time.Now()
			report.Status = models.HealthCheckStatusFailed
			report.CompletedAt = &now
			if updateErr := h.healthCheckRepo.Update(bgCtx, report); updateErr != nil {
				logger.Error("Failed to update failed report", zap.Error(updateErr))
			}
			return
		}

		// Copy results from orchestrator's report to our persisted report
		report.Status = updatedReport.Status
		report.CompletedAt = updatedReport.CompletedAt
		report.Duration = updatedReport.Duration
		report.Results = updatedReport.Results
		report.Summary = updatedReport.Summary

		// Update the report in database
		if err := h.healthCheckRepo.Update(bgCtx, report); err != nil {
			logger.Error("Failed to update health check report", zap.Error(err))
			return
		}

		logger.Info("Health check completed",
			zap.String("workflow_id", workflowID),
			zap.String("report_id", report.ID),
			zap.String("status", string(report.Status)),
		)
	}()

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message":     "Health check started",
		"workflow_id": workflowID,
	})
}

// SetBaseline marks a health check report as the baseline
func (h *HealthCheckHandler) SetBaseline(c *fiber.Ctx) error {
	reportID := c.Params("report_id")
	ctx := context.Background()

	baselineService := healthcheck.NewBaselineService(h.healthCheckRepo)
	err := baselineService.SetAsBaseline(ctx, reportID)

	if err != nil {
		logger.Error("Failed to set baseline",
			zap.String("report_id", reportID),
			zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to set baseline",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Baseline set successfully",
	})
}

// GetBaseline retrieves the baseline report for a workflow
func (h *HealthCheckHandler) GetBaseline(c *fiber.Ctx) error {
	workflowID := c.Params("id")
	ctx := context.Background()

	baselineService := healthcheck.NewBaselineService(h.healthCheckRepo)
	baseline, err := baselineService.GetBaseline(ctx, workflowID)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No baseline found for this workflow",
		})
	}

	return c.JSON(baseline)
}

// CompareWithBaseline compares a report with the baseline
func (h *HealthCheckHandler) CompareWithBaseline(c *fiber.Ctx) error {
	reportID := c.Params("report_id")
	ctx := context.Background()

	// Get current report
	current, err := h.healthCheckRepo.GetByID(ctx, reportID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Report not found",
		})
	}

	// Get baseline
	baselineService := healthcheck.NewBaselineService(h.healthCheckRepo)
	baseline, err := baselineService.GetBaseline(ctx, current.WorkflowID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No baseline found for comparison",
		})
	}

	// Compare
	comparisons := baselineService.CompareWithBaseline(current, baseline)

	return c.JSON(fiber.Map{
		"current":     current,
		"baseline":    baseline,
		"comparisons": comparisons,
	})
}

// GetHealthCheckReport retrieves a health check report by ID
func (h *HealthCheckHandler) GetHealthCheckReport(c *fiber.Ctx) error {
	ctx := context.Background()
	reportID := c.Params("report_id")

	report, err := h.healthCheckRepo.GetByID(ctx, reportID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Health check report not found",
		})
	}

	return c.JSON(report)
}

// ListHealthChecks lists health check reports for a workflow
func (h *HealthCheckHandler) ListHealthChecks(c *fiber.Ctx) error {
	ctx := context.Background()
	workflowID := c.Params("id")

	// Get limit from query param (default 10)
	limit := 10
	if c.Query("limit") != "" {
		l := c.QueryInt("limit", 10)
		if l > 0 {
			limit = l
		}
	}

	reports, err := h.healthCheckRepo.ListByWorkflow(ctx, workflowID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch health checks",
		})
	}

	return c.JSON(fiber.Map{
		"workflow_id": workflowID,
		"reports":     reports,
		"total":       len(reports),
	})
}
