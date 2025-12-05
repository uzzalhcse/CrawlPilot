package storage

import "context"

// Writer defines the interface for extracted items storage
// Implementations: BatchWriter (pgx.Batch), CopyWriter (COPY protocol)
type Writer interface {
	// Add adds a single item to the buffer
	Add(ctx context.Context, item ExtractedItem) error

	// AddBatch adds multiple items at once
	AddBatch(ctx context.Context, items []ExtractedItem) error

	// Flush forces immediate write of buffered items
	Flush() error

	// Close gracefully shuts down the writer
	Close() error

	// Stats returns current writer statistics
	Stats() map[string]interface{}
}

// Ensure implementations satisfy the interface
var _ Writer = (*BatchWriter)(nil)
var _ Writer = (*CopyWriter)(nil)
