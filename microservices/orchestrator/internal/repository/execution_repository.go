package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/uzzalhcse/crawlify/microservices/shared/database"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
)

// postgresExecutionRepo implements ExecutionRepository using PostgreSQL
type postgresExecutionRepo struct {
	db *database.DB
}

// NewExecutionRepository creates a new PostgreSQL execution repository
func NewExecutionRepository(db *database.DB) ExecutionRepository {
	return &postgresExecutionRepo{db: db}
}

func (r *postgresExecutionRepo) Create(ctx context.Context, execution *models.Execution) error {
	if execution.ID == "" {
		execution.ID = uuid.New().String()
	}

	execution.StartedAt = time.Now()
	execution.Status = "running"

	// Marshal metadata to JSON string for PgBouncer simple protocol compatibility
	// Simple protocol requires string for JSONB columns
	var metadataJSON string
	if execution.Metadata != nil {
		jsonBytes, err := json.Marshal(execution.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = string(jsonBytes)
	} else {
		metadataJSON = "{}"
	}

	query := `
		INSERT INTO workflow_executions (id, workflow_id, status, started_at, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		execution.ID,
		execution.WorkflowID,
		execution.Status,
		execution.StartedAt,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	return nil
}

func (r *postgresExecutionRepo) Get(ctx context.Context, id string) (*models.Execution, error) {
	query := `
		SELECT id, workflow_id, status, started_at, completed_at, metadata
		FROM workflow_executions
		WHERE id = $1
	`

	var execution models.Execution

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&execution.ID,
		&execution.WorkflowID,
		&execution.Status,
		&execution.StartedAt,
		&execution.CompletedAt,
		&execution.Metadata,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("execution not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	return &execution, nil
}

func (r *postgresExecutionRepo) List(ctx context.Context, workflowID string, filters ListFilters) ([]*models.Execution, error) {
	query := `
		SELECT id, workflow_id, status, started_at, completed_at, metadata
		FROM workflow_executions
		WHERE workflow_id = $1
	`

	args := []interface{}{workflowID}
	argPos := 2

	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, filters.Status)
		argPos++
	}

	query += " ORDER BY started_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filters.Limit)
		argPos++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filters.Offset)
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	executions := make([]*models.Execution, 0)

	for rows.Next() {
		var execution models.Execution
		err := rows.Scan(
			&execution.ID,
			&execution.WorkflowID,
			&execution.Status,
			&execution.StartedAt,
			&execution.CompletedAt,
			&execution.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}
		executions = append(executions, &execution)
	}

	return executions, nil
}

func (r *postgresExecutionRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `
		UPDATE workflow_executions
		SET status = $2
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update execution status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("execution not found: %s", id)
	}

	return nil
}

func (r *postgresExecutionRepo) UpdateStats(ctx context.Context, id string, stats ExecutionStats) error {
	query := `
		UPDATE workflow_executions
		SET 
			urls_processed = COALESCE(urls_processed, 0) + $2,
			urls_discovered = COALESCE(urls_discovered, 0) + $3,
			items_extracted = COALESCE(items_extracted, 0) + $4,
			errors = COALESCE(errors, 0) + $5
		WHERE id = $1
	`

	_, err := r.db.Pool.Exec(ctx, query,
		id,
		stats.URLsProcessed,
		stats.URLsDiscovered,
		stats.ItemsExtracted,
		stats.Errors,
	)

	if err != nil {
		return fmt.Errorf("failed to update execution stats: %w", err)
	}

	return nil
}

// BatchUpdateStats updates multiple execution statistics in a single batch operation
// Critical for high-throughput scenarios - reduces 10k DB operations to 1
func (r *postgresExecutionRepo) BatchUpdateStats(ctx context.Context, updates []BatchExecutionStats) error {
	if len(updates) == 0 {
		return nil
	}

	// Use pgx.Batch for efficient batch operations
	batch := &pgx.Batch{}

	query := `
		UPDATE workflow_executions
		SET 
			urls_processed = COALESCE(urls_processed, 0) + $2,
			urls_discovered = COALESCE(urls_discovered, 0) + $3,
			items_extracted = COALESCE(items_extracted, 0) + $4,
			errors = COALESCE(errors, 0) + $5
		WHERE id = $1
	`

	for _, update := range updates {
		batch.Queue(query,
			update.ExecutionID,
			update.URLsProcessed,
			update.URLsDiscovered,
			update.ItemsExtracted,
			update.Errors,
		)
	}

	// Execute batch
	results := r.db.Pool.SendBatch(ctx, batch)
	defer results.Close()

	// Check for errors in batch results
	for i := 0; i < batch.Len(); i++ {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("batch update failed at index %d: %w", i, err)
		}
	}

	return nil
}

func (r *postgresExecutionRepo) Complete(ctx context.Context, id string, status string) error {
	now := time.Now()

	query := `
		UPDATE workflow_executions
		SET status = $2, completed_at = $3
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query, id, status, now)
	if err != nil {
		return fmt.Errorf("failed to complete execution: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("execution not found: %s", id)
	}

	return nil
}
