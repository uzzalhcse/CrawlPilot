package reporter

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

const (
	// Redis keys for tracking outstanding tasks
	outstandingKeyPrefix = "crawlify:outstanding:"
	completionTTL        = 24 * time.Hour // Keep counters for 24h
	// Batch flush interval - sync local counters to Redis periodically
	// This reduces 10K Redis ops/sec to ~0.2/sec per execution
	completionFlushInterval = 5 * time.Second
)

// executionCounterEntry holds local atomic counters for an execution
type executionCounterEntry struct {
	queued    atomic.Int64 // Tasks queued locally (not yet synced)
	completed atomic.Int64 // Tasks completed locally (not yet synced)
}

// CompletionTracker tracks outstanding tasks and detects execution completion
// using Redis for distributed coordination across multiple workers.
//
// HIGH-THROUGHPUT DESIGN:
// - Uses local atomic counters (lock-free) for per-task operations
// - Batches Redis sync every 5 seconds (not per-task)
// - Reduces Redis load from 10K+/sec to ~0.2/sec per execution
type CompletionTracker struct {
	cache           *cache.Cache
	statsReporter   *BatchedStatsReporter
	orchestratorURL string

	// Local counters per execution (batched for performance)
	counters sync.Map // map[executionID]*executionCounterEntry

	// Control channels
	stopCh   chan struct{}
	wg       sync.WaitGroup
	shutdown atomic.Bool
}

// NewCompletionTracker creates a new completion tracker with batched Redis sync
func NewCompletionTracker(redisCache *cache.Cache, statsReporter *BatchedStatsReporter, orchestratorURL string) *CompletionTracker {
	t := &CompletionTracker{
		cache:           redisCache,
		statsReporter:   statsReporter,
		orchestratorURL: orchestratorURL,
		stopCh:          make(chan struct{}),
	}

	// Start background flush loop
	t.wg.Add(1)
	go t.runFlushLoop()

	logger.Info("Completion tracker initialized",
		zap.Duration("flush_interval", completionFlushInterval),
		zap.Bool("redis_enabled", redisCache != nil),
	)

	return t
}

// TaskQueued increments the local queued count (lock-free, no network call)
// This is safe to call at 10K+ times/sec with zero blocking
func (t *CompletionTracker) TaskQueued(ctx context.Context, executionID string, count int64) error {
	if t.shutdown.Load() {
		return nil
	}

	entry := t.getOrCreateEntry(executionID)
	entry.queued.Add(count)

	logger.Debug("Tasks queued (local)",
		zap.String("execution_id", executionID),
		zap.Int64("added", count),
	)

	return nil
}

// TaskCompleted increments the local completed count (lock-free, no network call)
// Returns false always - completion detection happens in flush loop
func (t *CompletionTracker) TaskCompleted(ctx context.Context, executionID string) (bool, error) {
	if t.shutdown.Load() {
		return false, nil
	}

	entry := t.getOrCreateEntry(executionID)
	entry.completed.Add(1)

	logger.Debug("Task completed (local)",
		zap.String("execution_id", executionID),
	)

	// Don't return true here - completion is detected in flush loop
	return false, nil
}

// getOrCreateEntry gets or creates a counter entry for an execution
func (t *CompletionTracker) getOrCreateEntry(executionID string) *executionCounterEntry {
	entryI, _ := t.counters.LoadOrStore(executionID, &executionCounterEntry{})
	return entryI.(*executionCounterEntry)
}

// runFlushLoop periodically syncs local counters to Redis
func (t *CompletionTracker) runFlushLoop() {
	defer t.wg.Done()

	ticker := time.NewTicker(completionFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.flush()
		case <-t.stopCh:
			// Final flush on shutdown
			t.flush()
			return
		}
	}
}

// flush syncs all local counters to Redis and checks for completion
func (t *CompletionTracker) flush() {
	if t.cache == nil {
		return
	}

	ctx := context.Background()

	t.counters.Range(func(key, value any) bool {
		executionID := key.(string)
		entry := value.(*executionCounterEntry)

		// Atomically swap counters to zero and get current values
		queued := entry.queued.Swap(0)
		completed := entry.completed.Swap(0)

		// Calculate net change (queued - completed)
		// Positive = more tasks pending, Negative = tasks completed
		netChange := queued - completed

		if netChange == 0 && queued == 0 && completed == 0 {
			return true // No activity, skip
		}

		redisKey := t.makeKey(executionID)

		// Apply net change to Redis in a single operation
		var remaining int64
		var err error

		if netChange > 0 {
			remaining, err = t.cache.IncrBy(ctx, redisKey, netChange)
		} else if netChange < 0 {
			remaining, err = t.cache.DecrBy(ctx, redisKey, -netChange)
		} else {
			// netChange == 0 but we had activity - need to check current value
			val, getErr := t.cache.Get(ctx, redisKey)
			if getErr != nil {
				remaining = 0
			} else {
				fmt.Sscanf(val, "%d", &remaining)
			}
			err = nil
		}

		if err != nil {
			logger.Warn("Failed to sync completion counter to Redis",
				zap.String("execution_id", executionID),
				zap.Int64("queued", queued),
				zap.Int64("completed", completed),
				zap.Error(err),
			)
			// Re-add the values on failure
			entry.queued.Add(queued)
			entry.completed.Add(completed)
			return true
		}

		logger.Debug("Completion counters synced to Redis",
			zap.String("execution_id", executionID),
			zap.Int64("queued", queued),
			zap.Int64("completed", completed),
			zap.Int64("net_change", netChange),
			zap.Int64("remaining", remaining),
		)

		// Check if execution is complete
		if remaining <= 0 && completed > 0 {
			logger.Info("Execution complete - all tasks processed",
				zap.String("execution_id", executionID),
			)

			// Clean up Redis key
			t.cache.Delete(ctx, redisKey)

			// Clean up local entry
			t.counters.Delete(executionID)

			// Signal completion to orchestrator (async)
			if t.statsReporter != nil {
				go func(execID string) {
					if err := t.statsReporter.MarkComplete(execID, "completed"); err != nil {
						logger.Warn("Failed to signal execution completion",
							zap.String("execution_id", execID),
							zap.Error(err),
						)
					}
				}(executionID)
			}
		}

		return true
	})
}

// GetOutstandingCount returns the current outstanding task count from Redis
func (t *CompletionTracker) GetOutstandingCount(ctx context.Context, executionID string) (int64, error) {
	if t.cache == nil {
		return 0, nil
	}

	key := t.makeKey(executionID)
	val, err := t.cache.Get(ctx, key)
	if err != nil {
		// Key doesn't exist means 0 outstanding
		return 0, nil
	}

	var count int64
	fmt.Sscanf(val, "%d", &count)
	return count, nil
}

// Clear removes tracking data for an execution
func (t *CompletionTracker) Clear(ctx context.Context, executionID string) error {
	// Clear local counters
	t.counters.Delete(executionID)

	if t.cache == nil {
		return nil
	}

	key := t.makeKey(executionID)
	return t.cache.Delete(ctx, key)
}

// Close gracefully shuts down the completion tracker
func (t *CompletionTracker) Close() error {
	if t.shutdown.Swap(true) {
		return nil // Already shutdown
	}

	close(t.stopCh)

	// Wait for final flush with timeout
	done := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("Completion tracker closed")
	case <-time.After(10 * time.Second):
		logger.Warn("Completion tracker shutdown timeout")
	}

	return nil
}

func (t *CompletionTracker) makeKey(executionID string) string {
	return outstandingKeyPrefix + executionID
}
