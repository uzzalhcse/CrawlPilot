package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/uzzalhcse/crawlify/microservices/shared/database"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
)

// postgresWorkflowRepo implements WorkflowRepository using PostgreSQL
type postgresWorkflowRepo struct {
	db *database.DB
}

// NewWorkflowRepository creates a new PostgreSQL workflow repository
func NewWorkflowRepository(db *database.DB) WorkflowRepository {
	return &postgresWorkflowRepo{db: db}
}

func (r *postgresWorkflowRepo) Create(ctx context.Context, workflow *models.Workflow) error {
	if workflow.ID == "" {
		workflow.ID = uuid.New().String()
	}

	now := time.Now()
	workflow.CreatedAt = now
	workflow.UpdatedAt = now

	query := `
		INSERT INTO workflows (id, name, config, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		workflow.ID,
		workflow.Name,
		workflow.Config,
		workflow.Status,
		workflow.CreatedAt,
		workflow.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create workflow: %w", err)
	}

	return nil
}

func (r *postgresWorkflowRepo) Get(ctx context.Context, id string) (*models.Workflow, error) {
	query := `
		SELECT id, name, config, status, created_at, updated_at
		FROM workflows
		WHERE id = $1 AND deleted_at IS NULL
	`

	var workflow models.Workflow

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&workflow.ID,
		&workflow.Name,
		&workflow.Config,
		&workflow.Status,
		&workflow.CreatedAt,
		&workflow.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("workflow not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	return &workflow, nil
}

func (r *postgresWorkflowRepo) List(ctx context.Context, filters ListFilters) ([]*models.Workflow, error) {
	query := `
		SELECT id, name, config, status, created_at, updated_at
		FROM workflows
		WHERE deleted_at IS NULL
	`

	args := []interface{}{}
	argPos := 1

	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, filters.Status)
		argPos++
	}

	query += " ORDER BY created_at DESC"

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
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer rows.Close()

	workflows := make([]*models.Workflow, 0)

	for rows.Next() {
		var workflow models.Workflow
		err := rows.Scan(
			&workflow.ID,
			&workflow.Name,
			&workflow.Config,
			&workflow.Status,
			&workflow.CreatedAt,
			&workflow.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}
		workflows = append(workflows, &workflow)
	}

	return workflows, nil
}

func (r *postgresWorkflowRepo) Update(ctx context.Context, workflow *models.Workflow) error {
	workflow.UpdatedAt = time.Now()

	query := `
		UPDATE workflows
		SET name = $2, config = $3, status = $4, updated_at = $5
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Pool.Exec(ctx, query,
		workflow.ID,
		workflow.Name,
		workflow.Config,
		workflow.Status,
		workflow.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("workflow not found: %s", workflow.ID)
	}

	return nil
}

func (r *postgresWorkflowRepo) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE workflows
		SET deleted_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Pool.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("workflow not found: %s", id)
	}

	return nil
}

func (r *postgresWorkflowRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `
		UPDATE workflows
		SET status = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Pool.Exec(ctx, query, id, status, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update workflow status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("workflow not found: %s", id)
	}

	return nil
}
