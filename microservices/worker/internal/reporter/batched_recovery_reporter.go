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

// UpdateWithID combines an attempt ID with its update request
type UpdateWithID struct {
	AttemptID string
	Request   *UpdateAttemptRequest
}

// BatchedRecoveryReporter handles batched async reporting of recovery attempts
// Designed for high-throughput: 10K URLs/sec with ~1% error rate = 100 recoveries/sec
type BatchedRecoveryReporter struct {
	orchestratorURL string
	httpClient      *http.Client

	// Buffers for batching
	createBuffer   []*CreateAttemptRequest
	createBufferMu sync.Mutex

	updateBuffer   []*UpdateWithID
	updateBufferMu sync.Mutex

	// Control
	stopCh   chan struct{}
	doneCh   chan struct{}
	wg       sync.WaitGroup
	shutdown atomic.Bool

	// Stats
	createsSent   atomic.Int64
	batchesSent   atomic.Int64
	createsFailed atomic.Int64
	updatesSent   atomic.Int64
	updatesFailed atomic.Int64
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
		updateBuffer: make([]*UpdateWithID, 0, maxBatchSize),
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
		logger.Warn("BatchedRecoveryReporter shutdown, dropping attempt",
			zap.String("task_id", req.TaskID),
		)
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
	logger.Debug("Recovery attempt queued for batch",
		zap.String("task_id", req.TaskID),
		zap.String("domain", req.Domain),
		zap.String("status", req.Status),
		zap.Int("buffer_size", len(r.createBuffer)),
	)

	// Flush immediately if batch is full
	if len(r.createBuffer) >= maxBatchSize {
		go r.flushCreates()
	}
}

// UpdateAttemptAsync queues a recovery attempt update for batched sending
func (r *BatchedRecoveryReporter) UpdateAttemptAsync(ctx context.Context, attemptID string, req *UpdateAttemptRequest) {
	if r.shutdown.Load() || attemptID == "" {
		return
	}

	r.updateBufferMu.Lock()
	defer r.updateBufferMu.Unlock()

	// Check buffer limit
	if len(r.updateBuffer) >= maxPendingAttempts {
		r.updatesFailed.Add(1)
		return
	}

	r.updateBuffer = append(r.updateBuffer, &UpdateWithID{
		AttemptID: attemptID,
		Request:   req,
	})

	// Flush immediately if batch is full
	if len(r.updateBuffer) >= maxBatchSize {
		go r.flushUpdates()
	}
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
			r.flushUpdates()
		case <-statsTicker.C:
			r.logStats()
		case <-r.stopCh:
			r.flushCreates() // Final flush
			r.flushUpdates()
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

	logger.Debug("Flushing recovery attempts batch to orchestrator",
		zap.Int("batch_size", len(batch)),
		zap.String("orchestrator_url", r.orchestratorURL),
	)

	// Send batch to orchestrator
	if err := r.sendCreateBatch(batch); err != nil {
		r.createsFailed.Add(int64(len(batch)))
		logger.Warn("Failed to send recovery batch",
			zap.Int("batch_size", len(batch)),
			zap.Error(err),
		)
		return
	}

	logger.Debug("Recovery batch sent successfully",
		zap.Int("batch_size", len(batch)),
	)
	r.createsSent.Add(int64(len(batch)))
	r.batchesSent.Add(1)
}

// flushUpdates sends buffered updates as a batch to orchestrator
func (r *BatchedRecoveryReporter) flushUpdates() {
	r.updateBufferMu.Lock()
	if len(r.updateBuffer) == 0 {
		r.updateBufferMu.Unlock()
		return
	}

	// Take the batch and clear buffer
	batch := r.updateBuffer
	r.updateBuffer = make([]*UpdateWithID, 0, maxBatchSize)
	r.updateBufferMu.Unlock()

	// Send batch to orchestrator
	if err := r.sendUpdateBatch(batch); err != nil {
		r.updatesFailed.Add(int64(len(batch)))
		logger.Warn("Failed to send recovery update batch",
			zap.Int("batch_size", len(batch)),
			zap.Error(err),
		)
		return
	}

	r.updatesSent.Add(int64(len(batch)))
}

