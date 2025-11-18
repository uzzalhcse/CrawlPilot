package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

type ExecutionRepository struct {
	db *PostgresDB
}

func NewExecutionRepository(db *PostgresDB) *ExecutionRepository {
	return &ExecutionRepository{db: db}
}

func (r *ExecutionRepository) Create(ctx context.Context, execution *models.WorkflowExecution) error {
	if execution.ID == "" {
		execution.ID = uuid.New().String()
	}

	if execution.StartedAt.IsZero() {
		execution.StartedAt = time.Now()
	}

	if execution.Status == "" {
		execution.Status = models.ExecutionStatusRunning
	}

	// Prepare metadata JSONB with stats and context
	metadata := map[string]interface{}{
		"stats":   execution.Stats,
		"context": execution.Context,
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO workflow_executions (id, workflow_id, status, started_at, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING started_at
	`

	err = r.db.Pool.QueryRow(ctx, query,
		execution.ID,
		execution.WorkflowID,
		execution.Status,
		execution.StartedAt,
		metadataJSON,
	).Scan(&execution.StartedAt)

	if err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	return nil
}

func (r *ExecutionRepository) UpdateStatus(ctx context.Context, id string, status models.ExecutionStatus, err string) error {
	var query string
	var args []interface{}

	if status == models.ExecutionStatusCompleted || status == models.ExecutionStatusFailed || status == models.ExecutionStatusCancelled {
		query = `
			UPDATE workflow_executions
			SET status = $2, completed_at = NOW(), error = $3
			WHERE id = $1
		`
		args = []interface{}{id, status, err}
	} else {
		query = `
			UPDATE workflow_executions
			SET status = $2
			WHERE id = $1
		`
		args = []interface{}{id, status}
	}

	result, cmdErr := r.db.Pool.Exec(ctx, query, args...)
	if cmdErr != nil {
		return fmt.Errorf("failed to update execution status: %w", cmdErr)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("execution not found: %s", id)
	}

	return nil
}

func (r *ExecutionRepository) UpdateStats(ctx context.Context, id string, stats models.ExecutionStats) error {
	// Store stats in metadata as JSONB
	statsWrapper := map[string]interface{}{
		"stats": stats,
	}

	statsJSON, err := json.Marshal(statsWrapper)
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	query := `
		UPDATE workflow_executions
		SET metadata = COALESCE(metadata, '{}'::jsonb) || $2::jsonb
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query, id, statsJSON)
	if err != nil {
		return fmt.Errorf("failed to update execution stats: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("execution not found: %s", id)
	}

	return nil
}
