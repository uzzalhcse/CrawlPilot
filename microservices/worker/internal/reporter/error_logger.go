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
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

const (
	// errorFlushInterval is how often we flush errors to orchestrator
	errorFlushInterval = 5 * time.Second
	// maxErrorsPerExecution is the maximum errors to store per execution
	maxErrorsPerExecution = 1000
)

// BatchedErrorLogger aggregates errors locally and flushes periodically
// Prevents DB overload from high error rates
type BatchedErrorLogger struct {
	orchestratorURL string
	httpClient      *http.Client

	// Local buffers (per execution ID)
	buffers sync.Map // map[executionID]*errorBuffer

	// Control
	stopCh   chan struct{}
	doneCh   chan struct{}
	wg       sync.WaitGroup
	shutdown atomic.Bool
}

// errorBuffer holds errors for a single execution
type errorBuffer struct {
	mu     sync.Mutex
	errors []models.ExecutionError
	count  atomic.Int64 // Total logged (may exceed stored if maxed)
}

// ErrorEntry is the input for logging an error
type ErrorEntry struct {
	ExecutionID string
	URL         string
	ErrorType   string // timeout, blocked, parse_error, network, extraction
	Message     string
	PhaseID     string
	RetryCount  int
}

// NewBatchedErrorLogger creates a new batched error logger
func NewBatchedErrorLogger(orchestratorURL string) *BatchedErrorLogger {
	l := &BatchedErrorLogger{
		orchestratorURL: orchestratorURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	// Start background flush goroutine
	l.wg.Add(1)
	go l.runFlushLoop()

	logger.Info("Batched error logger initialized",
		zap.String("orchestrator_url", orchestratorURL),
		zap.Duration("flush_interval", errorFlushInterval),
		zap.Int("max_per_execution", maxErrorsPerExecution),
	)

	return l
}

// Log records an error for an execution (non-blocking)
func (l *BatchedErrorLogger) Log(entry ErrorEntry) {
	if l.shutdown.Load() {
		return
	}

	// Get or create buffer for this execution
	bufferI, _ := l.buffers.LoadOrStore(entry.ExecutionID, &errorBuffer{
		errors: make([]models.ExecutionError, 0, 100),
	})
	buf := bufferI.(*errorBuffer)

	buf.mu.Lock()
	defer buf.mu.Unlock()

	// Track total count
	buf.count.Add(1)

	// Only store if under limit
	if len(buf.errors) < maxErrorsPerExecution {
		buf.errors = append(buf.errors, models.ExecutionError{
			ExecutionID: entry.ExecutionID,
			URL:         entry.URL,
			ErrorType:   entry.ErrorType,
			Message:     entry.Message,
			PhaseID:     entry.PhaseID,
			RetryCount:  entry.RetryCount,
			CreatedAt:   time.Now(),
		})
	}
}

// runFlushLoop periodically flushes accumulated errors
func (l *BatchedErrorLogger) runFlushLoop() {
	defer l.wg.Done()
	defer close(l.doneCh)

	ticker := time.NewTicker(errorFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.flush()
		case <-l.stopCh:
			l.flush() // Final flush
			return
		}
	}
}

// flush sends all accumulated errors to orchestrator
func (l *BatchedErrorLogger) flush() {
	if l.orchestratorURL == "" {
		return
	}

	// Collect all errors from all buffers
	allErrors := make(map[string][]models.ExecutionError)

	l.buffers.Range(func(key, value any) bool {
		executionID := key.(string)
		buf := value.(*errorBuffer)

		buf.mu.Lock()
		if len(buf.errors) > 0 {
			// Copy and clear
			errors := make([]models.ExecutionError, len(buf.errors))
			copy(errors, buf.errors)
			buf.errors = buf.errors[:0]
			allErrors[executionID] = errors
		}
		buf.mu.Unlock()

		return true
	})

	if len(allErrors) == 0 {
		return
	}

	// Send to orchestrator
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := l.sendBatchedErrors(ctx, allErrors); err != nil {
		logger.Warn("Failed to send batched errors",
			zap.Int("executions", len(allErrors)),
			zap.Error(err),
		)
		// Re-add errors on failure (best effort)
		for execID, errors := range allErrors {
			bufferI, _ := l.buffers.LoadOrStore(execID, &errorBuffer{})
			buf := bufferI.(*errorBuffer)
			buf.mu.Lock()
			buf.errors = append(errors, buf.errors...)
			buf.mu.Unlock()
		}
		return
	}

	totalErrors := 0
	for _, errors := range allErrors {
		totalErrors += len(errors)
	}

	logger.Info("Error batch flushed",
		zap.Int("executions", len(allErrors)),
		zap.Int("total_errors", totalErrors),
	)
}

// BatchedErrorsRequest is the request body for batch errors endpoint
type BatchedErrorsRequest struct {
	Errors    map[string][]models.ExecutionError `json:"errors"` // keyed by execution_id
	Timestamp time.Time                          `json:"timestamp"`
}

// sendBatchedErrors sends errors to orchestrator
func (l *BatchedErrorLogger) sendBatchedErrors(ctx context.Context, allErrors map[string][]models.ExecutionError) error {
	reqBody := BatchedErrorsRequest{
		Errors:    allErrors,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal errors: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/internal/errors/batch", l.orchestratorURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// Close gracefully shuts down the error logger
func (l *BatchedErrorLogger) Close() error {
	if l.shutdown.Swap(true) {
		return nil
	}

	close(l.stopCh)

	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("Batched error logger closed")
	case <-time.After(10 * time.Second):
		logger.Warn("Batched error logger shutdown timeout")
	}

	return nil
}

// Stats returns current logger statistics
func (l *BatchedErrorLogger) Stats() map[string]interface{} {
	count := 0
	var totalQueued int64

	l.buffers.Range(func(key, value any) bool {
		count++
		buf := value.(*errorBuffer)
		buf.mu.Lock()
		totalQueued += int64(len(buf.errors))
		buf.mu.Unlock()
		return true
	})

	return map[string]interface{}{
		"active_executions": count,
		"queued_errors":     totalQueued,
		"flush_interval":    errorFlushInterval.String(),
		"max_per_execution": maxErrorsPerExecution,
		"shutdown":          l.shutdown.Load(),
	}
}
