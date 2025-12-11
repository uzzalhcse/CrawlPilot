package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/camoufox-go"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/services"
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
	Camoufox  *camoufox.Camoufox
	CloseCh   chan struct{}
}

// NewBrowserManager creates a new browser manager
func NewBrowserManager(profileRepo repository.BrowserProfileRepository) *BrowserManager {
	return &BrowserManager{
		profileRepo: profileRepo,
		sessions:    make(map[string]*ManagedBrowserSession),
	}
}

// Note: ensureScheme has been moved to shared/services/browser_factory.go as services.EnsureScheme

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

	// Launch browser
	var browser playwright.Browser
	var pw *playwright.Playwright
	var cam *camoufox.Camoufox

	if profile.DriverType == "camoufox" {
		// Use shared browser factory for consistent configuration (includes proxy validation)
		opts, err := services.BuildCamoufoxOptions(profile, false /* headless - always headed for manual interaction */)
		if err != nil {
			return fmt.Errorf("failed to build camoufox options: %w", err)
		}
		opts.Debug = true // Enable debug for orchestrator

		// Launch Camoufox using shared factory options
		cam, err = camoufox.NewBrowser(opts)
		if err != nil {
			return fmt.Errorf("failed to launch camoufox: %w", err)
		}
		browser = cam.Browser()
	} else {
		// Initialize Playwright for standard browsers
		var err error
		pw, err = playwright.Run()
		if err != nil {
			return fmt.Errorf("failed to start playwright: %w", err)
		}

		// Use shared browser factory for launch options
		browserType := services.GetPlaywrightBrowserType(profile)
		launchOpts := services.BuildPlaywrightLaunchOptions(profile, false /* headless - always headed for manual interaction */)

		// Standard Playwright launch
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
	}

	// Create context options using shared browser factory (only for Playwright, not Camoufox)
	var contextOpts playwright.BrowserNewContextOptions
	var fingerprint *services.Fingerprint
	if cam == nil {
		// Get fingerprint service for init script injection
		fpService, _ := services.GetFingerprintService()

		// Use shared browser factory for context options (includes fingerprint and proxy validation)
		var err error
		contextOpts, err = services.BuildPlaywrightContextOptions(profile, fpService)
		if err != nil {
			browser.Close()
			if pw != nil {
				pw.Stop()
			}
			return fmt.Errorf("failed to build context options: %w", err)
		}

		// Get fingerprint for init script (if needed)
		if fpService != nil {
			browserType := services.GetPlaywrightBrowserType(profile)
			targetOS := "linux"
			if profile.TargetOS != "" {
				targetOS = profile.TargetOS
			}
			maxWidth := profile.ScreenWidth
			if maxWidth == 0 {
				maxWidth = 1920
			}
			maxHeight := profile.ScreenHeight
			if maxHeight == 0 {
				maxHeight = 1080
			}
			fingerprint, _ = fpService.GenerateFingerprintWithBrowser(targetOS, browserType, maxWidth, maxHeight)
		}
	}

	// Create context
	var browserCtx playwright.BrowserContext
	if cam != nil {
		// Camoufox handles context creation with fingerprint
		browserCtx, err = cam.NewContext()
	} else {
		browserCtx, err = browser.NewContext(contextOpts)
	}

	if err != nil {
		browser.Close()
		if pw != nil {
			pw.Stop()
		}
		if cam != nil {
			cam.Close()
		}
		return fmt.Errorf("failed to create browser context: %w", err)
	}

	// Inject additional fingerprint properties via init script (only for non-Camoufox)
	// Inject additional fingerprint properties via init script (only for non-Camoufox)
	if cam == nil && fingerprint != nil {
		initScript := buildFingerprintInjectionScript(fingerprint)
		if err := browserCtx.AddInitScript(playwright.Script{
			Content: playwright.String(initScript),
		}); err != nil {
			browser.Close()
			if pw != nil {
				pw.Stop()
			}
			return fmt.Errorf("failed to add init script: %w", err)
		}
	}

	// Create page
	page, err := browserCtx.NewPage()
	if err != nil {
		browserCtx.Close()
		browser.Close()
		if pw != nil {
			pw.Stop()
		}
		return fmt.Errorf("failed to create page: %w", err)
	}

	resp, err := page.Goto("https://www.browserscan.net/")

	if err != nil {
		return fmt.Errorf("failed to navigate to browserscan: %w", err)
	}
	fmt.Printf("Response status: %d\n", resp.Status())

	// Store session
	session := &ManagedBrowserSession{
		ProfileID: profileID,
		Pw:        pw,
		Browser:   browser,
		Context:   browserCtx,
		Page:      page,
		Camoufox:  cam,
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
	// Update DB status
	profile, err := m.profileRepo.Get(ctx, profileID)
	if err == nil {
		profile.Status = "active" // Reset to active (ready)
		if err := m.profileRepo.Update(ctx, profile); err != nil {
			logger.Error("Failed to update profile status to active", zap.Error(err))
		}
	}
	m.mu.Lock()
	session, exists := m.sessions[profileID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("browser profile %s is not running", profileID)
	}
	delete(m.sessions, profileID)
	m.mu.Unlock()

	// Close resources
	if session.Camoufox != nil {
		if err := session.Camoufox.Close(); err != nil {
			logger.Warn("Failed to close camoufox cleanly", zap.Error(err))
		}
	} else {
		if err := session.Browser.Close(); err != nil {
			logger.Warn("Failed to close browser cleanly", zap.Error(err))
		}
		if session.Pw != nil {
			if err := session.Pw.Stop(); err != nil {
				logger.Warn("Failed to stop playwright cleanly", zap.Error(err))
			}
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

			// Stop playwright/camoufox
			go func() {
				if session.Camoufox != nil {
					if err := session.Camoufox.Close(); err != nil {
						logger.Warn("Failed to close camoufox after manual close", zap.Error(err))
					}
				} else if session.Pw != nil {
					if err := session.Pw.Stop(); err != nil {
						logger.Warn("Failed to stop playwright after manual close", zap.Error(err))
					}
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

func buildFingerprintInjectionScript(fp *services.Fingerprint) string {
	return fmt.Sprintf(`
		// Inject BrowserForge fingerprint properties
		Object.defineProperty(navigator, 'hardwareConcurrency', {
			get: () => %d,
			configurable: false
		});
		
		Object.defineProperty(navigator, 'platform', {
			get: () => '%s',
			configurable: false
		});
		
		Object.defineProperty(navigator, 'maxTouchPoints', {
			get: () => %d,
			configurable: false
		});
		
		// Hide that we're using Playwright
		delete navigator.webdriver;
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined,
			configurable: false
		});
	`, fp.Navigator.HardwareConcurrency, fp.Navigator.Platform, fp.Navigator.MaxTouchPoints)
}
