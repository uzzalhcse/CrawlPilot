package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// IncidentHandler handles incident HTTP requests
type IncidentHandler struct {
	incidentRepo *repository.IncidentRepository
}

// NewIncidentHandler creates a new incident handler
func NewIncidentHandler(incidentRepo *repository.IncidentRepository) *IncidentHandler {
	return &IncidentHandler{
		incidentRepo: incidentRepo,
	}
}

// GetAllIncidents handles GET /api/v1/incidents
func (h *IncidentHandler) GetAllIncidents(c *fiber.Ctx) error {
	status := c.Query("status", "")
	priority := c.Query("priority", "")
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	incidents, total, err := h.incidentRepo.GetAllIncidents(c.Context(), status, priority, limit, offset)
	if err != nil {
		logger.Error("Failed to get incidents", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get incidents",
		})
	}

	return c.JSON(fiber.Map{
		"incidents": incidents,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// GetIncident handles GET /api/v1/incidents/:id
func (h *IncidentHandler) GetIncident(c *fiber.Ctx) error {
	id := c.Params("id")

	incident, err := h.incidentRepo.GetIncidentByID(c.Context(), id)
	if err != nil {
		logger.Error("Failed to get incident", zap.String("id", id), zap.Error(err))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "incident not found",
		})
	}

	return c.JSON(incident)
}

// UpdateIncidentStatusRequest represents a status update request
type UpdateIncidentStatusRequest struct {
	Status     string `json:"status"`
	Resolution string `json:"resolution,omitempty"`
}

// UpdateIncidentStatus handles PATCH /api/v1/incidents/:id/status
func (h *IncidentHandler) UpdateIncidentStatus(c *fiber.Ctx) error {
	id := c.Params("id")

	var req UpdateIncidentStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate status
	validStatuses := map[string]bool{
		"open": true, "in_progress": true, "resolved": true, "ignored": true,
	}
	if !validStatuses[req.Status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid status. Must be: open, in_progress, resolved, or ignored",
		})
	}

	if err := h.incidentRepo.UpdateIncidentStatus(c.Context(), id, req.Status, req.Resolution); err != nil {
		logger.Error("Failed to update incident status", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update incident status",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"id":      id,
		"status":  req.Status,
	})
}

// AssignIncidentRequest represents an assignment request
type AssignIncidentRequest struct {
	UserID string `json:"user_id"`
}

// AssignIncident handles PATCH /api/v1/incidents/:id/assign
func (h *IncidentHandler) AssignIncident(c *fiber.Ctx) error {
	id := c.Params("id")

	var req AssignIncidentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.incidentRepo.AssignIncident(c.Context(), id, req.UserID); err != nil {
		logger.Error("Failed to assign incident", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to assign incident",
		})
	}

	return c.JSON(fiber.Map{
		"success":     true,
		"id":          id,
		"assigned_to": req.UserID,
	})
}

// GetIncidentStats handles GET /api/v1/incidents/stats
func (h *IncidentHandler) GetIncidentStats(c *fiber.Ctx) error {
	stats, err := h.incidentRepo.GetIncidentStats(c.Context())
	if err != nil {
		logger.Error("Failed to get incident stats", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get incident statistics",
		})
	}

	return c.JSON(stats)
}

// GetDomainStats handles GET /api/v1/incidents/domains
func (h *IncidentHandler) GetDomainStats(c *fiber.Ctx) error {
	stats, err := h.incidentRepo.GetDomainIncidentStats(c.Context())
	if err != nil {
		logger.Error("Failed to get domain incident stats", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get domain statistics",
		})
	}

	return c.JSON(fiber.Map{
		"domains": stats,
	})
}

// ResolveIncident handles POST /api/v1/incidents/:id/resolve (shortcut)
func (h *IncidentHandler) ResolveIncident(c *fiber.Ctx) error {
	id := c.Params("id")

	var req struct {
		Resolution string `json:"resolution"`
	}
	if err := c.BodyParser(&req); err != nil {
		req.Resolution = "Manually resolved"
	}

	if err := h.incidentRepo.UpdateIncidentStatus(c.Context(), id, "resolved", req.Resolution); err != nil {
		logger.Error("Failed to resolve incident", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to resolve incident",
		})
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"id":         id,
		"status":     "resolved",
		"resolution": req.Resolution,
	})
}
