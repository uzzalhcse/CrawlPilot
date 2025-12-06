package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// RecoveryHandler handles recovery system configuration HTTP requests
type RecoveryHandler struct {
	configRepo *repository.SystemConfigRepository
}

// NewRecoveryHandler creates a new recovery handler
func NewRecoveryHandler(configRepo *repository.SystemConfigRepository) *RecoveryHandler {
	return &RecoveryHandler{
		configRepo: configRepo,
	}
}

// =====================================================
// SYSTEM CONFIGURATION ENDPOINTS
// =====================================================

// GetAllConfigs handles GET /api/v1/recovery/config
func (h *RecoveryHandler) GetAllConfigs(c *fiber.Ctx) error {
	configs, err := h.configRepo.GetAllConfigs(c.Context())
	if err != nil {
		logger.Error("Failed to get configs", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get configurations",
		})
	}

	// Group by category for frontend convenience
	grouped := make(map[string][]repository.ConfigItem)
	for _, cfg := range configs {
		grouped[cfg.Category] = append(grouped[cfg.Category], cfg)
	}

	return c.JSON(fiber.Map{
		"configs": configs,
		"grouped": grouped,
		"count":   len(configs),
	})
}

// GetConfigsByCategory handles GET /api/v1/recovery/config/:category
func (h *RecoveryHandler) GetConfigsByCategory(c *fiber.Ctx) error {
	category := c.Params("category")

	configs, err := h.configRepo.GetConfigsByCategory(c.Context(), category)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get configurations",
		})
	}

	return c.JSON(fiber.Map{
		"configs":  configs,
		"category": category,
		"count":    len(configs),
	})
}

// UpdateConfigRequest represents a config update request
type UpdateConfigRequest struct {
	Value interface{} `json:"value"`
}

// UpdateConfig handles PUT /api/v1/recovery/config/:key
func (h *RecoveryHandler) UpdateConfig(c *fiber.Ctx) error {
	key := c.Params("key")

	var req UpdateConfigRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.configRepo.UpdateConfig(c.Context(), key, req.Value); err != nil {
		logger.Error("Failed to update config", zap.String("key", key), zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update configuration",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"key":     key,
		"value":   req.Value,
	})
}

// UpdateMultipleConfigs handles PUT /api/v1/recovery/config
type UpdateMultipleConfigsRequest struct {
	Updates map[string]interface{} `json:"updates"`
}

func (h *RecoveryHandler) UpdateMultipleConfigs(c *fiber.Ctx) error {
	var req UpdateMultipleConfigsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.configRepo.UpdateConfigs(c.Context(), req.Updates); err != nil {
		logger.Error("Failed to update configs", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update configurations",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"updated": len(req.Updates),
	})
}

// =====================================================
// RECOVERY RULES ENDPOINTS
// =====================================================

// GetAllRules handles GET /api/v1/recovery/rules
func (h *RecoveryHandler) GetAllRules(c *fiber.Ctx) error {
	rules, err := h.configRepo.GetAllRules(c.Context())
	if err != nil {
		logger.Error("Failed to get rules", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get recovery rules",
		})
	}

	return c.JSON(fiber.Map{
		"rules": rules,
		"count": len(rules),
	})
}

// GetRule handles GET /api/v1/recovery/rules/:id
func (h *RecoveryHandler) GetRule(c *fiber.Ctx) error {
	id := c.Params("id")

	rule, err := h.configRepo.GetRuleByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "rule not found",
		})
	}

	return c.JSON(rule)
}

// CreateRuleRequest represents a rule creation request
type CreateRuleRequest struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Priority     int                    `json:"priority"`
	Enabled      bool                   `json:"enabled"`
	Pattern      string                 `json:"pattern"`
	Conditions   []interface{}          `json:"conditions,omitempty"`
	Action       string                 `json:"action"`
	ActionParams map[string]interface{} `json:"action_params,omitempty"`
	MaxRetries   int                    `json:"max_retries"`
	RetryDelay   int                    `json:"retry_delay"`
}

// CreateRule handles POST /api/v1/recovery/rules
func (h *RecoveryHandler) CreateRule(c *fiber.Ctx) error {
	var req CreateRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" || req.Pattern == "" || req.Action == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name, pattern, and action are required",
		})
	}

	rule := &repository.RecoveryRule{
		ID:           uuid.New().String(),
		Name:         req.Name,
		Description:  req.Description,
		Priority:     req.Priority,
		Enabled:      req.Enabled,
		Pattern:      req.Pattern,
		Conditions:   req.Conditions,
		Action:       req.Action,
		ActionParams: req.ActionParams,
		MaxRetries:   req.MaxRetries,
		RetryDelay:   req.RetryDelay,
	}

	if rule.MaxRetries == 0 {
		rule.MaxRetries = 3
	}
	if rule.RetryDelay == 0 {
		rule.RetryDelay = 5
	}

	if err := h.configRepo.CreateRule(c.Context(), rule); err != nil {
		logger.Error("Failed to create rule", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create rule",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(rule)
}

// UpdateRule handles PUT /api/v1/recovery/rules/:id
func (h *RecoveryHandler) UpdateRule(c *fiber.Ctx) error {
	id := c.Params("id")

	var req CreateRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	rule := &repository.RecoveryRule{
		ID:           id,
		Name:         req.Name,
		Description:  req.Description,
		Priority:     req.Priority,
		Enabled:      req.Enabled,
		Pattern:      req.Pattern,
		Conditions:   req.Conditions,
		Action:       req.Action,
		ActionParams: req.ActionParams,
		MaxRetries:   req.MaxRetries,
		RetryDelay:   req.RetryDelay,
	}

	if err := h.configRepo.UpdateRule(c.Context(), rule); err != nil {
		logger.Error("Failed to update rule", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update rule",
		})
	}

	return c.JSON(rule)
}

