package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// PluginHandler handles plugin marketplace API requests
type PluginHandler struct {
	pluginRepo *storage.PluginRepository
	logger     *zap.Logger
}

// NewPluginHandler creates a new plugin handler
func NewPluginHandler(pluginRepo *storage.PluginRepository, logger *zap.Logger) *PluginHandler {
	return &PluginHandler{
		pluginRepo: pluginRepo,
		logger:     logger,
	}
}

// CreatePlugin handles POST /api/v1/plugins
func (h *PluginHandler) CreatePlugin(c *fiber.Ctx) error {
	var plugin models.Plugin
	if err := c.BodyParser(&plugin); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if plugin.ID == "" {
		plugin.ID = uuid.New().String()
	}

	if plugin.Name == "" || plugin.Slug == "" || plugin.PhaseType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing required fields",
		})
	}

	if err := h.pluginRepo.CreatePlugin(c.Context(), &plugin); err != nil {
		h.logger.Error("Failed to create plugin", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create plugin",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(plugin)
}

// ListPlugins handles GET /api/v1/plugins
func (h *PluginHandler) ListPlugins(c *fiber.Ctx) error {
	filters := models.PluginFilters{
		Query:      c.Query("q"),
		Category:   c.Query("category"),
		PhaseType:  models.PhaseType(c.Query("phase_type")),
		PluginType: models.PluginType(c.Query("plugin_type")),
		SortBy:     c.Query("sort_by", "popular"),
		SortOrder:  c.Query("sort_order", "desc"),
		Limit:      getIntQuery(c, "limit", 50),
		Offset:     getIntQuery(c, "offset", 0),
	}

	if verified := c.Query("verified"); verified != "" {
		isVerified := verified == "true"
		filters.IsVerified = &isVerified
	}

	plugins, err := h.pluginRepo.ListPlugins(c.Context(), filters)
	if err != nil {
		h.logger.Error("Failed to list plugins", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list plugins",
		})
	}

	return c.JSON(plugins)
}

// GetPlugin handles GET /api/v1/plugins/:slug
func (h *PluginHandler) GetPlugin(c *fiber.Ctx) error {
	slug := c.Params("slug")

	plugin, err := h.pluginRepo.GetPluginBySlug(c.Context(), slug)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Plugin not found",
		})
	}

	return c.JSON(plugin)
}

// UpdatePlugin handles PUT /api/v1/plugins/:id
func (h *PluginHandler) UpdatePlugin(c *fiber.Ctx) error {
	pluginID := c.Params("id")

	var plugin models.Plugin
	if err := c.BodyParser(&plugin); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	plugin.ID = pluginID

	if err := h.pluginRepo.UpdatePlugin(c.Context(), &plugin); err != nil {
		h.logger.Error("Failed to update plugin", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update plugin",
		})
	}

	return c.JSON(plugin)
}

// DeletePlugin handles DELETE /api/v1/plugins/:id
func (h *PluginHandler) DeletePlugin(c *fiber.Ctx) error {
	pluginID := c.Params("id")

	if err := h.pluginRepo.DeletePlugin(c.Context(), pluginID); err != nil {
		h.logger.Error("Failed to delete plugin", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete plugin",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// PublishVersion handles POST /api/v1/plugins/:id/versions
func (h *PluginHandler) PublishVersion(c *fiber.Ctx) error {
	pluginID := c.Params("id")

	var version models.PluginVersion
	if err := c.BodyParser(&version); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	version.ID = uuid.New().String()
	version.PluginID = pluginID

	if err := h.pluginRepo.PublishVersion(c.Context(), &version); err != nil {
		h.logger.Error("Failed to publish version", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to publish version",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(version)
}

// ListVersions handles GET /api/v1/plugins/:id/versions
func (h *PluginHandler) ListVersions(c *fiber.Ctx) error {
	pluginID := c.Params("id")

	versions, err := h.pluginRepo.ListVersions(c.Context(), pluginID)
	if err != nil {
		h.logger.Error("Failed to list versions", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list versions",
		})
	}

	return c.JSON(versions)
}

// GetVersion handles GET /api/v1/plugins/:id/versions/:version
func (h *PluginHandler) GetVersion(c *fiber.Ctx) error {
	versionID := c.Params("version")

	version, err := h.pluginRepo.GetVersionByID(c.Context(), versionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Version not found",
		})
	}

	return c.JSON(version)
}

// InstallPlugin handles POST /api/v1/plugins/:id/install
func (h *PluginHandler) InstallPlugin(c *fiber.Ctx) error {
	pluginID := c.Params("id")

	workspaceID := c.Get("X-Workspace-ID", "default")

	version, err := h.pluginRepo.GetLatestVersion(c.Context(), pluginID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No stable version available",
		})
	}

	installation := &models.PluginInstallation{
		ID:              uuid.New().String(),
		PluginID:        pluginID,
		PluginVersionID: version.ID,
		WorkspaceID:     workspaceID,
	}

	if err := h.pluginRepo.InstallPlugin(c.Context(), installation); err != nil {
		h.logger.Error("Failed to install plugin", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to install plugin",
		})
	}

	return c.JSON(installation)
}

