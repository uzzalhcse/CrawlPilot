package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/service"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// ExecutionHandler handles execution HTTP requests
type ExecutionHandler struct {
	executionSvc *service.ExecutionService
}

// NewExecutionHandler creates a new execution handler
func NewExecutionHandler(executionSvc *service.ExecutionService) *ExecutionHandler {
	return &ExecutionHandler{
		executionSvc: executionSvc,
	}
}

// StartExecution handles POST /api/v1/workflows/:id/execute
func (h *ExecutionHandler) StartExecution(c *fiber.Ctx) error {
	workflowID := c.Params("id")

	execution, err := h.executionSvc.StartExecution(c.Context(), workflowID)
	if err != nil {
		logger.Error("Failed to start execution",
			zap.String("workflow_id", workflowID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"execution_id": execution.ID,
		"workflow_id":  execution.WorkflowID,
		"status":       execution.Status,
		"started_at":   execution.StartedAt,
	})
}

// GetExecution handles GET /api/v1/executions/:id
func (h *ExecutionHandler) GetExecution(c *fiber.Ctx) error {
	id := c.Params("id")

	execution, err := h.executionSvc.GetExecution(c.Context(), id)
	if err != nil {
		logger.Error("Failed to get execution", zap.Error(err))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "execution not found",
		})
	}

	return c.JSON(execution)
}

// ListExecutions handles GET /api/v1/workflows/:id/executions
func (h *ExecutionHandler) ListExecutions(c *fiber.Ctx) error {
	workflowID := c.Params("id")

	filters := repository.ListFilters{
		Limit:  c.QueryInt("limit", 50),
		Offset: c.QueryInt("offset", 0),
		Status: c.Query("status", ""),
	}

	executions, err := h.executionSvc.ListExecutions(c.Context(), workflowID, filters)
	if err != nil {
		logger.Error("Failed to list executions", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list executions",
		})
	}

	return c.JSON(fiber.Map{
		"executions": executions,
		"count":      len(executions),
	})
}

// StopExecution handles DELETE /api/v1/executions/:id
func (h *ExecutionHandler) StopExecution(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.executionSvc.StopExecution(c.Context(), id); err != nil {
		logger.Error("Failed to stop execution", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// GetExecutionErrors handles GET /api/v1/executions/:id/errors
func (h *ExecutionHandler) GetExecutionErrors(c *fiber.Ctx) error {
	id := c.Params("id")
	limit := c.QueryInt("limit", 100)
	offset := c.QueryInt("offset", 0)

	errors, err := h.executionSvc.GetExecutionErrors(c.Context(), id, limit, offset)
	if err != nil {
		logger.Error("Failed to get execution errors",
			zap.String("execution_id", id),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get errors",
		})
	}

	return c.JSON(fiber.Map{
		"errors": errors,
		"count":  len(errors),
	})
}
