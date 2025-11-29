package handlers

import (
	"context"
	"time"

	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

type BrowserProfileHandler struct {
	repo            *storage.BrowserProfileRepository
	fingerprintGen  *browser.FingerprintGenerator
	browserLauncher *browser.BrowserLauncher
	activeSessions  map[string]playwright.Browser
	sessionMutex    sync.Mutex
}

func NewBrowserProfileHandler(repo *storage.BrowserProfileRepository, launcher *browser.BrowserLauncher) *BrowserProfileHandler {
	return &BrowserProfileHandler{
		repo:            repo,
		fingerprintGen:  browser.NewFingerprintGenerator(),
		browserLauncher: launcher,
		activeSessions:  make(map[string]playwright.Browser),
	}
}

// CreateProfile creates a new browser profile
func (h *BrowserProfileHandler) CreateProfile(c *fiber.Ctx) error {
	var req models.BrowserProfileCreateRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	profile := &models.BrowserProfile{
		ID:                  uuid.New().String(),
		Name:                req.Name,
		Description:         req.Description,
		Status:              "active",
		Folder:              req.Folder,
		Tags:                req.Tags,
		BrowserType:         req.BrowserType,
		ExecutablePath:      req.ExecutablePath,
		CDPEndpoint:         req.CDPEndpoint,
		LaunchArgs:          req.LaunchArgs,
		UserAgent:           req.UserAgent,
		Platform:            req.Platform,
		ScreenWidth:         req.ScreenWidth,
		ScreenHeight:        req.ScreenHeight,
		Timezone:            req.Timezone,
		Locale:              req.Locale,
		Languages:           req.Languages,
		WebGLVendor:         req.WebGLVendor,
		WebGLRenderer:       req.WebGLRenderer,
		CanvasNoise:         req.CanvasNoise,
		HardwareConcurrency: req.HardwareConcurrency,
		DeviceMemory:        req.DeviceMemory,
		Fonts:               req.Fonts,
		DoNotTrack:          req.DoNotTrack,
		DisableWebRTC:       req.DisableWebRTC,
		GeoLatitude:         req.GeoLatitude,
		GeoLongitude:        req.GeoLongitude,
		GeoAccuracy:         req.GeoAccuracy,
		ProxyEnabled:        req.ProxyEnabled,
		ProxyType:           req.ProxyType,
		ProxyServer:         req.ProxyServer,
		ProxyUsername:       req.ProxyUsername,
		ProxyPassword:       req.ProxyPassword,
		ClearOnClose:        req.ClearOnClose,
		UsageCount:          0,
	}

	// Validate profile
	if err := profile.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := h.repo.Create(context.Background(), profile); err != nil {
		logger.Error("Failed to create browser profile", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create profile",
		})
	}

	logger.Info("Browser profile created", zap.String("profile_id", profile.ID), zap.String("name", profile.Name))
	return c.Status(fiber.StatusCreated).JSON(profile)
}

// ListProfiles lists all browser profiles
func (h *BrowserProfileHandler) ListProfiles(c *fiber.Ctx) error {
	status := c.Query("status", "")
	folder := c.Query("folder", "")
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	profiles, err := h.repo.List(context.Background(), status, folder, limit, offset)
	if err != nil {
		logger.Error("Failed to list browser profiles", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list profiles",
		})
	}

	return c.JSON(fiber.Map{
		"profiles": profiles,
		"count":    len(profiles),
	})
}

// GetProfile retrieves a browser profile by ID
func (h *BrowserProfileHandler) GetProfile(c *fiber.Ctx) error {
	id := c.Params("id")

	profile, err := h.repo.GetByID(context.Background(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Profile not found",
		})
	}

	return c.JSON(profile)
}

