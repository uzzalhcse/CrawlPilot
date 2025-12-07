package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/uzzalhcse/crawlify/microservices/shared/database"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
)

// ProbeRepository handles probe-related database operations
type ProbeRepository interface {
	// SaveProbeResult stores a probe execution result
	SaveProbeResult(ctx context.Context, result *models.ProbeResult) error

	// GetProbeResults retrieves probe results for a workflow
	GetProbeResults(ctx context.Context, workflowID string, limit int) ([]*models.ProbeResult, error)

	// GetLatestProbeResult gets the most recent probe result for a workflow
	GetLatestProbeResult(ctx context.Context, workflowID string) (*models.ProbeResult, error)

	// SaveSampleURL saves or updates a sample URL for a phase
	SaveSampleURL(ctx context.Context, workflowID, phaseID, sampleURL string) error

	// GetSampleURL retrieves a sample URL for a phase
	GetSampleURL(ctx context.Context, workflowID, phaseID string) (string, error)

	// GetAllSampleURLs retrieves all sample URLs for a workflow
	GetAllSampleURLs(ctx context.Context, workflowID string) (map[string]string, error)

	// UpdateProbeStatus updates the probe_status column on an execution
	UpdateProbeStatus(ctx context.Context, executionID, status string) error

	// GetBaselineProbeResult gets the last healthy probe result for baseline comparison
	GetBaselineProbeResult(ctx context.Context, workflowID string) (*models.ProbeResult, error)

	// UpdateNodeSelector updates a workflow node's selector (for auto-fix)
	UpdateNodeSelector(ctx context.Context, workflowID, nodeID, newSelector, reason string) error

	// DisableNode marks a workflow node as disabled (for skip_node fix)
	DisableNode(ctx context.Context, workflowID, nodeID, reason string) error
}

// probeRepository implements ProbeRepository
type probeRepository struct {
	db *database.DB
}

// NewProbeRepository creates a new probe repository
func NewProbeRepository(db *database.DB) ProbeRepository {
	return &probeRepository{db: db}
}

// SaveProbeResult stores a probe execution result
func (r *probeRepository) SaveProbeResult(ctx context.Context, result *models.ProbeResult) error {
	// Marshal phases to JSON string for JSONB column
	phasesJSON, err := json.Marshal(result.Phases)
	if err != nil {
		return fmt.Errorf("failed to marshal phases: %w", err)
	}

	query := `
		INSERT INTO probe_results (execution_id, workflow_id, status, duration_ms, phases)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	err = r.db.Pool.QueryRow(ctx, query,
		result.ExecutionID,
		result.WorkflowID,
		result.Status,
		result.Duration,
		string(phasesJSON),
	).Scan(&result.ID, &result.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to save probe result: %w", err)
	}

	return nil
}

// GetProbeResults retrieves probe results for a workflow
func (r *probeRepository) GetProbeResults(ctx context.Context, workflowID string, limit int) ([]*models.ProbeResult, error) {
	query := `
		SELECT id, execution_id, workflow_id, status, duration_ms, phases, created_at
		FROM probe_results
		WHERE workflow_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Pool.Query(ctx, query, workflowID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query probe results: %w", err)
	}
	defer rows.Close()

	var results []*models.ProbeResult
	for rows.Next() {
		var result models.ProbeResult
		var phasesJSON []byte
		err := rows.Scan(
			&result.ID,
			&result.ExecutionID,
			&result.WorkflowID,
			&result.Status,
			&result.Duration,
			&phasesJSON,
			&result.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan probe result: %w", err)
		}
		// Unmarshal phases JSON
		if len(phasesJSON) > 0 {
			if err := json.Unmarshal(phasesJSON, &result.Phases); err != nil {
				return nil, fmt.Errorf("failed to unmarshal phases: %w", err)
			}
		}
		results = append(results, &result)
	}

	return results, nil
}

