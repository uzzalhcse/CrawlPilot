package handlers

import (
	"time"

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

// BatchStatsUpdate represents a single execution's stats in a batch
type BatchStatsUpdate struct {
	ExecutionID    string `json:"execution_id"`
	URLsProcessed  int    `json:"urls_processed"`
	URLsDiscovered int    `json:"urls_discovered"`
	ItemsExtracted int    `json:"items_extracted"`
	Errors         int    `json:"errors"`
}

// BatchStatsRequest is the request body for batch stats updates
type BatchStatsRequest struct {
	Updates   []BatchStatsUpdate `json:"updates"`
	Timestamp time.Time          `json:"timestamp"`
	WorkerID  string             `json:"worker_id,omitempty"`
}

// BatchUpdateStats handles POST /api/v1/internal/stats/batch
// This endpoint reduces database load by accepting multiple execution stats in one request
// Critical for high-throughput scenarios (10k+ URLs/sec)
func (h *StatsHandler) BatchUpdateStats(c *fiber.Ctx) error {
	var req BatchStatsRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if len(req.Updates) == 0 {
		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"message":   "no updates to process",
			"processed": 0,
		})
	}

	// Convert to repository format
	statsUpdates := make([]repository.BatchExecutionStats, 0, len(req.Updates))
	for _, update := range req.Updates {
		statsUpdates = append(statsUpdates, repository.BatchExecutionStats{
			ExecutionID:    update.ExecutionID,
			URLsProcessed:  update.URLsProcessed,
			URLsDiscovered: update.URLsDiscovered,
			ItemsExtracted: update.ItemsExtracted,
			Errors:         update.Errors,
		})
	}

	// Single batch database operation
	if err := h.executionRepo.BatchUpdateStats(c.Context(), statsUpdates); err != nil {
		logger.Error("Failed to batch update execution stats",
			zap.Int("count", len(req.Updates)),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to batch update stats",
		})
	}

	logger.Info("Batch stats updated",
		zap.Int("executions", len(req.Updates)),
		zap.String("worker_id", req.WorkerID),
	)

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message":   "stats updated",
		"processed": len(req.Updates),
	})
}
