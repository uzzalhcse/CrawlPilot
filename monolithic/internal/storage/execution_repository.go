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

func (r *ExecutionRepository) GetByID(ctx context.Context, id string) (*models.WorkflowExecution, error) {
	query := `
		SELECT id, workflow_id, status, started_at, completed_at, error, metadata
		FROM workflow_executions
		WHERE id = $1
	`

	var execution models.WorkflowExecution
	var metadataJSON []byte
	var completedAt *time.Time
	var errorMsg *string

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&execution.ID,
		&execution.WorkflowID,
		&execution.Status,
		&execution.StartedAt,
		&completedAt,
		&errorMsg,
		&metadataJSON,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	if completedAt != nil {
		execution.CompletedAt = completedAt
	}

	if errorMsg != nil {
		execution.Error = *errorMsg
	}

	// Parse metadata
	if len(metadataJSON) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(metadataJSON, &metadata); err == nil {
			if statsData, ok := metadata["stats"]; ok {
				statsBytes, _ := json.Marshal(statsData)
				json.Unmarshal(statsBytes, &execution.Stats)
			}
		}
	}

	return &execution, nil
}

func (r *ExecutionRepository) List(ctx context.Context, workflowID, status string, limit, offset int) ([]*models.WorkflowExecution, error) {
	query := `
		SELECT id, workflow_id, status, started_at, completed_at, error, metadata
		FROM workflow_executions
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if workflowID != "" {
		query += fmt.Sprintf(" AND workflow_id = $%d", argIndex)
		args = append(args, workflowID)
		argIndex++
	}

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	query += " ORDER BY started_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	var executions []*models.WorkflowExecution

	for rows.Next() {
		var execution models.WorkflowExecution
		var metadataJSON []byte
		var completedAt *time.Time
		var errorMsg *string

		err := rows.Scan(
			&execution.ID,
			&execution.WorkflowID,
			&execution.Status,
			&execution.StartedAt,
			&completedAt,
			&errorMsg,
			&metadataJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}

		if completedAt != nil {
			execution.CompletedAt = completedAt
		}

		if errorMsg != nil {
			execution.Error = *errorMsg
		}

		// Parse metadata
		if len(metadataJSON) > 0 {
			var metadata map[string]interface{}
			if err := json.Unmarshal(metadataJSON, &metadata); err == nil {
				if statsData, ok := metadata["stats"]; ok {
					statsBytes, _ := json.Marshal(statsData)
					json.Unmarshal(statsBytes, &execution.Stats)
				}
			}
		}

		executions = append(executions, &execution)
	}

	return executions, nil
}

func (r *ExecutionRepository) Count(ctx context.Context, workflowID, status string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM workflow_executions
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if workflowID != "" {
		query += fmt.Sprintf(" AND workflow_id = $%d", argIndex)
		args = append(args, workflowID)
		argIndex++
	}

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
	}

	var count int
	err := r.db.Pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count executions: %w", err)
	}

	return count, nil
}
