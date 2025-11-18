package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

type NodeExecutionRepository struct {
	db *PostgresDB
}

func NewNodeExecutionRepository(db *PostgresDB) *NodeExecutionRepository {
	return &NodeExecutionRepository{db: db}
}

// Create creates a new node execution record
func (r *NodeExecutionRepository) Create(ctx context.Context, nodeExec *models.NodeExecution) error {
	query := `
		INSERT INTO node_executions (id, execution_id, node_id, status, url_id, parent_node_execution_id, 
									 node_type, started_at, input, retry_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	if nodeExec.ID == "" {
		nodeExec.ID = uuid.New().String()
	}

	if nodeExec.StartedAt.IsZero() {
		nodeExec.StartedAt = time.Now()
	}

	_, err := r.db.Pool.Exec(ctx, query,
		nodeExec.ID,
		nodeExec.ExecutionID,
		nodeExec.NodeID,
		nodeExec.Status,
		nodeExec.URLID,
		nodeExec.ParentNodeExecutionID,
		nodeExec.NodeType,
		nodeExec.StartedAt,
		nodeExec.Input,
		nodeExec.RetryCount,
	)

	return err
}

// Update updates an existing node execution record
func (r *NodeExecutionRepository) Update(ctx context.Context, nodeExec *models.NodeExecution) error {
	query := `
		UPDATE node_executions 
		SET status = $1, completed_at = $2, output = $3, error = $4, retry_count = $5,
		    urls_discovered = $6, items_extracted = $7, error_message = $8, duration_ms = $9
		WHERE id = $10
	`

	// Calculate duration if completed
	var durationMs *int
	if nodeExec.CompletedAt != nil && !nodeExec.StartedAt.IsZero() {
		duration := nodeExec.CompletedAt.Sub(nodeExec.StartedAt).Milliseconds()
		durationInt := int(duration)
		durationMs = &durationInt
	}

	_, err := r.db.Pool.Exec(ctx, query,
		nodeExec.Status,
		nodeExec.CompletedAt,
		nodeExec.Output,
		nodeExec.Error,
		nodeExec.RetryCount,
		nodeExec.URLsDiscovered,
		nodeExec.ItemsExtracted,
		nodeExec.ErrorMessage,
		durationMs,
		nodeExec.ID,
	)

	return err
}

// UpdateStatus updates the status of a node execution
func (r *NodeExecutionRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE node_executions SET status = $1 WHERE id = $2`
	_, err := r.db.Pool.Exec(ctx, query, status, id)
	return err
}

// MarkCompleted marks a node execution as completed
func (r *NodeExecutionRepository) MarkCompleted(ctx context.Context, id string, output interface{}) error {
	query := `
		UPDATE node_executions 
		SET status = $1, completed_at = $2, output = $3
		WHERE id = $4
	`

	var outputJSON []byte
	var err error
	if output != nil {
		outputJSON, err = json.Marshal(output)
		if err != nil {
			return err
		}
	}

	_, err = r.db.Pool.Exec(ctx, query, "completed", time.Now(), outputJSON, id)
	return err
}

func (r *NodeExecutionRepository) MarkCompletedWithTime(ctx context.Context, id string, output interface{}, completedAt time.Time) error {
	query := `
		UPDATE node_executions 
		SET status = $1, completed_at = $2, output = $3
		WHERE id = $4
	`

	var outputJSON []byte
	var err error
	if output != nil {
		outputJSON, err = json.Marshal(output)
		if err != nil {
			return err
		}
	}

	_, err = r.db.Pool.Exec(ctx, query, "completed", completedAt, outputJSON, id)
	return err
}

// MarkFailed marks a node execution as failed
func (r *NodeExecutionRepository) MarkFailed(ctx context.Context, id string, errorMsg string) error {
	query := `
		UPDATE node_executions 
		SET status = $1, completed_at = $2, error = $3
		WHERE id = $4
	`

	_, err := r.db.Pool.Exec(ctx, query, "failed", time.Now(), errorMsg, id)
	return err
}

// GetByExecutionID retrieves all node executions for a given execution
func (r *NodeExecutionRepository) GetByExecutionID(ctx context.Context, executionID string) ([]*models.NodeExecution, error) {
	query := `
		SELECT id, execution_id, node_id, status, url_id, parent_node_execution_id, node_type,
		       urls_discovered, items_extracted, error_message, duration_ms,
		       started_at, completed_at, input, output, error, retry_count
		FROM node_executions
		WHERE execution_id = $1
		ORDER BY started_at ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodeExecutions []*models.NodeExecution
	for rows.Next() {
		var ne models.NodeExecution
		var completedAt sql.NullTime
		var input, output sql.NullString
		var errorMsg sql.NullString

		err := rows.Scan(
			&ne.ID,
			&ne.ExecutionID,
			&ne.NodeID,
			&ne.Status,
			&ne.URLID,
			&ne.ParentNodeExecutionID,
			&ne.NodeType,
			&ne.URLsDiscovered,
			&ne.ItemsExtracted,
			&ne.ErrorMessage,
			&ne.DurationMs,
			&ne.StartedAt,
			&completedAt,
			&input,
			&output,
			&errorMsg,
			&ne.RetryCount,
		)
		if err != nil {
			return nil, err
		}

		if completedAt.Valid {
			ne.CompletedAt = &completedAt.Time
		}

		if input.Valid {
			ne.Input = json.RawMessage(input.String)
		}

		if output.Valid {
			ne.Output = json.RawMessage(output.String)
		}

		if errorMsg.Valid {
			ne.Error = errorMsg.String
		}

		nodeExecutions = append(nodeExecutions, &ne)
	}

	return nodeExecutions, rows.Err()
}

// GetByID retrieves a node execution by ID
func (r *NodeExecutionRepository) GetByID(ctx context.Context, id string) (*models.NodeExecution, error) {
	query := `
		SELECT id, execution_id, node_id, status, url_id, parent_node_execution_id, node_type,
		       urls_discovered, items_extracted, error_message, duration_ms,
		       started_at, completed_at, input, output, error, retry_count
		FROM node_executions
		WHERE id = $1
	`

	var ne models.NodeExecution
	var completedAt sql.NullTime
	var input, output sql.NullString
	var errorMsg sql.NullString

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&ne.ID,
		&ne.ExecutionID,
		&ne.NodeID,
		&ne.Status,
		&ne.URLID,
		&ne.ParentNodeExecutionID,
		&ne.NodeType,
		&ne.URLsDiscovered,
		&ne.ItemsExtracted,
		&ne.ErrorMessage,
		&ne.DurationMs,
		&ne.StartedAt,
		&completedAt,
		&input,
		&output,
		&errorMsg,
		&ne.RetryCount,
	)

	if err != nil {
		return nil, err
	}

	if completedAt.Valid {
		ne.CompletedAt = &completedAt.Time
	}

	if input.Valid {
		ne.Input = json.RawMessage(input.String)
	}

	if output.Valid {
		ne.Output = json.RawMessage(output.String)
	}

	if errorMsg.Valid {
		ne.Error = errorMsg.String
	}

	return &ne, nil
}

// GetStatsByExecutionID retrieves statistics for node executions
func (r *NodeExecutionRepository) GetStatsByExecutionID(ctx context.Context, executionID string) (map[string]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM node_executions
		WHERE execution_id = $1
		GROUP BY status
	`

	rows, err := r.db.Pool.Query(ctx, query, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}

	return stats, rows.Err()
}
