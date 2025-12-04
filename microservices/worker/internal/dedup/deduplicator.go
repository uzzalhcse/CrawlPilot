package dedup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
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

// IsDuplicate checks if a URL has already been processed in this phase
func (d *URLDeduplicator) IsDuplicate(ctx context.Context, executionID, phaseID, url string) (bool, error) {
	key := fmt.Sprintf("crawlify:dedup:%s:%s:%s", executionID, phaseID, url)

	exists, err := d.cache.Exists(ctx, key)
	if err != nil {
		return false, err
	}

	if !exists {
		// Mark as seen with 24-hour expiry
		err = d.cache.Set(ctx, key, "1", 24*time.Hour)
		if err != nil {
			return false, err
		}
		return false, nil
	}

	return true, nil
}

// MarkAsProcessed explicitly marks a URL as processed
func (d *URLDeduplicator) MarkAsProcessed(ctx context.Context, executionID, url string) error {
	key := d.makeKey(executionID, url)
	return d.cache.Set(ctx, key, "1", 24*time.Hour)
}

// GetProcessedCount returns the count of processed URLs for an execution (approximate)
func (d *URLDeduplicator) GetProcessedCount(ctx context.Context, executionID string) (int, error) {
	// This is an approximation using a separate counter
	counterKey := fmt.Sprintf("dedup:exec:%s:count", executionID)

	count, err := d.cache.Get(ctx, counterKey)
	if err != nil {
		// Counter doesn't exist yet
		return 0, nil
	}

	var result int
	fmt.Sscanf(count, "%d", &result)
	return result, nil
}

// IncrementProcessedCount increments the processed URL counter
func (d *URLDeduplicator) IncrementProcessedCount(ctx context.Context, executionID string) error {
	counterKey := fmt.Sprintf("dedup:exec:%s:count", executionID)
	_, err := d.cache.Increment(ctx, counterKey)
	if err != nil {
		return err
	}

	// Set expiration on first increment
	return d.cache.Expire(ctx, counterKey, 24*time.Hour)
}

// makeKey generates a Redis key for a URL
func (d *URLDeduplicator) makeKey(executionID, url string) string {
	// Hash the URL to keep key size manageable
	urlHash := hashURL(url)
	return fmt.Sprintf("dedup:exec:%s:url:%s", executionID, urlHash)
}

// hashURL creates a SHA256 hash of the URL
func hashURL(url string) string {
	hash := sha256.Sum256([]byte(url))
	return hex.EncodeToString(hash[:])
}

// Clear removes all deduplication data for an execution
func (d *URLDeduplicator) Clear(ctx context.Context, executionID string) error {
	// Note: This is best effort. In production, you might want to track all keys
	// or use Redis SCAN to find and delete all matching keys
	counterKey := fmt.Sprintf("dedup:exec:%s:count", executionID)
	return d.cache.Delete(ctx, counterKey)
}
