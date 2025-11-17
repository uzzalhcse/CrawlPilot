package storage

import (
	"context"
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

	query := `
		INSERT INTO workflow_executions (id, workflow_id, status, started_at, stats, context)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING started_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		execution.ID,
		execution.WorkflowID,
		execution.Status,
		execution.StartedAt,
		execution.Stats,
		execution.Context,
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
	query := `
		UPDATE workflow_executions
		SET stats = $2
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query, id, stats)
	if err != nil {
		return fmt.Errorf("failed to update execution stats: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("execution not found: %s", id)
	}

	return nil
}
