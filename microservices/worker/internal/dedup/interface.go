package dedup

import "context"

// Deduplicator defines the interface for URL deduplication strategies
type Deduplicator interface {
	// IsDuplicate checks if a URL has been processed for this execution/phase
	// Returns true if duplicate (should skip), false if new (should process)
	IsDuplicate(ctx context.Context, executionID, phaseID, url string) (bool, error)
}

// BatchDeduplicator extends Deduplicator with batch operations for higher throughput
// Implementations can optionally implement this for Redis pipeline optimization
type BatchDeduplicator interface {
	Deduplicator
	// FilterDuplicatesBatch checks multiple URLs and returns only unique ones (new URLs)
	// Uses Redis pipeline to execute all SetNX ops in minimal round-trips
	// maxBatchSize limits the number of keys per pipeline operation (0 = default 1000)
	FilterDuplicatesBatch(ctx context.Context, executionID, phaseID string, urls []string, maxBatchSize int) ([]string, error)
}

// Ensure both implementations satisfy the interface
var _ Deduplicator = (*URLDeduplicator)(nil)
var _ Deduplicator = (*BloomDeduplicator)(nil)

// URLDeduplicator also satisfies BatchDeduplicator
var _ BatchDeduplicator = (*URLDeduplicator)(nil)
