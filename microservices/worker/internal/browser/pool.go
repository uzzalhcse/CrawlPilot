package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/camoufox-go"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/shared/services"
	"go.uber.org/zap"
)

// PooledContext wraps a browser context with metadata for lifecycle management
type PooledContext struct {
	ctx       playwright.BrowserContext
	createdAt time.Time
}

// Pool manages a pool of browser contexts
type Pool struct {
	browser     playwright.Browser
	contexts    chan *PooledContext
	config      *config.BrowserConfig
	profile     *models.BrowserProfile // Optional profile for fingerprint settings
	mu          sync.Mutex
	activeCount int
	pw          *playwright.Playwright
	cam         *camoufox.Camoufox
	semaphore   chan struct{} // Limits concurrent browser operations
}

// ProxyConfig holds proxy settings for a browser context
type ProxyConfig struct {
	Server   string // proxy server URL (e.g., http://host:port)
	Username string
	Password string
}

// NewPool creates a new browser pool
func NewPool(cfg *config.BrowserConfig) (*Pool, error) {
	return newPoolInternal(cfg, nil)
}

// NewPoolWithProfile creates a new browser pool with profile-specific settings
// This enables profile-based browser type selection and fingerprint configuration
func NewPoolWithProfile(cfg *config.BrowserConfig, profile *models.BrowserProfile) (*Pool, error) {
	return newPoolInternal(cfg, profile)
}

// newPoolInternal is the internal pool creation function
func newPoolInternal(cfg *config.BrowserConfig, profile *models.BrowserProfile) (*Pool, error) {
	// Launch browser
	var browser playwright.Browser
	var pw *playwright.Playwright
	var cam *camoufox.Camoufox
	var browserType string = "camoufox"

	if profile != nil && profile.DriverType == "camoufox" {
		// Configure Camoufox options
		opts := camoufox.Options{
			Headless:        cfg.Headless,
			Humanize:        profile.Humanize,
			VirtualHeadless: profile.VirtualHeadless,
			BlockImages:     profile.BlockImages,
			BlockWebGL:      profile.BlockWebGL,
			OS:              profile.TargetOS,
			GeoIP:           profile.GeoIP,
		}

		// Launch Camoufox
		var err error
		cam, err = camoufox.NewBrowser(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to launch camoufox: %w", err)
		}
		browser = cam.Browser()
	} else {
		// Initialize Playwright for standard browsers
		var err error
		pw, err = playwright.Run()
		if err != nil {
			return nil, fmt.Errorf("failed to start playwright: %w", err)
		}

		// Determine browser type from profile or default to Chromium
		browserType = "chromium"
		if profile != nil && profile.BrowserType != "" {
			browserType = profile.BrowserType
		}

		// Build launch options
		launchOpts := playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(cfg.Headless),
			Args: []string{
				"--no-sandbox",
				"--disable-setuid-sandbox",
				"--disable-dev-shm-usage",
				"--disable-gpu",
			},
		}

		// Add custom executable path from profile
		if profile != nil && profile.ExecutablePath != "" {
			launchOpts.ExecutablePath = playwright.String(profile.ExecutablePath)
		}

		// Add extra launch args from profile
		if profile != nil && len(profile.LaunchArgs) > 0 {
			launchOpts.Args = append(launchOpts.Args, profile.LaunchArgs...)
		}

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
			return nil, fmt.Errorf("failed to launch %s browser: %w", browserType, err)
		}
	}

	// Initialize semaphore for max_concurrency (defaults to pool_size if not set)
	maxConcurrency := cfg.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = cfg.PoolSize
	}

	pool := &Pool{
		browser:   browser,
		contexts:  make(chan *PooledContext, cfg.PoolSize),
		config:    cfg,
		profile:   profile,
		pw:        pw,
		cam:       cam,
		semaphore: make(chan struct{}, maxConcurrency),
	}

	// Pre-create contexts with profile fingerprint applied
	for i := 0; i < cfg.PoolSize; i++ {
		ctx, err := pool.createContextWithProfile(nil)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to create initial context: %w", err)
		}
		pool.contexts <- &PooledContext{
			ctx:       ctx,
			createdAt: time.Now(),
		}
	}

	logger.Info("Browser pool initialized",
		zap.Int("pool_size", cfg.PoolSize),
		zap.Int("max_concurrency", maxConcurrency),
		zap.Int("context_lifetime_sec", cfg.ContextLifetime),
		zap.Bool("headless", cfg.Headless),
		zap.String("browser_type", browserType),
	)

	return pool, nil
}

// createContext creates a new browser context with optional proxy
func (p *Pool) createContext(proxy *ProxyConfig) (playwright.BrowserContext, error) {
	return p.createContextWithProfile(proxy)
}

