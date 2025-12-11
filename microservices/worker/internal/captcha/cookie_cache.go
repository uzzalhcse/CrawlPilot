package captcha

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

const (
	// SessionCacheKeyPrefix is the Redis key prefix for CAPTCHA sessions (cookies + fingerprint)
	SessionCacheKeyPrefix = "captcha:session:"

	// DefaultSessionTTL is the default TTL for cached sessions (30 minutes)
	// Cloudflare cf_clearance typically expires in 30 mins to 2 hours
	DefaultSessionTTL = 30 * time.Minute

	// LockKeyPrefix is the prefix for domain locks during CAPTCHA solving
	LockKeyPrefix = "captcha:lock:"

	// LockTTL is how long a domain lock is held during CAPTCHA solving
	// Reduced from 2 min to 30 sec for faster recovery at high volume
	LockTTL = 30 * time.Second
)

// CookieCache manages domain-scoped CAPTCHA sessions (cookies + fingerprint) for sharing across browser instances
type CookieCache struct {
	cache *cache.Cache
	ttl   time.Duration
}

// NewCookieCache creates a new CAPTCHA cookie cache
func NewCookieCache(redisCache *cache.Cache) *CookieCache {
	return &CookieCache{
		cache: redisCache,
		ttl:   DefaultSessionTTL,
	}
}

// CachedSession represents a stored CAPTCHA session (cookies + fingerprint) for a domain
type CachedSession struct {
	Domain      string              `json:"domain"`
	Cookies     []*SerializedCookie `json:"cookies"`
	Fingerprint *CachedFingerprint  `json:"fingerprint,omitempty"` // Locked fingerprint for session reuse
	SolvedAt    time.Time           `json:"solved_at"`
	ExpiresAt   time.Time           `json:"expires_at"`
}

// CachedFingerprint contains fingerprint data for session locking
// This is a subset of camoufox.Fingerprint that can be serialized to Redis
type CachedFingerprint struct {
	// Navigator properties
	UserAgent           string   `json:"user_agent"`
	Platform            string   `json:"platform,omitempty"`
	Language            string   `json:"language,omitempty"`
	Languages           []string `json:"languages,omitempty"`
	HardwareConcurrency int      `json:"hardware_concurrency,omitempty"`

	// Screen properties
	ScreenWidth  int `json:"screen_width,omitempty"`
	ScreenHeight int `json:"screen_height,omitempty"`
}

// SerializedCookie is a JSON-serializable version of http.Cookie
type SerializedCookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain"`
	Path     string    `json:"path"`
	Expires  time.Time `json:"expires"`
	Secure   bool      `json:"secure"`
	HttpOnly bool      `json:"http_only"`
	SameSite int       `json:"same_site"`
}

// GetSession retrieves cached session (cookies + fingerprint) for a domain
func (c *CookieCache) GetSession(ctx context.Context, domain string) (*CachedSession, error) {
	if c.cache == nil {
		return nil, nil // No cache available
	}

	key := SessionCacheKeyPrefix + domain
	var cached CachedSession
	err := c.cache.GetJSON(ctx, key, &cached)
	if err != nil {
		return nil, nil // Cache miss, not an error
	}

	// Check if session is still valid
	if time.Now().After(cached.ExpiresAt) {
		logger.Debug("CAPTCHA session expired",
			zap.String("domain", domain),
			zap.Time("expired_at", cached.ExpiresAt),
		)
		return nil, nil
	}

	logger.Info("Using cached CAPTCHA session",
		zap.String("domain", domain),
		zap.Int("cookie_count", len(cached.Cookies)),
		zap.Bool("has_fingerprint", cached.Fingerprint != nil),
		zap.Duration("age", time.Since(cached.SolvedAt)),
	)

	return &cached, nil
}

// GetCookies retrieves cached cookies for a domain (backward compatible)
func (c *CookieCache) GetCookies(ctx context.Context, domain string) ([]*http.Cookie, error) {
	session, err := c.GetSession(ctx, domain)
	if err != nil || session == nil {
		return nil, err
	}

	// Convert serialized cookies back to http.Cookie
	cookies := make([]*http.Cookie, len(session.Cookies))
	for i, sc := range session.Cookies {
		cookies[i] = &http.Cookie{
			Name:     sc.Name,
			Value:    sc.Value,
			Domain:   sc.Domain,
			Path:     sc.Path,
			Expires:  sc.Expires,
			Secure:   sc.Secure,
			HttpOnly: sc.HttpOnly,
			SameSite: http.SameSite(sc.SameSite),
		}
	}

	return cookies, nil
}