// UpdateProfile updates a browser profile
func (h *BrowserProfileHandler) UpdateProfile(c *fiber.Ctx) error {
	id := c.Params("id")

	// Get existing profile
	profile, err := h.repo.GetByID(context.Background(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Profile not found",
		})
	}

	var req models.BrowserProfileUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Apply updates
	if req.Name != nil {
		profile.Name = *req.Name
	}
	if req.Description != nil {
		profile.Description = *req.Description
	}
	if req.Status != nil {
		profile.Status = *req.Status
	}
	if req.Folder != nil {
		profile.Folder = req.Folder
	}
	if req.Tags != nil {
		profile.Tags = req.Tags
	}
	if req.BrowserType != nil {
		profile.BrowserType = *req.BrowserType
	}
	if req.ExecutablePath != nil {
		profile.ExecutablePath = req.ExecutablePath
	}
	if req.CDPEndpoint != nil {
		profile.CDPEndpoint = req.CDPEndpoint
	}
	if req.LaunchArgs != nil {
		profile.LaunchArgs = req.LaunchArgs
	}
	if req.UserAgent != nil {
		profile.UserAgent = *req.UserAgent
	}
	if req.Platform != nil {
		profile.Platform = *req.Platform
	}
	if req.ScreenWidth != nil {
		profile.ScreenWidth = *req.ScreenWidth
	}
	if req.ScreenHeight != nil {
		profile.ScreenHeight = *req.ScreenHeight
	}
	if req.Timezone != nil {
		profile.Timezone = *req.Timezone
	}
	if req.Locale != nil {
		profile.Locale = *req.Locale
	}
	if req.Languages != nil {
		profile.Languages = req.Languages
	}
	if req.WebGLVendor != nil {
		profile.WebGLVendor = req.WebGLVendor
	}
	if req.WebGLRenderer != nil {
		profile.WebGLRenderer = req.WebGLRenderer
	}
	if req.CanvasNoise != nil {
		profile.CanvasNoise = *req.CanvasNoise
	}
	if req.HardwareConcurrency != nil {
		profile.HardwareConcurrency = *req.HardwareConcurrency
	}
	if req.DeviceMemory != nil {
		profile.DeviceMemory = *req.DeviceMemory
	}
	if req.Fonts != nil {
		profile.Fonts = req.Fonts
	}
	if req.DoNotTrack != nil {
		profile.DoNotTrack = *req.DoNotTrack
	}
	if req.DisableWebRTC != nil {
		profile.DisableWebRTC = *req.DisableWebRTC
	}
	if req.GeoLatitude != nil {
		profile.GeoLatitude = req.GeoLatitude
	}
	if req.GeoLongitude != nil {
		profile.GeoLongitude = req.GeoLongitude
	}
	if req.GeoAccuracy != nil {
		profile.GeoAccuracy = req.GeoAccuracy
	}
	if req.ProxyEnabled != nil {
		profile.ProxyEnabled = *req.ProxyEnabled
	}
	if req.ProxyType != nil {
		profile.ProxyType = req.ProxyType
	}
	if req.ProxyServer != nil {
		profile.ProxyServer = req.ProxyServer
	}
	if req.ProxyUsername != nil {
		profile.ProxyUsername = req.ProxyUsername
	}
	if req.ProxyPassword != nil {
		profile.ProxyPassword = req.ProxyPassword
	}
	if req.ClearOnClose != nil {
		profile.ClearOnClose = *req.ClearOnClose
	}

	// Validate updated profile
	if err := profile.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := h.repo.Update(context.Background(), profile); err != nil {
		logger.Error("Failed to update browser profile", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update profile",
		})
	}

	logger.Info("Browser profile updated", zap.String("profile_id", id))
	return c.JSON(profile)
}

// DeleteProfile deletes a browser profile
func (h *BrowserProfileHandler) DeleteProfile(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.repo.Delete(context.Background(), id); err != nil {
		logger.Error("Failed to delete browser profile", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete profile",
		})
	}

	logger.Info("Browser profile deleted", zap.String("profile_id", id))
	return c.Status(fiber.StatusNoContent).Send(nil)
}

// DuplicateProfile duplicates a browser profile
func (h *BrowserProfileHandler) DuplicateProfile(c *fiber.Ctx) error {
	id := c.Params("id")

	// Get the original profile
	originalProfile, err := h.repo.GetByID(context.Background(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Profile not found",
		})
	}

	// Create a new profile with copied settings
	newProfile := *originalProfile
	newProfile.ID = uuid.New().String()
	newProfile.Name = originalProfile.Name + " (Copy)"
	newProfile.UsageCount = 0
	newProfile.LastUsedAt = nil

	if err := h.repo.Create(context.Background(), &newProfile); err != nil {
		logger.Error("Failed to duplicate browser profile", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to duplicate profile",
		})
	}

	logger.Info("Browser profile duplicated",
		zap.String("original_id", id),
		zap.String("new_id", newProfile.ID))
	return c.Status(fiber.StatusCreated).JSON(newProfile)
}

