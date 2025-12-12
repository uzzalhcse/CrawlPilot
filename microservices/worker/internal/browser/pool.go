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

// SessionChecker checks if a domain needs session reuse (anti-bot sessions)
// This interface allows dependency injection of SmartUnblocker without circular imports
type SessionChecker interface {
	NeedsSessionReuse(ctx context.Context, domain string) bool
}

// PooledContext wraps a browser context with metadata for lifecycle management
type PooledContext struct {
	ctx         playwright.BrowserContext
	createdAt   time.Time
	Fingerprint *services.Fingerprint // BrowserForge fingerprint used for this context

	// Session tracking: which domains have anti-bot cookies in this context
	cookieDomains map[string]bool // domain -> has anti-bot cookies
	domainMu      sync.RWMutex
}

// HasCookiesFor checks if this context has anti-bot cookies for a domain
func (pc *PooledContext) HasCookiesFor(domain string) bool {
	pc.domainMu.RLock()
	defer pc.domainMu.RUnlock()
	return pc.cookieDomains[domain]
}

// MarkCookiesFor marks that this context has anti-bot cookies for a domain
func (pc *PooledContext) MarkCookiesFor(domain string) {
	pc.domainMu.Lock()
	defer pc.domainMu.Unlock()
	if pc.cookieDomains == nil {
		pc.cookieDomains = make(map[string]bool)
	}
	pc.cookieDomains[domain] = true
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

	// Session-aware context selection
	sessionChecker SessionChecker            // Optional: checks if domain needs session reuse
	domainContexts map[string]*PooledContext // domain -> context with anti-bot cookies (for reuse)
	domainMu       sync.RWMutex
}

// SetSessionChecker sets the session checker for session-aware context selection
// This should be called after pool creation to wire up SmartUnblocker
func (p *Pool) SetSessionChecker(checker SessionChecker) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sessionChecker = checker
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
		// Use shared browser factory for consistent Camoufox configuration
		opts, err := services.BuildCamoufoxOptions(profile, cfg.Headless)
		if err != nil {
			return nil, fmt.Errorf("failed to build camoufox options: %w", err)
		}

		// Launch Camoufox
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

		// Use shared browser factory for browser type and launch options
		browserType = services.GetPlaywrightBrowserType(profile)
		launchOpts := services.BuildPlaywrightLaunchOptions(profile, cfg.Headless)

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
		ctx, fp, err := pool.createContextWithProfile(nil)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to create initial context: %w", err)
		}
		pool.contexts <- &PooledContext{
			ctx:         ctx,
			createdAt:   time.Now(),
			Fingerprint: fp,
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
func (p *Pool) createContext(proxy *ProxyConfig) (playwright.BrowserContext, *services.Fingerprint, error) {
	return p.createContextWithProfile(proxy)
}

// createContextWithProfile creates a new browser context with profile fingerprint settings
// Returns the context and the fingerprint used for session caching
func (p *Pool) createContextWithProfile(proxy *ProxyConfig) (playwright.BrowserContext, *services.Fingerprint, error) {
	var fingerprint *services.Fingerprint

	// Use shared browser factory for context options (includes fingerprints and proxy validation)
	fpService, _ := services.GetFingerprintService()

	opts, err := services.BuildPlaywrightContextOptions(p.profile, fpService)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build context options: %w", err)
	}

	// Get fingerprint for init script injection (if using standard Playwright, not Camoufox)
	if p.cam == nil && fpService != nil && p.profile != nil {
		browserType := services.GetPlaywrightBrowserType(p.profile)
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
		fingerprint, _ = fpService.GenerateFingerprintWithBrowser(targetOS, browserType, maxWidth, maxHeight)
	}

	// Apply Camoufox fingerprint (overrides factory opts for Camoufox)
	if p.cam != nil {
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
	}

	// Do Not Track (from profile, not in shared factory)
	if p.profile != nil && p.profile.DoNotTrack {
		opts.ExtraHttpHeaders = map[string]string{
			"DNT": "1",
		}
	}

	// Geolocation accuracy (from profile, not in shared factory)
	if p.profile != nil && opts.Geolocation != nil && p.profile.GeolocationAccuracy != nil {
		opts.Geolocation.Accuracy = playwright.Float(float64(*p.profile.GeolocationAccuracy))
	}

	// Add proxy if provided (overrides profile proxy)
	if proxy != nil && proxy.Server != "" {
		proxyURL, _ := services.EnsureScheme(proxy.Server)
		opts.Proxy = &playwright.Proxy{
			Server: proxyURL,
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
		return nil, nil, err
	}

	// Inject additional fingerprint properties via init script
	if fingerprint != nil {
		initScript := buildFingerprintInjectionScript(fingerprint)
		ctx.AddInitScript(playwright.Script{Content: playwright.String(initScript)})
	}

	return ctx, fingerprint, nil
}

// CreateContextWithProxy is a public method to create a context with proxy
// Use this when recovery needs a fresh context with a different proxy
func (p *Pool) CreateContextWithProxy(proxy *ProxyConfig) (playwright.BrowserContext, error) {
	ctx, _, err := p.CreateContextWithProxyAndFingerprint(proxy)
	return ctx, err
}

// CreateContextWithProxyAndFingerprint creates a context with proxy and returns the fingerprint
// Use this for session caching with proxied requests
func (p *Pool) CreateContextWithProxyAndFingerprint(proxy *ProxyConfig) (playwright.BrowserContext, *services.Fingerprint, error) {
	p.mu.Lock()
	p.activeCount++
	p.mu.Unlock()

	ctx, fp, err := p.createContext(proxy)
	if err != nil {
		p.mu.Lock()
		p.activeCount--
		p.mu.Unlock()
		return nil, nil, err
	}

	logger.Debug("Created proxy context with fingerprint",
		zap.Int("active_contexts", p.activeCount),
	)

	return ctx, fp, nil
}

// Acquire gets a browser context from the pool
func (p *Pool) Acquire(ctx context.Context) (playwright.BrowserContext, error) {
	browserCtx, _, err := p.AcquireWithFingerprint(ctx)
	return browserCtx, err
}

// AcquireWithFingerprint gets a browser context and its fingerprint from the pool
// Use this for session caching to get the BrowserForge fingerprint used for this context
func (p *Pool) AcquireWithFingerprint(ctx context.Context) (playwright.BrowserContext, *services.Fingerprint, error) {
	// Acquire semaphore slot for max_concurrency limiting
	select {
	case p.semaphore <- struct{}{}:
		// Got semaphore slot, proceed
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, nil, fmt.Errorf("timeout waiting for concurrency slot")
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
				newCtx, newFP, err := p.createContextWithProfile(nil)
				if err != nil {
					p.mu.Lock()
					p.activeCount--
					p.mu.Unlock()
					<-p.semaphore // Release semaphore on error
					return nil, nil, fmt.Errorf("failed to recreate expired context: %w", err)
				}
				pooledCtx.ctx = newCtx
				pooledCtx.Fingerprint = newFP
				pooledCtx.createdAt = time.Now()
			}
		}

		logger.Debug("Context acquired",
			zap.Int("active_contexts", p.activeCount),
		)

		return pooledCtx.ctx, pooledCtx.Fingerprint, nil
	case <-ctx.Done():
		<-p.semaphore // Release semaphore on cancel
		return nil, nil, ctx.Err()
	case <-time.After(30 * time.Second):
		<-p.semaphore // Release semaphore on timeout
		return nil, nil, fmt.Errorf("timeout waiting for browser context")
	}
}

