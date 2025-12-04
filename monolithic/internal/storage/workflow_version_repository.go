package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

type WorkflowVersionRepository struct {
	db *PostgresDB
}

func NewWorkflowVersionRepository(db *PostgresDB) *WorkflowVersionRepository {
	return &WorkflowVersionRepository{db: db}
}

// Create creates a new workflow version
func (r *WorkflowVersionRepository) Create(ctx context.Context, version *models.WorkflowVersion) error {
	if version.ID == "" {
		version.ID = uuid.New().String()
	}

	query := `
		INSERT INTO workflow_versions (id, workflow_id, version, config, change_reason, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING created_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		version.ID,
		version.WorkflowID,
		version.Version,
		version.Config,
		version.ChangeReason,
	).Scan(&version.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create workflow version: %w", err)
	}

	return nil
}

// GetLatest retrieves the latest version for a workflow
func (r *WorkflowVersionRepository) GetLatest(ctx context.Context, workflowID string) (*models.WorkflowVersion, error) {
	query := `
		SELECT id, workflow_id, version, config, change_reason, created_at
		FROM workflow_versions
		WHERE workflow_id = $1
		ORDER BY version DESC
		LIMIT 1
	`

	var version models.WorkflowVersion
	var configJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, workflowID).Scan(
		&version.ID,
		&version.WorkflowID,
		&version.Version,
		&configJSON,
		&version.ChangeReason,
		&version.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No versions found
		}
		return nil, fmt.Errorf("failed to get latest workflow version: %w", err)
	}

	if err := json.Unmarshal(configJSON, &version.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &version, nil
}

// GetByVersion retrieves a specific version for a workflow
func (r *WorkflowVersionRepository) GetByVersion(ctx context.Context, workflowID string, versionNum int) (*models.WorkflowVersion, error) {
	query := `
		SELECT id, workflow_id, version, config, change_reason, created_at
		FROM workflow_versions
		WHERE workflow_id = $1 AND version = $2
	`

	var version models.WorkflowVersion
	var configJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, workflowID, versionNum).Scan(
		&version.ID,
		&version.WorkflowID,
		&version.Version,
		&configJSON,
		&version.ChangeReason,
		&version.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("workflow version not found: %s v%d", workflowID, versionNum)
		}
		return nil, fmt.Errorf("failed to get workflow version: %w", err)
	}

	if err := json.Unmarshal(configJSON, &version.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &version, nil
}

// List retrieves versions for a workflow with pagination
func (r *WorkflowVersionRepository) List(ctx context.Context, workflowID string, limit, offset int) ([]*models.WorkflowVersion, error) {
	query := `
		SELECT id, workflow_id, version, config, change_reason, created_at
		FROM workflow_versions
		WHERE workflow_id = $1
		ORDER BY version DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Pool.Query(ctx, query, workflowID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow versions: %w", err)
	}
	defer rows.Close()

	var versions []*models.WorkflowVersion
	for rows.Next() {
		var version models.WorkflowVersion
		var configJSON []byte

		err := rows.Scan(
			&version.ID,
			&version.WorkflowID,
			&version.Version,
			&configJSON,
			&version.ChangeReason,
			&version.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow version: %w", err)
		}

		if err := json.Unmarshal(configJSON, &version.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		versions = append(versions, &version)
	}

	return versions, nil
}
