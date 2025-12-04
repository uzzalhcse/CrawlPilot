package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/monitoring"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/internal/workflow"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// MonitoringHandler handles monitoring API requests
type MonitoringHandler struct {
	workflowRepo    *storage.WorkflowRepository
	monitoringRepo  *storage.MonitoringRepository
	browserPool     *browser.BrowserPool
	nodeRegistry    *workflow.NodeRegistry
	snapshotService *monitoring.SnapshotService
}

// NewMonitoringHandler creates a new monitoring handler
func NewMonitoringHandler(
	workflowRepo *storage.WorkflowRepository,
	monitoringRepo *storage.MonitoringRepository,
	browserPool *browser.BrowserPool,
	nodeRegistry *workflow.NodeRegistry,
	snapshotService *monitoring.SnapshotService,
) *MonitoringHandler {
	return &MonitoringHandler{
		workflowRepo:    workflowRepo,
		monitoringRepo:  monitoringRepo,
		browserPool:     browserPool,
		nodeRegistry:    nodeRegistry,
		snapshotService: snapshotService,
	}
}

// RunMonitoring triggers a monitoring for a workflow
func (h *MonitoringHandler) RunMonitoring(c *fiber.Ctx) error {
	ctx := context.Background()
	// Make a copy of the string to be safe for goroutine usage
	paramID := c.Params("id")
	workflowID := string(append([]byte(nil), paramID...))

	// Parse config from request body (optional)
	config := &models.MonitoringConfig{
		MaxURLsPerPhase:    1,
		MaxPaginationPages: 2,
		MaxDepth:           2,
		TimeoutSeconds:     300,
		SkipDataStorage:    true,
	}

	// Try to parse custom config if provided
	if err := c.BodyParser(config); err != nil {
		// Use defaults if parse fails
		logger.Debug("Using default monitoring config")
	}

	// Get workflow
	wf, err := h.workflowRepo.GetByID(ctx, workflowID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Workflow not found",
		})
	}

	// Run monitoring in background
	go func() {
		bgCtx := context.Background()

		// Create orchestrator with snapshot service
		orchestrator := monitoring.NewOrchestrator(h.browserPool, h.nodeRegistry, config, h.snapshotService)

		// Create initial report to get ID
		report := &models.MonitoringReport{
			ID:         uuid.New().String(), // Generate UUID
			WorkflowID: workflowID,
			Status:     models.MonitoringStatusRunning,
			StartedAt:  time.Now(),
		}
		logger.Info("Creating initial monitoring report",
			zap.String("report_id", report.ID),
			zap.String("workflow_id", workflowID))

		if err := h.monitoringRepo.Create(bgCtx, report); err != nil {
			logger.Error("Failed to create monitoring report", zap.Error(err))
			return
		}

		// Add workflow_id and report_id to context for snapshot capture
		ctxWithIDs := context.WithValue(bgCtx, "workflowID", workflowID)
		ctxWithIDs = context.WithValue(ctxWithIDs, "reportID", report.ID)

		logger.Info("Starting monitoring with context IDs",
			zap.String("workflow_id", workflowID),
			zap.String("report_id", report.ID))

		// Run monitoring with context
		updatedReport, err := orchestrator.RunMonitoring(ctxWithIDs, wf)

		if err != nil {
			logger.Error("Monitoring failed", zap.Error(err), zap.String("workflow_id", workflowID))
			// Update report status to failed
			now := time.Now()
			report.Status = models.MonitoringStatusFailed
			report.CompletedAt = &now
			if updateErr := h.monitoringRepo.Update(bgCtx, report); updateErr != nil {
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
		if err := h.monitoringRepo.Update(bgCtx, report); err != nil {
			logger.Error("Failed to update monitoring report", zap.Error(err))
			return
		}

		logger.Info("Monitoring completed",
			zap.String("workflow_id", workflowID),
			zap.String("report_id", report.ID),
			zap.String("status", string(report.Status)),
		)
	}()

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message":     "Monitoring started",
		"workflow_id": workflowID,
	})
}

// SetBaseline marks a monitoring report as the baseline
func (h *MonitoringHandler) SetBaseline(c *fiber.Ctx) error {
	reportID := c.Params("report_id")
	ctx := context.Background()

	baselineService := monitoring.NewBaselineService(h.monitoringRepo)
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
func (h *MonitoringHandler) GetBaseline(c *fiber.Ctx) error {
	workflowID := c.Params("id")
	ctx := context.Background()

	baselineService := monitoring.NewBaselineService(h.monitoringRepo)
	baseline, err := baselineService.GetBaseline(ctx, workflowID)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No baseline found for this workflow",
		})
	}

	return c.JSON(baseline)
}

// CompareWithBaseline compares a report with the baseline
func (h *MonitoringHandler) CompareWithBaseline(c *fiber.Ctx) error {
	reportID := c.Params("report_id")
	ctx := context.Background()

	// Get current report
	current, err := h.monitoringRepo.GetByID(ctx, reportID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Report not found",
		})
	}

	// Get baseline
	baselineService := monitoring.NewBaselineService(h.monitoringRepo)
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

// GetMonitoringReport retrieves a monitoring report by ID
func (h *MonitoringHandler) GetMonitoringReport(c *fiber.Ctx) error {
	ctx := context.Background()
	reportID := c.Params("report_id")

	report, err := h.monitoringRepo.GetByID(ctx, reportID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Monitoring report not found",
		})
	}

	return c.JSON(report)
}

// ListMonitoring lists monitoring reports for a workflow
func (h *MonitoringHandler) ListMonitoring(c *fiber.Ctx) error {
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

	reports, err := h.monitoringRepo.ListByWorkflow(ctx, workflowID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch monitorings",
		})
	}

	return c.JSON(fiber.Map{
		"workflow_id": workflowID,
		"reports":     reports,
		"total":       len(reports),
	})
}
