package handlers

import (
	"math/rand"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// BrowserProfileHandler handles browser profile HTTP requests
type BrowserProfileHandler struct {
	profileRepo repository.BrowserProfileRepository
}

// NewBrowserProfileHandler creates a new browser profile handler
func NewBrowserProfileHandler(profileRepo repository.BrowserProfileRepository) *BrowserProfileHandler {
	return &BrowserProfileHandler{
		profileRepo: profileRepo,
	}
}

// CreateProfile handles POST /api/v1/profiles
func (h *BrowserProfileHandler) CreateProfile(c *fiber.Ctx) error {
	var profile models.BrowserProfile

	if err := c.BodyParser(&profile); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	profile.SetDefaults()

	if err := h.profileRepo.Create(c.Context(), &profile); err != nil {
		logger.Error("Failed to create browser profile", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(profile)
}

// GetProfile handles GET /api/v1/profiles/:id
func (h *BrowserProfileHandler) GetProfile(c *fiber.Ctx) error {
	id := c.Params("id")

	profile, err := h.profileRepo.Get(c.Context(), id)
	if err != nil {
		logger.Error("Failed to get browser profile", zap.Error(err))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "profile not found",
		})
	}

	return c.JSON(profile)
}

// ListProfiles handles GET /api/v1/profiles
func (h *BrowserProfileHandler) ListProfiles(c *fiber.Ctx) error {
	filters := repository.BrowserProfileFilters{
		Limit:      c.QueryInt("limit", 50),
		Offset:     c.QueryInt("offset", 0),
		Status:     c.Query("status", ""),
		Folder:     c.Query("folder", ""),
		DriverType: c.Query("driver_type", ""),
	}

	profiles, err := h.profileRepo.List(c.Context(), filters)
	if err != nil {
		logger.Error("Failed to list browser profiles", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list profiles",
		})
	}

	return c.JSON(fiber.Map{
		"profiles": profiles,
		"count":    len(profiles),
	})
}