// createContextWithProfile creates a new browser context with profile fingerprint settings
func (p *Pool) createContextWithProfile(proxy *ProxyConfig) (playwright.BrowserContext, error) {
	// Generate BrowserForge fingerprint
	var fingerprint *services.Fingerprint
	if p.profile != nil {
		fpService, err := services.GetFingerprintService()
		if err == nil {
			targetOS := "linux"
			if p.profile.TargetOS != "" {
				targetOS = p.profile.TargetOS
			}

			maxWidth := p.profile.ScreenWidth
			if maxWidth == 0 {
				maxWidth = 1920
			}
			maxHeight := p.profile.ScreenHeight
			if maxHeight == 0 {
				maxHeight = 1080
			}

			fingerprint, _ = fpService.GenerateFingerprint(targetOS, maxWidth, maxHeight)
		}
	}

	opts := playwright.BrowserNewContextOptions{
		IgnoreHttpsErrors: playwright.Bool(true),
		JavaScriptEnabled: playwright.Bool(true),
	}

	// Apply BrowserForge fingerprint
	if p.cam != nil {
		// Use Camoufox fingerprint
		fp := p.cam.Fingerprint()
		if fp != nil {
			opts.UserAgent = playwright.String(fp.Navigator.UserAgent)
			if fp.Screen.Width > 0 && fp.Screen.Height > 0 {
				opts.Screen = &playwright.Size{
					Width:  fp.Screen.Width,
					Height: fp.Screen.Height,
				}
				opts.Viewport = &playwright.Size{
					Width:  fp.Screen.InnerWidth,
					Height: fp.Screen.InnerHeight,
				}
			}
			if fp.Navigator.Language != "" {
				opts.Locale = playwright.String(fp.Navigator.Language)
			}
		}
	} else if fingerprint != nil {
		// Use standard BrowserForge fingerprint
		opts.UserAgent = playwright.String(fingerprint.Navigator.UserAgent)

		if fingerprint.Screen.Width > 0 && fingerprint.Screen.Height > 0 {
			opts.Screen = &playwright.Size{
				Width:  fingerprint.Screen.Width,
				Height: fingerprint.Screen.Height,
			}
			opts.Viewport = &playwright.Size{
				Width:  fingerprint.Screen.InnerWidth,
				Height: fingerprint.Screen.InnerHeight,
			}
		}

		if fingerprint.Navigator.Language != "" {
			opts.Locale = playwright.String(fingerprint.Navigator.Language)
		}
	}

	// Apply user-configurable settings from profile (can override fingerprint)
	if p.profile != nil {
		// Timezone override
		if p.profile.Timezone != "" {
			opts.TimezoneId = playwright.String(p.profile.Timezone)
		}

		// Locale override
		if p.profile.Locale != "" {
			opts.Locale = playwright.String(p.profile.Locale)
		}

		// Geolocation
		if p.profile.GeolocationLatitude != nil && p.profile.GeolocationLongitude != nil {
			opts.Geolocation = &playwright.Geolocation{
				Latitude:  *p.profile.GeolocationLatitude,
				Longitude: *p.profile.GeolocationLongitude,
			}
			if p.profile.GeolocationAccuracy != nil {
				opts.Geolocation.Accuracy = playwright.Float(float64(*p.profile.GeolocationAccuracy))
			}
			opts.Permissions = []string{"geolocation"}
		}

		// Do Not Track
		if p.profile.DoNotTrack {
			opts.ExtraHttpHeaders = map[string]string{
				"DNT": "1",
			}
		}

		// Proxy from profile (if not overridden by proxy parameter)
		if proxy == nil && p.profile.ProxyEnabled && p.profile.ProxyServer != "" {
			proxyURL := p.profile.ProxyServer
			if p.profile.ProxyType != "" && p.profile.ProxyType != "http" {
				proxyURL = p.profile.ProxyType + "://" + proxyURL
			}
			opts.Proxy = &playwright.Proxy{
				Server: proxyURL,
			}
			if p.profile.ProxyUsername != "" {
				opts.Proxy.Username = playwright.String(p.profile.ProxyUsername)
				opts.Proxy.Password = playwright.String(p.profile.ProxyPassword)
			}
		}
	}

	// Add proxy if provided (overrides profile proxy)
	if proxy != nil && proxy.Server != "" {
		opts.Proxy = &playwright.Proxy{
			Server: proxy.Server,
		}
		if proxy.Username != "" {
			opts.Proxy.Username = playwright.String(proxy.Username)
			opts.Proxy.Password = playwright.String(proxy.Password)
		}
		logger.Debug("Creating context with proxy",
			zap.String("server", proxy.Server),
		)
	}

	ctx, err := p.browser.NewContext(opts)
	if err != nil {
		return nil, err
	}

	// Inject additional fingerprint properties via init script
	if fingerprint != nil {
		initScript := buildFingerprintInjectionScript(fingerprint)
		ctx.AddInitScript(playwright.Script{Content: playwright.String(initScript)})
	}

	return ctx, nil
}

