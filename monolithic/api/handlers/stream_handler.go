package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type StreamHandler struct {
	executionHandler *ExecutionHandler
}

func NewStreamHandler(executionHandler *ExecutionHandler) *StreamHandler {
	return &StreamHandler{
		executionHandler: executionHandler,
	}
}

// StreamExecutionEvents handles SSE connections for execution updates
func (h *StreamHandler) StreamExecutionEvents(c *fiber.Ctx) error {
	executionID := c.Params("id")
	if executionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "execution_id is required",
		})
	}

	// Set headers for SSE
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// Get the event broadcaster from the execution handler
	broadcaster := h.executionHandler.GetEventBroadcaster()
	if broadcaster == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Event broadcaster not available",
		})
	}

	// Subscribe to events
	eventChan := broadcaster.Subscribe()

	// Create a context for the request to handle cancellation
	ctx := c.Context()

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		defer broadcaster.Unsubscribe(eventChan)

		// Send initial connection message
		fmt.Fprintf(w, "event: connected\ndata: \"connected\"\n\n")
		w.Flush()

		for {
			select {
			case event := <-eventChan:
				// Filter by execution ID
				if event.ExecutionID != executionID {
					continue
				}

				// Marshal event data
				data, err := json.Marshal(event)
				if err != nil {
					logger.Error("Failed to marshal event", zap.Error(err))
					continue
				}

				// Write SSE event
				_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(data))
				if err != nil {
					logger.Debug("Client disconnected while writing event", zap.String("execution_id", executionID), zap.Error(err))
					return // Exit the StreamWriter goroutine
				}

				// Flush the response
				if err := w.Flush(); err != nil {
					logger.Debug("Client disconnected during flush", zap.String("execution_id", executionID), zap.Error(err))
					return // Exit the StreamWriter goroutine
				}

				// Close stream when execution completes or fails
				if event.Type == "execution_completed" || event.Type == "execution_failed" {
					logger.Debug("Execution finished, closing SSE stream",
						zap.String("execution_id", executionID),
						zap.String("event_type", event.Type))
					// Send a final comment to ensure clean closure
					fmt.Fprintf(w, ": stream closed\n\n")
					w.Flush()
					return // Exit the StreamWriter goroutine
				}

			case <-ctx.Done():
				logger.Debug("Client closed connection", zap.String("execution_id", executionID))
				return // Exit the StreamWriter goroutine
			case <-time.After(30 * time.Second):
				// Send keepalive ping
				fmt.Fprintf(w, ": keepalive\n\n")
				w.Flush()
			}
		}
	}))

	return nil
}
