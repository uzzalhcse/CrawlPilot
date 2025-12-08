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

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

const (
	// recoveryFlushInterval is how often we flush recovery attempts to orchestrator
	recoveryFlushInterval = 500 * time.Millisecond
	// maxBatchSize is the maximum number of attempts per batch
	maxBatchSize = 50
	// maxPendingAttempts is the maximum pending attempts to buffer
	maxPendingAttempts = 10000
)

// BatchedRecoveryReporter handles batched async reporting of recovery attempts
// Designed for high-throughput: 10K URLs/sec with ~1% error rate = 100 recoveries/sec
type BatchedRecoveryReporter struct {
	orchestratorURL string
	httpClient      *http.Client

	// Buffers for batching
	createBuffer   []*CreateAttemptRequest
	createBufferMu sync.Mutex

	// Control
	stopCh   chan struct{}
	doneCh   chan struct{}
	wg       sync.WaitGroup
	shutdown atomic.Bool

	// Stats
	createsSent    atomic.Int64
	batchesSent    atomic.Int64
	createsFailed  atomic.Int64
	updatesDropped atomic.Int64
}

// NewBatchedRecoveryReporter creates a new batched async recovery reporter
func NewBatchedRecoveryReporter(orchestratorURL string) *BatchedRecoveryReporter {
	r := &BatchedRecoveryReporter{
		orchestratorURL: orchestratorURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 50,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		createBuffer: make([]*CreateAttemptRequest, 0, maxBatchSize),
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}

	// Start batch flush loop
	r.wg.Add(1)
	go r.runFlushLoop()

	logger.Info("Batched recovery reporter initialized",
		zap.String("orchestrator_url", orchestratorURL),
		zap.Duration("flush_interval", recoveryFlushInterval),
		zap.Int("max_batch_size", maxBatchSize),
	)

	return r
}

// CreateAttemptAsync queues a recovery attempt for batched creation (non-blocking)
func (r *BatchedRecoveryReporter) CreateAttemptAsync(ctx context.Context, req *CreateAttemptRequest) {
	if r.shutdown.Load() {
		return
	}

	r.createBufferMu.Lock()
	defer r.createBufferMu.Unlock()

	// Check buffer limit
	if len(r.createBuffer) >= maxPendingAttempts {
		r.createsFailed.Add(1)
		logger.Warn("Recovery buffer full, dropping",
			zap.String("execution_id", req.ExecutionID),
		)
		return
	}

	r.createBuffer = append(r.createBuffer, req)

	// Flush immediately if batch is full
	if len(r.createBuffer) >= maxBatchSize {
		go r.flushCreates()
	}
}

// UpdateAttemptAsync is a no-op for async batched mode (updates are not tracked individually)
// For high-throughput, we only track aggregate stats rather than individual attempt outcomes
func (r *BatchedRecoveryReporter) UpdateAttemptAsync(ctx context.Context, attemptID string, req *UpdateAttemptRequest) {
	// In high-throughput mode, we skip individual updates to reduce DB load
	// The frontend shows pending attempts; successful task completions update via other means
	r.updatesDropped.Add(1)
}

// runFlushLoop periodically flushes accumulated attempts
func (r *BatchedRecoveryReporter) runFlushLoop() {
	defer r.wg.Done()
	defer close(r.doneCh)

	ticker := time.NewTicker(recoveryFlushInterval)
	defer ticker.Stop()

	statsTicker := time.NewTicker(30 * time.Second)
	defer statsTicker.Stop()

	for {
		select {
		case <-ticker.C:
			r.flushCreates()
		case <-statsTicker.C:
			r.logStats()
		case <-r.stopCh:
			r.flushCreates() // Final flush
			return
		}
	}
}

// flushCreates sends buffered creates as a batch to orchestrator
func (r *BatchedRecoveryReporter) flushCreates() {
	r.createBufferMu.Lock()
	if len(r.createBuffer) == 0 {
		r.createBufferMu.Unlock()
		return
	}

	// Take the batch and clear buffer
	batch := r.createBuffer
	r.createBuffer = make([]*CreateAttemptRequest, 0, maxBatchSize)
	r.createBufferMu.Unlock()

	// Send batch to orchestrator
	if err := r.sendCreateBatch(batch); err != nil {
		r.createsFailed.Add(int64(len(batch)))
		logger.Warn("Failed to send recovery batch",
			zap.Int("batch_size", len(batch)),
			zap.Error(err),
		)
		return
	}

	r.createsSent.Add(int64(len(batch)))
	r.batchesSent.Add(1)
}

// BatchCreateRequest matches the orchestrator API
type BatchCreateRequest struct {
	Attempts []*CreateAttemptRequest `json:"attempts"`
}

// sendCreateBatch sends a batch of attempts to orchestrator
func (r *BatchedRecoveryReporter) sendCreateBatch(batch []*CreateAttemptRequest) error {
	url := fmt.Sprintf("%s/api/v1/internal/recovery/attempts/batch", r.orchestratorURL)

	reqBody := BatchCreateRequest{Attempts: batch}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("orchestrator returned error: %d", resp.StatusCode)
	}

	return nil
}

// logStats logs current statistics
func (r *BatchedRecoveryReporter) logStats() {
	creates := r.createsSent.Load()
	batches := r.batchesSent.Load()
	fails := r.createsFailed.Load()
	drops := r.updatesDropped.Load()

	r.createBufferMu.Lock()
	bufferLen := len(r.createBuffer)
	r.createBufferMu.Unlock()

	if creates > 0 || fails > 0 {
		avgBatchSize := float64(0)
		if batches > 0 {
			avgBatchSize = float64(creates) / float64(batches)
		}
		logger.Info("Recovery reporter stats",
			zap.Int64("creates_sent", creates),
			zap.Int64("batches_sent", batches),
			zap.Float64("avg_batch_size", avgBatchSize),
			zap.Int64("creates_failed", fails),
			zap.Int64("updates_dropped", drops),
			zap.Int("buffer_len", bufferLen),
		)
	}
}

// Close gracefully shuts down the reporter
func (r *BatchedRecoveryReporter) Close() error {
	if r.shutdown.Swap(true) {
		return nil
	}

	close(r.stopCh)

	select {
	case <-r.doneCh:
		logger.Info("Batched recovery reporter closed",
			zap.Int64("total_creates", r.createsSent.Load()),
			zap.Int64("total_batches", r.batchesSent.Load()),
		)
	case <-time.After(10 * time.Second):
		logger.Warn("Batched recovery reporter shutdown timeout")
	}

	return nil
}

// Stats returns current reporter statistics
func (r *BatchedRecoveryReporter) Stats() map[string]interface{} {
	r.createBufferMu.Lock()
	bufferLen := len(r.createBuffer)
	r.createBufferMu.Unlock()

	return map[string]interface{}{
		"creates_sent":    r.createsSent.Load(),
		"batches_sent":    r.batchesSent.Load(),
		"creates_failed":  r.createsFailed.Load(),
		"updates_dropped": r.updatesDropped.Load(),
		"buffer_len":      bufferLen,
		"shutdown":        r.shutdown.Load(),
	}
}
