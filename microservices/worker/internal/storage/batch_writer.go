package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/uzzalhcse/crawlify/microservices/shared/database"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// ExtractedItem represents a single extracted item for batch writing
type ExtractedItem struct {
	ExecutionID string
	WorkflowID  string
	TaskID      string
	URL         string
	Data        map[string]interface{}
}

// BatchWriterConfig holds configuration for the batch writer
type BatchWriterConfig struct {
	BufferSize   int           // Max items before auto-flush (default: 100)
	FlushTimeout time.Duration // Max time before auto-flush (default: 5s)
}

// DefaultBatchWriterConfig returns optimized defaults for high throughput
// At 10k URLs/sec with 1 item/URL = 10k items/sec
// Buffer of 500 fills in ~50ms, giving ~20 flushes/sec (well within DB capacity)
func DefaultBatchWriterConfig() BatchWriterConfig {
	return BatchWriterConfig{
		BufferSize:   500,             // 5x larger for fewer flushes
		FlushTimeout: 2 * time.Second, // Faster periodic flush
	}
}

// BatchWriter provides async buffered writes for extracted items
// Uses bulk INSERT for efficient database operations
type BatchWriter struct {
	db     *database.DB
	config BatchWriterConfig

	buffer []ExtractedItem
	mu     sync.Mutex

	flushCh  chan struct{}
	doneCh   chan struct{}
	shutdown bool

	// Metrics
	totalFlushed int64
	flushCount   int64
}

// NewBatchWriter creates a new batch writer with the given configuration
func NewBatchWriter(db *database.DB, config BatchWriterConfig) *BatchWriter {
	if config.BufferSize <= 0 {
		config.BufferSize = 100
	}
	if config.FlushTimeout <= 0 {
		config.FlushTimeout = 5 * time.Second
	}

	bw := &BatchWriter{
		db:      db,
		config:  config,
		buffer:  make([]ExtractedItem, 0, config.BufferSize),
		flushCh: make(chan struct{}, 1),
		doneCh:  make(chan struct{}),
	}

	// Start background flush goroutine
	go bw.runFlushLoop()

	logger.Info("Batch writer initialized",
		zap.Int("buffer_size", config.BufferSize),
		zap.Duration("flush_timeout", config.FlushTimeout),
	)

	return bw
}

// Add adds an item to the buffer. Triggers flush if buffer is full.
func (bw *BatchWriter) Add(ctx context.Context, item ExtractedItem) error {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if bw.shutdown {
		return fmt.Errorf("batch writer is shutting down")
	}

	bw.buffer = append(bw.buffer, item)

	// Trigger flush if buffer is full
	if len(bw.buffer) >= bw.config.BufferSize {
		select {
		case bw.flushCh <- struct{}{}:
		default:
			// Flush already pending
		}
	}

	return nil
}

// AddBatch adds multiple items at once (more efficient than individual Add calls)
func (bw *BatchWriter) AddBatch(ctx context.Context, items []ExtractedItem) error {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if bw.shutdown {
		return fmt.Errorf("batch writer is shutting down")
	}

	bw.buffer = append(bw.buffer, items...)

	// Trigger flush if buffer is full
	if len(bw.buffer) >= bw.config.BufferSize {
		select {
		case bw.flushCh <- struct{}{}:
		default:
		}
	}

	return nil
}

// runFlushLoop runs the background flush loop
func (bw *BatchWriter) runFlushLoop() {
	ticker := time.NewTicker(bw.config.FlushTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-bw.flushCh:
			// Triggered by buffer full
			if err := bw.flush(); err != nil {
				logger.Error("Batch flush failed", zap.Error(err))
			}

		case <-ticker.C:
			// Periodic flush
			bw.mu.Lock()
			hasItems := len(bw.buffer) > 0
			bw.mu.Unlock()

			if hasItems {
				if err := bw.flush(); err != nil {
					logger.Error("Periodic batch flush failed", zap.Error(err))
				}
			}

		case <-bw.doneCh:
			// Shutdown - final flush
			if err := bw.flush(); err != nil {
				logger.Error("Final batch flush failed", zap.Error(err))
			}
			return
		}
	}
}

// flush writes all buffered items to the database using bulk INSERT
func (bw *BatchWriter) flush() error {
	bw.mu.Lock()
	if len(bw.buffer) == 0 {
		bw.mu.Unlock()
		return nil
	}

	// Copy and clear buffer
	items := make([]ExtractedItem, len(bw.buffer))
	copy(items, bw.buffer)
	bw.buffer = bw.buffer[:0]
	bw.mu.Unlock()

	// Execute bulk insert
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := bw.bulkInsert(ctx, items)
	if err != nil {
		// On failure, try to re-add items to buffer (best effort)
		bw.mu.Lock()
		bw.buffer = append(items, bw.buffer...)
		bw.mu.Unlock()
		return err
	}

	bw.mu.Lock()
	bw.totalFlushed += int64(len(items))
	bw.flushCount++
	bw.mu.Unlock()

	logger.Info("Batch flushed",
		zap.Int("items", len(items)),
		zap.Int64("total_flushed", bw.totalFlushed),
	)

	return nil
}

// bulkInsert performs efficient bulk insert using pgx.Batch
func (bw *BatchWriter) bulkInsert(ctx context.Context, items []ExtractedItem) error {
	if len(items) == 0 {
		return nil
	}

	// Use pgx.Batch for efficient bulk operations
	batch := &pgx.Batch{}

	query := `
		INSERT INTO extracted_items (execution_id, workflow_id, task_id, url, data)
		VALUES ($1, $2, $3, $4, $5)
	`

	for _, item := range items {
		dataBytes, err := json.Marshal(item.Data)
		if err != nil {
			logger.Warn("Failed to marshal item data", zap.Error(err))
			continue
		}
		// Convert to string for PgBouncer simple protocol compatibility
		dataJSON := string(dataBytes)

		batch.Queue(query, item.ExecutionID, item.WorkflowID, item.TaskID, item.URL, dataJSON)
	}

	// Execute batch
	results := bw.db.Pool.SendBatch(ctx, batch)
	defer results.Close()

	// Check results
	var lastErr error
	for i := 0; i < batch.Len(); i++ {
		_, err := results.Exec()
		if err != nil {
			logger.Warn("Batch item insert failed",
				zap.Int("index", i),
				zap.Error(err),
			)
			lastErr = err
		}
	}

	return lastErr
}

// Flush forces an immediate flush of all buffered items
func (bw *BatchWriter) Flush() error {
	return bw.flush()
}

// Close gracefully shuts down the batch writer
// Waits for pending flush and final buffer flush
func (bw *BatchWriter) Close() error {
	bw.mu.Lock()
	if bw.shutdown {
		bw.mu.Unlock()
		return nil
	}
	bw.shutdown = true
	bw.mu.Unlock()

	// Signal shutdown and wait for flush loop to finish
	close(bw.doneCh)

	// Give some time for final flush
	time.Sleep(100 * time.Millisecond)

	logger.Info("Batch writer closed",
		zap.Int64("total_flushed", bw.totalFlushed),
		zap.Int64("flush_count", bw.flushCount),
	)

	return nil
}

// Stats returns current batch writer statistics
func (bw *BatchWriter) Stats() map[string]interface{} {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	return map[string]interface{}{
		"buffer_size":   len(bw.buffer),
		"buffer_cap":    bw.config.BufferSize,
		"total_flushed": bw.totalFlushed,
		"flush_count":   bw.flushCount,
		"flush_timeout": bw.config.FlushTimeout.String(),
		"shutdown":      bw.shutdown,
	}
}
