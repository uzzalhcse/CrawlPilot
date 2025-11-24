package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
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
}

// NewHealthCheckHandler creates a new health check handler
func NewHealthCheckHandler(
	workflowRepo *storage.WorkflowRepository,
	healthCheckRepo *storage.HealthCheckRepository,
	browserPool *browser.BrowserPool,
	nodeRegistry *workflow.NodeRegistry,
) *HealthCheckHandler {
	return &HealthCheckHandler{
		workflowRepo:    workflowRepo,
		healthCheckRepo: healthCheckRepo,
		browserPool:     browserPool,
		nodeRegistry:    nodeRegistry,
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

		orchestrator := healthcheck.NewOrchestrator(h.browserPool, h.nodeRegistry, config)
		report, err := orchestrator.RunHealthCheck(bgCtx, wf)

		if err != nil {
			logger.Error("Health check failed", zap.Error(err), zap.String("workflow_id", workflowID))
			return
		}

		// Save report
		if err := h.healthCheckRepo.Create(bgCtx, report); err != nil {
			logger.Error("Failed to save health check report", zap.Error(err))
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