// CreateContextWithProxy is a public method to create a context with proxy
// Use this when recovery needs a fresh context with a different proxy
func (p *Pool) CreateContextWithProxy(proxy *ProxyConfig) (playwright.BrowserContext, error) {
	p.mu.Lock()
	p.activeCount++
	p.mu.Unlock()

	ctx, err := p.createContext(proxy)
	if err != nil {
		p.mu.Lock()
		p.activeCount--
		p.mu.Unlock()
		return nil, err
	}

	logger.Debug("Created proxy context",
		zap.Int("active_contexts", p.activeCount),
	)

	return ctx, nil
}

// Acquire gets a browser context from the pool
func (p *Pool) Acquire(ctx context.Context) (playwright.BrowserContext, error) {
	// Acquire semaphore slot for max_concurrency limiting
	select {
	case p.semaphore <- struct{}{}:
		// Got semaphore slot, proceed
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for concurrency slot")
	}

	// Get context from pool
	select {
	case pooledCtx := <-p.contexts:
		p.mu.Lock()
		p.activeCount++
		p.mu.Unlock()

		// Check context_lifetime and recycle if expired
		if p.config.ContextLifetime > 0 {
			age := time.Since(pooledCtx.createdAt)
			lifetime := time.Duration(p.config.ContextLifetime) * time.Second
			if age > lifetime {
				logger.Debug("Context expired, recycling",
					zap.Duration("age", age),
					zap.Duration("lifetime", lifetime),
				)
				pooledCtx.ctx.Close()
				newCtx, err := p.createContextWithProfile(nil)
				if err != nil {
					p.mu.Lock()
					p.activeCount--
					p.mu.Unlock()
					<-p.semaphore // Release semaphore on error
					return nil, fmt.Errorf("failed to recreate expired context: %w", err)
				}
				pooledCtx.ctx = newCtx
				pooledCtx.createdAt = time.Now()
			}
		}

		logger.Debug("Context acquired",
			zap.Int("active_contexts", p.activeCount),
		)

		return pooledCtx.ctx, nil
	case <-ctx.Done():
		<-p.semaphore // Release semaphore on cancel
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		<-p.semaphore // Release semaphore on timeout
		return nil, fmt.Errorf("timeout waiting for browser context")
	}
}

// Release returns a browser context to the pool
func (p *Pool) Release(browserCtx playwright.BrowserContext) {
	// Release semaphore slot
	select {
	case <-p.semaphore:
		// Released semaphore slot
	default:
		// Semaphore was empty (shouldn't happen normally)
		logger.Warn("Release called but semaphore was empty")
	}

	// Clean up context (close all pages, clear cookies, etc.)
	pages := browserCtx.Pages()
	for _, page := range pages {
		page.Close()
	}

	// Clear storage
	browserCtx.ClearCookies()

	p.mu.Lock()
	p.activeCount--
	p.mu.Unlock()

	// Create pooled context wrapper
	pooledCtx := &PooledContext{
		ctx:       browserCtx,
		createdAt: time.Now(), // Reset creation time on release
	}

	// Return to pool
	select {
	case p.contexts <- pooledCtx:
		logger.Debug("Context released",
			zap.Int("active_contexts", p.activeCount),
		)
	default:
		// Pool is full, close this context and create a new one
		browserCtx.Close()
		newCtx, err := p.createContext(nil)
		if err != nil {
			logger.Error("Failed to recreate context", zap.Error(err))
			return
		}
		p.contexts <- &PooledContext{
			ctx:       newCtx,
			createdAt: time.Now(),
		}
	}
}

// Close closes the browser pool
func (p *Pool) Close() error {
	close(p.contexts)

	// Close all contexts
	for pooledCtx := range p.contexts {
		pooledCtx.ctx.Close()
	}

	// Close browser
	if p.cam != nil {
		if err := p.cam.Close(); err != nil {
			logger.Error("Failed to close camoufox", zap.Error(err))
		}
	} else if p.browser != nil {
		if err := p.browser.Close(); err != nil {
			logger.Error("Failed to close browser", zap.Error(err))
		}
	}

	// Stop Playwright
	if p.pw != nil {
		if err := p.pw.Stop(); err != nil {
			logger.Error("Failed to stop playwright", zap.Error(err))
		}
	}

	logger.Info("Browser pool closed")
	return nil
}

// Stats returns pool statistics
func (p *Pool) Stats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	return map[string]interface{}{
		"pool_size":          p.config.PoolSize,
		"active_contexts":    p.activeCount,
		"available_contexts": len(p.contexts),
	}
}

// buildFingerprintInjectionScript creates a JavaScript init script to inject fingerprint properties
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
