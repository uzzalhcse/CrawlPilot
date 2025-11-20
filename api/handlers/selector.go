package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"go.uber.org/zap"
)

type SelectorHandler struct {
	selectorManager *browser.ElementSelectorManager
}

func NewSelectorHandler(browserPool *browser.BrowserPool) *SelectorHandler {
	return &SelectorHandler{
		selectorManager: browser.NewElementSelectorManager(browserPool),
	}
}

// CreateSelectorSession creates a new visual selector session
func (h *SelectorHandler) CreateSelectorSession(c *fiber.Ctx) error {
	var req struct {
		URL string `json:"url"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "URL is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session, err := h.selectorManager.CreateSession(ctx, req.URL)
	if err != nil {
		logger.Error("Failed to create selector session", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create selector session",
		})
	}

	logger.Info("Selector session created",
		zap.String("session_id", session.ID),
		zap.String("url", req.URL),
	)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"session_id": session.ID,
		"url":        session.URL,
		"message":    "Browser window opened. Select elements in the browser window, then close it when done.",
	})
}

// GetSelectedFields retrieves the fields selected in a session
func (h *SelectorHandler) GetSelectedFields(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")

	fields, err := h.selectorManager.GetSelectedFields(sessionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Session not found",
		})
	}

	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"fields":     fields,
		"count":      len(fields),
	})
}

// CloseSelectorSession closes a selector session
func (h *SelectorHandler) CloseSelectorSession(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")

	if err := h.selectorManager.CloseSession(sessionID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Session not found",
		})
	}

	logger.Info("Selector session closed", zap.String("session_id", sessionID))

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// GetSessionStatus checks if a session is still active
func (h *SelectorHandler) GetSessionStatus(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")

	session, err := h.selectorManager.GetSession(sessionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Session not found",
		})
	}

	return c.JSON(fiber.Map{
		"session_id":    session.ID,
		"url":           session.URL,
		"created_at":    session.CreatedAt,
		"last_activity": session.LastActivity,
		"active":        true,
		"fields_count":  len(session.SelectedFields),
	})
}
