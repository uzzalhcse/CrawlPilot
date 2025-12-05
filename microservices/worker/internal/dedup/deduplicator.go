package dedup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

const (
	// Key format: crawlify:dedup:exec:{execution_id}:phase:{phase_id}:url:{url_hash}
	dedupKeyPrefix = "crawlify:dedup:exec"
	// TTL for deduplication keys (24 hours)
	dedupTTL = 24 * time.Hour
)

// URLDeduplicator handles URL deduplication using Redis
type URLDeduplicator struct {
	cache *cache.Cache
}

// NewURLDeduplicator creates a new URL deduplicator
func NewURLDeduplicator(cache *cache.Cache) *URLDeduplicator {
	return &URLDeduplicator{
		cache: cache,
	}
}

// IsDuplicate checks if a URL has already been processed in this phase.
// Uses atomic SetNX to check AND mark in one operation (best practice for dedup).
// Returns true if URL was already processed, false if this is the first time.
func (d *URLDeduplicator) IsDuplicate(ctx context.Context, executionID, phaseID, url string) (bool, error) {
	if d.cache == nil {
		// No cache available, treat as not duplicate
		logger.Warn("Deduplicator cache not available, skipping dedup check")
		return false, nil
	}

	key := d.makeKey(executionID, phaseID, url)

	// Atomic check + mark using SetNX (SET if Not eXists)
	// Returns true if key was SET (meaning URL is NEW)
	// Returns false if key already EXISTS (meaning URL is DUPLICATE)
	wasSet, err := d.cache.SetNX(ctx, key, "1", dedupTTL)
	if err != nil {
		logger.Warn("Deduplication SetNX failed",
			zap.String("key", key),
			zap.Error(err),
		)
		return false, err
	}

	// wasSet=true means we SET the key → URL is NEW (not duplicate)
	// wasSet=false means key existed → URL is DUPLICATE
	isDuplicate := !wasSet

	if isDuplicate {
		logger.Debug("Duplicate URL detected",
			zap.String("execution_id", executionID),
			zap.String("phase_id", phaseID),
			zap.String("url_hash", hashURL(url)[:16]),
		)
	}

	return isDuplicate, nil
}

// MarkAsProcessed explicitly marks a URL as processed (for manual marking)
func (d *URLDeduplicator) MarkAsProcessed(ctx context.Context, executionID, phaseID, url string) error {
	if d.cache == nil {
		return nil
	}

	key := d.makeKey(executionID, phaseID, url)
	return d.cache.Set(ctx, key, "1", dedupTTL)
}

// GetProcessedCount returns the approximate count of processed URLs for an execution
func (d *URLDeduplicator) GetProcessedCount(ctx context.Context, executionID string) (int64, error) {
	if d.cache == nil {
		return 0, nil
	}

	counterKey := fmt.Sprintf("%s:%s:count", dedupKeyPrefix, executionID)
	count, err := d.cache.Get(ctx, counterKey)
	if err != nil {
		// Counter doesn't exist yet
		return 0, nil
	}

	var result int64
	fmt.Sscanf(count, "%d", &result)
	return result, nil
}

// IncrementProcessedCount increments the processed URL counter
func (d *URLDeduplicator) IncrementProcessedCount(ctx context.Context, executionID string) error {
	if d.cache == nil {
		return nil
	}

	counterKey := fmt.Sprintf("%s:%s:count", dedupKeyPrefix, executionID)
	_, err := d.cache.Increment(ctx, counterKey)
	if err != nil {
		return err
	}

	// Set expiration on counter
	return d.cache.Expire(ctx, counterKey, dedupTTL)
}

// makeKey generates a consistent Redis key for a URL
// Format: crawlify:dedup:exec:{execution_id}:phase:{phase_id}:url:{url_hash}
func (d *URLDeduplicator) makeKey(executionID, phaseID, url string) string {
	urlHash := hashURL(url)
	return fmt.Sprintf("%s:%s:phase:%s:url:%s", dedupKeyPrefix, executionID, phaseID, urlHash)
}

// hashURL creates a SHA256 hash of the URL for consistent key length
func hashURL(url string) string {
	hash := sha256.Sum256([]byte(url))
	return hex.EncodeToString(hash[:])
}

// Clear removes all deduplication data for an execution
func (d *URLDeduplicator) Clear(ctx context.Context, executionID string) error {
	if d.cache == nil {
		return nil
	}

	// Note: In production, use Redis SCAN to find and delete all matching keys
	// For now, just clear the counter
	counterKey := fmt.Sprintf("%s:%s:count", dedupKeyPrefix, executionID)
	return d.cache.Delete(ctx, counterKey)
}
