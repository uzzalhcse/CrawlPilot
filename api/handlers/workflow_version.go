package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// WorkflowVersionHandler handles workflow version requests
type WorkflowVersionHandler struct {
	versionRepo  *storage.WorkflowVersionRepository
	workflowRepo *storage.WorkflowRepository
}

// NewWorkflowVersionHandler creates a new workflow version handler
func NewWorkflowVersionHandler(versionRepo *storage.WorkflowVersionRepository, workflowRepo *storage.WorkflowRepository) *WorkflowVersionHandler {
	return &WorkflowVersionHandler{
		versionRepo:  versionRepo,
		workflowRepo: workflowRepo,
	}
}

// ListVersions retrieves versions for a workflow
// GET /api/v1/workflows/:id/versions
func (h *WorkflowVersionHandler) ListVersions(c *fiber.Ctx) error {
	workflowID := c.Params("id")
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	versions, err := h.versionRepo.List(c.Context(), workflowID, limit, offset)
	if err != nil {
		logger.Error("Failed to list workflow versions", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list workflow versions",
		})
	}

	return c.JSON(versions)
}

// RollbackVersion rolls back a workflow to a specific version
// POST /api/v1/workflows/:id/rollback/:version
func (h *WorkflowVersionHandler) RollbackVersion(c *fiber.Ctx) error {
	workflowID := c.Params("id")
	versionStr := c.Params("version")

	versionNum, err := strconv.Atoi(versionStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid version number",
		})
	}

	// Get the target version
	targetVersion, err := h.versionRepo.GetByVersion(c.Context(), workflowID, versionNum)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Version not found",
		})
	}

	// Get current workflow
	workflow, err := h.workflowRepo.GetByID(c.Context(), workflowID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Workflow not found",
		})
	}

	// Create a new version (backup of current state) before rollback?
	// Or just create a new version representing the rollback state?
	// Let's create a new version representing the rollback state (Version N+1)
	// This preserves history better than just overwriting.

	newVersionNum := workflow.Version + 1

	newVersion := &models.WorkflowVersion{
		WorkflowID:   workflowID,
		Version:      newVersionNum,
		Config:       targetVersion.Config, // Restore config from target version
		ChangeReason: "Rollback to version " + versionStr,
	}

	if err := h.versionRepo.Create(c.Context(), newVersion); err != nil {
		logger.Error("Failed to create rollback version", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create rollback version",
		})
	}

	// Update workflow with restored config and new version
	workflow.Config = targetVersion.Config
	workflow.Version = newVersionNum

	if err := h.workflowRepo.Update(c.Context(), workflow); err != nil {
		logger.Error("Failed to update workflow", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update workflow",
		})
	}

	logger.Info("Workflow rolled back",
		zap.String("workflow_id", workflowID),
		zap.Int("from_version", versionNum),
		zap.Int("to_version", newVersionNum))

	return c.JSON(workflow)
}
