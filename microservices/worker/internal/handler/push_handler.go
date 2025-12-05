package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/executor"
	"go.uber.org/zap"
)

// PubSubPushMessage represents the Cloud Run Pub/Sub push message format
// See: https://cloud.google.com/pubsub/docs/push
type PubSubPushMessage struct {
	Message struct {
		Data        string            `json:"data"`        // Base64 encoded message data
		Attributes  map[string]string `json:"attributes"`  // Message attributes
		MessageID   string            `json:"messageId"`   // Unique message ID
		PublishTime string            `json:"publishTime"` // RFC 3339 publish time
	} `json:"message"`
	Subscription string `json:"subscription"` // Subscription that triggered this push
}

// PushHandler handles Pub/Sub push messages for Cloud Run
type PushHandler struct {
	executor *executor.TaskExecutor
	timeout  time.Duration
}

// NewPushHandler creates a new push handler
func NewPushHandler(exec *executor.TaskExecutor) *PushHandler {
	return &PushHandler{
		executor: exec,
		timeout:  5 * time.Minute, // Task execution timeout
	}
}

// Handler returns the Fiber handler function for /tasks/push endpoint
func (h *PushHandler) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		startTime := time.Now()

		// Parse push message
		var pushMsg PubSubPushMessage
		if err := c.BodyParser(&pushMsg); err != nil {
			logger.Error("Failed to parse push message", zap.Error(err))
			// Return 400 to indicate bad request (Pub/Sub will retry)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid message format",
			})
		}

		// Validate message
		if pushMsg.Message.Data == "" {
			logger.Warn("Empty message data received")
			// Ack empty messages to avoid infinite retry
			return c.SendStatus(fiber.StatusNoContent)
		}

		// Decode base64 data
		taskData, err := base64.StdEncoding.DecodeString(pushMsg.Message.Data)
		if err != nil {
			logger.Error("Failed to decode message data",
				zap.String("message_id", pushMsg.Message.MessageID),
				zap.Error(err),
			)
			// Return 400 for invalid encoding
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid base64 encoding",
			})
		}

		// Parse task
		var task models.Task
		if err := json.Unmarshal(taskData, &task); err != nil {
			logger.Error("Failed to unmarshal task",
				zap.String("message_id", pushMsg.Message.MessageID),
				zap.Error(err),
			)
			// Return 400 for invalid task format
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid task format",
			})
		}

		logger.Info("Processing push task",
			zap.String("task_id", task.TaskID),
			zap.String("execution_id", task.ExecutionID),
			zap.String("url", task.URL),
			zap.String("message_id", pushMsg.Message.MessageID),
		)

		// Execute task with timeout
		ctx, cancel := context.WithTimeout(c.Context(), h.timeout)
		defer cancel()

		if err := h.executor.Execute(ctx, &task); err != nil {
			logger.Error("Task execution failed",
				zap.String("task_id", task.TaskID),
				zap.String("message_id", pushMsg.Message.MessageID),
				zap.Error(err),
			)
			// Return 500 to trigger retry
			// Pub/Sub will retry with exponential backoff
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "task execution failed",
				"task_id": task.TaskID,
			})
		}

		duration := time.Since(startTime)
		logger.Info("Push task completed",
			zap.String("task_id", task.TaskID),
			zap.Duration("duration", duration),
		)

		// Return 200 or 204 to acknowledge message
		return c.SendStatus(fiber.StatusNoContent)
	}
}
