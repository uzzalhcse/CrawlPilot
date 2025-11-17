package handlers

import (
	"context"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/queue"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/internal/workflow"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

type ExecutionHandler struct {
	workflowRepo      *storage.WorkflowRepository
	executionRepo     *storage.ExecutionRepository
	extractedDataRepo *storage.ExtractedDataRepository
	nodeExecRepo      *storage.NodeExecutionRepository
	browserPool       *browser.BrowserPool
	urlQueue          *queue.URLQueue
	executor          *workflow.Executor
	executions        sync.Map // Track running executions
}

func NewExecutionHandler(
	workflowRepo *storage.WorkflowRepository,
	executionRepo *storage.ExecutionRepository,
	extractedDataRepo *storage.ExtractedDataRepository,
	nodeExecRepo *storage.NodeExecutionRepository,
	browserPool *browser.BrowserPool,
	urlQueue *queue.URLQueue,
) *ExecutionHandler {
	return &ExecutionHandler{
		workflowRepo:      workflowRepo,
		executionRepo:     executionRepo,
		extractedDataRepo: extractedDataRepo,
		nodeExecRepo:      nodeExecRepo,
		browserPool:       browserPool,
		urlQueue:          urlQueue,
		executor:          workflow.NewExecutor(browserPool, urlQueue, extractedDataRepo, nodeExecRepo, executionRepo),
	}
}

// StartExecution starts a workflow execution
func (h *ExecutionHandler) StartExecution(c *fiber.Ctx) error {
	workflowID := c.Params("id")

	// Get workflow
	wf, err := h.workflowRepo.GetByID(context.Background(), workflowID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Workflow not found",
		})
	}

	logger.Info("Workflow loaded",
		zap.String("workflow_id", wf.ID),
		zap.Int("start_urls_count", len(wf.Config.StartURLs)),
		zap.Any("start_urls", wf.Config.StartURLs),
	)

	// Check if workflow is active
	if wf.Status != models.WorkflowStatusActive {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Workflow must be in active status to execute",
		})
	}

	// Create execution ID
	executionID := uuid.New().String()

	// Create execution record in database BEFORE starting
	execution := &models.WorkflowExecution{
		ID:         executionID,
		WorkflowID: workflowID,
		Status:     models.ExecutionStatusRunning,
		Stats:      models.ExecutionStats{},
		Context:    models.NewExecutionContext(),
	}

	if err := h.executionRepo.Create(context.Background(), execution); err != nil {
		logger.Error("Failed to create execution record", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create execution",
		})
	}

	// Start execution in background
	go func() {
		ctx := context.Background()
		h.executions.Store(executionID, true)
		defer h.executions.Delete(executionID)

		logger.Info("Starting workflow execution",
			zap.String("workflow_id", workflowID),
			zap.String("execution_id", executionID),
		)

		if err := h.executor.ExecuteWorkflow(ctx, wf, executionID); err != nil {
			logger.Error("Workflow execution failed",
				zap.Error(err),
				zap.String("execution_id", executionID),
			)
		} else {
			logger.Info("Workflow execution completed",
				zap.String("execution_id", executionID),
			)
		}
	}()

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message":      "Workflow execution started",
		"execution_id": executionID,
		"workflow_id":  workflowID,
	})
}

// GetExecutionStatus gets the status of an execution
func (h *ExecutionHandler) GetExecutionStatus(c *fiber.Ctx) error {
	executionID := c.Params("execution_id")

	// Check if execution is running
	_, running := h.executions.Load(executionID)

	// Get queue statistics
	stats, err := h.urlQueue.GetStats(context.Background(), executionID)
	if err != nil {
		logger.Error("Failed to get execution stats", zap.Error(err))
		stats = make(map[string]int)
	}

	return c.JSON(fiber.Map{
		"execution_id": executionID,
		"running":      running,
		"stats":        stats,
	})
}

// StopExecution stops a running execution
func (h *ExecutionHandler) StopExecution(c *fiber.Ctx) error {
	executionID := c.Params("execution_id")

	// Check if execution exists
	_, running := h.executions.Load(executionID)
	if !running {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Execution not found or already stopped",
		})
	}

	// Remove from running executions
	h.executions.Delete(executionID)

	logger.Info("Execution stopped", zap.String("execution_id", executionID))

	return c.JSON(fiber.Map{
		"message":      "Execution stopped",
		"execution_id": executionID,
	})
}

// GetQueueStats gets queue statistics for an execution
func (h *ExecutionHandler) GetQueueStats(c *fiber.Ctx) error {
	executionID := c.Params("execution_id")

	stats, err := h.urlQueue.GetStats(context.Background(), executionID)
	if err != nil {
		logger.Error("Failed to get queue stats", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get queue statistics",
		})
	}

	pendingCount, err := h.urlQueue.GetPendingCount(context.Background(), executionID)
	if err != nil {
		logger.Error("Failed to get pending count", zap.Error(err))
		pendingCount = 0
	}

	return c.JSON(fiber.Map{
		"execution_id":  executionID,
		"stats":         stats,
		"pending_count": pendingCount,
	})
}

// GetExtractedData retrieves extracted data for an execution
func (h *ExecutionHandler) GetExtractedData(c *fiber.Ctx) error {
	executionID := c.Params("execution_id")

	// Parse query parameters for pagination
	limit := c.QueryInt("limit", 100)
	offset := c.QueryInt("offset", 0)

	// Validate pagination parameters
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Get extracted data
	data, err := h.extractedDataRepo.GetByExecutionID(context.Background(), executionID, limit, offset)
	if err != nil {
		logger.Error("Failed to get extracted data", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve extracted data",
		})
	}

	// Get total count
	total, err := h.extractedDataRepo.Count(context.Background(), executionID)
	if err != nil {
		logger.Error("Failed to count extracted data", zap.Error(err))
		total = 0
	}

	return c.JSON(fiber.Map{
		"execution_id": executionID,
		"data":         data,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	})
}