// GenerateFingerprint generates a random browser fingerprint
func (h *BrowserProfileHandler) GenerateFingerprint(c *fiber.Ctx) error {
	var req struct {
		BrowserType string `json:"browser_type"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.BrowserType == "" {
		req.BrowserType = "chromium"
	}

	fingerprint, err := h.fingerprintGen.GenerateFingerprint(req.BrowserType)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fingerprint)
}

// GetFolders retrieves all unique folder names
func (h *BrowserProfileHandler) GetFolders(c *fiber.Ctx) error {
	folders, err := h.repo.GetFolders(context.Background())
	if err != nil {
		logger.Error("Failed to get folders", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get folders",
		})
	}

	return c.JSON(fiber.Map{
		"folders": folders,
	})
}

// GetBrowserTypes returns available browser types and their capabilities
func (h *BrowserProfileHandler) GetBrowserTypes(c *fiber.Ctx) error {
	browserTypes := []map[string]interface{}{
		{
			"type":        "chromium",
			"name":        "Chromium",
			"description": "Google Chrome, Microsoft Edge, Brave, and other Chromium-based browsers",
			"icon":        "chrome",
		},
		{
			"type":        "firefox",
			"name":        "Firefox",
			"description": "Mozilla Firefox browser",
			"icon":        "firefox",
		},
		{
			"type":        "webkit",
			"name":        "WebKit",
			"description": "Safari browser (macOS/iOS)",
			"icon":        "safari",
		},
	}

	return c.JSON(fiber.Map{
		"browser_types": browserTypes,
	})
}

// TestBrowserConfig tests a browser configuration before saving
func (h *BrowserProfileHandler) TestBrowserConfig(c *fiber.Ctx) error {
	var req struct {
		BrowserType    string  `json:"browser_type"`
		ExecutablePath *string `json:"executable_path,omitempty"`
		CDPEndpoint    *string `json:"cdp_endpoint,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Create a temporary profile for testing
	testProfile := models.DefaultBrowserProfile("Test Profile")
	testProfile.BrowserType = req.BrowserType
	testProfile.ExecutablePath = req.ExecutablePath
	testProfile.CDPEndpoint = req.CDPEndpoint

	// Validate the configuration
	if err := testProfile.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   err.Error(),
			"success": false,
		})
	}

	// Try to launch the browser
	launchOpts := &browser.LaunchOptions{
		Headless: true,
		Timeout:  30000,
		Profile:  testProfile,
	}

	ctx := context.Background()
	browser, err := h.browserLauncher.LaunchBrowserWithProfile(ctx, launchOpts)
	if err != nil {
		logger.Warn("Browser configuration test failed", zap.Error(err))
		return c.JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}
	defer browser.Close()

	logger.Info("Browser configuration test successful",
		zap.String("browser_type", req.BrowserType))
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Browser configuration is valid and working",
	})
}

// TestProfile tests a complete profile by launching a browser
func (h *BrowserProfileHandler) TestProfile(c *fiber.Ctx) error {
	id := c.Params("id")

	profile, err := h.repo.GetByID(context.Background(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Profile not found",
		})
	}

	// Try to launch the browser with this profile
	launchOpts := &browser.LaunchOptions{
		Headless: true,
		Timeout:  30000,
		Profile:  profile,
	}

	ctx := context.Background()
	browser, err := h.browserLauncher.LaunchBrowserWithProfile(ctx, launchOpts)
	if err != nil {
		logger.Warn("Profile test failed", zap.Error(err), zap.String("profile_id", id))
		return c.JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}
	defer browser.Close()

	logger.Info("Profile test successful", zap.String("profile_id", id))
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Profile is valid and working",
	})
}

