package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/service"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// ScheduleHandler handles schedule HTTP requests
type ScheduleHandler struct {
	scheduleSvc  *service.ScheduleService
	executionSvc *service.ExecutionService
}

// NewScheduleHandler creates a new schedule handler
func NewScheduleHandler(scheduleSvc *service.ScheduleService, executionSvc *service.ExecutionService) *ScheduleHandler {
	return &ScheduleHandler{
		scheduleSvc:  scheduleSvc,
		executionSvc: executionSvc,
	}
}

// CreateScheduleRequest represents the request body for creating a schedule
type CreateScheduleRequest struct {
	WorkflowID     string `json:"workflow_id"`
	Name           string `json:"name"`
	CronExpression string `json:"cron_expression"`
	Timezone       string `json:"timezone"`
}

// UpdateScheduleRequest represents the request body for updating a schedule
type UpdateScheduleRequest struct {
	Name           string `json:"name"`
	CronExpression string `json:"cron_expression"`
	Timezone       string `json:"timezone"`
	IsEnabled      *bool  `json:"is_enabled"`
}

// CreateSchedule handles POST /api/v1/schedules
func (h *ScheduleHandler) CreateSchedule(c *fiber.Ctx) error {
	var req CreateScheduleRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate required fields
	if req.WorkflowID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "workflow_id is required",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name is required",
		})
	}

	if req.CronExpression == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cron_expression is required",
		})
	}

	schedule := &models.Schedule{
		WorkflowID:     req.WorkflowID,
		Name:           req.Name,
		CronExpression: req.CronExpression,
		Timezone:       req.Timezone,
	}

	if err := h.scheduleSvc.CreateSchedule(c.Context(), schedule); err != nil {
		logger.Error("Failed to create schedule", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(schedule)
}

// GetSchedule handles GET /api/v1/schedules/:id
func (h *ScheduleHandler) GetSchedule(c *fiber.Ctx) error {
	id := c.Params("id")

	schedule, err := h.scheduleSvc.GetSchedule(c.Context(), id)
	if err != nil {
		logger.Error("Failed to get schedule", zap.Error(err))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "schedule not found",
		})
	}

	return c.JSON(schedule)
}

// ListSchedules handles GET /api/v1/schedules
func (h *ScheduleHandler) ListSchedules(c *fiber.Ctx) error {
	filters := repository.ScheduleFilters{
		Limit:      c.QueryInt("limit", 50),
		Offset:     c.QueryInt("offset", 0),
		WorkflowID: c.Query("workflow_id", ""),
	}

	// Parse is_enabled filter
	if c.Query("is_enabled") != "" {
		isEnabled := c.Query("is_enabled") == "true"
		filters.IsEnabled = &isEnabled
	}

	schedules, err := h.scheduleSvc.ListSchedules(c.Context(), filters)
	if err != nil {
		logger.Error("Failed to list schedules", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list schedules",
		})
	}

	// Calculate stats
	activeCount := 0
	pausedCount := 0
	for _, s := range schedules {
		if s.IsEnabled {
			activeCount++
		} else {
			pausedCount++
		}
	}

	return c.JSON(fiber.Map{
		"schedules": schedules,
		"count":     len(schedules),
		"stats": fiber.Map{
			"total":  len(schedules),
			"active": activeCount,
			"paused": pausedCount,
		},
	})
}

// UpdateSchedule handles PUT /api/v1/schedules/:id
func (h *ScheduleHandler) UpdateSchedule(c *fiber.Ctx) error {
	id := c.Params("id")

	var req UpdateScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Get existing schedule
	schedule, err := h.scheduleSvc.GetSchedule(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "schedule not found",
		})
	}

	// Update fields
	if req.Name != "" {
		schedule.Name = req.Name
	}
	if req.CronExpression != "" {
		schedule.CronExpression = req.CronExpression
	}
	if req.Timezone != "" {
		schedule.Timezone = req.Timezone
	}
	if req.IsEnabled != nil {
		schedule.IsEnabled = *req.IsEnabled
	}

	if err := h.scheduleSvc.UpdateSchedule(c.Context(), schedule); err != nil {
		logger.Error("Failed to update schedule", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(schedule)
}

// DeleteSchedule handles DELETE /api/v1/schedules/:id
func (h *ScheduleHandler) DeleteSchedule(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.scheduleSvc.DeleteSchedule(c.Context(), id); err != nil {
		logger.Error("Failed to delete schedule", zap.Error(err))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ToggleSchedule handles PATCH /api/v1/schedules/:id/toggle
func (h *ScheduleHandler) ToggleSchedule(c *fiber.Ctx) error {
	id := c.Params("id")

	schedule, err := h.scheduleSvc.ToggleSchedule(c.Context(), id)
	if err != nil {
		logger.Error("Failed to toggle schedule", zap.Error(err))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(schedule)
}

// RunNow handles POST /api/v1/schedules/:id/run
// Manually triggers a scheduled workflow execution
func (h *ScheduleHandler) RunNow(c *fiber.Ctx) error {
	id := c.Params("id")

	// Get schedule
	schedule, err := h.scheduleSvc.GetSchedule(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "schedule not found",
		})
	}

	// Start execution
	executionID, err := h.executionSvc.StartScheduledExecution(c.Context(), schedule.WorkflowID, schedule.ID)
	if err != nil {
		logger.Error("Failed to run scheduled workflow", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"execution_id": executionID,
		"schedule_id":  schedule.ID,
		"workflow_id":  schedule.WorkflowID,
	})
}

// GetCronPresets handles GET /api/v1/schedules/presets
func (h *ScheduleHandler) GetCronPresets(c *fiber.Ctx) error {
	presets := h.scheduleSvc.GetCronPresets()
	return c.JSON(fiber.Map{
		"presets": presets,
	})
}
