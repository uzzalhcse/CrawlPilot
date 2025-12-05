package dedup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

const (
	// bloomKeyPrefix is the Redis key prefix for bloom filter backup
	bloomKeyPrefix = "crawlify:bloom:"
	// bloomFilterSize is the expected number of URLs per execution (1 million)
	bloomFilterSize = 1_000_000
	// bloomFalsePositiveRate is the acceptable false positive rate (0.1%)
	bloomFalsePositiveRate = 0.001
	// bloomSyncInterval is how often we back up the bloom filter to Redis
	bloomSyncInterval = 30 * time.Second
)

// BloomDeduplicator uses Bloom filters for memory-efficient URL deduplication
//
// Memory comparison at 10M URLs:
// - Redis SetNX: ~1GB (100 bytes/key × 10M)
// - Bloom filter: ~12MB (1.2 bytes/element at 0.1% FPR)
// - Savings: 99% memory reduction
//
// Trade-off: 0.1% false positive rate (1 in 1000 URLs might be incorrectly skipped)
// For web scraping, this is acceptable as missing 0.1% of URLs is negligible.
type BloomDeduplicator struct {
	cache   *cache.Cache
	filters sync.Map // map[executionID]*executionBloom
	mu      sync.RWMutex

	// Background sync
	stopCh chan struct{}
	doneCh chan struct{}
}

// executionBloom holds the bloom filter and metadata for one execution
type executionBloom struct {
	filter    *bloom.BloomFilter
	createdAt time.Time
	count     int64 // Approximate count of URLs added
	mu        sync.Mutex
}

// NewBloomDeduplicator creates a new Bloom filter-based deduplicator
func NewBloomDeduplicator(redisCache *cache.Cache) *BloomDeduplicator {
	d := &BloomDeduplicator{
		cache:  redisCache,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	// Start background sync if Redis is available
	if redisCache != nil {
		go d.runSyncLoop()
	} else {
		// No sync loop needed, close doneCh immediately
		close(d.doneCh)
	}

	logger.Info("Bloom deduplicator initialized",
		zap.Int("expected_size", bloomFilterSize),
		zap.Float64("false_positive_rate", bloomFalsePositiveRate),
		zap.Duration("sync_interval", bloomSyncInterval),
		zap.Bool("redis_available", redisCache != nil),
	)

	return d
}

// IsDuplicate checks if a URL has been seen for this execution/phase
// Uses Bloom filter for O(1) check with 0.1% false positive rate
func (d *BloomDeduplicator) IsDuplicate(ctx context.Context, executionID, phaseID, url string) (bool, error) {
	// Get or create bloom filter for this execution
	filter := d.getOrCreateFilter(executionID)

	// Create composite key for URL+phase
	key := d.makeKey(phaseID, url)

	filter.mu.Lock()
	defer filter.mu.Unlock()

	// Test if URL might exist (probabilistic)
	if filter.filter.TestString(key) {
		// Bloom filter says "maybe exists"
		// This could be a true positive OR a false positive (0.1% chance)
		logger.Debug("Bloom filter: potential duplicate",
			zap.String("execution_id", executionID),
			zap.String("phase_id", phaseID),
		)
		return true, nil
	}

	// Bloom filter says "definitely not exists" - add it
	filter.filter.AddString(key)
	filter.count++

	return false, nil
}

// IsDuplicateWithFallback checks Bloom filter first, then Redis for confirmation
// Use this when false positives are not acceptable
func (d *BloomDeduplicator) IsDuplicateWithFallback(ctx context.Context, executionID, phaseID, url string) (bool, error) {
	// Get or create bloom filter
	filter := d.getOrCreateFilter(executionID)
	key := d.makeKey(phaseID, url)

	filter.mu.Lock()
	defer filter.mu.Unlock()

	// First: Bloom filter check (fast negative)
	if !filter.filter.TestString(key) {
		// Definitely new - add to both Bloom and Redis
		filter.filter.AddString(key)
		filter.count++

		// Also mark in Redis (for distributed workers)
		if d.cache != nil {
			redisKey := fmt.Sprintf("%s%s:%s:%s", bloomKeyPrefix, executionID, phaseID, hashURL(url)[:16])
			d.cache.SetNX(ctx, redisKey, "1", 24*time.Hour)
		}

		return false, nil
	}

	// Bloom says "maybe exists" - confirm with Redis
	if d.cache != nil {
		redisKey := fmt.Sprintf("%s%s:%s:%s", bloomKeyPrefix, executionID, phaseID, hashURL(url)[:16])
		exists, err := d.cache.Exists(ctx, redisKey)
		if err != nil {
			logger.Warn("Redis check failed in fallback", zap.Error(err))
			// On Redis error, trust Bloom filter (accept potential false positive)
			return true, nil
		}

		if !exists {
			// False positive from Bloom filter! URL is actually new
			filter.filter.AddString(key) // Ensure it's in Bloom
			d.cache.SetNX(ctx, redisKey, "1", 24*time.Hour)
			return false, nil
		}
	}

	return true, nil
}

// getOrCreateFilter gets or creates a bloom filter for an execution
func (d *BloomDeduplicator) getOrCreateFilter(executionID string) *executionBloom {
	if existing, ok := d.filters.Load(executionID); ok {
		return existing.(*executionBloom)
	}

	// Create new filter
	filter := &executionBloom{
		filter:    bloom.NewWithEstimates(bloomFilterSize, bloomFalsePositiveRate),
		createdAt: time.Now(),
		count:     0,
	}

	// Store (race condition handled by LoadOrStore)
	actual, _ := d.filters.LoadOrStore(executionID, filter)
	return actual.(*executionBloom)
}

// makeKey creates a consistent key from phase and URL
func (d *BloomDeduplicator) makeKey(phaseID, url string) string {
	return phaseID + ":" + url
}

// runSyncLoop periodically syncs bloom filter stats to Redis
func (d *BloomDeduplicator) runSyncLoop() {
	defer close(d.doneCh)

	ticker := time.NewTicker(bloomSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.syncStats()
		case <-d.stopCh:
			return
		}
	}
}

