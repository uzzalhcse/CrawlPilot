package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/internal/config"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"go.uber.org/zap"
)

type BrowserPool struct {
	config      *config.BrowserConfig
	playwright  *playwright.Playwright
	browser     playwright.Browser
	contexts    chan playwright.BrowserContext
	mu          sync.RWMutex
	closed      bool
	contextPool sync.Pool
	launcher    *BrowserLauncher
	profileRepo *storage.BrowserProfileRepository
}

type BrowserContext struct {
	Context       playwright.BrowserContext
	Page          playwright.Page
	pool          *BrowserPool
	headedBrowser playwright.Browser // For headed sessions
	isHeaded      bool
}

func NewBrowserPool(cfg *config.BrowserConfig, profileRepo *storage.BrowserProfileRepository) (*BrowserPool, error) {
	// Install Playwright browsers if needed
	err := playwright.Install(&playwright.RunOptions{
		Verbose: false,
	})
	if err != nil {
		logger.Warn("Failed to install playwright browsers", zap.Error(err))
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to start playwright: %w", err)
	}

	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(cfg.Headless),
		Timeout:  playwright.Float(float64(cfg.Timeout)),
	}

	browser, err := pw.Chromium.Launch(launchOptions)
	if err != nil {
		pw.Stop()
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	pool := &BrowserPool{
		config:      cfg,
		playwright:  pw,
		browser:     browser,
		contexts:    make(chan playwright.BrowserContext, cfg.PoolSize),
		closed:      false,
		launcher:    NewBrowserLauncher(pw),
		profileRepo: profileRepo,
	}

	// Pre-create browser contexts
	for i := 0; i < cfg.PoolSize; i++ {
		ctx, err := pool.createContext()
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to create browser context: %w", err)
		}
		pool.contexts <- ctx
	}

	logger.Info("Browser pool initialized",
		zap.Int("pool_size", cfg.PoolSize),
		zap.Bool("headless", cfg.Headless),
	)

	return pool, nil
}

func (p *BrowserPool) createContext() (playwright.BrowserContext, error) {
	options := playwright.BrowserNewContextOptions{
		UserAgent:         playwright.String("Crawlify/1.0"),
		AcceptDownloads:   playwright.Bool(false),
		IgnoreHttpsErrors: playwright.Bool(true),
		JavaScriptEnabled: playwright.Bool(true),
		Viewport: &playwright.Size{
			Width:  1920,
			Height: 1080,
		},
	}

	// Configure proxy if enabled
	if p.config.Proxy.Enabled && p.config.Proxy.Server != "" {
		proxyConfig := &playwright.Proxy{
			Server: p.config.Proxy.Server,
		}

		// Add authentication if provided
		if p.config.Proxy.Username != "" && p.config.Proxy.Password != "" {
			proxyConfig.Username = playwright.String(p.config.Proxy.Username)
			proxyConfig.Password = playwright.String(p.config.Proxy.Password)
		}

		options.Proxy = proxyConfig

		logger.Info("Using proxy configuration",
			zap.String("server", p.config.Proxy.Server),
			zap.String("username", p.config.Proxy.Username),
		)
	}

	return p.browser.NewContext(options)
}

// Acquire gets a browser context from the pool
func (p *BrowserPool) Acquire(ctx context.Context, headed ...bool) (*BrowserContext, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("browser pool is closed")
	}
	p.mu.RUnlock()

	// If headed mode is requested, create a new browser instance
	isHeaded := len(headed) > 0 && headed[0]
	if isHeaded {
		return p.createHeadedContext(ctx)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case browserCtx := <-p.contexts:
		// Create a new page in the context
		page, err := browserCtx.NewPage()
		if err != nil {
			// Context is invalid, create a new one
			browserCtx.Close()
			newCtx, err := p.createContext()
			if err != nil {
				return nil, fmt.Errorf("failed to create new context: %w", err)
			}
			page, err = newCtx.NewPage()
			if err != nil {
				return nil, fmt.Errorf("failed to create new page: %w", err)
			}
			browserCtx = newCtx
		}

		return &BrowserContext{
			Context: browserCtx,
			Page:    page,
			pool:    p,
		}, nil
	}
}

