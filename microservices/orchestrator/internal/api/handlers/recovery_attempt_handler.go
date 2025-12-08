package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// RecoveryAttemptHandler handles recovery attempt HTTP requests
type RecoveryAttemptHandler struct {
	attemptRepo *repository.RecoveryAttemptRepository
}

// NewRecoveryAttemptHandler creates a new recovery attempt handler
func NewRecoveryAttemptHandler(attemptRepo *repository.RecoveryAttemptRepository) *RecoveryAttemptHandler {
	return &RecoveryAttemptHandler{
		attemptRepo: attemptRepo,
	}
}

// GetAttempts handles GET /api/v1/recovery/attempts
func (h *RecoveryAttemptHandler) GetAttempts(c *fiber.Ctx) error {
	executionID := c.Query("execution_id")
	status := c.Query("status")
	pattern := c.Query("pattern")

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	attempts, total, err := h.attemptRepo.GetAttempts(c.Context(), executionID, status, pattern, limit, offset)
	if err != nil {
		logger.Error("Failed to get recovery attempts", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get recovery attempts",
		})
	}

	return c.JSON(fiber.Map{
		"attempts": attempts,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetAttemptStats handles GET /api/v1/recovery/attempts/stats
func (h *RecoveryAttemptHandler) GetAttemptStats(c *fiber.Ctx) error {
	stats, err := h.attemptRepo.GetStats(c.Context())
	if err != nil {
		logger.Error("Failed to get recovery attempt stats", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get stats",
		})
	}

	patternStats, _ := h.attemptRepo.GetPatternStats(c.Context())
	stats["by_pattern"] = patternStats

	return c.JSON(stats)
}

// CreateAttemptRequest represents a request to create a recovery attempt
type CreateAttemptRequest struct {
	ExecutionID  string `json:"execution_id"`
	TaskID       string `json:"task_id"`
	WorkflowID   string `json:"workflow_id"`
	URL          string `json:"url"`
	Domain       string `json:"domain"`
	ErrorPattern string `json:"error_pattern"`
	ErrorMessage string `json:"error_message,omitempty"`
	StatusCode   int    `json:"status_code,omitempty"`
}

// CreateAttempt handles POST /api/v1/internal/recovery/attempt
// Called by worker immediately when error is detected
func (h *RecoveryAttemptHandler) CreateAttempt(c *fiber.Ctx) error {
	var req CreateAttemptRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.ExecutionID == "" || req.ErrorPattern == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "execution_id and error_pattern are required",
		})
	}

	attempt := &repository.RecoveryAttempt{
		ExecutionID:  req.ExecutionID,
		TaskID:       req.TaskID,
		WorkflowID:   req.WorkflowID,
		URL:          req.URL,
		Domain:       req.Domain,
		ErrorPattern: req.ErrorPattern,
		ErrorMessage: req.ErrorMessage,
		StatusCode:   req.StatusCode,
	}

	if err := h.attemptRepo.CreateAttempt(c.Context(), attempt); err != nil {
		logger.Error("Failed to create recovery attempt", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create recovery attempt",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(attempt)
}

// BatchCreateAttemptRequest represents a batch of recovery attempts to create
type BatchCreateAttemptRequest struct {
	Attempts []CreateAttemptRequest `json:"attempts"`
}

// CreateAttemptsBatch handles POST /api/v1/internal/recovery/attempts/batch
// Called by worker to create multiple recovery attempts in one request (high-throughput)
func (h *RecoveryAttemptHandler) CreateAttemptsBatch(c *fiber.Ctx) error {
	var req BatchCreateAttemptRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if len(req.Attempts) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "no attempts provided",
		})
	}

	// Limit batch size to prevent abuse
	if len(req.Attempts) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "batch size exceeds maximum of 100",
		})
	}

	// Convert to repository type
	attempts := make([]repository.RecoveryAttempt, len(req.Attempts))
	for i, a := range req.Attempts {
		attempts[i] = repository.RecoveryAttempt{
			ExecutionID:  a.ExecutionID,
			TaskID:       a.TaskID,
			WorkflowID:   a.WorkflowID,
			URL:          a.URL,
			Domain:       a.Domain,
			ErrorPattern: a.ErrorPattern,
			ErrorMessage: a.ErrorMessage,
			StatusCode:   a.StatusCode,
		}
	}

	inserted, err := h.attemptRepo.CreateAttemptsBatch(c.Context(), attempts)
	if err != nil {
		logger.Error("Failed to batch create recovery attempts", zap.Error(err), zap.Int("count", len(attempts)))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create recovery attempts",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success":  true,
		"inserted": inserted,
	})
}

// UpdateAttemptRequest represents a request to update a recovery attempt
type UpdateAttemptRequest struct {
	Action       string `json:"action"`
	Source       string `json:"source"` // rule, ai, default, none
	Status       string `json:"status"` // success, failed
	RuleID       string `json:"rule_id,omitempty"`
	AIReasoning  string `json:"ai_reasoning,omitempty"`
	RetryDelayMs int    `json:"retry_delay_ms,omitempty"`
	DurationMs   int    `json:"duration_ms,omitempty"`
}

// UpdateAttempt handles PATCH /api/v1/internal/recovery/attempt/:id
// Called by worker after recovery execution to update outcome
func (h *RecoveryAttemptHandler) UpdateAttempt(c *fiber.Ctx) error {
	id := c.Params("id")

	var req UpdateAttemptRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "status is required",
		})
	}

	err := h.attemptRepo.UpdateAttempt(
		c.Context(), id,
		req.Action, req.Source, req.Status,
		req.RuleID, req.AIReasoning,
		req.RetryDelayMs, req.DurationMs,
	)
	if err != nil {
		logger.Error("Failed to update recovery attempt", zap.String("id", id), zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update recovery attempt",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"id":      id,
		"status":  req.Status,
	})
}