// SetSession stores a complete session (cookies + fingerprint) for a domain
func (c *CookieCache) SetSession(ctx context.Context, domain string, cookies []*http.Cookie, fingerprint *CachedFingerprint) error {
	if c.cache == nil {
		return nil // No cache available
	}

	// Filter for relevant Cloudflare cookies
	cfCookies := filterCloudflareCookies(cookies)
	if len(cfCookies) == 0 {
		logger.Debug("No Cloudflare cookies found to cache",
			zap.String("domain", domain),
		)
		return nil
	}

	// Serialize cookies
	serialized := make([]*SerializedCookie, len(cfCookies))
	for i, cookie := range cfCookies {
		serialized[i] = &SerializedCookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Expires:  cookie.Expires,
			Secure:   cookie.Secure,
			HttpOnly: cookie.HttpOnly,
			SameSite: int(cookie.SameSite),
		}
	}

	cached := CachedSession{
		Domain:      domain,
		Cookies:     serialized,
		Fingerprint: fingerprint,
		SolvedAt:    time.Now(),
		ExpiresAt:   time.Now().Add(c.ttl),
	}

	key := SessionCacheKeyPrefix + domain
	err := c.cache.SetJSON(ctx, key, cached, c.ttl)
	if err != nil {
		return fmt.Errorf("failed to cache session: %w", err)
	}

	logger.Info("Cached CAPTCHA session",
		zap.String("domain", domain),
		zap.Int("cookie_count", len(cfCookies)),
		zap.Bool("has_fingerprint", fingerprint != nil),
		zap.Duration("ttl", c.ttl),
	)

	return nil
}

// SetCookies stores cookies for a domain after CAPTCHA is solved (backward compatible, no fingerprint)
func (c *CookieCache) SetCookies(ctx context.Context, domain string, cookies []*http.Cookie) error {
	return c.SetSession(ctx, domain, cookies, nil)
}

// TryLock attempts to acquire a lock for solving CAPTCHA on a domain
// Returns true if lock acquired, false if another instance is already solving
func (c *CookieCache) TryLock(ctx context.Context, domain string) (bool, error) {
	if c.cache == nil {
		return true, nil // No cache = always proceed
	}

	lockKey := LockKeyPrefix + domain
	acquired, err := c.cache.SetNX(ctx, lockKey, "solving", LockTTL)
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if acquired {
		logger.Debug("Acquired CAPTCHA solve lock",
			zap.String("domain", domain),
		)
	}

	return acquired, nil
}

// Unlock releases the CAPTCHA solving lock for a domain
func (c *CookieCache) Unlock(ctx context.Context, domain string) error {
	if c.cache == nil {
		return nil
	}

	lockKey := LockKeyPrefix + domain
	return c.cache.Delete(ctx, lockKey)
}

// WaitForSolve waits for another instance to finish solving, then returns cached cookies
func (c *CookieCache) WaitForSolve(ctx context.Context, domain string, maxWait time.Duration) ([]*http.Cookie, error) {
	if c.cache == nil {
		return nil, nil
	}

	deadline := time.Now().Add(maxWait)
	checkInterval := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		// Check if cookies are now available
		cookies, err := c.GetCookies(ctx, domain)
		if err == nil && len(cookies) > 0 {
			return cookies, nil
		}

		// Check if lock is still held
		lockKey := LockKeyPrefix + domain
		exists, _ := c.cache.Exists(ctx, lockKey)
		if !exists {
			// Lock released but no cookies = solving failed
			return nil, nil
		}

		// Wait before checking again
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(checkInterval):
			continue
		}
	}

	logger.Warn("Timeout waiting for CAPTCHA solve",
		zap.String("domain", domain),
		zap.Duration("waited", maxWait),
	)

	return nil, nil
}

// ExtractDomain extracts the domain from a URL for cache keying
func ExtractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return parsed.Host
}

// filterCloudflareCookies filters for Cloudflare-specific cookies
func filterCloudflareCookies(cookies []*http.Cookie) []*http.Cookie {
	cfCookieNames := map[string]bool{
		"cf_clearance": true,
		"__cf_bm":      true,
		"cf_chl_prog":  true,
		"cf_chl_seq_":  true,
	}

	var filtered []*http.Cookie
	for _, cookie := range cookies {
		// Check exact match or prefix match
		if cfCookieNames[cookie.Name] {
			filtered = append(filtered, cookie)
			continue
		}
		// Check prefix matches
		for prefix := range cfCookieNames {
			if len(cookie.Name) >= len(prefix) && cookie.Name[:len(prefix)] == prefix {
				filtered = append(filtered, cookie)
				break
			}
		}
	}

	return filtered
}
