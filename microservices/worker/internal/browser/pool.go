package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
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
	// Initialize Playwright
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to start playwright: %w", err)
	}

	// Determine browser type from profile or default to Chromium
	browserType := "chromium"
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

	// Launch browser based on type
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
		return nil, fmt.Errorf("failed to launch %s browser: %w", browserType, err)
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
	opts := playwright.BrowserNewContextOptions{
		IgnoreHttpsErrors: playwright.Bool(true),
		JavaScriptEnabled: playwright.Bool(true),
	}

	// Apply profile fingerprint settings if available
	if p.profile != nil {
		// User agent
		if p.profile.UserAgent != "" {
			opts.UserAgent = playwright.String(p.profile.UserAgent)
		} else {
			opts.UserAgent = playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		}

		// Viewport size - set via Screen property in playwright-go
		if p.profile.ScreenWidth > 0 && p.profile.ScreenHeight > 0 {
			opts.Screen = &playwright.Size{
				Width:  p.profile.ScreenWidth,
				Height: p.profile.ScreenHeight,
			}
		}

		// Locale
		if p.profile.Locale != "" {
			opts.Locale = playwright.String(p.profile.Locale)
		}

		// Timezone
		if p.profile.Timezone != "" {
			opts.TimezoneId = playwright.String(p.profile.Timezone)
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
	} else {
		opts.UserAgent = playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
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
	if p.browser != nil {
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
