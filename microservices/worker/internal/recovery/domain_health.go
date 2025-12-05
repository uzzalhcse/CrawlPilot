package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// DomainHealth tracks the health status of domains in Redis
// Uses atomic operations to prevent race conditions across workers
type DomainHealth struct {
	cache *cache.Cache
	ttl   time.Duration
}

// Redis key structure for domain health
const (
	fieldDomainSuccessCount    = "success_count"
	fieldDomainFailureCount    = "failure_count"
	fieldDomainConsecutiveFail = "consecutive_fails"
	fieldDomainLastPattern     = "last_pattern"
	fieldDomainIsBlocked       = "is_blocked"
	fieldDomainBlockedUntil    = "blocked_until"
	fieldDomainLastSuccess     = "last_success"
	fieldDomainLastFailure     = "last_failure"
	fieldDomainWorkingProxies  = "working_proxies"
)

// NewDomainHealth creates a new domain health tracker
func NewDomainHealth(c *cache.Cache) *DomainHealth {
	return &DomainHealth{
		cache: c,
		ttl:   2 * time.Hour,
	}
}

// keyFor returns the Redis key for a domain
func (d *DomainHealth) keyFor(domain string) string {
	return fmt.Sprintf("domain:health:%s", domain)
}

// Get retrieves the health status for a domain
func (d *DomainHealth) Get(ctx context.Context, domain string) (*DomainStatus, error) {
	if d.cache == nil {
		return &DomainStatus{Domain: domain}, nil
	}

	key := d.keyFor(domain)
	data, err := d.cache.HGetAll(ctx, key)
	if err != nil || len(data) == 0 {
		return &DomainStatus{Domain: domain}, nil
	}

	status := &DomainStatus{Domain: domain}

	if v, ok := data[fieldDomainSuccessCount]; ok {
		val, _ := strconv.ParseInt(v, 10, 64)
		status.SuccessCount = int(val)
	}
	if v, ok := data[fieldDomainFailureCount]; ok {
		val, _ := strconv.ParseInt(v, 10, 64)
		status.FailureCount = int(val)
	}
	if v, ok := data[fieldDomainConsecutiveFail]; ok {
		val, _ := strconv.ParseInt(v, 10, 64)
		status.ConsecutiveFails = int(val)
	}
	if v, ok := data[fieldDomainLastPattern]; ok {
		status.LastPattern = ErrorPattern(v)
	}
	if v, ok := data[fieldDomainIsBlocked]; ok {
		status.IsBlocked = v == "true"
	}
	if v, ok := data[fieldDomainBlockedUntil]; ok {
		status.BlockedUntil, _ = time.Parse(time.RFC3339, v)
	}
	if v, ok := data[fieldDomainLastSuccess]; ok {
		status.LastSuccess, _ = time.Parse(time.RFC3339, v)
	}
	if v, ok := data[fieldDomainLastFailure]; ok {
		status.LastFailure, _ = time.Parse(time.RFC3339, v)
	}
	if v, ok := data[fieldDomainWorkingProxies]; ok {
		json.Unmarshal([]byte(v), &status.WorkingProxies)
	}

	// Check if block has expired
	if status.IsBlocked && time.Now().After(status.BlockedUntil) {
		status.IsBlocked = false
		// Async cleanup
		go d.cache.HSet(ctx, key, fieldDomainIsBlocked, "false")
	}

	return status, nil
}

// IsHealthy checks if a domain is healthy (not blocked/limited)
func (d *DomainHealth) IsHealthy(ctx context.Context, domain string) (bool, error) {
	status, err := d.Get(ctx, domain)
	if err != nil {
		return true, err // Assume healthy on error
	}
	return !status.IsBlocked, nil
}

// RecordSuccess records a successful request - ATOMIC
func (d *DomainHealth) RecordSuccess(ctx context.Context, domain string, proxyID string) error {
	if d.cache == nil {
		return nil
	}

	key := d.keyFor(domain)

	// Atomic increment success count
	if _, err := d.cache.HIncrBy(ctx, key, fieldDomainSuccessCount, 1); err != nil {
		return err
	}

	// Atomic reset consecutive failures
	d.cache.HSet(ctx, key, fieldDomainConsecutiveFail, "0")
	d.cache.HSet(ctx, key, fieldDomainLastSuccess, time.Now().Format(time.RFC3339))

	// Set TTL
	d.cache.Expire(ctx, key, d.ttl)

	// Add working proxy if not already tracked
	if proxyID != "" {
		go d.addWorkingProxy(ctx, key, proxyID)
	}

	return nil
}

