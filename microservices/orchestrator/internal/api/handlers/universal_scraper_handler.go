package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/service"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// UniversalScraperHandler handles universal scraper HTTP requests
type UniversalScraperHandler struct {
	scraperSvc *service.UniversalScraperService
}

// NewUniversalScraperHandler creates a new universal scraper handler
func NewUniversalScraperHandler(scraperSvc *service.UniversalScraperService) *UniversalScraperHandler {
	return &UniversalScraperHandler{
		scraperSvc: scraperSvc,
	}
}

// Scrape handles POST /api/v1/scrape
// Submits a URL for scraping and returns a scrape ID for polling
func (h *UniversalScraperHandler) Scrape(c *fiber.Ctx) error {
	var input models.ScrapeRequestInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate URL
	if input.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "URL is required",
		})
	}

	// Set defaults
	input.ValidateAndSetDefaults()

	// Create scrape request
	// Default headless to true if not specified
	headless := true
	if input.Headless != nil {
		headless = *input.Headless
	}

	req := &models.ScrapeRequest{
		ID:              uuid.New().String(),
		URL:             input.URL,
		Driver:          input.Driver,
		ProfileID:       input.ProfileID,
		OutputFormat:    input.OutputFormat,
		Timeout:         input.Timeout,
		WaitForSelector: input.WaitForSelector,
		Headless:        headless,
		UseProxy:        input.UseProxy,
		ProxyTier:       input.ProxyTier,
		Status:          models.ScrapeStatusPending,
	}

	// Submit to queue
	if err := h.scraperSvc.SubmitScrape(c.Context(), req); err != nil {
		logger.Error("Failed to submit scrape request",
			zap.String("url", req.URL),
			zap.Error(err),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to submit scrape request",
		})
	}

	logger.Info("Scrape request submitted",
		zap.String("scrape_id", req.ID),
		zap.String("url", req.URL),
		zap.String("driver", req.Driver),
	)

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"scrape_id": req.ID,
		"status":    req.Status,
		"url":       req.URL,
	})
}

// GetResult handles GET /api/v1/scrape/:id
// Polls for scrape result
func (h *UniversalScraperHandler) GetResult(c *fiber.Ctx) error {
	id := c.Params("id")

	result, err := h.scraperSvc.GetResult(c.Context(), id)
	if err != nil {
		logger.Error("Failed to get scrape result",
			zap.String("scrape_id", id),
			zap.Error(err),
		)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  "Scrape result not found",
			"status": models.ScrapeStatusPending,
		})
	}

	return c.JSON(result)
}

// ListDrivers handles GET /api/v1/scrape/drivers
// Returns available drivers
func (h *UniversalScraperHandler) ListDrivers(c *fiber.Ctx) error {
	drivers := []fiber.Map{
		{"id": "http", "name": "HTTP Client (Fast)", "description": "Fast HTTP requests without JavaScript rendering"},
		{"id": "playwright", "name": "Playwright", "description": "Full browser automation with JavaScript support"},
		{"id": "camoufox", "name": "Camoufox (Stealth)", "description": "Stealth browser with anti-detection fingerprinting"},
	}

	return c.JSON(fiber.Map{
		"drivers": drivers,
	})
}

// ListProfiles handles GET /api/v1/scrape/profiles
// Returns available browser profiles for dropdown
func (h *UniversalScraperHandler) ListProfiles(c *fiber.Ctx) error {
	profiles, err := h.scraperSvc.ListProfiles(c.Context())
	if err != nil {
		logger.Error("Failed to list profiles", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list profiles",
		})
	}

	return c.JSON(fiber.Map{
		"profiles": profiles,
	})
}
