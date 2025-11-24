package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/internal/healthcheck"
	"github.com/uzzalhcse/crawlify/internal/healthcheck/notifications"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// ScheduleHandler handles health check schedule endpoints
type ScheduleHandler struct {
	scheduleRepo     *storage.HealthCheckScheduleRepository
	schedulerService *healthcheck.SchedulerService
}

// NewScheduleHandler creates a new schedule handler
func NewScheduleHandler(
	scheduleRepo *storage.HealthCheckScheduleRepository,
	schedulerService *healthcheck.SchedulerService,
) *ScheduleHandler {
	return &ScheduleHandler{
		scheduleRepo:     scheduleRepo,
		schedulerService: schedulerService,
	}
}

// GetSchedule retrieves the schedule for a workflow
func (h *ScheduleHandler) GetSchedule(c *fiber.Ctx) error {
	workflowID := c.Params("id")
	ctx := context.Background()

	schedule, err := h.scheduleRepo.GetByWorkflowID(ctx, workflowID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Schedule not found",
		})
	}

	return c.JSON(schedule)
}

// CreateSchedule creates or updates a health check schedule
func (h *ScheduleHandler) CreateSchedule(c *fiber.Ctx) error {
	workflowID := c.Params("id")
	ctx := context.Background()

	var req struct {
		Schedule           string                     `json:"schedule"`
		Enabled            bool                       `json:"enabled"`
		NotificationConfig *models.NotificationConfig `json:"notification_config,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Check if schedule already exists
	existingSchedule, err := h.scheduleRepo.GetByWorkflowID(ctx, workflowID)
	if err == nil && existingSchedule != nil {
		// Update existing
		existingSchedule.Schedule = req.Schedule
		existingSchedule.Enabled = req.Enabled
		existingSchedule.NotificationConfig = req.NotificationConfig

		err = h.scheduleRepo.Update(ctx, existingSchedule)
		if err != nil {
			logger.Error("Failed to update schedule", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update schedule",
			})
		}

		return c.JSON(existingSchedule)
	}

	// Create new
	schedule, err := h.schedulerService.CreateSchedule(ctx, workflowID, req.Schedule, req.NotificationConfig)
	if err != nil {
		logger.Error("Failed to create schedule", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create schedule",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(schedule)
}

// DeleteSchedule deletes a health check schedule
func (h *ScheduleHandler) DeleteSchedule(c *fiber.Ctx) error {
	workflowID := c.Params("id")
	ctx := context.Background()

	err := h.scheduleRepo.Delete(ctx, workflowID)
	if err != nil {
		logger.Error("Failed to delete schedule", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete schedule",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// TestNotification sends a test notification
func (h *ScheduleHandler) TestNotification(c *fiber.Ctx) error {
	workflowID := c.Params("id")

	var req struct {
		NotificationConfig *models.NotificationConfig `json:"notification_config"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Create a dummy report for testing
	testReport := &models.HealthCheckReport{
		ID:           "test-notification",
		WorkflowID:   workflowID,
		WorkflowName: "Test Workflow",
		Status:       models.HealthCheckStatusHealthy,
		Summary: &models.HealthCheckSummary{
			TotalNodes:   5,
			PassedNodes:  5,
			FailedNodes:  0,
			WarningNodes: 0,
		},
	}

	// Send test notification
	if req.NotificationConfig != nil && req.NotificationConfig.Slack != nil {
		slackNotifier := notifications.NewSlackNotifier()
		err := slackNotifier.Send(req.NotificationConfig.Slack, testReport, "Test Workflow")
		if err != nil {
			logger.Error("Failed to send test notification", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to send notification: " + err.Error(),
			})
		}
	}

	return c.JSON(fiber.Map{
		"message": "Test notification sent successfully",
	})
}
