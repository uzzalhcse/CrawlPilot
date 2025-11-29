package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/internal/error_recovery"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"go.uber.org/zap"
)

type ErrorRecoveryHandler struct {
	repo *storage.ErrorRecoveryRepository
}

func NewErrorRecoveryHandler(repo *storage.ErrorRecoveryRepository) *ErrorRecoveryHandler {
	return &ErrorRecoveryHandler{repo: repo}
}

// GetConfig retrieves a configuration value
func (h *ErrorRecoveryHandler) GetConfig(c *fiber.Ctx) error {
	key := c.Params("key")
	logger.Debug("📖 Fetching error recovery config", zap.String("key", key))
	config, err := h.repo.GetConfig(context.Background(), key)
	if err != nil {
		logger.Error("Failed to get config", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get config"})
	}
	if config == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Config not found"})
	}
	return c.JSON(fiber.Map{"key": key, "value": config})
}

// UpdateConfig updates a configuration value
func (h *ErrorRecoveryHandler) UpdateConfig(c *fiber.Ctx) error {
	var req struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	logger.Debug("✍️ Updating error recovery config", zap.String("key", req.Key), zap.Any("value", req.Value))

	if err := h.repo.UpdateConfig(context.Background(), req.Key, req.Value); err != nil {
		logger.Error("Failed to update config", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update config"})
	}

	return c.JSON(fiber.Map{"message": "Config updated successfully"})
}

// ListRules returns all error recovery rules
func (h *ErrorRecoveryHandler) ListRules(c *fiber.Ctx) error {
	logger.Debug("📋 Listing all error recovery rules")
	rules, err := h.repo.ListRules(context.Background())
	if err != nil {
		logger.Error("Failed to list rules", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list rules"})
	}
	return c.JSON(rules)
}

// CreateRule creates a new rule
func (h *ErrorRecoveryHandler) CreateRule(c *fiber.Ctx) error {
	var rule error_recovery.ContextAwareRule
	if err := c.BodyParser(&rule); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Ensure ID is generated if not provided (though NewContextAwareRule handles it, JSON unmarshal might overwrite)
	if rule.ID == "" {
		newRule := error_recovery.NewContextAwareRule(rule.Name)
		rule.ID = newRule.ID
		rule.CreatedAt = newRule.CreatedAt
		rule.UpdatedAt = newRule.UpdatedAt
	}

	if err := h.repo.CreateRule(context.Background(), &rule); err != nil {
		logger.Error("Failed to create rule", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create rule"})
	}

	return c.Status(fiber.StatusCreated).JSON(rule)
}

// GetRule retrieves a rule by ID
func (h *ErrorRecoveryHandler) GetRule(c *fiber.Ctx) error {
	id := c.Params("id")
	rule, err := h.repo.GetRule(context.Background(), id)
	if err != nil {
		logger.Error("Failed to get rule", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get rule"})
	}
	if rule == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Rule not found"})
	}
	return c.JSON(rule)
}

// UpdateRule updates a rule
func (h *ErrorRecoveryHandler) UpdateRule(c *fiber.Ctx) error {
	id := c.Params("id")
	var rule error_recovery.ContextAwareRule
	if err := c.BodyParser(&rule); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	rule.ID = id // Ensure ID matches URL

	if err := h.repo.UpdateRule(context.Background(), &rule); err != nil {
		logger.Error("Failed to update rule", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update rule"})
	}

	return c.JSON(rule)
}

// DeleteRule deletes a rule
func (h *ErrorRecoveryHandler) DeleteRule(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.repo.DeleteRule(context.Background(), id); err != nil {
		logger.Error("Failed to delete rule", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete rule"})
	}
	return c.Status(fiber.StatusNoContent).Send(nil)
}
