package handlers

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/internal/workflow"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

type WorkflowHandler struct {
	repo   *storage.WorkflowRepository
	parser *workflow.Parser
}

func NewWorkflowHandler(repo *storage.WorkflowRepository) *WorkflowHandler {
	return &WorkflowHandler{
		repo:   repo,
		parser: workflow.NewParser(),
	}
}

// CreateWorkflow creates a new workflow
func (h *WorkflowHandler) CreateWorkflow(c *fiber.Ctx) error {
	var req struct {
		Name        string                `json:"name"`
		Description string                `json:"description"`
		Config      models.WorkflowConfig `json:"config"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate workflow configuration
	if err := h.parser.Validate(&req.Config); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid workflow configuration: %v", err),
		})
	}

	workflow := &models.Workflow{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Config:      req.Config,
		Status:      models.WorkflowStatusDraft,
	}

	if err := h.repo.Create(context.Background(), workflow); err != nil {
		logger.Error("Failed to create workflow", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create workflow",
		})
	}

	logger.Info("Workflow created", zap.String("workflow_id", workflow.ID))
	return c.Status(fiber.StatusCreated).JSON(workflow)
}

// GetWorkflow retrieves a workflow by ID
func (h *WorkflowHandler) GetWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")

	workflow, err := h.repo.GetByID(context.Background(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Workflow not found",
		})
	}

	return c.JSON(workflow)
}

// ListWorkflows lists all workflows
func (h *WorkflowHandler) ListWorkflows(c *fiber.Ctx) error {
	status := models.WorkflowStatus(c.Query("status", ""))
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	workflows, err := h.repo.List(context.Background(), status, limit, offset)
	if err != nil {
		logger.Error("Failed to list workflows", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list workflows",
		})
	}

	return c.JSON(fiber.Map{
		"workflows": workflows,
		"count":     len(workflows),
	})
}

// UpdateWorkflow updates a workflow
func (h *WorkflowHandler) UpdateWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")

	var req struct {
		Name        string                `json:"name"`
		Description string                `json:"description"`
		Config      models.WorkflowConfig `json:"config"`
		Status      models.WorkflowStatus `json:"status"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate workflow configuration
	if err := h.parser.Validate(&req.Config); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid workflow configuration: %v", err),
		})
	}

	workflow := &models.Workflow{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Config:      req.Config,
		Status:      req.Status,
	}

	if err := h.repo.Update(context.Background(), workflow); err != nil {
		logger.Error("Failed to update workflow", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update workflow",
		})
	}

	logger.Info("Workflow updated", zap.String("workflow_id", id))
	return c.JSON(workflow)
}

// DeleteWorkflow deletes a workflow
func (h *WorkflowHandler) DeleteWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.repo.Delete(context.Background(), id); err != nil {
		logger.Error("Failed to delete workflow", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete workflow",
		})
	}

	logger.Info("Workflow deleted", zap.String("workflow_id", id))
	return c.Status(fiber.StatusNoContent).Send(nil)
}

// UpdateWorkflowStatus updates workflow status
func (h *WorkflowHandler) UpdateWorkflowStatus(c *fiber.Ctx) error {
	id := c.Params("id")

	var req struct {
		Status models.WorkflowStatus `json:"status"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.repo.UpdateStatus(context.Background(), id, req.Status); err != nil {
		logger.Error("Failed to update workflow status", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update workflow status",
		})
	}

	logger.Info("Workflow status updated",
		zap.String("workflow_id", id),
		zap.String("status", string(req.Status)),
	)

	return c.JSON(fiber.Map{
		"message": "Workflow status updated successfully",
		"status":  req.Status,
	})
}
