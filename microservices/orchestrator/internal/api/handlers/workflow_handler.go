package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/service"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// WorkflowHandler handles workflow HTTP requests
type WorkflowHandler struct {
	workflowSvc *service.WorkflowService
}

// NewWorkflowHandler creates a new workflow handler
func NewWorkflowHandler(workflowSvc *service.WorkflowService) *WorkflowHandler {
	return &WorkflowHandler{
		workflowSvc: workflowSvc,
	}
}

// CreateWorkflow handles POST /api/v1/workflows
func (h *WorkflowHandler) CreateWorkflow(c *fiber.Ctx) error {
	var workflow models.Workflow

	if err := c.BodyParser(&workflow); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.workflowSvc.CreateWorkflow(c.Context(), &workflow); err != nil {
		logger.Error("Failed to create workflow", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(workflow)
}

// GetWorkflow handles GET /api/v1/workflows/:id
func (h *WorkflowHandler) GetWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")

	workflow, err := h.workflowSvc.GetWorkflow(c.Context(), id)
	if err != nil {
		logger.Error("Failed to get workflow", zap.Error(err))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "workflow not found",
		})
	}

	return c.JSON(workflow)
}

// ListWorkflows handles GET /api/v1/workflows
func (h *WorkflowHandler) ListWorkflows(c *fiber.Ctx) error {
	filters := repository.ListFilters{
		Limit:  c.QueryInt("limit", 50),
		Offset: c.QueryInt("offset", 0),
		Status: c.Query("status", ""),
	}

	workflows, err := h.workflowSvc.ListWorkflows(c.Context(), filters)
	if err != nil {
		logger.Error("Failed to list workflows", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list workflows",
		})
	}

	return c.JSON(fiber.Map{
		"workflows": workflows,
		"count":     len(workflows),
	})
}

// UpdateWorkflow handles PUT /api/v1/workflows/:id
func (h *WorkflowHandler) UpdateWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")

	var workflow models.Workflow
	if err := c.BodyParser(&workflow); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	workflow.ID = id

	if err := h.workflowSvc.UpdateWorkflow(c.Context(), &workflow); err != nil {
		logger.Error("Failed to update workflow", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(workflow)
}

// DeleteWorkflow handles DELETE /api/v1/workflows/:id
func (h *WorkflowHandler) DeleteWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.workflowSvc.DeleteWorkflow(c.Context(), id); err != nil {
		logger.Error("Failed to delete workflow", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}
