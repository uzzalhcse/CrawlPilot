package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
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
	ExecutionID    string                    `json:"execution_id"`
	URLsProcessed  int                       `json:"urls_processed"`
	URLsDiscovered int                       `json:"urls_discovered"`
	ItemsExtracted int                       `json:"items_extracted"`
	Errors         int                       `json:"errors"`
	PhaseStats     map[string]PhaseStatEntry `json:"phase_stats,omitempty"`
}

// PhaseStatEntry holds per-phase statistics
type PhaseStatEntry struct {
	Processed  int `json:"processed"`
	Errors     int `json:"errors"`
	DurationMs int `json:"duration_ms"`
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

	// Single batch database operation for overall stats
	if err := h.executionRepo.BatchUpdateStats(c.Context(), statsUpdates); err != nil {
		logger.Error("Failed to batch update execution stats",
			zap.Int("count", len(req.Updates)),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to batch update stats",
		})
	}

	// Update phase stats for each execution
	for _, update := range req.Updates {
		if len(update.PhaseStats) > 0 {
			// Convert to models format
			phaseStats := make(map[string]models.PhaseStatEntry)
			for phaseID, entry := range update.PhaseStats {
				phaseStats[phaseID] = models.PhaseStatEntry{
					Processed:  entry.Processed,
					Errors:     entry.Errors,
					DurationMs: int64(entry.DurationMs),
				}
			}
			if err := h.executionRepo.UpdatePhaseStats(c.Context(), update.ExecutionID, phaseStats); err != nil {
				logger.Warn("Failed to update phase stats",
					zap.String("execution_id", update.ExecutionID),
					zap.Error(err),
				)
			}
		}
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

// CompleteExecutionRequest is the request body for marking execution complete
type CompleteExecutionRequest struct {
	Status string `json:"status"` // "completed" or "failed"
	Reason string `json:"reason,omitempty"`
}

// CompleteExecution handles POST /api/v1/internal/executions/:id/complete
// Workers call this when they detect an execution should be marked as completed
func (h *StatsHandler) CompleteExecution(c *fiber.Ctx) error {
	executionID := c.Params("id")

	var req CompleteExecutionRequest
	if err := c.BodyParser(&req); err != nil {
		// Default to completed if no body
		req.Status = "completed"
	}

	// Validate status
	if req.Status != "completed" && req.Status != "failed" {
		req.Status = "completed"
	}

	// Mark execution as complete
	if err := h.executionRepo.Complete(c.Context(), executionID, req.Status); err != nil {
		logger.Error("Failed to complete execution",
			zap.String("execution_id", executionID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to complete execution",
		})
	}

	logger.Info("Execution completed",
		zap.String("execution_id", executionID),
		zap.String("status", req.Status),
		zap.String("reason", req.Reason),
	)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "execution completed",
		"status":  req.Status,
	})
}

// BatchErrorsRequest is the request body for batch error inserts
type BatchErrorsRequest struct {
	Errors    map[string][]ErrorEntry `json:"errors"` // keyed by execution_id
	Timestamp time.Time               `json:"timestamp"`
}

// ErrorEntry represents a single error entry
type ErrorEntry struct {
	URL        string    `json:"url"`
	ErrorType  string    `json:"error_type"`
	Message    string    `json:"message"`
	PhaseID    string    `json:"phase_id,omitempty"`
	RetryCount int       `json:"retry_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// BatchInsertErrors handles POST /api/v1/internal/errors/batch
// This endpoint receives batched error logs from workers
func (h *StatsHandler) BatchInsertErrors(c *fiber.Ctx) error {
	var req BatchErrorsRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	totalErrors := 0
	for executionID, errors := range req.Errors {
		// Convert to models format
		modelErrors := make([]models.ExecutionError, 0, len(errors))
		for _, e := range errors {
			modelErrors = append(modelErrors, models.ExecutionError{
				ExecutionID: executionID,
				URL:         e.URL,
				ErrorType:   e.ErrorType,
				Message:     e.Message,
				PhaseID:     e.PhaseID,
				RetryCount:  e.RetryCount,
				CreatedAt:   e.CreatedAt,
			})
		}

		// Batch insert for this execution
		if err := h.executionRepo.BatchInsertErrors(c.Context(), modelErrors); err != nil {
			logger.Error("Failed to batch insert errors",
				zap.String("execution_id", executionID),
				zap.Int("count", len(errors)),
				zap.Error(err),
			)
			// Continue with other executions
			continue
		}
		totalErrors += len(errors)
	}

	logger.Info("Batch errors inserted",
		zap.Int("executions", len(req.Errors)),
		zap.Int("total_errors", totalErrors),
	)

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message":   "errors inserted",
		"processed": totalErrors,
	})
}
