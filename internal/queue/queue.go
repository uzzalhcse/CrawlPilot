package queue

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

type URLQueue struct {
	db       *storage.PostgresDB
	workerID string
}

func NewURLQueue(db *storage.PostgresDB) *URLQueue {
	return &URLQueue{
		db:       db,
		workerID: uuid.New().String(),
	}
}

// Enqueue adds a URL to the queue with deduplication
func (q *URLQueue) Enqueue(ctx context.Context, item *models.URLQueueItem) error {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}

	// Calculate URL hash for deduplication
	item.URLHash = q.hashURL(item.URL)

	// Handle empty metadata - use NULL for JSONB column
	var metadata interface{}
	if item.Metadata == "" {
		metadata = nil
	} else {
		metadata = item.Metadata
	}

	const query = `
		INSERT INTO url_queue (id, execution_id, url, url_hash, depth, priority, status, parent_url_id, discovered_by_node, url_type, marker, phase_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
		ON CONFLICT (execution_id, url_hash) DO UPDATE SET
			priority = GREATEST(url_queue.priority, EXCLUDED.priority),
			status = CASE
				WHEN url_queue.status = 'failed' THEN 'pending'
				ELSE url_queue.status
			END
		RETURNING id
	` // The executor will set it appropriately based on the node
	// if item.URLType == "" {
	// 	item.URLType = "page"
	// }

	var returnedID string
	err := q.db.Pool.QueryRow(ctx, query,
		item.ID,
		item.ExecutionID,
		item.URL,
		item.URLHash,
		item.Depth,
		item.Priority,
		models.QueueItemStatusPending,
		item.ParentURLID,
		item.DiscoveredByNode,
		item.URLType,
		item.Marker,
		item.PhaseID,
		metadata,
	).Scan(&returnedID)

	if err != nil {
		if err == pgx.ErrNoRows {
			// Item already exists (duplicate), not an error
			return nil
		}
		return fmt.Errorf("failed to enqueue URL: %w", err)
	}

	// If an item was returned, it means it was inserted or updated.
	// We don't need to update item.CreatedAt here as it's not returned.
	// The ID is returned, but item.ID is already set.
	return nil
}

// EnqueueBatch adds multiple URLs to the queue efficiently
func (q *URLQueue) EnqueueBatch(ctx context.Context, items []*models.URLQueueItem) error {
	if len(items) == 0 {
		return nil
	}

	batch := &pgx.Batch{}

	for _, item := range items {
		if item.ID == "" {
			item.ID = uuid.New().String()
		}
		item.URLHash = q.hashURL(item.URL)

		// Handle empty metadata - use NULL for JSONB column
		var metadata interface{}
		if item.Metadata == "" {
			metadata = nil
		} else {
			metadata = item.Metadata
		}

		// Don't set default url_type - let it remain empty if not provided
		// The executor will set it appropriately based on the node
		// if item.URLType == "" {
		// 	item.URLType = "page"
		// }

		const query = `
			INSERT INTO url_queue (id, execution_id, url, url_hash, depth, priority, status, parent_url_id, discovered_by_node, url_type, marker, phase_id, metadata, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
			ON CONFLICT (execution_id, url_hash) DO UPDATE SET
				priority = GREATEST(url_queue.priority, EXCLUDED.priority),
				status = CASE
					WHEN url_queue.status = 'failed' THEN 'pending'
					ELSE url_queue.status
				END
			RETURNING id
		`
		batch.Queue(query,
			item.ID,
			item.ExecutionID,
			item.URL,
			item.URLHash,
			item.Depth,
			item.Priority,
			models.QueueItemStatusPending,
			item.ParentURLID,
			item.DiscoveredByNode,
			item.URLType,
			item.Marker,
			item.PhaseID,
			metadata,
		)
	}

	br := q.db.Pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(items); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to enqueue batch at index %d: %w", i, err)
		}
	}

	return nil
}

