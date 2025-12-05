package dedup

import "context"

// Deduplicator defines the interface for URL deduplication strategies
type Deduplicator interface {
	// IsDuplicate checks if a URL has been processed for this execution/phase
	// Returns true if duplicate (should skip), false if new (should process)
	IsDuplicate(ctx context.Context, executionID, phaseID, url string) (bool, error)
}

// Ensure both implementations satisfy the interface
var _ Deduplicator = (*URLDeduplicator)(nil)
var _ Deduplicator = (*BloomDeduplicator)(nil)
