package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// StatsHandler handles internal stats updates from workers
type StatsHandler struct {
	executionRepo repository.ExecutionRepository
}

// NewStatsHandler creates a new stats handler
func NewStatsHandler(executionRepo repository.ExecutionRepository) *StatsHandler {
	return &StatsHandler{
		executionRepo: executionRepo,
	}
}

// UpdateExecutionStats handles POST /api/v1/internal/executions/:id/stats
func (h *StatsHandler) UpdateExecutionStats(c *fiber.Ctx) error {
	executionID := c.Params("id")

	// Parse request body
	var statsUpdate struct {
		URLsProcessed  int `json:"urls_processed"`
		URLsDiscovered int `json:"urls_discovered"`
		ItemsExtracted int `json:"items_extracted"`
		Errors         int `json:"errors"`
	}

	if err := c.BodyParser(&statsUpdate); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Update stats
	stats := repository.ExecutionStats{
		URLsProcessed:  statsUpdate.URLsProcessed,
		URLsDiscovered: statsUpdate.URLsDiscovered,
		ItemsExtracted: statsUpdate.ItemsExtracted,
		Errors:         statsUpdate.Errors,
	}

	if err := h.executionRepo.UpdateStats(c.Context(), executionID, stats); err != nil {
		logger.Error("Failed to update execution stats",
			zap.String("execution_id", executionID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update stats",
		})
	}

	logger.Debug("Execution stats updated",
		zap.String("execution_id", executionID),
		zap.Int("urls_processed", statsUpdate.URLsProcessed),
	)

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message": "stats updated",
	})
}
