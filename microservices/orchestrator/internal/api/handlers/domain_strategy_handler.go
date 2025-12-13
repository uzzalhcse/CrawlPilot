package handlers

import (
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// DomainStrategyHandler handles domain strategy HTTP requests
type DomainStrategyHandler struct {
	strategyRepo *repository.DomainStrategyRepository
}

// NewDomainStrategyHandler creates a new domain strategy handler
func NewDomainStrategyHandler(strategyRepo *repository.DomainStrategyRepository) *DomainStrategyHandler {
	return &DomainStrategyHandler{
		strategyRepo: strategyRepo,
	}
}

// GetAll handles GET /api/v1/recovery/domains
// Returns all learned domain strategies with optional status filtering
func (h *DomainStrategyHandler) GetAll(c *fiber.Ctx) error {
	status := c.Query("status", "")
	limit := c.QueryInt("limit", 100)
	offset := c.QueryInt("offset", 0)

	strategies, total, err := h.strategyRepo.GetAll(c.Context(), status, limit, offset)
	if err != nil {
		logger.Error("Failed to get domain strategies", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get domain strategies",
		})
	}

	// Get stats for overview
	stats, _ := h.strategyRepo.GetStats(c.Context())

	return c.JSON(fiber.Map{
		"strategies": strategies,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
		"stats":      stats,
	})
}

// GetByDomain handles GET /api/v1/recovery/domains/:domain
// Returns a single domain strategy by domain name
func (h *DomainStrategyHandler) GetByDomain(c *fiber.Ctx) error {
	domain := c.Params("domain")

	// URL decode the domain param (in case it contains : like 34.85.113.40:8585)
	decodedDomain, err := url.PathUnescape(domain)
	if err != nil {
		decodedDomain = domain
	}

	strategy, err := h.strategyRepo.GetByDomain(c.Context(), decodedDomain)
	if err != nil {
		logger.Debug("Domain strategy not found", zap.String("domain", decodedDomain), zap.Error(err))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "domain strategy not found",
		})
	}

	return c.JSON(strategy)
}

// Delete handles DELETE /api/v1/recovery/domains/:domain
// Clears learned strategy for a single domain
func (h *DomainStrategyHandler) Delete(c *fiber.Ctx) error {
	domain := c.Params("domain")

	// URL decode
	decodedDomain, err := url.PathUnescape(domain)
	if err != nil {
		decodedDomain = domain
	}

	if err := h.strategyRepo.Delete(c.Context(), decodedDomain); err != nil {
		logger.Error("Failed to delete domain strategy", zap.String("domain", decodedDomain), zap.Error(err))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "domain not found or already deleted",
		})
	}

	logger.Info("Domain strategy cleared", zap.String("domain", decodedDomain))
	return c.JSON(fiber.Map{
		"success": true,
		"domain":  decodedDomain,
		"message": "Domain learning cleared. Next crawl will start fresh.",
	})
}

// DeleteAll handles DELETE /api/v1/recovery/domains
// Clears ALL learned strategies (requires confirmation in body)
func (h *DomainStrategyHandler) DeleteAll(c *fiber.Ctx) error {
	// Require confirmation to prevent accidental deletion
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := c.BodyParser(&req); err != nil || !req.Confirm {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "confirmation required: send {\"confirm\": true}",
		})
	}

	count, err := h.strategyRepo.DeleteAll(c.Context())
	if err != nil {
		logger.Error("Failed to delete all domain strategies", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to clear domain strategies",
		})
	}

	logger.Info("All domain strategies cleared", zap.Int64("count", count))
	return c.JSON(fiber.Map{
		"success": true,
		"deleted": count,
		"message": "All domain learning cleared. Next crawls will start fresh.",
	})
}

// GetStats handles GET /api/v1/recovery/domains/stats
// Returns aggregate statistics for domain strategies
func (h *DomainStrategyHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.strategyRepo.GetStats(c.Context())
	if err != nil {
		logger.Error("Failed to get domain strategy stats", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get stats",
		})
	}

	return c.JSON(stats)
}

// Update handles PATCH /api/v1/recovery/domains/:domain
// Updates a domain strategy's settings
func (h *DomainStrategyHandler) Update(c *fiber.Ctx) error {
	domain := c.Params("domain")

	// URL decode
	decodedDomain, err := url.PathUnescape(domain)
	if err != nil {
		decodedDomain = domain
	}

	var update repository.DomainStrategyUpdate
	if err := c.BodyParser(&update); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	strategy, err := h.strategyRepo.Update(c.Context(), decodedDomain, update)
	if err != nil {
		logger.Error("Failed to update domain strategy", zap.String("domain", decodedDomain), zap.Error(err))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	logger.Info("Domain strategy updated",
		zap.String("domain", decodedDomain),
		zap.Any("update", update),
	)
	return c.JSON(strategy)
}

// Create handles POST /api/v1/recovery/domains
// Creates or updates a domain strategy manually
func (h *DomainStrategyHandler) Create(c *fiber.Ctx) error {
	var req struct {
		Domain          string `json:"domain"`
		RecommendedTier int    `json:"recommended_tier"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Domain == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "domain is required",
		})
	}

	strategy, err := h.strategyRepo.Create(c.Context(), req.Domain, req.RecommendedTier)
	if err != nil {
		logger.Error("Failed to create domain strategy", zap.String("domain", req.Domain), zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	logger.Info("Domain strategy created",
		zap.String("domain", req.Domain),
		zap.Int("recommended_tier", req.RecommendedTier),
	)
	return c.Status(fiber.StatusCreated).JSON(strategy)
}
