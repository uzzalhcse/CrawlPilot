package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// Pool manages a pool of browser contexts
type Pool struct {
	browser     playwright.Browser
	contexts    chan playwright.BrowserContext
	config      *config.BrowserConfig
	mu          sync.Mutex
	activeCount int
	pw          *playwright.Playwright
}

// NewPool creates a new browser pool
func NewPool(cfg *config.BrowserConfig) (*Pool, error) {
	// Initialize Playwright
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to start playwright: %w", err)
	}

	// Launch browser
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(cfg.Headless), // Use config setting
		Args: []string{
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-dev-shm-usage",
			"--disable-gpu",
			// Note: --single-process removed - it breaks multiple contexts
			// For Cloud Run with high memory (4GB+), leave default multi-process
		},
	})
	if err != nil {
		pw.Stop()
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	pool := &Pool{
		browser:  browser,
		contexts: make(chan playwright.BrowserContext, cfg.PoolSize),
		config:   cfg,
		pw:       pw,
	}

	// Pre-create contexts
	for i := 0; i < cfg.PoolSize; i++ {
		ctx, err := pool.createContext()
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to create initial context: %w", err)
		}
		pool.contexts <- ctx
	}

	logger.Info("Browser pool initialized",
		zap.Int("pool_size", cfg.PoolSize),
		zap.Bool("headless", cfg.Headless),
	)

	return pool, nil
}

// createContext creates a new browser context
func (p *Pool) createContext() (playwright.BrowserContext, error) {
	ctx, err := p.browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
		// ViewportSize removed - not needed, use default
		IgnoreHttpsErrors: playwright.Bool(true),
		JavaScriptEnabled: playwright.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	return ctx, nil
}

// Acquire gets a browser context from the pool
func (p *Pool) Acquire(ctx context.Context) (playwright.BrowserContext, error) {
	select {
	case browserCtx := <-p.contexts:
		p.mu.Lock()
		p.activeCount++
		p.mu.Unlock()

		logger.Debug("Context acquired",
			zap.Int("active_contexts", p.activeCount),
		)

		return browserCtx, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for browser context")
	}
}

// Release returns a browser context to the pool
func (p *Pool) Release(browserCtx playwright.BrowserContext) {
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

	// Return to pool
	select {
	case p.contexts <- browserCtx:
		logger.Debug("Context released",
			zap.Int("active_contexts", p.activeCount),
		)
	default:
		// Pool is full, close this context and create a new one
		browserCtx.Close()
		newCtx, err := p.createContext()
		if err != nil {
			logger.Error("Failed to recreate context", zap.Error(err))
			return
		}
		p.contexts <- newCtx
	}
}

// Close closes the browser pool
func (p *Pool) Close() error {
	close(p.contexts)

	// Close all contexts
	for ctx := range p.contexts {
		ctx.Close()
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