// BatchCreateRequest matches the orchestrator API
type BatchCreateRequest struct {
	Attempts []*CreateAttemptRequest `json:"attempts"`
}

// BatchUpdateRequest for batch updates
type BatchUpdateRequest struct {
	Updates []BatchUpdateItem `json:"updates"`
}

// BatchUpdateItem is a single update in a batch
type BatchUpdateItem struct {
	AttemptID    string `json:"attempt_id"`
	Action       string `json:"action"`
	Source       string `json:"source"`
	Status       string `json:"status"`
	RuleID       string `json:"rule_id,omitempty"`
	AIReasoning  string `json:"ai_reasoning,omitempty"`
	RetryDelayMs int    `json:"retry_delay_ms,omitempty"`
	DurationMs   int    `json:"duration_ms,omitempty"`
	ProxyID      string `json:"proxy_id,omitempty"`
	ProxyTier    int    `json:"proxy_tier,omitempty"`
	TierFrom     int    `json:"tier_from,omitempty"`
	TierTo       int    `json:"tier_to,omitempty"`
	RetryCount   int    `json:"retry_count,omitempty"`
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

// sendUpdateBatch sends a batch of updates to orchestrator
func (r *BatchedRecoveryReporter) sendUpdateBatch(batch []*UpdateWithID) error {
	url := fmt.Sprintf("%s/api/v1/internal/recovery/attempts/batch-update", r.orchestratorURL)

	items := make([]BatchUpdateItem, 0, len(batch))
	for _, u := range batch {
		items = append(items, BatchUpdateItem{
			AttemptID:    u.AttemptID,
			Action:       u.Request.Action,
			Source:       u.Request.Source,
			Status:       u.Request.Status,
			RuleID:       u.Request.RuleID,
			AIReasoning:  u.Request.AIReasoning,
			RetryDelayMs: u.Request.RetryDelayMs,
			DurationMs:   u.Request.DurationMs,
			ProxyID:      u.Request.ProxyID,
			ProxyTier:    u.Request.ProxyTier,
			TierFrom:     u.Request.TierFrom,
			TierTo:       u.Request.TierTo,
			RetryCount:   u.Request.RetryCount,
		})
	}

	reqBody := BatchUpdateRequest{Updates: items}
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
	createFails := r.createsFailed.Load()
	updates := r.updatesSent.Load()
	updateFails := r.updatesFailed.Load()

	r.createBufferMu.Lock()
	createBufferLen := len(r.createBuffer)
	r.createBufferMu.Unlock()

	r.updateBufferMu.Lock()
	updateBufferLen := len(r.updateBuffer)
	r.updateBufferMu.Unlock()

	if creates > 0 || createFails > 0 || updates > 0 {
		avgBatchSize := float64(0)
		if batches > 0 {
			avgBatchSize = float64(creates) / float64(batches)
		}
		logger.Info("Recovery reporter stats",
			zap.Int64("creates_sent", creates),
			zap.Int64("updates_sent", updates),
			zap.Int64("batches_sent", batches),
			zap.Float64("avg_batch_size", avgBatchSize),
			zap.Int64("creates_failed", createFails),
			zap.Int64("updates_failed", updateFails),
			zap.Int("create_buffer_len", createBufferLen),
			zap.Int("update_buffer_len", updateBufferLen),
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
			zap.Int64("total_updates", r.updatesSent.Load()),
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
	createBufferLen := len(r.createBuffer)
	r.createBufferMu.Unlock()

	r.updateBufferMu.Lock()
	updateBufferLen := len(r.updateBuffer)
	r.updateBufferMu.Unlock()

	return map[string]interface{}{
		"creates_sent":      r.createsSent.Load(),
		"updates_sent":      r.updatesSent.Load(),
		"batches_sent":      r.batchesSent.Load(),
		"creates_failed":    r.createsFailed.Load(),
		"updates_failed":    r.updatesFailed.Load(),
		"create_buffer_len": createBufferLen,
		"update_buffer_len": updateBufferLen,
		"shutdown":          r.shutdown.Load(),
	}
}
