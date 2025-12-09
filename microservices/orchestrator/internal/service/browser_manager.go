package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// BrowserManager manages active browser sessions
type BrowserManager struct {
	profileRepo repository.BrowserProfileRepository
	sessions    map[string]*ManagedBrowserSession
	mu          sync.RWMutex
}

// ManagedBrowserSession represents an active browser session managed by BrowserManager
type ManagedBrowserSession struct {
	ProfileID string
	Pw        *playwright.Playwright
	Browser   playwright.Browser
	Context   playwright.BrowserContext
	Page      playwright.Page
	CloseCh   chan struct{}
}

// NewBrowserManager creates a new browser manager
func NewBrowserManager(profileRepo repository.BrowserProfileRepository) *BrowserManager {
	return &BrowserManager{
		profileRepo: profileRepo,
		sessions:    make(map[string]*ManagedBrowserSession),
	}
}

// Launch launches a browser for the given profile
func (m *BrowserManager) Launch(ctx context.Context, profileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already running
	if _, exists := m.sessions[profileID]; exists {
		return fmt.Errorf("browser profile %s is already running", profileID)
	}

	// Get profile details
	profile, err := m.profileRepo.Get(ctx, profileID)
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	// Initialize Playwright
	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("failed to start playwright: %w", err)
	}

	// Determine browser type
	browserType := profile.BrowserType
	if browserType == "" {
		browserType = "chromium"
	}

	// Launch options
	launchOpts := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false), // Always headed for manual interaction
		Args: []string{
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-dev-shm-usage",
			"--disable-gpu",
		},
	}

	if profile.ExecutablePath != "" {
		launchOpts.ExecutablePath = playwright.String(profile.ExecutablePath)
	}

	if len(profile.LaunchArgs) > 0 {
		launchOpts.Args = append(launchOpts.Args, profile.LaunchArgs...)
	}

	// Launch browser
	var browser playwright.Browser
	switch browserType {
	case "firefox":
		browser, err = pw.Firefox.Launch(launchOpts)
	case "webkit":
		browser, err = pw.WebKit.Launch(launchOpts)
	default:
		browser, err = pw.Chromium.Launch(launchOpts)
	}
	if err != nil {
		pw.Stop()
		return fmt.Errorf("failed to launch %s browser: %w", browserType, err)
	}

	// Create context options with fingerprint
	contextOpts := playwright.BrowserNewContextOptions{
		IgnoreHttpsErrors: playwright.Bool(true),
		JavaScriptEnabled: playwright.Bool(true),
	}

	// Apply fingerprint settings
	if profile.UserAgent != "" {
		contextOpts.UserAgent = playwright.String(profile.UserAgent)
	}
	if profile.ScreenWidth > 0 && profile.ScreenHeight > 0 {
		contextOpts.Screen = &playwright.Size{
			Width:  profile.ScreenWidth,
			Height: profile.ScreenHeight,
		}
		contextOpts.Viewport = &playwright.Size{
			Width:  profile.ScreenWidth,
			Height: profile.ScreenHeight,
		}
	}
	if profile.Locale != "" {
		contextOpts.Locale = playwright.String(profile.Locale)
	}
	if profile.Timezone != "" {
		contextOpts.TimezoneId = playwright.String(profile.Timezone)
	}
	if profile.GeolocationLatitude != nil && profile.GeolocationLongitude != nil {
		contextOpts.Geolocation = &playwright.Geolocation{
			Latitude:  *profile.GeolocationLatitude,
			Longitude: *profile.GeolocationLongitude,
		}
		contextOpts.Permissions = []string{"geolocation"}
	}

	// Proxy configuration
	if profile.ProxyEnabled && profile.ProxyServer != "" {
		proxyURL := profile.ProxyServer
		if profile.ProxyType != "" && profile.ProxyType != "http" {
			proxyURL = profile.ProxyType + "://" + proxyURL
		}
		contextOpts.Proxy = &playwright.Proxy{
			Server: proxyURL,
		}
		if profile.ProxyUsername != "" {
			contextOpts.Proxy.Username = playwright.String(profile.ProxyUsername)
			contextOpts.Proxy.Password = playwright.String(profile.ProxyPassword)
		}
	}

	// Create context
	browserCtx, err := browser.NewContext(contextOpts)
	if err != nil {
		browser.Close()
		pw.Stop()
		return fmt.Errorf("failed to create browser context: %w", err)
	}

	// Create page
	page, err := browserCtx.NewPage()
	if err != nil {
		browserCtx.Close()
		browser.Close()
		pw.Stop()
		return fmt.Errorf("failed to create page: %w", err)
	}

	// Store session
	session := &ManagedBrowserSession{
		ProfileID: profileID,
		Pw:        pw,
		Browser:   browser,
		Context:   browserCtx,
		Page:      page,
		CloseCh:   make(chan struct{}),
	}
	m.sessions[profileID] = session

	// Update DB status
	profile.Status = "running"
	if err := m.profileRepo.Update(ctx, profile); err != nil {
		logger.Error("Failed to update profile status to running", zap.Error(err))
	}

	// Monitor for closure
	go m.monitorSession(profileID, session)

	return nil
}

// Stop stops the browser for the given profile
func (m *BrowserManager) Stop(ctx context.Context, profileID string) error {
	m.mu.Lock()
	session, exists := m.sessions[profileID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("browser profile %s is not running", profileID)
	}
	delete(m.sessions, profileID)
	m.mu.Unlock()

	// Close resources
	if err := session.Browser.Close(); err != nil {
		logger.Warn("Failed to close browser cleanly", zap.Error(err))
	}
	if err := session.Pw.Stop(); err != nil {
		logger.Warn("Failed to stop playwright cleanly", zap.Error(err))
	}

	// Update DB status
	profile, err := m.profileRepo.Get(ctx, profileID)
	if err == nil {
		profile.Status = "active" // Reset to active (ready)
		if err := m.profileRepo.Update(ctx, profile); err != nil {
			logger.Error("Failed to update profile status to active", zap.Error(err))
		}
	}

	return nil
}

func (m *BrowserManager) monitorSession(profileID string, session *ManagedBrowserSession) {
	cleanup := func(source string) {
		m.mu.Lock()
		defer m.mu.Unlock()

		if _, exists := m.sessions[profileID]; exists {
			logger.Info("Browser session closed manually",
				zap.String("profile_id", profileID),
				zap.String("source", source),
			)
			delete(m.sessions, profileID)

			// Stop playwright
			go func() {
				if err := session.Pw.Stop(); err != nil {
					logger.Warn("Failed to stop playwright after manual close", zap.Error(err))
				}
			}()

			// Update DB status
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			profile, err := m.profileRepo.Get(ctx, profileID)
			if err == nil {
				profile.Status = "active"
				if err := m.profileRepo.Update(ctx, profile); err != nil {
					logger.Error("Failed to update profile status after manual close", zap.Error(err))
				}
			}
		}
	}

	// Wait for browser to be disconnected
	session.Browser.On("disconnected", func() {
		cleanup("browser_disconnected")
	})

	// Also wait for context close (closing the window often closes the context first)
	session.Context.On("close", func() {
		cleanup("context_closed")
	})
}