// DeleteRule handles DELETE /api/v1/recovery/rules/:id
func (h *RecoveryHandler) DeleteRule(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.configRepo.DeleteRule(c.Context(), id); err != nil {
		logger.Error("Failed to delete rule", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete rule",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ToggleRuleRequest represents a rule toggle request
type ToggleRuleRequest struct {
	Enabled bool `json:"enabled"`
}

// ToggleRule handles PATCH /api/v1/recovery/rules/:id/toggle
func (h *RecoveryHandler) ToggleRule(c *fiber.Ctx) error {
	id := c.Params("id")

	var req ToggleRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.configRepo.ToggleRule(c.Context(), id, req.Enabled); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to toggle rule",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"id":      id,
		"enabled": req.Enabled,
	})
}

// =====================================================
// PROXIES ENDPOINTS
// =====================================================

// GetAllProxies handles GET /api/v1/recovery/proxies
func (h *RecoveryHandler) GetAllProxies(c *fiber.Ctx) error {
	proxies, err := h.configRepo.GetAllProxies(c.Context())
	if err != nil {
		logger.Error("Failed to get proxies", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get proxies",
		})
	}

	stats, _ := h.configRepo.GetProxyStats(c.Context())

	return c.JSON(fiber.Map{
		"proxies": proxies,
		"stats":   stats,
		"count":   len(proxies),
	})
}

// CreateProxyRequest represents a proxy creation request
type CreateProxyRequest struct {
	ProxyID      string `json:"proxy_id"`
	Server       string `json:"server"`
	ProxyAddress string `json:"proxy_address"`
	Port         int    `json:"port"`
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	ProxyType    string `json:"proxy_type"`
}

// CreateProxy handles POST /api/v1/recovery/proxies
func (h *RecoveryHandler) CreateProxy(c *fiber.Ctx) error {
	var req CreateProxyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.ProxyAddress == "" || req.Port == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "proxy_address and port are required",
		})
	}

	id := uuid.New().String()
	proxyID := req.ProxyID
	if proxyID == "" {
		proxyID = id
	}

	proxy := &repository.Proxy{
		ID:           id,
		ProxyID:      proxyID,
		Server:       req.Server,
		ProxyAddress: req.ProxyAddress,
		Port:         req.Port,
		Username:     req.Username,
		Password:     req.Password,
		ProxyType:    req.ProxyType,
		IsHealthy:    true,
	}

	if proxy.ProxyType == "" {
		proxy.ProxyType = "static"
	}
	if proxy.Server == "" {
		proxy.Server = req.ProxyAddress
	}

	if err := h.configRepo.CreateProxy(c.Context(), proxy); err != nil {
		logger.Error("Failed to create proxy", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create proxy",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(proxy)
}

// UpdateProxy handles PUT /api/v1/recovery/proxies/:id
func (h *RecoveryHandler) UpdateProxy(c *fiber.Ctx) error {
	id := c.Params("id")

	var req CreateProxyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	proxy := &repository.Proxy{
		ID:           id,
		Server:       req.Server,
		ProxyAddress: req.ProxyAddress,
		Port:         req.Port,
		Username:     req.Username,
		Password:     req.Password,
		ProxyType:    req.ProxyType,
		IsHealthy:    true,
	}

	if err := h.configRepo.UpdateProxy(c.Context(), proxy); err != nil {
		logger.Error("Failed to update proxy", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update proxy",
		})
	}

	return c.JSON(proxy)
}

// DeleteProxy handles DELETE /api/v1/recovery/proxies/:id
func (h *RecoveryHandler) DeleteProxy(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.configRepo.DeleteProxy(c.Context(), id); err != nil {
		logger.Error("Failed to delete proxy", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete proxy",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ToggleProxy handles PATCH /api/v1/recovery/proxies/:id/toggle
func (h *RecoveryHandler) ToggleProxy(c *fiber.Ctx) error {
	id := c.Params("id")

	var req ToggleRuleRequest // Reuse same struct
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.configRepo.ToggleProxy(c.Context(), id, req.Enabled); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to toggle proxy",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"id":      id,
		"enabled": req.Enabled,
	})
}

// GetProxyStats handles GET /api/v1/recovery/proxies/stats
func (h *RecoveryHandler) GetProxyStats(c *fiber.Ctx) error {
	stats, err := h.configRepo.GetProxyStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get proxy stats",
		})
	}
	return c.JSON(stats)
}