// addWorkingProxy adds a proxy to the working proxies list atomically
func (d *DomainHealth) addWorkingProxy(ctx context.Context, key, proxyID string) {
	// Get current working proxies
	data, _ := d.cache.HGet(ctx, key, fieldDomainWorkingProxies)
	var proxies []string
	if data != "" {
		json.Unmarshal([]byte(data), &proxies)
	}

	// Check if already in list
	for _, p := range proxies {
		if p == proxyID {
			return
		}
	}

	// Add if under limit
	if len(proxies) < 10 {
		proxies = append(proxies, proxyID)
		data, _ := json.Marshal(proxies)
		d.cache.HSet(ctx, key, fieldDomainWorkingProxies, string(data))
	}
}

// RecordFailure records a failed request - ATOMIC
func (d *DomainHealth) RecordFailure(ctx context.Context, domain string, pattern ErrorPattern) error {
	if d.cache == nil {
		return nil
	}

	key := d.keyFor(domain)

	// Atomic increment failure count
	d.cache.HIncrBy(ctx, key, fieldDomainFailureCount, 1)

	// Atomic increment consecutive failures
	consecutiveFails, err := d.cache.HIncrBy(ctx, key, fieldDomainConsecutiveFail, 1)
	if err != nil {
		consecutiveFails = 1
	}

	// Update metadata
	d.cache.HSet(ctx, key, fieldDomainLastPattern, string(pattern))
	d.cache.HSet(ctx, key, fieldDomainLastFailure, time.Now().Format(time.RFC3339))

	// Set TTL
	d.cache.Expire(ctx, key, d.ttl)

	// Auto-block logic based on consecutive failures
	if int(consecutiveFails) >= 5 {
		waitDuration := calculateBlockDuration(int(consecutiveFails), pattern)
		d.cache.HSet(ctx, key, fieldDomainIsBlocked, "true")
		d.cache.HSet(ctx, key, fieldDomainBlockedUntil, time.Now().Add(waitDuration).Format(time.RFC3339))

		logger.Warn("Domain auto-blocked due to consecutive failures",
			zap.String("domain", domain),
			zap.Int64("consecutive_fails", consecutiveFails),
			zap.Duration("blocked_for", waitDuration),
		)
	}

	return nil
}

// Block manually blocks a domain for a duration
func (d *DomainHealth) Block(ctx context.Context, domain string, duration time.Duration, reason string) error {
	if d.cache == nil {
		return nil
	}

	key := d.keyFor(domain)

	d.cache.HSet(ctx, key, fieldDomainIsBlocked, "true")
	d.cache.HSet(ctx, key, fieldDomainBlockedUntil, time.Now().Add(duration).Format(time.RFC3339))
	d.cache.Expire(ctx, key, d.ttl)

	logger.Info("Domain blocked",
		zap.String("domain", domain),
		zap.Duration("duration", duration),
		zap.String("reason", reason),
	)

	return nil
}

// Unblock removes a domain from blocklist
func (d *DomainHealth) Unblock(ctx context.Context, domain string) error {
	if d.cache == nil {
		return nil
	}

	key := d.keyFor(domain)

	d.cache.HSet(ctx, key, fieldDomainIsBlocked, "false")
	d.cache.HSet(ctx, key, fieldDomainConsecutiveFail, "0")

	return nil
}

// GetWorkingProxies returns proxies that have worked for a domain
func (d *DomainHealth) GetWorkingProxies(ctx context.Context, domain string) ([]string, error) {
	status, err := d.Get(ctx, domain)
	if err != nil {
		return nil, err
	}
	return status.WorkingProxies, nil
}

// GetAllBlocked returns all currently blocked domains
func (d *DomainHealth) GetAllBlocked(ctx context.Context) ([]DomainStatus, error) {
	if d.cache == nil {
		return nil, nil
	}

	// Scan for all domain:health:* keys
	keys, err := d.cache.Keys(ctx, "domain:health:*")
	if err != nil {
		return nil, err
	}

	blocked := make([]DomainStatus, 0)
	for _, key := range keys {
		domain := key[len("domain:health:"):]
		status, err := d.Get(ctx, domain)
		if err != nil {
			continue
		}

		if status.IsBlocked && time.Now().Before(status.BlockedUntil) {
			blocked = append(blocked, *status)
		}
	}

	return blocked, nil
}

// calculateBlockDuration determines how long to block based on failure severity
func calculateBlockDuration(consecutiveFails int, pattern ErrorPattern) time.Duration {
	base := 1 * time.Minute

	// Pattern-based multiplier
	multiplier := 1.0
	switch pattern {
	case PatternBlocked:
		multiplier = 5.0 // 5 min per failure
	case PatternRateLimited:
		multiplier = 2.0 // 2 min per failure
	case PatternCaptcha:
		multiplier = 10.0 // 10 min, needs human intervention
	default:
		multiplier = 1.0
	}

	// Exponential backoff for consecutive failures
	duration := base * time.Duration(float64(consecutiveFails)*multiplier)

	// Cap at 1 hour
	if duration > 1*time.Hour {
		duration = 1 * time.Hour
	}

	return duration
}