// Dequeue retrieves and locks the next URL to process
func (q *URLQueue) Dequeue(ctx context.Context, executionID string) (*models.URLQueueItem, error) {
	// Use advisory lock to prevent race conditions
	query := `
		UPDATE url_queue
		SET status = $1, locked_at = NOW(), locked_by = $2
		WHERE id = (
			SELECT id FROM url_queue
			WHERE execution_id = $3
			AND status = $4
			AND (locked_at IS NULL OR locked_at < NOW() - INTERVAL '5 minutes')
			ORDER BY priority DESC, created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, execution_id, url, url_hash, depth, priority, status, parent_url_id, discovered_by_node, url_type, marker, phase_id, retry_count, error, metadata, created_at, processed_at, locked_at, locked_by
	`

	var item models.URLQueueItem
	var errorVal, metadataVal *string
	err := q.db.Pool.QueryRow(ctx, query,
		models.QueueItemStatusProcessing,
		q.workerID,
		executionID,
		models.QueueItemStatusPending,
	).Scan(
		&item.ID,
		&item.ExecutionID,
		&item.URL,
		&item.URLHash,
		&item.Depth,
		&item.Priority,
		&item.Status,
		&item.ParentURLID,
		&item.DiscoveredByNode,
		&item.URLType,
		&item.Marker,
		&item.PhaseID,
		&item.RetryCount,
		&errorVal,
		&metadataVal,
		&item.CreatedAt,
		&item.ProcessedAt,
		&item.LockedAt,
		&item.LockedBy,
	)

	// Handle NULL values
	if errorVal != nil {
		item.Error = *errorVal
	}
	if metadataVal != nil {
		item.Metadata = *metadataVal
	}

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No items available
		}
		return nil, fmt.Errorf("failed to dequeue URL: %w", err)
	}

	return &item, nil
}

