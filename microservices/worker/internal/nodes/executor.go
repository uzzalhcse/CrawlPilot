package nodes

import (
	"context"
	"net/http"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/captcha"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/driver"
)

// ExecutionContext holds the current execution state
type ExecutionContext struct {
	Page                    driver.Page
	Task                    *models.Task
	Variables               map[string]interface{}
	ExtractedItems          []map[string]interface{}                   // Items extracted during execution
	DiscoveredURLs          []string                                   // URLs discovered during execution
	MissingRequiredFields   []string                                   // Required fields that failed to extract (for probe detection)
	BranchNodes             []models.Node                              // Nodes to execute from conditional branches
	SwitchDriver            func(string) error                         // Callback to switch driver (legacy)
	SwitchDriverWithProfile func(driverType, profileID string) error   // Switch driver with optional profile
	SwitchDriverWithBrowser func(driverType, browserName string) error // Switch HTTP driver with browser name for JA3
	OnWarning               func(field, message string)                // Callback for logging warnings
	CaptchaCookieCache      CaptchaCookieCacheInterface                // Cookie cache for CAPTCHA bypass sharing

	// Session detection callback for SmartUnblocker integration
	// Called when anti-bot cookies (Cloudflare, Datadome, etc.) are detected after navigation
	OnAntiBotCookiesDetected func(domain string, cookieNames []string)
}

// CaptchaCookieCacheInterface defines the interface for CAPTCHA cookie/session caching
// Enhanced with GetSession/SetSession for performance at scale (avoids double Redis calls)
type CaptchaCookieCacheInterface interface {
	// Session-based methods (optimized - includes fingerprint)
	GetSession(ctx context.Context, domain string) (*captcha.CachedSession, error)
	SetSession(ctx context.Context, domain string, cookies []*http.Cookie, fingerprint *captcha.CachedFingerprint) error

	// Cookie-only methods (backward compatible)
	GetCookies(ctx context.Context, domain string) ([]*http.Cookie, error)
	SetCookies(ctx context.Context, domain string, cookies []*http.Cookie) error

	// Distributed locking for CAPTCHA solving
	TryLock(ctx context.Context, domain string) (bool, error)
	Unlock(ctx context.Context, domain string) error
	WaitForSolve(ctx context.Context, domain string, maxWait time.Duration) ([]*http.Cookie, error)
}

// NodeExecutor defines the interface for node execution
type NodeExecutor interface {
	// Execute runs the node logic
	Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error

	// Type returns the node type this executor handles
	Type() string
}

// Result holds the result of a node execution
type Result struct {
	Success bool
	Data    map[string]interface{}
	Error   error
}