// AcquireWithProfile acquires a browser context using a specific browser profile
func (p *BrowserPool) AcquireWithProfile(ctx context.Context, profileID string) (*BrowserContext, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("browser pool is closed")
	}
	p.mu.RUnlock()

	// Check if profileRepo is available
	if p.profileRepo == nil {
		return nil, fmt.Errorf("profile repository not available")
	}

	// Get the profile from database
	profile, err := p.profileRepo.GetByID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get browser profile: %w", err)
	}

	// Launch browser with profile configuration
	launchOpts := &LaunchOptions{
		Headless: p.config.Headless, // Use pool's headless setting
		Timeout:  float64(p.config.Timeout),
		Profile:  profile,
	}

	browser, err := p.launcher.LaunchBrowserWithProfile(ctx, launchOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser with profile: %w", err)
	}

	// Create context with fingerprinting
	browserCtx, err := p.launcher.CreateContextWithFingerprint(browser, profile)
	if err != nil {
		browser.Close()
		return nil, fmt.Errorf("failed to create context with fingerprint: %w", err)
	}

	// Create page
	page, err := browserCtx.NewPage()
	if err != nil {
		browserCtx.Close()
		browser.Close()
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	logger.Info("Acquired browser context with profile",
		zap.String("profile_id", profileID),
		zap.String("browser_type", profile.BrowserType))

	return &BrowserContext{
		Context:       browserCtx,
		Page:          page,
		pool:          p,
		headedBrowser: browser, // Store browser instance to close later
		isHeaded:      !p.config.Headless,
	}, nil
}

// createHeadedContext creates a new headed browser context (not from pool)
func (p *BrowserPool) createHeadedContext(ctx context.Context) (*BrowserContext, error) {
	// Launch a new headed browser instance
	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
		Timeout:  playwright.Float(float64(p.config.Timeout)),
	}

	headedBrowser, err := p.playwright.Chromium.Launch(launchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to launch headed browser: %w", err)
	}

	options := playwright.BrowserNewContextOptions{
		UserAgent:         playwright.String("Crawlify/1.0 ElementSelector"),
		AcceptDownloads:   playwright.Bool(false),
		IgnoreHttpsErrors: playwright.Bool(true),
		JavaScriptEnabled: playwright.Bool(true),
		Viewport: &playwright.Size{
			Width:  1920,
			Height: 1080,
		},
	}

	browserCtx, err := headedBrowser.NewContext(options)
	if err != nil {
		headedBrowser.Close()
		return nil, fmt.Errorf("failed to create headed context: %w", err)
	}

	page, err := browserCtx.NewPage()
	if err != nil {
		browserCtx.Close()
		headedBrowser.Close()
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	return &BrowserContext{
		Context:       browserCtx,
		Page:          page,
		pool:          p,
		headedBrowser: headedBrowser,
		isHeaded:      true,
	}, nil
}

// Release returns a browser context to the pool
func (p *BrowserPool) Release(bc *BrowserContext) {
	if bc == nil || bc.Context == nil {
		return
	}

	// If this is a headed session, close everything
	if bc.isHeaded {
		if bc.Page != nil {
			bc.Page.Close()
		}
		if bc.Context != nil {
			bc.Context.Close()
		}
		if bc.headedBrowser != nil {
			bc.headedBrowser.Close()
		}
		return
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		bc.Context.Close()
		return
	}

	// Close the page but keep the context
	if bc.Page != nil {
		bc.Page.Close()
	}

	// Clear cookies and storage
	bc.Context.ClearCookies()

	select {
	case p.contexts <- bc.Context:
		// Successfully returned to pool
	default:
		// Pool is full, close the context
		bc.Context.Close()
	}
}

// Close closes all browser contexts and the browser
func (p *BrowserPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	close(p.contexts)

	// Close all contexts in the pool
	for ctx := range p.contexts {
		ctx.Close()
	}

	if p.browser != nil {
		p.browser.Close()
	}

	if p.playwright != nil {
		p.playwright.Stop()
	}

	logger.Info("Browser pool closed")
}

// NewPage creates a new page with the given options
func (bc *BrowserContext) NewPage() (playwright.Page, error) {
	return bc.Context.NewPage()
}

// SetCookies sets cookies for the browser context
func (bc *BrowserContext) SetCookies(cookies []playwright.OptionalCookie) error {
	return bc.Context.AddCookies(cookies)
}

// SetHeaders sets extra HTTP headers for all requests
func (bc *BrowserContext) SetHeaders(headers map[string]string) error {
	return bc.Context.SetExtraHTTPHeaders(headers)
}

// Navigate navigates to a URL
func (bc *BrowserContext) Navigate(url string, options ...playwright.PageGotoOptions) (playwright.Response, error) {
	var opts playwright.PageGotoOptions
	if len(options) > 0 {
		opts = options[0]
	} else {
		opts = playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			Timeout:   playwright.Float(30000 * 2),
		}
	}

	return bc.Page.Goto(url, opts)
}

// WaitForSelector waits for a selector to appear
func (bc *BrowserContext) WaitForSelector(selector string, timeout time.Duration) error {
	_, err := bc.Page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(float64(timeout.Milliseconds())),
		State:   playwright.WaitForSelectorStateVisible,
	})
	return err
}

// Content returns the page HTML content
func (bc *BrowserContext) Content() (string, error) {
	return bc.Page.Content()
}

// Screenshot takes a screenshot of the page
func (bc *BrowserContext) Screenshot(options playwright.PageScreenshotOptions) ([]byte, error) {
	return bc.Page.Screenshot(options)
}

// GetLauncher returns the browser launcher instance
func (p *BrowserPool) GetLauncher() *BrowserLauncher {
	return p.launcher
}