// UpdateProfile handles PUT /api/v1/profiles/:id
func (h *BrowserProfileHandler) UpdateProfile(c *fiber.Ctx) error {
	id := c.Params("id")

	var profile models.BrowserProfile
	if err := c.BodyParser(&profile); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	profile.ID = id

	if err := h.profileRepo.Update(c.Context(), &profile); err != nil {
		logger.Error("Failed to update browser profile", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(profile)
}

// DeleteProfile handles DELETE /api/v1/profiles/:id
func (h *BrowserProfileHandler) DeleteProfile(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.profileRepo.Delete(c.Context(), id); err != nil {
		logger.Error("Failed to delete browser profile", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// DuplicateProfile handles POST /api/v1/profiles/:id/duplicate
func (h *BrowserProfileHandler) DuplicateProfile(c *fiber.Ctx) error {
	id := c.Params("id")

	duplicate, err := h.profileRepo.Duplicate(c.Context(), id)
	if err != nil {
		logger.Error("Failed to duplicate browser profile", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(duplicate)
}

// GenerateFingerprint handles POST /api/v1/profiles/generate-fingerprint
func (h *BrowserProfileHandler) GenerateFingerprint(c *fiber.Ctx) error {
	var req struct {
		BrowserType string `json:"browser_type"`
	}

	if err := c.BodyParser(&req); err != nil {
		req.BrowserType = "chromium"
	}

	fingerprint := generateRandomFingerprint(req.BrowserType)
	return c.JSON(fingerprint)
}

// GetBrowserTypes handles GET /api/v1/profiles/browser-types
func (h *BrowserProfileHandler) GetBrowserTypes(c *fiber.Ctx) error {
	browserTypes := []struct {
		Type        string `json:"type"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	}{
		{Type: "chromium", Name: "Chromium", Description: "Google Chrome, Microsoft Edge, Brave", Icon: "chrome"},
		{Type: "firefox", Name: "Firefox", Description: "Mozilla Firefox", Icon: "firefox"},
		{Type: "webkit", Name: "WebKit", Description: "Safari (macOS/iOS)", Icon: "safari"},
	}

	return c.JSON(fiber.Map{
		"browser_types": browserTypes,
	})
}

// GetFolders handles GET /api/v1/profiles/folders
func (h *BrowserProfileHandler) GetFolders(c *fiber.Ctx) error {
	folders, err := h.profileRepo.GetFolders(c.Context())
	if err != nil {
		logger.Error("Failed to get folders", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get folders",
		})
	}

	return c.JSON(fiber.Map{
		"folders": folders,
	})
}

// TestBrowserConfig handles POST /api/v1/profiles/test-browser-config
func (h *BrowserProfileHandler) TestBrowserConfig(c *fiber.Ctx) error {
	var req struct {
		BrowserType    string `json:"browser_type"`
		ExecutablePath string `json:"executable_path"`
		CDPEndpoint    string `json:"cdp_endpoint"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Basic validation - actual browser testing would require spawning a browser
	// which is done by the worker, not orchestrator
	switch req.BrowserType {
	case "chromium", "firefox", "webkit", "":
		// Valid
	default:
		return c.JSON(fiber.Map{
			"success": false,
			"message": "Invalid browser type",
			"error":   "browser_type must be chromium, firefox, or webkit",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Browser configuration is valid",
	})
}

// Fingerprint generation data
type Fingerprint struct {
	UserAgent           string   `json:"UserAgent"`
	Platform            string   `json:"Platform"`
	ScreenWidth         int      `json:"ScreenWidth"`
	ScreenHeight        int      `json:"ScreenHeight"`
	Timezone            string   `json:"Timezone"`
	Locale              string   `json:"Locale"`
	Languages           []string `json:"Languages"`
	WebGLVendor         string   `json:"WebGLVendor"`
	WebGLRenderer       string   `json:"WebGLRenderer"`
	HardwareConcurrency int      `json:"HardwareConcurrency"`
	DeviceMemory        int      `json:"DeviceMemory"`
	Fonts               []string `json:"Fonts"`
}

func generateRandomFingerprint(browserType string) Fingerprint {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	// User agents per browser type
	chromiumUAs := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}

	firefoxUAs := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0",
	}

	webkitUAs := []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Safari/605.1.15",
	}

	// Select user agent based on browser type
	var userAgents []string
	switch browserType {
	case "firefox":
		userAgents = firefoxUAs
	case "webkit":
		userAgents = webkitUAs
	default:
		userAgents = chromiumUAs
	}

	platforms := []string{"Win32", "MacIntel", "Linux x86_64"}
	resolutions := [][2]int{{1920, 1080}, {1366, 768}, {1440, 900}, {2560, 1440}, {1536, 864}}
	timezones := []string{
		"America/New_York", "America/Los_Angeles", "America/Chicago",
		"Europe/London", "Europe/Paris", "Europe/Berlin",
		"Asia/Tokyo", "Asia/Shanghai", "Asia/Singapore",
	}
	locales := []string{"en-US", "en-GB", "de-DE", "fr-FR", "es-ES", "ja-JP", "zh-CN"}

	webglVendors := []string{
		"Intel Inc.", "NVIDIA Corporation", "AMD", "Google Inc. (NVIDIA)",
		"Google Inc. (Intel)", "Google Inc. (AMD)",
	}

	webglRenderers := []string{
		"Intel Iris OpenGL Engine",
		"Intel(R) UHD Graphics 630",
		"NVIDIA GeForce GTX 1080 Ti/PCIe/SSE2",
		"AMD Radeon Pro 5500M OpenGL Engine",
		"ANGLE (NVIDIA, NVIDIA GeForce GTX 1660 SUPER Direct3D11 vs_5_0 ps_5_0)",
		"ANGLE (Intel, Intel(R) UHD Graphics 620 Direct3D11 vs_5_0 ps_5_0, D3D11)",
	}

	hardwareConcurrencies := []int{4, 6, 8, 12, 16}
	deviceMemories := []int{4, 8, 16, 32}

	res := resolutions[rnd.Intn(len(resolutions))]
	locale := locales[rnd.Intn(len(locales))]

	return Fingerprint{
		UserAgent:           userAgents[rnd.Intn(len(userAgents))],
		Platform:            platforms[rnd.Intn(len(platforms))],
		ScreenWidth:         res[0],
		ScreenHeight:        res[1],
		Timezone:            timezones[rnd.Intn(len(timezones))],
		Locale:              locale,
		Languages:           []string{locale, locale[:2]},
		WebGLVendor:         webglVendors[rnd.Intn(len(webglVendors))],
		WebGLRenderer:       webglRenderers[rnd.Intn(len(webglRenderers))],
		HardwareConcurrency: hardwareConcurrencies[rnd.Intn(len(hardwareConcurrencies))],
		DeviceMemory:        deviceMemories[rnd.Intn(len(deviceMemories))],
		Fonts: []string{
			"Arial", "Arial Black", "Comic Sans MS", "Courier New", "Georgia",
			"Impact", "Times New Roman", "Trebuchet MS", "Verdana",
		},
	}
}