// AcquireForDomain acquires a context suitable for the given domain
// If the domain needs session reuse (detected anti-bot cookies), it will try to reuse a context that has cookies for that domain
// Returns the context, fingerprint, and pooledContext for domain registration later
func (p *Pool) AcquireForDomain(ctx context.Context, domain string) (playwright.BrowserContext, *services.Fingerprint, *PooledContext, error) {
	// Check if domain needs session reuse
	needsSession := false
	if p.sessionChecker != nil {
		needsSession = p.sessionChecker.NeedsSessionReuse(ctx, domain)
	}

	// If session needed, try to find existing context with cookies for this domain
	if needsSession {
		p.domainMu.RLock()
		existingCtx := p.domainContexts[domain]
		p.domainMu.RUnlock()

		if existingCtx != nil && existingCtx.HasCookiesFor(domain) {
			// Acquire semaphore for this context
			select {
			case p.semaphore <- struct{}{}:
				p.mu.Lock()
				p.activeCount++
				p.mu.Unlock()

				logger.Info("Reusing session context for anti-bot domain",
					zap.String("domain", domain),
					zap.Int("active_contexts", p.activeCount),
				)

				return existingCtx.ctx, existingCtx.Fingerprint, existingCtx, nil
			case <-ctx.Done():
				return nil, nil, nil, ctx.Err()
			case <-time.After(5 * time.Second):
				// Fallback to regular acquire if can't get semaphore
				logger.Debug("Could not acquire session context, falling back to regular")
			}
		}
	}

	// Regular acquire
	browserCtx, fp, err := p.AcquireWithFingerprint(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	// Wrap in PooledContext for tracking
	pooledCtx := &PooledContext{
		ctx:         browserCtx,
		createdAt:   time.Now(),
		Fingerprint: fp,
	}

	return browserCtx, fp, pooledCtx, nil
}

// RegisterDomainContext registers a context as having anti-bot cookies for a domain
// Call this after CAPTCHA is solved and cookies are present
func (p *Pool) RegisterDomainContext(domain string, pooledCtx *PooledContext) {
	if pooledCtx == nil {
		return
	}

	pooledCtx.MarkCookiesFor(domain)

	p.domainMu.Lock()
	if p.domainContexts == nil {
		p.domainContexts = make(map[string]*PooledContext)
	}
	p.domainContexts[domain] = pooledCtx
	p.domainMu.Unlock()

	logger.Debug("Registered domain context for session reuse",
		zap.String("domain", domain),
	)
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
		newCtx, newFP, err := p.createContext(nil)
		if err != nil {
			logger.Error("Failed to recreate context", zap.Error(err))
			return
		}
		p.contexts <- &PooledContext{
			ctx:         newCtx,
			createdAt:   time.Now(),
			Fingerprint: newFP,
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