// UninstallPlugin handles DELETE /api/v1/plugins/:id/uninstall
func (h *PluginHandler) UninstallPlugin(c *fiber.Ctx) error {
	pluginID := c.Params("id")

	workspaceID := c.Get("X-Workspace-ID", "default")

	if err := h.pluginRepo.UninstallPlugin(c.Context(), pluginID, workspaceID); err != nil {
		h.logger.Error("Failed to uninstall plugin", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to uninstall plugin",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ListInstalledPlugins handles GET /api/v1/plugins/installed
func (h *PluginHandler) ListInstalledPlugins(c *fiber.Ctx) error {
	workspaceID := c.Get("X-Workspace-ID", "default")

	plugins, err := h.pluginRepo.ListInstalledPlugins(c.Context(), workspaceID)
	if err != nil {
		h.logger.Error("Failed to list installed plugins", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list installed plugins",
		})
	}

	return c.JSON(plugins)
}

// CreateReview handles POST /api/v1/plugins/:id/reviews
func (h *PluginHandler) CreateReview(c *fiber.Ctx) error {
	pluginID := c.Params("id")

	var review models.PluginReview
	if err := c.BodyParser(&review); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	review.ID = uuid.New().String()
	review.PluginID = pluginID

	if review.UserID == "" {
		review.UserID = "anonymous" // TODO: Get from auth context
	}

	if review.Rating < 1 || review.Rating > 5 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Rating must be between 1 and 5",
		})
	}

	if err := h.pluginRepo.CreateReview(c.Context(), &review); err != nil {
		h.logger.Error("Failed to create review", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create review",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(review)
}

// ListReviews handles GET /api/v1/plugins/:id/reviews
func (h *PluginHandler) ListReviews(c *fiber.Ctx) error {
	pluginID := c.Params("id")

	limit := getIntQuery(c, "limit", 20)
	offset := getIntQuery(c, "offset", 0)

	reviews, err := h.pluginRepo.ListReviews(c.Context(), pluginID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list reviews", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list reviews",
		})
	}

	return c.JSON(reviews)
}

// GetCategories handles GET /api/v1/plugins/categories
func (h *PluginHandler) GetCategories(c *fiber.Ctx) error {
	categories, err := h.pluginRepo.GetCategories(c.Context())
	if err != nil {
		h.logger.Error("Failed to get categories", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get categories",
		})
	}

	return c.JSON(categories)
}

// SearchPlugins handles GET /api/v1/plugins/search
func (h *PluginHandler) SearchPlugins(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Search query required",
		})
	}

	limit := getIntQuery(c, "limit", 20)

	plugins, err := h.pluginRepo.SearchPlugins(c.Context(), query, limit)
	if err != nil {
		h.logger.Error("Failed to search plugins", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to search plugins",
		})
	}

	return c.JSON(plugins)
}

// GetPopularPlugins handles GET /api/v1/plugins/popular
func (h *PluginHandler) GetPopularPlugins(c *fiber.Ctx) error {
	limit := getIntQuery(c, "limit", 10)

	plugins, err := h.pluginRepo.GetPopularPlugins(c.Context(), limit)
	if err != nil {
		h.logger.Error("Failed to get popular plugins", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get popular plugins",
		})
	}

	return c.JSON(plugins)
}

// Helper function to get integer query parameter
func getIntQuery(c *fiber.Ctx, key string, defaultValue int) int {
	value := c.Query(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}