// GetLatestProbeResult gets the most recent probe result for a workflow
func (r *probeRepository) GetLatestProbeResult(ctx context.Context, workflowID string) (*models.ProbeResult, error) {
	query := `
		SELECT id, execution_id, workflow_id, status, duration_ms, phases, created_at
		FROM probe_results
		WHERE workflow_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var result models.ProbeResult
	var phasesJSON []byte
	err := r.db.Pool.QueryRow(ctx, query, workflowID).Scan(
		&result.ID,
		&result.ExecutionID,
		&result.WorkflowID,
		&result.Status,
		&result.Duration,
		&phasesJSON,
		&result.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get latest probe result: %w", err)
	}

	// Unmarshal phases JSON
	if len(phasesJSON) > 0 {
		if err := json.Unmarshal(phasesJSON, &result.Phases); err != nil {
			return nil, fmt.Errorf("failed to unmarshal phases: %w", err)
		}
	}

	return &result, nil
}

// SaveSampleURL saves or updates a sample URL for a phase
func (r *probeRepository) SaveSampleURL(ctx context.Context, workflowID, phaseID, sampleURL string) error {
	query := `
		INSERT INTO probe_sample_urls (workflow_id, phase_id, sample_url, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (workflow_id, phase_id) 
		DO UPDATE SET sample_url = EXCLUDED.sample_url, updated_at = NOW()
	`

	_, err := r.db.Pool.Exec(ctx, query, workflowID, phaseID, sampleURL)
	if err != nil {
		return fmt.Errorf("failed to save sample URL: %w", err)
	}

	return nil
}

// GetSampleURL retrieves a sample URL for a phase
func (r *probeRepository) GetSampleURL(ctx context.Context, workflowID, phaseID string) (string, error) {
	query := `
		SELECT sample_url FROM probe_sample_urls
		WHERE workflow_id = $1 AND phase_id = $2
	`

	var sampleURL string
	err := r.db.Pool.QueryRow(ctx, query, workflowID, phaseID).Scan(&sampleURL)
	if err != nil {
		return "", err // May be sql.ErrNoRows
	}

	return sampleURL, nil
}

// GetAllSampleURLs retrieves all sample URLs for a workflow
func (r *probeRepository) GetAllSampleURLs(ctx context.Context, workflowID string) (map[string]string, error) {
	query := `
		SELECT phase_id, sample_url FROM probe_sample_urls
		WHERE workflow_id = $1
	`

	rows, err := r.db.Pool.Query(ctx, query, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sample URLs: %w", err)
	}
	defer rows.Close()

	urls := make(map[string]string)
	for rows.Next() {
		var phaseID, sampleURL string
		if err := rows.Scan(&phaseID, &sampleURL); err != nil {
			return nil, fmt.Errorf("failed to scan sample URL: %w", err)
		}
		urls[phaseID] = sampleURL
	}

	return urls, nil
}

// UpdateProbeStatus updates the probe_status column on an execution
func (r *probeRepository) UpdateProbeStatus(ctx context.Context, executionID, status string) error {
	query := `
		UPDATE workflow_executions
		SET probe_status = $2
		WHERE id = $1
	`

	_, err := r.db.Pool.Exec(ctx, query, executionID, status)
	if err != nil {
		return fmt.Errorf("failed to update probe status: %w", err)
	}

	return nil
}

// GetBaselineProbeResult gets the last healthy probe result for baseline comparison
func (r *probeRepository) GetBaselineProbeResult(ctx context.Context, workflowID string) (*models.ProbeResult, error) {
	query := `
		SELECT id, execution_id, workflow_id, status, duration_ms, phases, created_at
		FROM probe_results
		WHERE workflow_id = $1 AND status = 'healthy'
		ORDER BY created_at DESC
		LIMIT 1
	`

	var result models.ProbeResult
	var phasesJSON []byte
	err := r.db.Pool.QueryRow(ctx, query, workflowID).Scan(
		&result.ID,
		&result.ExecutionID,
		&result.WorkflowID,
		&result.Status,
		&result.Duration,
		&phasesJSON,
		&result.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get baseline probe result: %w", err)
	}

	// Unmarshal phases JSON
	if len(phasesJSON) > 0 {
		if err := json.Unmarshal(phasesJSON, &result.Phases); err != nil {
			return nil, fmt.Errorf("failed to unmarshal phases: %w", err)
		}
	}

	return &result, nil
}

// UpdateNodeSelector updates a workflow node's selector (for auto-fix)
func (r *probeRepository) UpdateNodeSelector(ctx context.Context, workflowID, nodeID, newSelector, reason string) error {
	// Update the node config in workflows table
	// This uses jsonb_set to update the specific node's selector property
	query := `
		WITH phase_update AS (
			SELECT w.id as workflow_id, 
				   p.ordinal as phase_idx,
				   n.ordinal as node_idx,
				   p.element ->> 'id' as phase_id
			FROM workflows w,
				 LATERAL jsonb_array_elements(w.config->'phases') WITH ORDINALITY p(element, ordinal),
				 LATERAL jsonb_array_elements(p.element->'nodes') WITH ORDINALITY n(element, ordinal)
			WHERE w.id = $1 AND n.element ->> 'id' = $2
			LIMIT 1
		)
		UPDATE workflows w
		SET config = jsonb_set(
			config,
			ARRAY['phases', (pu.phase_idx - 1)::text, 'nodes', (pu.node_idx - 1)::text, 'config', 'selector'],
			to_jsonb($3::text)
		),
		updated_at = NOW()
		FROM phase_update pu
		WHERE w.id = pu.workflow_id
	`

	result, err := r.db.Pool.Exec(ctx, query, workflowID, nodeID, newSelector)
	if err != nil {
		return fmt.Errorf("failed to update node selector: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("node %s not found in workflow %s", nodeID, workflowID)
	}

	return nil
}

// DisableNode marks a workflow node as disabled (for skip_node fix)
func (r *probeRepository) DisableNode(ctx context.Context, workflowID, nodeID, reason string) error {
	// Update the node config to add disabled=true
	query := `
		WITH phase_update AS (
			SELECT w.id as workflow_id, 
				   p.ordinal as phase_idx,
				   n.ordinal as node_idx
			FROM workflows w,
				 LATERAL jsonb_array_elements(w.config->'phases') WITH ORDINALITY p(element, ordinal),
				 LATERAL jsonb_array_elements(p.element->'nodes') WITH ORDINALITY n(element, ordinal)
			WHERE w.id = $1 AND n.element ->> 'id' = $2
			LIMIT 1
		)
		UPDATE workflows w
		SET config = jsonb_set(
			config,
			ARRAY['phases', (pu.phase_idx - 1)::text, 'nodes', (pu.node_idx - 1)::text, 'disabled'],
			'true'
		),
		updated_at = NOW()
		FROM phase_update pu
		WHERE w.id = pu.workflow_id
	`

	result, err := r.db.Pool.Exec(ctx, query, workflowID, nodeID)
	if err != nil {
		return fmt.Errorf("failed to disable node: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("node %s not found in workflow %s", nodeID, workflowID)
	}

	return nil
}
