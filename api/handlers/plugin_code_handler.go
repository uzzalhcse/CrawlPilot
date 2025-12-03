package handlers

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/plugin"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"go.uber.org/zap"
)

// PluginCodeHandler handles plugin code API requests
type PluginCodeHandler struct {
	sourceManager *plugin.SourceManager
	builder       *plugin.Builder
	pluginRepo    *storage.PluginRepository
}

// NewPluginCodeHandler creates a new plugin code handler
func NewPluginCodeHandler(
	sourceManager *plugin.SourceManager,
	builder *plugin.Builder,
	pluginRepo *storage.PluginRepository,
) *PluginCodeHandler {
	return &PluginCodeHandler{
		sourceManager: sourceManager,
		builder:       builder,
		pluginRepo:    pluginRepo,
	}
}

// GetPluginSource handles GET /api/v1/plugins/:id/code
func (h *PluginCodeHandler) GetPluginSource(c *fiber.Ctx) error {
	pluginID := c.Params("id")
	ctx := c.Context()

	// Get plugin to find slug
	p, err := h.pluginRepo.GetPluginByID(ctx, pluginID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Plugin not found",
		})
	}

	files, err := h.sourceManager.GetSource(p.Slug)
	if err != nil {
		logger.Error("Failed to get plugin source", zap.String("plugin_id", pluginID), zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get plugin source",
		})
	}

	return c.JSON(fiber.Map{
		"files": files,
	})
}

// UpdatePluginSource handles PUT /api/v1/plugins/:id/code
func (h *PluginCodeHandler) UpdatePluginSource(c *fiber.Ctx) error {
	pluginID := c.Params("id")
	ctx := c.Context()

	var req struct {
		Files map[string]string `json:"files"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get plugin to find slug
	p, err := h.pluginRepo.GetPluginByID(ctx, pluginID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Plugin not found",
		})
	}

	if err := h.sourceManager.UpdateSource(p.Slug, req.Files); err != nil {
		logger.Error("Failed to update plugin source", zap.String("plugin_id", pluginID), zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to update plugin source: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Source updated successfully",
	})
}

// BuildPlugin handles POST /api/v1/plugins/:id/build
func (h *PluginCodeHandler) BuildPlugin(c *fiber.Ctx) error {
	pluginID := c.Params("id")
	ctx := c.Context()

	// Get plugin to find slug
	p, err := h.pluginRepo.GetPluginByID(ctx, pluginID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Plugin not found",
		})
	}

	jobID, err := h.builder.Build(p.Slug)
	if err != nil {
		logger.Error("Failed to trigger build", zap.String("plugin_id", pluginID), zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to trigger build",
		})
	}

	return c.JSON(fiber.Map{
		"build_id": jobID,
		"message":  "Build triggered successfully",
	})
}

// GetBuildStatus handles GET /api/v1/builds/:build_id/status
func (h *PluginCodeHandler) GetBuildStatus(c *fiber.Ctx) error {
	buildID := c.Params("build_id")

	job, err := h.builder.GetBuildJob(buildID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Build job not found",
		})
	}

	return c.JSON(job)
}

// ScaffoldPlugin handles POST /api/v1/plugins/scaffold
func (h *PluginCodeHandler) ScaffoldPlugin(c *fiber.Ctx) error {
	var req plugin.PluginScaffoldConfig

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" || req.Slug == "" || req.PhaseType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name, slug, and phase_type are required",
		})
	}

	// Create template generator
	templateGen := plugin.NewTemplateGenerator(h.sourceManager.BasePath)

	// Generate plugin files
	pluginModel, err := templateGen.GeneratePlugin(&req)
	if err != nil {
		logger.Error("Failed to scaffold plugin", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to scaffold plugin: %v", err),
		})
	}

	// Save to database
	ctx := c.Context()
	if err := h.pluginRepo.CreatePlugin(ctx, pluginModel); err != nil {
		logger.Error("Failed to save plugin to database", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save plugin",
		})
	}

	logger.Info("Plugin scaffolded successfully",
		zap.String("plugin_id", pluginModel.ID),
		zap.String("slug", pluginModel.Slug))

	return c.Status(fiber.StatusCreated).JSON(pluginModel)
}

// GetPluginReadme handles GET /api/v1/plugins/:id/readme
func (h *PluginCodeHandler) GetPluginReadme(c *fiber.Ctx) error {
	pluginID := c.Params("id")

	// Get plugin to find its slug
	plugin, err := h.pluginRepo.GetPluginByID(c.Context(), pluginID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Plugin not found",
		})
	}

	// Read README.md from plugin directory
	readmePath := filepath.Join(h.sourceManager.BasePath, plugin.Slug, "README.md")
	logger.Info("Attempting to read README",
		zap.String("plugin_id", pluginID),
		zap.String("plugin_slug", plugin.Slug),
		zap.String("readme_path", readmePath))

	content, err := os.ReadFile(readmePath)
	if err != nil {
		// README not found, log and return empty content
		logger.Warn("README not found",
			zap.String("plugin_slug", plugin.Slug),
			zap.String("path", readmePath),
			zap.Error(err))
		return c.JSON(fiber.Map{
			"content": "",
		})
	}

	return c.JSON(fiber.Map{
		"content": string(content),
	})
}
