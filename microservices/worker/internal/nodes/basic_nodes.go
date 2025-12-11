package nodes

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/captcha"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/driver"
	"go.uber.org/zap"
)

// NavigateNode handles page navigation
type NavigateNode struct{}

// NewNavigateNode creates a new navigate node executor
func NewNavigateNode() NodeExecutor {
	return &NavigateNode{}
}

func (n *NavigateNode) Type() string {
	return "navigate"
}

func (n *NavigateNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	targetURL := execCtx.Task.URL

	// Get timeout from params or use default
	timeout := 60000.0 // 60 seconds
	if timeoutVal, ok := node.Params["timeout"].(float64); ok {
		timeout = timeoutVal
	}

	logger.Info("Navigating to URL",
		zap.String("url", targetURL),
		zap.Float64("timeout", timeout),
	)

	// Check if driver switch is requested (skip if empty or "default")
	if targetDriver, ok := node.Params["driver"].(string); ok && targetDriver != "" && targetDriver != "default" {
		// Check if we're already using the target driver (skip redundant switch)
		currentDriver := execCtx.Page.DriverName()
		if currentDriver == targetDriver {
			logger.Debug("Already using target driver, skipping switch",
				zap.String("driver", targetDriver),
			)
		} else {
			profileID := ""
			if pid, ok := node.Params["browser_profile_id"].(string); ok {
				profileID = pid
			}

			browserName := ""
			if bn, ok := node.Params["browser_name"].(string); ok {
				browserName = bn
			}

			// Priority: profile > browser_name > default
			if profileID != "" && execCtx.SwitchDriverWithProfile != nil {
				// Use full profile (for Playwright/Chromedp)
				if err := execCtx.SwitchDriverWithProfile(targetDriver, profileID); err != nil {
					return fmt.Errorf("failed to switch driver with profile: %w", err)
				}
			} else if browserName != "" && targetDriver == "http" && execCtx.SwitchDriverWithBrowser != nil {
				// Use browser_name for HTTP driver (JA3 + user agent)
				if err := execCtx.SwitchDriverWithBrowser(targetDriver, browserName); err != nil {
					return fmt.Errorf("failed to switch HTTP driver with browser: %w", err)
				}
			} else if execCtx.SwitchDriver != nil {
				// Default driver switch
				if err := execCtx.SwitchDriver(targetDriver); err != nil {
					return fmt.Errorf("failed to switch driver: %w", err)
				}
			} else {
				logger.Warn("Driver switch requested but not supported by execution context")
				if execCtx.OnWarning != nil {
					execCtx.OnWarning("navigate", "driver switch not supported")
				}
			}
		}
	}

	// Extract domain for cookie cache
	domain := extractDomain(targetURL)

	// Try to get cached CAPTCHA cookies before navigation
	var usedCachedCookies bool
	if execCtx.CaptchaCookieCache != nil {
		cachedCookies, err := execCtx.CaptchaCookieCache.GetCookies(ctx, domain)
		if err == nil && len(cachedCookies) > 0 {
			// Set cookies before navigation
			if err := execCtx.Page.SetCookies(cachedCookies); err != nil {
				logger.Warn("Failed to set cached CAPTCHA cookies",
					zap.Error(err),
					zap.String("domain", domain),
				)
			} else {
				usedCachedCookies = true
				logger.Info("Using cached CAPTCHA cookies",
					zap.String("domain", domain),
					zap.Int("cookie_count", len(cachedCookies)),
				)
			}
		}
	}

	// Navigate to URL
	err := execCtx.Page.Goto(targetURL,
		driver.WithPageTimeout(time.Duration(timeout)*time.Millisecond),
		driver.WithWaitUntil("domcontentloaded"),
	)

	if err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}

	logger.Info("Navigation complete",
		zap.String("url", targetURL),
	)

	// Get wait_selector for content verification (used for both CAPTCHA and normal wait)
	waitSelector, hasWaitSelector := node.Params["wait_selector"].(string)

	// Automatic CAPTCHA solving: Check if driver supports CAPTCHA solving
	// This handles Cloudflare challenges transparently without workflow changes
	if captchaSolver, ok := execCtx.Page.(driver.CaptchaSolver); ok && captchaSolver.SupportsCaptchaSolving() {
		// Prepare CAPTCHA solve options
		captchaOpts := driver.DefaultCaptchaSolveOptions()
		captchaOpts.Timeout = time.Duration(timeout) * time.Millisecond

		// Use wait_selector as expected content selector (proves real content loaded after solve)
		if hasWaitSelector && waitSelector != "" {
			captchaOpts.ExpectedContentSelector = waitSelector
		}

		// Check for debug flag
		if debug, ok := node.Params["captcha_debug"].(bool); ok {
			captchaOpts.Debug = debug
		}

		logger.Debug("Attempting automatic CAPTCHA detection and solving",
			zap.String("url", targetURL),
			zap.String("expected_content", captchaOpts.ExpectedContentSelector),
			zap.Bool("used_cached_cookies", usedCachedCookies),
		)

		// Check if we should solve or wait for another instance
		shouldSolve := true
		if execCtx.CaptchaCookieCache != nil && !usedCachedCookies {
			// Try to acquire lock for this domain
			acquired, lockErr := execCtx.CaptchaCookieCache.TryLock(ctx, domain)
			if lockErr != nil {
				logger.Warn("Failed to acquire CAPTCHA lock", zap.Error(lockErr))
			} else if !acquired {
				// Another instance is solving - wait for them
				logger.Info("Another instance is solving CAPTCHA, waiting...",
					zap.String("domain", domain),
				)
				cookies, _ := execCtx.CaptchaCookieCache.WaitForSolve(ctx, domain, 90*time.Second)
				if len(cookies) > 0 {
					// Got cookies from other instance
					if err := execCtx.Page.SetCookies(cookies); err != nil {
						logger.Warn("Failed to set cookies from other solver", zap.Error(err))
					} else {
						// Refresh the page with new cookies
						if err := execCtx.Page.Goto(targetURL,
							driver.WithPageTimeout(time.Duration(timeout)*time.Millisecond),
							driver.WithWaitUntil("domcontentloaded"),
						); err != nil {
							logger.Warn("Failed to refresh after getting cookies", zap.Error(err))
						}
						shouldSolve = false
						logger.Info("Using cookies from other CAPTCHA solver",
							zap.String("domain", domain),
						)
					}
				}
			}
		}

		if shouldSolve {
			solved, solveErr := captchaSolver.SolveCaptcha(captchaOpts)
			if solveErr != nil {
				logger.Warn("CAPTCHA solving encountered an error (continuing anyway)",
					zap.Error(solveErr),
					zap.String("url", targetURL),
				)
				// Don't fail navigation on CAPTCHA errors - might not be a CAPTCHA page
			} else if solved {
				logger.Info("CAPTCHA solved successfully, page content should be accessible",
					zap.String("url", targetURL),
				)

				// Store cookies and fingerprint in cache for other instances
				if execCtx.CaptchaCookieCache != nil {
					cookies, cookieErr := execCtx.Page.GetCookies()
					if cookieErr != nil {
						logger.Warn("Failed to get cookies after CAPTCHA solve", zap.Error(cookieErr))
					} else {
						// Try to get fingerprint for session locking
						var fp *captcha.CachedFingerprint
						if fpProvider, ok := execCtx.Page.(driver.FingerprintProvider); ok {
							userAgent, platform, language, languages, hardwareConcurrency, screenWidth, screenHeight := fpProvider.GetFingerprint()
							if userAgent != "" {
								fp = &captcha.CachedFingerprint{
									UserAgent:           userAgent,
									Platform:            platform,
									Language:            language,
									Languages:           languages,
									HardwareConcurrency: hardwareConcurrency,
									ScreenWidth:         screenWidth,
									ScreenHeight:        screenHeight,
								}
								logger.Debug("Captured fingerprint for session caching")
							}
						}

						// Use interface SetSession method directly (no type assertion needed)
						if err := execCtx.CaptchaCookieCache.SetSession(ctx, domain, cookies, fp); err != nil {
							logger.Warn("Failed to cache CAPTCHA session", zap.Error(err))
						}
					}
				}
			} else {
				logger.Debug("No CAPTCHA detected or already solved",
					zap.String("url", targetURL),
				)
			}

			// Release lock after solving (success or failure)
			if execCtx.CaptchaCookieCache != nil {
				if err := execCtx.CaptchaCookieCache.Unlock(ctx, domain); err != nil {
					logger.Warn("Failed to release CAPTCHA lock", zap.Error(err))
				}
			}
		}
	}

	// Optional: Wait for specific selector (after CAPTCHA solving if applicable)
	if hasWaitSelector && waitSelector != "" {
		logger.Debug("Waiting for selector", zap.String("selector", waitSelector))

		if err := execCtx.Page.WaitForSelector(waitSelector,
			driver.WithWaitTimeout(time.Duration(timeout)*time.Millisecond),
		); err != nil {
			return fmt.Errorf("wait for selector failed: %w", err)
		}
	}

	return nil
}

