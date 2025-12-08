package handlers

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/service"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// VisualSelectorSession represents a visual selector browser session
type VisualSelectorSession struct {
	ID             string                 `json:"id"`
	URL            string                 `json:"url"`
	WorkflowID     string                 `json:"workflow_id,omitempty"`
	Status         string                 `json:"status"` // launching, running, completed, failed, timeout
	Fields         map[string]interface{} `json:"fields,omitempty"`
	ExistingFields map[string]interface{} `json:"existing_fields,omitempty"`
	Error          string                 `json:"error,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
}

// VisualSelectorHandler handles visual selector HTTP requests
type VisualSelectorHandler struct {
	sessions    sync.Map // sessionID -> *VisualSelectorSession
	selectorSvc *service.VisualSelectorService
}

// NewVisualSelectorHandler creates a new visual selector handler
func NewVisualSelectorHandler(selectorSvc *service.VisualSelectorService) *VisualSelectorHandler {
	return &VisualSelectorHandler{
		selectorSvc: selectorSvc,
	}
}

// StartSessionRequest is the request body for starting a visual selector session
type StartSessionRequest struct {
	URL            string                 `json:"url"`
	WorkflowID     string                 `json:"workflow_id,omitempty"`
	ExistingFields map[string]interface{} `json:"existing_fields,omitempty"` // Pre-populate with existing extract fields
}

// StartSession handles POST /api/v1/visual-selector/start
func (h *VisualSelectorHandler) StartSession(c *fiber.Ctx) error {
	var req StartSessionRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "url is required",
		})
	}

	// Create session
	session := &VisualSelectorSession{
		ID:             uuid.New().String(),
		URL:            req.URL,
		WorkflowID:     req.WorkflowID,
		Status:         "launching",
		ExistingFields: req.ExistingFields,
		CreatedAt:      time.Now(),
	}

	// Store session
	h.sessions.Store(session.ID, session)

	logger.Info("Starting visual selector session",
		zap.String("session_id", session.ID),
		zap.String("url", req.URL),
		zap.String("workflow_id", req.WorkflowID),
	)

	// Launch browser session asynchronously
	go func() {
		if err := h.selectorSvc.LaunchSession(session.ID, req.URL, req.ExistingFields); err != nil {
			logger.Error("Failed to launch visual selector session",
				zap.String("session_id", session.ID),
				zap.Error(err),
			)
			session.Status = "failed"
			session.Error = err.Error()
			h.sessions.Store(session.ID, session)
		} else {
			session.Status = "running"
			h.sessions.Store(session.ID, session)
		}
	}()

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"session_id": session.ID,
		"status":     session.Status,
	})
}

// GetResult handles GET /api/v1/visual-selector/:id/result
func (h *VisualSelectorHandler) GetResult(c *fiber.Ctx) error {
	sessionID := c.Params("id")

	sessionData, ok := h.sessions.Load(sessionID)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "session not found",
		})
	}

	session := sessionData.(*VisualSelectorSession)

	// Check if fields were collected via exposeFunction (Playwright bridge)
	if session.Status == "running" {
		if fields, completed := h.selectorSvc.GetSessionFields(sessionID); completed {
			now := time.Now()
			session.Fields = fields
			session.Status = "completed"
			session.CompletedAt = &now
			h.sessions.Store(sessionID, session)
		}
	}

	response := fiber.Map{
		"session_id":  session.ID,
		"status":      session.Status,
		"url":         session.URL,
		"workflow_id": session.WorkflowID,
		"created_at":  session.CreatedAt,
	}

	if session.Status == "completed" {
		response["fields"] = session.Fields
		response["completed_at"] = session.CompletedAt
	}

	if session.Status == "failed" {
		response["error"] = session.Error
	}

	return c.JSON(response)
}

// SaveFieldsRequest is the request body for saving fields from SelectFlow
type SaveFieldsRequest struct {
	Fields map[string]interface{} `json:"fields"`
}

// SaveFields handles POST /api/v1/visual-selector/:id/fields
// This is called by SelectFlow when the user clicks "Done"
func (h *VisualSelectorHandler) SaveFields(c *fiber.Ctx) error {
	sessionID := c.Params("id")

	sessionData, ok := h.sessions.Load(sessionID)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "session not found",
		})
	}

	session := sessionData.(*VisualSelectorSession)

	var req SaveFieldsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Update session with fields
	now := time.Now()
	session.Fields = req.Fields
	session.Status = "completed"
	session.CompletedAt = &now
	h.sessions.Store(sessionID, session)

	logger.Info("Visual selector session completed",
		zap.String("session_id", sessionID),
		zap.Int("field_count", len(req.Fields)),
	)

	// Close browser if still running
	h.selectorSvc.CloseSession(sessionID)

	return c.JSON(fiber.Map{
		"status":      "completed",
		"field_count": len(req.Fields),
	})
}

// CloseSession handles DELETE /api/v1/visual-selector/:id
func (h *VisualSelectorHandler) CloseSession(c *fiber.Ctx) error {
	sessionID := c.Params("id")

	sessionData, ok := h.sessions.Load(sessionID)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "session not found",
		})
	}

	session := sessionData.(*VisualSelectorSession)
	session.Status = "closed"
	h.sessions.Store(sessionID, session)

	// Close browser
	h.selectorSvc.CloseSession(sessionID)

	logger.Info("Visual selector session closed",
		zap.String("session_id", sessionID),
	)

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// CleanupStaleSessions removes sessions older than the specified duration
func (h *VisualSelectorHandler) CleanupStaleSessions(maxAge time.Duration) {
	now := time.Now()
	h.sessions.Range(func(key, value interface{}) bool {
		session := value.(*VisualSelectorSession)
		if now.Sub(session.CreatedAt) > maxAge {
			h.sessions.Delete(key)
			h.selectorSvc.CloseSession(session.ID)
			logger.Info("Cleaned up stale visual selector session",
				zap.String("session_id", session.ID),
			)
		}
		return true
	})
}