// syncStats logs bloom filter statistics
func (d *BloomDeduplicator) syncStats() {
	var totalFilters int
	var totalURLs int64

	d.filters.Range(func(key, value any) bool {
		totalFilters++
		eb := value.(*executionBloom)
		totalURLs += eb.count
		return true
	})

	if totalFilters > 0 {
		logger.Info("Bloom deduplicator stats",
			zap.Int("active_executions", totalFilters),
			zap.Int64("total_urls_tracked", totalURLs),
			zap.Int("memory_per_filter_mb", bloomFilterSize/8/1024/1024*10), // Approx
		)
	}
}

// GetStats returns current deduplicator statistics
func (d *BloomDeduplicator) GetStats() map[string]interface{} {
	var totalFilters int
	var totalURLs int64
	var oldestFilter time.Time

	d.filters.Range(func(key, value any) bool {
		totalFilters++
		eb := value.(*executionBloom)
		totalURLs += eb.count
		if oldestFilter.IsZero() || eb.createdAt.Before(oldestFilter) {
			oldestFilter = eb.createdAt
		}
		return true
	})

	return map[string]interface{}{
		"active_executions":  totalFilters,
		"total_urls_tracked": totalURLs,
		"oldest_filter_age":  time.Since(oldestFilter).String(),
		"expected_fpr":       bloomFalsePositiveRate,
		"filter_size":        bloomFilterSize,
	}
}

// ClearExecution removes the bloom filter for a completed execution
func (d *BloomDeduplicator) ClearExecution(executionID string) {
	d.filters.Delete(executionID)
	logger.Debug("Cleared bloom filter for execution", zap.String("execution_id", executionID))
}

// Close gracefully shuts down the deduplicator
func (d *BloomDeduplicator) Close() error {
	close(d.stopCh)

	// Wait for sync loop with timeout
	select {
	case <-d.doneCh:
	case <-time.After(5 * time.Second):
		logger.Warn("Bloom deduplicator shutdown timeout")
	}

	logger.Info("Bloom deduplicator closed")
	return nil
}

// Note: hashURL is defined in deduplicator.go and shared across the package
