package recovery

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// ErrorTracker tracks errors using Redis for distributed coordination
// Uses atomic operations to prevent race conditions across workers
type ErrorTracker struct {
	cache  *cache.Cache
	config *TrackerConfig
}

// TrackerConfig holds configuration for the error tracker
type TrackerConfig struct {
	WindowSize           int           // Number of results to track (default: 100)
	ErrorRateThreshold   float64       // Trigger if error rate exceeds this (default: 0.10 = 10%)
	ConsecutiveThreshold int           // Trigger after N consecutive errors (default: 3)
	WindowTTL            time.Duration // How long to keep window data (default: 1 hour)
}

// DefaultTrackerConfig returns sensible defaults
func DefaultTrackerConfig() *TrackerConfig {
	return &TrackerConfig{
		WindowSize:           100,
		ErrorRateThreshold:   0.10, // 10%
		ConsecutiveThreshold: 3,
		WindowTTL:            1 * time.Hour,
	}
}

// Redis key structure for distributed error tracking
// Using Hash for atomic field updates
const (
	// Hash fields for domain stats
	fieldSuccessCount    = "success"
	fieldFailureCount    = "failure"
	fieldConsecutiveErrs = "consecutive_errs"
	fieldLastPattern     = "last_pattern"
	fieldLastUpdated     = "last_updated"
)

// NewErrorTracker creates a new error tracker
func NewErrorTracker(c *cache.Cache, config *TrackerConfig) *ErrorTracker {
	if config == nil {
		config = DefaultTrackerConfig()
	}

	return &ErrorTracker{
		cache:  c,
		config: config,
	}
}

// keyFor returns the Redis key for a domain's stats
func (t *ErrorTracker) keyFor(domain string) string {
	return fmt.Sprintf("error:stats:%s", domain)
}

// RecordSuccess records a successful request - ATOMIC
func (t *ErrorTracker) RecordSuccess(ctx context.Context, domain string) error {
	if t.cache == nil {
		return nil
	}

	key := t.keyFor(domain)

	// Atomic increment success count
	if _, err := t.cache.HIncrBy(ctx, key, fieldSuccessCount, 1); err != nil {
		return err
	}

	// Atomic reset consecutive errors
	if err := t.cache.HSet(ctx, key, fieldConsecutiveErrs, "0"); err != nil {
		return err
	}

	// Update timestamp
	t.cache.HSet(ctx, key, fieldLastUpdated, time.Now().Format(time.RFC3339))

	// Set TTL to keep data fresh
	t.cache.Expire(ctx, key, t.config.WindowTTL)

	return nil
}

// RecordFailure records a failed request and returns whether recovery should trigger - ATOMIC
func (t *ErrorTracker) RecordFailure(ctx context.Context, domain string, pattern ErrorPattern) (shouldRecover bool, reason string) {
	if t.cache == nil {
		return false, ""
	}

	key := t.keyFor(domain)

	// Atomic increment failure count
	t.cache.HIncrBy(ctx, key, fieldFailureCount, 1)

	// Atomic increment consecutive errors
	consecutiveErrs, err := t.cache.HIncrBy(ctx, key, fieldConsecutiveErrs, 1)
	if err != nil {
		consecutiveErrs = 1
	}

	// Update pattern and timestamp
	t.cache.HSet(ctx, key, fieldLastPattern, string(pattern))
	t.cache.HSet(ctx, key, fieldLastUpdated, time.Now().Format(time.RFC3339))

	// Set TTL
	t.cache.Expire(ctx, key, t.config.WindowTTL)

	// Check if recovery should trigger
	return t.shouldTriggerRecovery(ctx, key, int(consecutiveErrs), pattern)
}

// shouldTriggerRecovery determines if recovery should be triggered
func (t *ErrorTracker) shouldTriggerRecovery(ctx context.Context, key string, consecutiveErrs int, pattern ErrorPattern) (bool, string) {
	// Always trigger immediately for critical patterns
	if pattern == PatternCaptcha || pattern == PatternAuthRequired {
		return true, fmt.Sprintf("Critical pattern detected: %s", pattern)
	}

	// Check consecutive errors threshold
	if consecutiveErrs >= t.config.ConsecutiveThreshold {
		return true, fmt.Sprintf("%d consecutive errors (threshold: %d)",
			consecutiveErrs, t.config.ConsecutiveThreshold)
	}

	// Check error rate threshold
	stats := t.getStats(ctx, key)
	total := stats.SuccessCount + stats.FailureCount
	if total >= 10 { // Need at least 10 samples for meaningful rate
		errorRate := float64(stats.FailureCount) / float64(total)
		if errorRate >= t.config.ErrorRateThreshold {
			return true, fmt.Sprintf("Error rate %.1f%% exceeds threshold %.1f%% (window: %d)",
				errorRate*100, t.config.ErrorRateThreshold*100, total)
		}
	}

	return false, ""
}