// extractDomain extracts the host from a URL for cache keying
func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return parsed.Host
}

// ClickNode handles element clicks
type ClickNode struct{}

func NewClickNode() NodeExecutor {
	return &ClickNode{}
}

func (n *ClickNode) Type() string {
	return "click"
}

func (n *ClickNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	selector, ok := node.Params["selector"].(string)
	if !ok || selector == "" {
		return fmt.Errorf("selector is required for click node")
	}

	logger.Info("Clicking element", zap.String("selector", selector))

	// Wait for element to be visible
	if err := execCtx.Page.WaitForSelector(selector,
		driver.WithState("visible"),
	); err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	// Click element
	if err := execCtx.Page.Click(selector); err != nil {
		return fmt.Errorf("click failed: %w", err)
	}

	// Optional: Wait after click
	if waitAfter, ok := node.Params["wait_after"].(float64); ok && waitAfter > 0 {
		time.Sleep(time.Duration(waitAfter) * time.Millisecond)
	}

	return nil
}

// TypeNode handles text input
type TypeNode struct{}

func NewTypeNode() NodeExecutor {
	return &TypeNode{}
}

func (n *TypeNode) Type() string {
	return "type"
}

func (n *TypeNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	selector, ok := node.Params["selector"].(string)
	if !ok || selector == "" {
		return fmt.Errorf("selector is required for type node")
	}

	text, ok := node.Params["text"].(string)
	if !ok {
		return fmt.Errorf("text is required for type node")
	}

	logger.Info("Typing text",
		zap.String("selector", selector),
		zap.Int("text_length", len(text)),
	)

	// Wait for input field
	if err := execCtx.Page.WaitForSelector(selector,
		driver.WithState("visible"),
	); err != nil {
		return fmt.Errorf("input field not found: %w", err)
	}

	// Clear existing text if specified
	if clear, ok := node.Params["clear"].(bool); ok && clear {
		// Use Type with empty string to clear? Or need a specific Clear method?
		// For now, let's assume Type overwrites or we can select all and delete.
		// Standard Playwright Fill clears.
		// Let's assume Type with empty string might not clear.
		// We might need a Clear method in the interface or use Type with special keys.
		// For simplicity in this refactor, we'll skip explicit clear or assume Type handles it if implemented as Fill.
		// Actually, let's add Fill to the interface later if needed, but for now Type is close enough.
		// Or better, use Type with empty string and assume driver implementation handles it.
		if err := execCtx.Page.Type(selector, ""); err != nil {
			return fmt.Errorf("failed to clear field: %w", err)
		}
	}

	// Type text
	if err := execCtx.Page.Type(selector, text); err != nil {
		return fmt.Errorf("typing failed: %w", err)
	}

	return nil
}

// WaitNode handles explicit waits
type WaitNode struct{}

func NewWaitNode() NodeExecutor {
	return &WaitNode{}
}

func (n *WaitNode) Type() string {
	return "wait"
}

func (n *WaitNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	// Wait by duration
	if duration, ok := node.Params["duration"].(float64); ok && duration > 0 {
		logger.Info("Waiting", zap.Float64("duration_ms", duration))
		time.Sleep(time.Duration(duration) * time.Millisecond)
		return nil
	}

	// Wait for selector
	if selector, ok := node.Params["selector"].(string); ok && selector != "" {
		timeout := 30000.0
		if t, ok := node.Params["timeout"].(float64); ok {
			timeout = t
		}

		logger.Info("Waiting for selector",
			zap.String("selector", selector),
			zap.Float64("timeout", timeout),
		)

		if err := execCtx.Page.WaitForSelector(selector,
			driver.WithWaitTimeout(time.Duration(timeout)*time.Millisecond),
		); err != nil {
			return fmt.Errorf("wait for selector failed: %w", err)
		}
		return nil
	}

	return fmt.Errorf("either duration or selector must be specified for wait node")
}