// MarkCompleted marks a URL as successfully processed
func (q *URLQueue) MarkCompleted(ctx context.Context, id string) error {
	query := `
		UPDATE url_queue
		SET status = $1, processed_at = NOW()
		WHERE id = $2 AND locked_by = $3
	`

	result, err := q.db.Pool.Exec(ctx, query, models.QueueItemStatusCompleted, id, q.workerID)
	if err != nil {
		return fmt.Errorf("failed to mark URL as completed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("URL not found or not locked by this worker: %s", id)
	}

	return nil
}

// MarkFailed marks a URL as failed
func (q *URLQueue) MarkFailed(ctx context.Context, id string, errMsg string, retry bool) error {
	var query string
	if retry {
		query = `
			UPDATE url_queue
			SET status = $1, retry_count = retry_count + 1, error = $2, locked_at = NULL, locked_by = NULL
			WHERE id = $3 AND locked_by = $4
		`
	} else {
		query = `
			UPDATE url_queue
			SET status = $1, retry_count = retry_count + 1, error = $2, processed_at = NOW()
			WHERE id = $3 AND locked_by = $4
		`
	}

	status := models.QueueItemStatusFailed
	if retry {
		status = models.QueueItemStatusPending
	}

	result, err := q.db.Pool.Exec(ctx, query, status, errMsg, id, q.workerID)
	if err != nil {
		return fmt.Errorf("failed to mark URL as failed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("URL not found or not locked by this worker: %s", id)
	}

	return nil
}

// GetStats returns queue statistics for an execution
func (q *URLQueue) GetStats(ctx context.Context, executionID string) (map[string]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM url_queue
		WHERE execution_id = $1
		GROUP BY status
	`

	rows, err := q.db.Pool.Query(ctx, query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan queue stats: %w", err)
		}
		stats[status] = count
	}

	return stats, nil
}

// CleanupStaleItems releases items locked for too long
func (q *URLQueue) CleanupStaleItems(ctx context.Context, timeout time.Duration) (int, error) {
	query := `
		UPDATE url_queue
		SET status = $1, locked_at = NULL, locked_by = NULL
		WHERE status = $2
		AND locked_at < $3
	`

	result, err := q.db.Pool.Exec(ctx, query,
		models.QueueItemStatusPending,
		models.QueueItemStatusProcessing,
		time.Now().Add(-timeout),
	)

	if err != nil {
		return 0, fmt.Errorf("failed to cleanup stale items: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// IsDuplicate checks if a URL already exists in the queue for this execution
func (q *URLQueue) IsDuplicate(ctx context.Context, executionID, url string) (bool, error) {
	urlHash := q.hashURL(url)

	query := `SELECT EXISTS(SELECT 1 FROM url_queue WHERE execution_id = $1 AND url_hash = $2)`

	var exists bool
	err := q.db.Pool.QueryRow(ctx, query, executionID, urlHash).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check duplicate: %w", err)
	}

	return exists, nil
}

// hashURL generates a SHA-256 hash of the URL for deduplication
func (q *URLQueue) hashURL(url string) string {
	hash := sha256.Sum256([]byte(url))
	return fmt.Sprintf("%x", hash)
}

// GetPendingCount returns the number of pending items for an execution
func (q *URLQueue) GetPendingCount(ctx context.Context, executionID string) (int, error) {
	query := `SELECT COUNT(*) FROM url_queue WHERE execution_id = $1 AND status = $2`

	var count int
	err := q.db.Pool.QueryRow(ctx, query, executionID, models.QueueItemStatusPending).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get pending count: %w", err)
	}

	return count, nil
}

// HasPendingDiscoveryURLs checks if there are any pending URLs in discovery phase
func (q *URLQueue) HasPendingDiscoveryURLs(ctx context.Context, executionID string, discoveryNodes []models.Node) (bool, error) {
	// Build list of discovery URL types
	var discoveryURLTypes []string
	discoveryURLTypes = append(discoveryURLTypes, "", "start", "page") // Include start URLs and default type

	for _, node := range discoveryNodes {
		if urlType, ok := node.Params["url_type"].(string); ok && urlType != "" {
			discoveryURLTypes = append(discoveryURLTypes, urlType)
		}
	}

	// Get the last discovery node URL types (these should NOT be counted as pending discovery)
	lastNodeTypes := make(map[string]bool)
	hasDependents := make(map[string]bool)
	for _, node := range discoveryNodes {
		for _, depID := range node.Dependencies {
			hasDependents[depID] = true
		}
	}

	for _, node := range discoveryNodes {
		if !hasDependents[node.ID] {
			if urlType, ok := node.Params["url_type"].(string); ok && urlType != "" {
				lastNodeTypes[urlType] = true
			}
		}
	}

	// Check if any URLs with discovery types (excluding last node types) are still pending or processing
	query := `
		SELECT EXISTS(
			SELECT 1 FROM url_queue 
			WHERE execution_id = $1 
			AND status IN ($2, $3)
			AND url_type = ANY($4)
		)
	`

	var hasPending bool
	err := q.db.Pool.QueryRow(ctx, query,
		executionID,
		models.QueueItemStatusPending,
		models.QueueItemStatusProcessing,
		discoveryURLTypes,
	).Scan(&hasPending)

	if err != nil {
		return false, fmt.Errorf("failed to check pending discovery URLs: %w", err)
	}

	// If we found pending URLs, we need to check if they're actually discovery URLs
	// and not extraction URLs (last node types)
	if hasPending && len(lastNodeTypes) > 0 {
		// Do a more refined check - exclude last node types
		var nonLastNodeTypes []string
		for _, urlType := range discoveryURLTypes {
			if !lastNodeTypes[urlType] {
				nonLastNodeTypes = append(nonLastNodeTypes, urlType)
			}
		}

		if len(nonLastNodeTypes) == 0 {
			// All discovery types are from last nodes, so no actual discovery pending
			return false, nil
		}

		// Re-check with refined list
		err = q.db.Pool.QueryRow(ctx, query,
			executionID,
			models.QueueItemStatusPending,
			models.QueueItemStatusProcessing,
			nonLastNodeTypes,
		).Scan(&hasPending)

		if err != nil {
			return false, fmt.Errorf("failed to check pending discovery URLs (refined): %w", err)
		}
	}

	return hasPending, nil
}

// RequeueForLater resets a URL back to pending status with lower priority
func (q *URLQueue) RequeueForLater(ctx context.Context, id string) error {
	query := `
		UPDATE url_queue
		SET status = $1, locked_at = NULL, locked_by = NULL, priority = priority - 1
		WHERE id = $2 AND locked_by = $3
	`

	result, err := q.db.Pool.Exec(ctx, query, models.QueueItemStatusPending, id, q.workerID)
	if err != nil {
		return fmt.Errorf("failed to requeue URL: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("URL not found or not locked by this worker: %s", id)
	}

	return nil
}

// UpdatePhaseID updates the phase ID of a queued item
func (q *URLQueue) UpdatePhaseID(ctx context.Context, id string, phaseID string) error {
	query := `
		UPDATE url_queue
		SET phase_id = $1
		WHERE id = $2
	`

	result, err := q.db.Pool.Exec(ctx, query, phaseID, id)
	if err != nil {
		return fmt.Errorf("failed to update phase ID: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("URL not found: %s", id)
	}

	return nil
}