// LaunchProfile launches a browser profile in standalone mode and keeps it running
func (h *BrowserProfileHandler) LaunchProfile(c *fiber.Ctx) error {
	id := c.Params("id")
	headless := c.QueryBool("headless", false) // Default to headed mode for standalone use

	h.sessionMutex.Lock()
	if _, exists := h.activeSessions[id]; exists {
		h.sessionMutex.Unlock()
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Profile is already running",
		})
	}
	h.sessionMutex.Unlock()

	profile, err := h.repo.GetByID(context.Background(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Profile not found",
		})
	}

	// Launch options
	launchOpts := &browser.LaunchOptions{
		Headless: headless,
		Timeout:  0, // No timeout for standalone session
		Profile:  profile,
	}

	ctx := context.Background()
	browserInstance, err := h.browserLauncher.LaunchBrowserWithProfile(ctx, launchOpts)
	if err != nil {
		logger.Error("Failed to launch profile", zap.Error(err), zap.String("profile_id", id))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to launch profile",
		})
	}

	// Create a browser context with fingerprinting to make the window visible
	browserContext, err := h.browserLauncher.CreateContextWithFingerprint(browserInstance, profile)
	if err != nil {
		browserInstance.Close()
		logger.Error("Failed to create browser context", zap.Error(err), zap.String("profile_id", id))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create browser context",
		})
	}

	// Create a new page to make the browser window visible
	page, err := browserContext.NewPage()
	if err != nil {
		browserContext.Close()
		browserInstance.Close()
		logger.Error("Failed to create page", zap.Error(err), zap.String("profile_id", id))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create page",
		})
	}

	// Navigate to a blank page or default URL
	if _, err := page.Goto("about:blank"); err != nil {
		logger.Warn("Failed to navigate to blank page", zap.Error(err))
	}

	// Update profile status to 'running' in database BEFORE returning response
	logger.Info("Updating profile status to running", zap.String("profile_id", id))
	profile.Status = "running"
	if err := h.repo.Update(context.Background(), profile); err != nil {
		logger.Error("Failed to update profile status to running", zap.Error(err), zap.String("profile_id", id))
		// Don't fail the launch, but log the error
	} else {
		logger.Info("Profile status updated to running in database", zap.String("profile_id", id))
	}

	// Store session (we store the browser, but the context/page keep it alive)
	h.sessionMutex.Lock()
	h.activeSessions[id] = browserInstance
	h.sessionMutex.Unlock()

	// Start a monitoring goroutine to check if the browser window is closed
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Check if we still have the session
			h.sessionMutex.Lock()
			_, exists := h.activeSessions[id]
			h.sessionMutex.Unlock()

			if !exists {
				// Session was removed (e.g. via StopProfile), stop monitoring
				return
			}

			// Check if browser context has any open pages
			// If user closes the window, pages count will be 0
			pages := browserContext.Pages()
			if len(pages) == 0 {
				logger.Info("No open pages found, assuming browser window closed", zap.String("profile_id", id))

				// Close browser resources
				browserInstance.Close()

				// Remove from active sessions
				h.sessionMutex.Lock()
				delete(h.activeSessions, id)
				h.sessionMutex.Unlock()

				// Update profile status back to 'active'
				ctx := context.Background()
				profile, err := h.repo.GetByID(ctx, id)
				if err == nil && profile.Status == "running" {
					profile.Status = "active"
					if err := h.repo.Update(ctx, profile); err != nil {
						logger.Error("Failed to update profile status after auto-close", zap.Error(err), zap.String("profile_id", id))
					} else {
						logger.Info("Profile status updated to active after auto-close", zap.String("profile_id", id))
					}
				}
				return // Exit monitoring loop
			}
		}
	}()

	// Also keep the disconnect listener as a backup (e.g. if browser crashes)
	browserInstance.OnDisconnected(func(browser playwright.Browser) {
		// This will handle crash cases, but the polling loop handles the "X" button better
		h.sessionMutex.Lock()
		if _, exists := h.activeSessions[id]; exists {
			delete(h.activeSessions, id)
			// Update status logic here as fallback...
			go func() {
				ctx := context.Background()
				if profile, err := h.repo.GetByID(ctx, id); err == nil && profile.Status == "running" {
					profile.Status = "active"
					h.repo.Update(ctx, profile)
				}
			}()
		}
		h.sessionMutex.Unlock()
	})

	logger.Info("Profile launched successfully with visible window", zap.String("profile_id", id))
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Profile launched successfully",
		"id":      id,
	})
}

// StopProfile stops a running standalone browser profile
func (h *BrowserProfileHandler) StopProfile(c *fiber.Ctx) error {
	id := c.Params("id")

	h.sessionMutex.Lock()
	browserInstance, exists := h.activeSessions[id]
	if !exists {
		h.sessionMutex.Unlock()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Profile is not running",
		})
	}
	delete(h.activeSessions, id)
	h.sessionMutex.Unlock()

	// Close the browser
	if err := browserInstance.Close(); err != nil {
		logger.Error("Failed to close browser", zap.Error(err), zap.String("profile_id", id))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to close browser",
		})
	}

	// Update profile status back to 'active' in database
	ctx := context.Background()
	profile, err := h.repo.GetByID(ctx, id)
	if err != nil {
		logger.Error("Failed to get profile for status update", zap.Error(err), zap.String("profile_id", id))
		// Continue even if this fails
	} else {
		profile.Status = "active"
		if err := h.repo.Update(ctx, profile); err != nil {
			logger.Error("Failed to update profile status after stop", zap.Error(err), zap.String("profile_id", id))
		}
	}

	logger.Info("Profile stopped successfully", zap.String("profile_id", id))
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Profile stopped successfully",
	})
}
