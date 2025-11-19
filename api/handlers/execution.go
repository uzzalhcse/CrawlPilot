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
	workflowRepo       *storage.WorkflowRepository
	executionRepo      *storage.ExecutionRepository
	extractedItemsRepo *storage.ExtractedItemsRepository
	nodeExecRepo       *storage.NodeExecutionRepository
	browserPool        *browser.BrowserPool
	urlQueue           *queue.URLQueue
	executor           *workflow.Executor
	executions         sync.Map // Track running executions
}

func NewExecutionHandler(
	workflowRepo *storage.WorkflowRepository,
	executionRepo *storage.ExecutionRepository,
	extractedItemsRepo *storage.ExtractedItemsRepository,
	nodeExecRepo *storage.NodeExecutionRepository,
	browserPool *browser.BrowserPool,
	urlQueue *queue.URLQueue,
) *ExecutionHandler {
	return &ExecutionHandler{
		workflowRepo:       workflowRepo,
		executionRepo:      executionRepo,
		extractedItemsRepo: extractedItemsRepo,
		nodeExecRepo:       nodeExecRepo,
		browserPool:        browserPool,
		urlQueue:           urlQueue,
		executor:           workflow.NewExecutor(browserPool, urlQueue, extractedItemsRepo, nodeExecRepo, executionRepo),
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
	ctx := context.Background()
	executionID := c.Params("execution_id")

	// Get execution from database
	execution, err := h.executionRepo.GetByID(ctx, executionID)
	if err != nil {
		logger.Error("Failed to get execution", zap.Error(err))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Execution not found",
		})
	}

	// Get workflow name
	workflow, err := h.workflowRepo.GetByID(ctx, execution.WorkflowID)
	if err == nil {
		execution.WorkflowName = workflow.Name
	}

	// Check if currently running in memory
	_, running := h.executions.Load(executionID)
	if running {
		execution.Status = models.ExecutionStatusRunning
	}

	// Get current queue statistics if running
	if running {
		stats, err := h.urlQueue.GetStats(ctx, executionID)
		if err == nil {
			execution.Stats = models.ExecutionStats{
				URLsProcessed:  stats["completed"],
				ItemsExtracted: stats["items_extracted"],
			}
		}
	}

	return c.JSON(execution)
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

	// Get extracted items (no pagination in repository method, returns all)
	items, err := h.extractedItemsRepo.GetByExecutionID(context.Background(), executionID)
	if err != nil {
		logger.Error("Failed to get extracted items", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve extracted data",
		})
	}

	// Get total count
	total, err := h.extractedItemsRepo.GetCount(context.Background(), executionID)
	if err != nil {
		logger.Error("Failed to count extracted items", zap.Error(err))
		total = 0
	}

	return c.JSON(fiber.Map{
		"execution_id": executionID,
		"items":        items,
		"total":        total,
	})
}

// ListExecutions lists all executions with optional filters
func (h *ExecutionHandler) ListExecutions(c *fiber.Ctx) error {
	ctx := context.Background()

	// Get query parameters
	workflowID := c.Query("workflow_id")
	status := c.Query("status")
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	// Get executions from database
	executions, err := h.executionRepo.List(ctx, workflowID, status, limit, offset)
	if err != nil {
		logger.Error("Failed to list executions", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve executions",
		})
	}

	// Get total count
	total, err := h.executionRepo.Count(ctx, workflowID, status)
	if err != nil {
		logger.Error("Failed to count executions", zap.Error(err))
		total = 0
	}

	// Enrich with workflow names and current running status
	for i := range executions {
		// Get workflow name
		workflow, err := h.workflowRepo.GetByID(ctx, executions[i].WorkflowID)
		if err == nil {
			executions[i].WorkflowName = workflow.Name
		}

		// Check if currently running in memory
		_, running := h.executions.Load(executions[i].ID)
		if running {
			executions[i].Status = models.ExecutionStatusRunning
		}

		// If running, get current stats
		if running {
			stats, err := h.urlQueue.GetStats(ctx, executions[i].ID)
			if err == nil {
				executions[i].Stats = models.ExecutionStats{
					URLsProcessed:  stats["completed"],
					ItemsExtracted: stats["items_extracted"],
				}
			}
		}
	}

	return c.JSON(fiber.Map{
		"executions": executions,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}
