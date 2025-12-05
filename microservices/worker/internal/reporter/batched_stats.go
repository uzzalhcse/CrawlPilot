package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

const (
	// statsFlushInterval is how often we flush stats to orchestrator
	statsFlushInterval = 5 * time.Second
	// statsRedisKeyPrefix is the Redis key prefix for stats counters
	statsRedisKeyPrefix = "crawlify:stats:"
	// statsRedisTTL is how long stats keys live in Redis
	statsRedisTTL = 1 * time.Hour
)

// BatchedStatsReporter aggregates stats locally and flushes periodically
// This reduces HTTP calls from 10k/sec to ~0.2/sec per execution
type BatchedStatsReporter struct {
	orchestratorURL string
	httpClient      *http.Client
	cache           *cache.Cache // Optional: for distributed aggregation

	// Local counters (per execution ID)
	counters sync.Map // map[executionID]*executionCounters

	// Control
	stopCh   chan struct{}
	doneCh   chan struct{}
	wg       sync.WaitGroup
	shutdown atomic.Bool
}

// executionCounters holds atomic counters for a single execution
type executionCounters struct {
	urlsProcessed  atomic.Int64
	urlsDiscovered atomic.Int64
	itemsExtracted atomic.Int64
	errors         atomic.Int64
}

// NewBatchedStatsReporter creates a new batched stats reporter
func NewBatchedStatsReporter(orchestratorURL string, redisCache *cache.Cache) *BatchedStatsReporter {
	r := &BatchedStatsReporter{
		orchestratorURL: orchestratorURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // Longer timeout for batch requests
		},
		cache:  redisCache,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	// Start background flush goroutine
	r.wg.Add(1)
	go r.runFlushLoop()

	logger.Info("Batched stats reporter initialized",
		zap.String("orchestrator_url", orchestratorURL),
		zap.Duration("flush_interval", statsFlushInterval),
		zap.Bool("redis_enabled", redisCache != nil),
	)

	return r
}

// Record records stats for an execution (atomic, no network calls)
func (r *BatchedStatsReporter) Record(executionID string, stats TaskStats) {
	if r.shutdown.Load() {
		return
	}

	// Get or create counters for this execution
	countersI, _ := r.counters.LoadOrStore(executionID, &executionCounters{})
	counters := countersI.(*executionCounters)

	// Atomic updates (lock-free)
	counters.urlsProcessed.Add(int64(stats.URLsProcessed))
	counters.urlsDiscovered.Add(int64(stats.URLsDiscovered))
	counters.itemsExtracted.Add(int64(stats.ItemsExtracted))
	counters.errors.Add(int64(stats.Errors))
}

// runFlushLoop periodically flushes accumulated stats
func (r *BatchedStatsReporter) runFlushLoop() {
	defer r.wg.Done()
	defer close(r.doneCh)

	ticker := time.NewTicker(statsFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.flush()
		case <-r.stopCh:
			// Final flush on shutdown
			r.flush()
			return
		}
	}
}

// flush sends all accumulated stats to orchestrator in ONE request
func (r *BatchedStatsReporter) flush() {
	if r.orchestratorURL == "" {
		logger.Debug("Orchestrator URL not configured, skipping stats flush")
		return
	}

	// Collect and reset all counters
	updates := make([]BatchedStatsUpdate, 0)

	r.counters.Range(func(key, value any) bool {
		executionID := key.(string)
		counters := value.(*executionCounters)

		// Swap counters to zero and get current values
		urlsProcessed := counters.urlsProcessed.Swap(0)
		urlsDiscovered := counters.urlsDiscovered.Swap(0)
		itemsExtracted := counters.itemsExtracted.Swap(0)
		errors := counters.errors.Swap(0)

		// Only include if there's data
		if urlsProcessed > 0 || urlsDiscovered > 0 || itemsExtracted > 0 || errors > 0 {
			updates = append(updates, BatchedStatsUpdate{
				ExecutionID:    executionID,
				URLsProcessed:  int(urlsProcessed),
				URLsDiscovered: int(urlsDiscovered),
				ItemsExtracted: int(itemsExtracted),
				Errors:         int(errors),
			})
		}

		return true
	})

	if len(updates) == 0 {
		return
	}

	// Send single batched request to orchestrator
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := r.sendBatchedStats(ctx, updates); err != nil {
		logger.Warn("Failed to send batched stats",
			zap.Int("count", len(updates)),
			zap.Error(err),
		)
		// Re-add stats on failure (best effort)
		for _, update := range updates {
			countersI, _ := r.counters.LoadOrStore(update.ExecutionID, &executionCounters{})
			counters := countersI.(*executionCounters)
			counters.urlsProcessed.Add(int64(update.URLsProcessed))
			counters.urlsDiscovered.Add(int64(update.URLsDiscovered))
			counters.itemsExtracted.Add(int64(update.ItemsExtracted))
			counters.errors.Add(int64(update.Errors))
		}
		return
	}

	logger.Info("Stats batch flushed",
		zap.Int("executions", len(updates)),
		zap.Int("total_urls_processed", sumURLsProcessed(updates)),
	)
}

// BatchedStatsUpdate represents stats update for one execution
type BatchedStatsUpdate struct {
	ExecutionID    string `json:"execution_id"`
	URLsProcessed  int    `json:"urls_processed"`
	URLsDiscovered int    `json:"urls_discovered"`
	ItemsExtracted int    `json:"items_extracted"`
	Errors         int    `json:"errors"`
}

// BatchedStatsRequest is the request body for batch stats endpoint
type BatchedStatsRequest struct {
	Updates   []BatchedStatsUpdate `json:"updates"`
	Timestamp time.Time            `json:"timestamp"`
	WorkerID  string               `json:"worker_id,omitempty"`
}

// sendBatchedStats sends a single batched request to orchestrator
func (r *BatchedStatsReporter) sendBatchedStats(ctx context.Context, updates []BatchedStatsUpdate) error {
	reqBody := BatchedStatsRequest{
		Updates:   updates,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal batched stats: %w", err)
	}

	// New batch endpoint (needs to be added to orchestrator)
	url := fmt.Sprintf("%s/api/v1/internal/stats/batch", r.orchestratorURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// Close gracefully shuts down the batched reporter
func (r *BatchedStatsReporter) Close() error {
	if r.shutdown.Swap(true) {
		return nil // Already shutdown
	}

	close(r.stopCh)

	// Wait for final flush with timeout
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("Batched stats reporter closed")
	case <-time.After(10 * time.Second):
		logger.Warn("Batched stats reporter shutdown timeout")
	}

	return nil
}

// Stats returns current reporter statistics
func (r *BatchedStatsReporter) Stats() map[string]interface{} {
	count := 0
	var totalQueued int64

	r.counters.Range(func(key, value any) bool {
		count++
		counters := value.(*executionCounters)
		totalQueued += counters.urlsProcessed.Load()
		return true
	})

	return map[string]interface{}{
		"active_executions": count,
		"queued_updates":    totalQueued,
		"flush_interval":    statsFlushInterval.String(),
		"shutdown":          r.shutdown.Load(),
	}
}

// sumURLsProcessed calculates total URLs processed in updates
func sumURLsProcessed(updates []BatchedStatsUpdate) int {
	total := 0
	for _, u := range updates {
		total += u.URLsProcessed
	}
	return total
}
