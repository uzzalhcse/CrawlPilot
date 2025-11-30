package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/internal/storage"
)

type ErrorRecoveryHistoryHandler struct {
	repo *storage.ErrorRecoveryHistoryRepository
}

func NewErrorRecoveryHistoryHandler(repo *storage.ErrorRecoveryHistoryRepository) *ErrorRecoveryHistoryHandler {
	return &ErrorRecoveryHistoryHandler{repo: repo}
}

// GetExecutionHistory retrieves all recovery events for an execution
// GET /api/v1/executions/:id/recovery-history
func (h *ErrorRecoveryHistoryHandler) GetExecutionHistory(c *fiber.Ctx) error {
	executionID := c.Params("id")

	id, err := uuid.Parse(executionID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid execution ID",
		})
	}

	records, err := h.repo.GetByExecutionID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch recovery history",
		})
	}

	return c.JSON(fiber.Map{
		"execution_id": executionID,
		"total_events": len(records),
		"events":       records,
	})
}

// GetWorkflowHistory retrieves recent recovery events for a workflow
// GET /api/v1/workflows/:id/recovery-history?limit=50
func (h *ErrorRecoveryHistoryHandler) GetWorkflowHistory(c *fiber.Ctx) error {
	workflowID := c.Params("id")

	id, err := uuid.Parse(workflowID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid workflow ID",
		})
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	records, err := h.repo.GetByWorkflowID(c.Context(), id, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch recovery history",
		})
	}

	return c.JSON(fiber.Map{
		"workflow_id":  workflowID,
		"limit":        limit,
		"total_events": len(records),
		"events":       records,
	})
}

// GetRecentHistory retrieves the most recent recovery events
// GET /api/v1/error-recovery/history/recent?limit=100
func (h *ErrorRecoveryHistoryHandler) GetRecentHistory(c *fiber.Ctx) error {
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	records, err := h.repo.GetRecent(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch recovery history",
		})
	}

	return c.JSON(fiber.Map{
		"limit":        limit,
		"total_events": len(records),
		"events":       records,
	})
}

// GetStats retrieves aggregated statistics
// GET /api/v1/error-recovery/history/stats?time_range=24h&workflow_id=...&domain=...
func (h *ErrorRecoveryHistoryHandler) GetStats(c *fiber.Ctx) error {
	filter := storage.StatsFilter{}

	// Parse time range
	if timeRange := c.Query("time_range"); timeRange != "" {
		duration, err := time.ParseDuration(timeRange)
		if err == nil {
			now := time.Now()
			startTime := now.Add(-duration)
			filter.StartTime = &startTime
			filter.EndTime = &now
		}
	}

	// Parse workflow ID
	if workflowIDStr := c.Query("workflow_id"); workflowIDStr != "" {
		if id, err := uuid.Parse(workflowIDStr); err == nil {
			filter.WorkflowID = &id
		}
	}

	// Parse domain
	if domain := c.Query("domain"); domain != "" {
		filter.Domain = &domain
	}

	// Parse error type
	if errorType := c.Query("error_type"); errorType != "" {
		filter.ErrorType = &errorType
	}

	stats, err := h.repo.GetStats(c.Context(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch statistics",
		})
	}

	// Add time range info to response
	response := fiber.Map{
		"stats": stats,
	}

	if filter.StartTime != nil && filter.EndTime != nil {
		response["time_range"] = fiber.Map{
			"start": filter.StartTime,
			"end":   filter.EndTime,
		}
	}

	return c.JSON(response)
}