// DomainStats holds statistics for a domain
type DomainStats struct {
	Domain          string
	SuccessCount    int64
	FailureCount    int64
	ConsecutiveErrs int64
	LastPattern     ErrorPattern
	LastUpdated     time.Time
}

// getStats retrieves current stats from Redis
func (t *ErrorTracker) getStats(ctx context.Context, key string) DomainStats {
	stats := DomainStats{}

	data, err := t.cache.HGetAll(ctx, key)
	if err != nil {
		return stats
	}

	if v, ok := data[fieldSuccessCount]; ok {
		stats.SuccessCount, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := data[fieldFailureCount]; ok {
		stats.FailureCount, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := data[fieldConsecutiveErrs]; ok {
		stats.ConsecutiveErrs, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := data[fieldLastPattern]; ok {
		stats.LastPattern = ErrorPattern(v)
	}
	if v, ok := data[fieldLastUpdated]; ok {
		stats.LastUpdated, _ = time.Parse(time.RFC3339, v)
	}

	return stats
}

// GetStats returns stats for a specific domain
func (t *ErrorTracker) GetStats(ctx context.Context, domain string) map[string]interface{} {
	if t.cache == nil {
		return nil
	}

	key := t.keyFor(domain)
	stats := t.getStats(ctx, key)

	total := stats.SuccessCount + stats.FailureCount
	errorRate := 0.0
	if total > 0 {
		errorRate = float64(stats.FailureCount) / float64(total)
	}

	return map[string]interface{}{
		"domain":           domain,
		"success_count":    stats.SuccessCount,
		"failure_count":    stats.FailureCount,
		"consecutive_errs": stats.ConsecutiveErrs,
		"total":            total,
		"error_rate":       errorRate,
		"last_pattern":     string(stats.LastPattern),
		"last_updated":     stats.LastUpdated,
	}
}

// GetAllStats returns stats for all tracked domains
func (t *ErrorTracker) GetAllStats(ctx context.Context) []map[string]interface{} {
	if t.cache == nil {
		return nil
	}

	// Scan for all error:stats:* keys
	keys, err := t.cache.Keys(ctx, "error:stats:*")
	if err != nil {
		logger.Warn("Failed to get error stats keys", zap.Error(err))
		return nil
	}

	result := make([]map[string]interface{}, 0, len(keys))
	for _, key := range keys {
		// Extract domain from key
		domain := key[len("error:stats:"):]
		stats := t.GetStats(ctx, domain)
		if stats != nil {
			result = append(result, stats)
		}
	}

	return result
}

// ShouldTriggerForDomain checks if recovery should be triggered for a domain
func (t *ErrorTracker) ShouldTriggerForDomain(ctx context.Context, domain string, pattern ErrorPattern) (bool, string) {
	if t.cache == nil {
		return false, ""
	}

	key := t.keyFor(domain)
	stats := t.getStats(ctx, key)
	return t.shouldTriggerRecovery(ctx, key, int(stats.ConsecutiveErrs), pattern)
}

// UpdateConfig allows dynamic configuration updates
func (t *ErrorTracker) UpdateConfig(windowSize int, errorRateThreshold float64, consecutiveThreshold int) {
	if windowSize > 0 {
		t.config.WindowSize = windowSize
	}
	if errorRateThreshold > 0 {
		t.config.ErrorRateThreshold = errorRateThreshold
	}
	if consecutiveThreshold > 0 {
		t.config.ConsecutiveThreshold = consecutiveThreshold
	}

	logger.Info("Error tracker config updated",
		zap.Int("window_size", t.config.WindowSize),
		zap.Float64("error_rate_threshold", t.config.ErrorRateThreshold),
		zap.Int("consecutive_threshold", t.config.ConsecutiveThreshold),
	)
}

// ResetDomain resets all stats for a domain
func (t *ErrorTracker) ResetDomain(ctx context.Context, domain string) error {
	if t.cache == nil {
		return nil
	}

	key := t.keyFor(domain)
	return t.cache.Delete(ctx, key)
}

// ResetConsecutiveErrors resets only consecutive errors for a domain
func (t *ErrorTracker) ResetConsecutiveErrors(ctx context.Context, domain string) error {
	if t.cache == nil {
		return nil
	}

	key := t.keyFor(domain)
	return t.cache.HSet(ctx, key, fieldConsecutiveErrs, "0")
}
