package storage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

type WorkflowRepository struct {
	db *PostgresDB
}

func NewWorkflowRepository(db *PostgresDB) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

func (r *WorkflowRepository) Create(ctx context.Context, workflow *models.Workflow) error {
	if workflow.ID == "" {
		workflow.ID = uuid.New().String()
	}

	// Default version is 1
	if workflow.Version == 0 {
		workflow.Version = 1
	}

	query := `
		INSERT INTO workflows (id, name, description, browser_profile_id, config, status, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		workflow.ID,
		workflow.Name,
		workflow.Description,
		workflow.BrowserProfileID, // NEW
		workflow.Config,
		workflow.Status,
		workflow.Version,
	).Scan(&workflow.CreatedAt, &workflow.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create workflow: %w", err)
	}

	return nil
}

func (r *WorkflowRepository) GetByID(ctx context.Context, id string) (*models.Workflow, error) {
	query := `
		SELECT id, name, description, browser_profile_id, config, status, version, created_at, updated_at
		FROM workflows
		WHERE id = $1
	`

	var workflow models.Workflow
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&workflow.ID,
		&workflow.Name,
		&workflow.Description,
		&workflow.BrowserProfileID, // NEW
		&workflow.Config,
		&workflow.Status,
		&workflow.Version,
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

func (r *WorkflowRepository) List(ctx context.Context, status models.WorkflowStatus, limit, offset int) ([]*models.Workflow, error) {
	query := `
		SELECT id, name, description, browser_profile_id, config, status, version, created_at, updated_at
		FROM workflows
	`
	args := []interface{}{}
	argPos := 1

	if status != "" {
		query += fmt.Sprintf(" WHERE status = $%d", argPos)
		args = append(args, status)
		argPos++
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, limit)
		argPos++
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, offset)
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer rows.Close()

	var workflows []*models.Workflow
	for rows.Next() {
		var workflow models.Workflow
		err := rows.Scan(
			&workflow.ID,
			&workflow.Name,
			&workflow.Description,
			&workflow.BrowserProfileID, // NEW
			&workflow.Config,
			&workflow.Status,
			&workflow.Version,
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

func (r *WorkflowRepository) Update(ctx context.Context, workflow *models.Workflow) error {
	// Increment version if not explicitly set (or if we want to force increment)
	// Usually the caller sets the new version, but we can also do it here.
	// Let's assume caller handles version logic or we just update what's passed.

	query := `
		UPDATE workflows
		SET name = $2, description = $3, browser_profile_id = $4, config = $5, status = $6, version = $7, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query,
		workflow.ID,
		workflow.Name,
		workflow.Description,
		workflow.BrowserProfileID, // NEW
		workflow.Config,
		workflow.Status,
		workflow.Version,
	)

	if err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("workflow not found: %s", workflow.ID)
	}

	return nil
}

func (r *WorkflowRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM workflows WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("workflow not found: %s", id)
	}

	return nil
}

func (r *WorkflowRepository) UpdateStatus(ctx context.Context, id string, status models.WorkflowStatus) error {
	query := `UPDATE workflows SET status = $2, updated_at = NOW() WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update workflow status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("workflow not found: %s", id)
	}

	return nil
}
