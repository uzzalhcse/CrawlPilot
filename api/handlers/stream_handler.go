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
				// Filter events by execution ID
				if event.ExecutionID == executionID {
					data, err := json.Marshal(event)
					if err != nil {
						logger.Error("Failed to marshal event", zap.Error(err))
						continue
					}

					// Send event
					fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(data))
					w.Flush()

					// If execution completed or failed, we can close the stream after a short delay
					// or let the client close it. Usually better to let client decide.
				}
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
				// Send keepalive ping
				fmt.Fprintf(w, ": keepalive\n\n")
				w.Flush()
			}
		}
	}))

	return nil
}
