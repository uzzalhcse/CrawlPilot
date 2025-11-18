package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/internal/config"
	"github.com/uzzalhcse/crawlify/internal/logger"
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
}

type BrowserContext struct {
	Context playwright.BrowserContext
	Page    playwright.Page
	pool    *BrowserPool
}

func NewBrowserPool(cfg *config.BrowserConfig) (*BrowserPool, error) {
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
		config:     cfg,
		playwright: pw,
		browser:    browser,
		contexts:   make(chan playwright.BrowserContext, cfg.PoolSize),
		closed:     false,
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
	return p.browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent:         playwright.String("Crawlify/1.0"),
		AcceptDownloads:   playwright.Bool(false),
		IgnoreHttpsErrors: playwright.Bool(true),
		JavaScriptEnabled: playwright.Bool(true),
		Viewport: &playwright.Size{
			Width:  1920,
			Height: 1080,
		},
	})
}

// Acquire gets a browser context from the pool
func (p *BrowserPool) Acquire(ctx context.Context) (*BrowserContext, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("browser pool is closed")
	}
	p.mu.RUnlock()

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

// Release returns a browser context to the pool
func (p *BrowserPool) Release(bc *BrowserContext) {
	if bc == nil || bc.Context == nil {
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
