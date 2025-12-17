package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
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

	// UpdateFieldSelector updates a field selector within an extract node (for auto-fix)
	UpdateFieldSelector(ctx context.Context, workflowID, nodeID, fieldName, newSelector, reason string) error

	// DisableNode marks a workflow node as disabled (for skip_node fix)
	DisableNode(ctx context.Context, workflowID, nodeID, reason string) error

	// GetRecentProbes gets recent probe results across all workflows
	GetRecentProbes(ctx context.Context, limit int) ([]*models.ProbeResult, int, error)

	// GetProbeStats returns aggregate probe statistics
	GetProbeStats(ctx context.Context) (*ProbeStatsResult, error)

	// GetAutoFixes retrieves auto-fix records with optional status filter
	GetAutoFixes(ctx context.Context, status string, limit int) ([]*models.AutoFixRecord, int, error)

	// UpdateAutoFixStatus updates the status of an auto-fix record
	UpdateAutoFixStatus(ctx context.Context, fixID, status string) error

	// CreateAutoFix records an AI-applied workflow fix
	CreateAutoFix(ctx context.Context, fix *models.AutoFixRecord) error

	// GetAutoFixByID retrieves a single auto-fix record by ID
	GetAutoFixByID(ctx context.Context, fixID string) (*models.AutoFixRecord, error)

	// GetSnapshotPath retrieves the DOM snapshot path for a specific node execution
	GetSnapshotPath(ctx context.Context, executionID, nodeID string) (string, error)
	UpdateAutoFix(ctx context.Context, fix *models.AutoFixRecord) error
}

// probeRepository implements ProbeRepository
type probeRepository struct {
	db *database.DB
}

// NewProbeRepository creates a new probe repository
func NewProbeRepository(db *database.DB) ProbeRepository {
	return &probeRepository{db: db}
}

// SaveProbeResult stores or updates a probe execution result
// If a result for this execution already exists, merges the new phases into it
func (r *probeRepository) SaveProbeResult(ctx context.Context, result *models.ProbeResult) error {
	// Marshal phases to JSON string for JSONB column
	phasesJSON, err := json.Marshal(result.Phases)
	if err != nil {
		return fmt.Errorf("failed to marshal phases: %w", err)
	}

	// Use UPSERT to merge phases when execution already exists
	// This handles the case where each phase reports separately
	query := `
		INSERT INTO probe_results (execution_id, workflow_id, status, duration_ms, phases)
		VALUES ($1, $2, $3, $4, $5::jsonb)
		ON CONFLICT (execution_id) DO UPDATE SET
			phases = (
				SELECT jsonb_agg(DISTINCT phase)
				FROM (
					SELECT jsonb_array_elements(probe_results.phases) AS phase
					UNION ALL
					SELECT jsonb_array_elements($5::jsonb) AS phase
				) combined
			),
			duration_ms = probe_results.duration_ms + EXCLUDED.duration_ms,
			status = CASE 
				WHEN probe_results.status = 'broken' OR EXCLUDED.status = 'broken' THEN 'broken'
				WHEN probe_results.status = 'degraded' OR EXCLUDED.status = 'degraded' THEN 'degraded'
				ELSE 'healthy'
			END
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
	// Update the node params in workflows table
	// The selector is stored at: phases[idx].nodes[idx].params.selector
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
			ARRAY['phases', (pu.phase_idx - 1)::text, 'nodes', (pu.node_idx - 1)::text, 'params', 'selector'],
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

// UpdateFieldSelector updates a field selector within an extract node (for auto-fix)
// Path: phases[idx].nodes[idx].params.fields.{fieldName}.selector
func (r *probeRepository) UpdateFieldSelector(ctx context.Context, workflowID, nodeID, fieldName, newSelector, reason string) error {
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
			ARRAY['phases', (pu.phase_idx - 1)::text, 'nodes', (pu.node_idx - 1)::text, 'params', 'fields', $4, 'selector'],
			to_jsonb($3::text)
		),
		updated_at = NOW()
		FROM phase_update pu
		WHERE w.id = pu.workflow_id
	`

	result, err := r.db.Pool.Exec(ctx, query, workflowID, nodeID, newSelector, fieldName)
	if err != nil {
		return fmt.Errorf("failed to update field selector: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("field %s in node %s not found in workflow %s", fieldName, nodeID, workflowID)
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

// ProbeStatsResult holds aggregate probe statistics
type ProbeStatsResult struct {
	Total      int            `json:"total"`
	Healthy    int            `json:"healthy"`
	Degraded   int            `json:"degraded"`
	Broken     int            `json:"broken"`
	ByWorkflow map[string]int `json:"by_workflow"`
}

// GetRecentProbes gets all recent probe executions sorted by date
func (r *probeRepository) GetRecentProbes(ctx context.Context, limit int) ([]*models.ProbeResult, int, error) {
	// Get all probe executions sorted by most recent first, joined with latest auto-fix
	query := `
		SELECT 
			p.id, p.execution_id, p.workflow_id, p.status, p.duration_ms, p.phases, p.created_at,
			f.id, f.workflow_id, f.execution_id, f.node_id, f.fix_type, f.old_selector, f.new_selector,
			f.reasoning, f.confidence, f.status, f.auto_applied, f.created_at
		FROM probe_results p
		LEFT JOIN LATERAL (
			SELECT * FROM workflow_auto_fixes 
			WHERE execution_id = p.execution_id
			ORDER BY created_at DESC
			LIMIT 1
		) f ON true
		ORDER BY p.created_at DESC
		LIMIT $1
	`

	rows, err := r.db.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query recent probes: %w", err)
	}
	defer rows.Close()

	var results []*models.ProbeResult
	for rows.Next() {
		var result models.ProbeResult
		var phasesJSON []byte

		// AutoFix fields (nullable)
		var fID, fWorkflowID, fExecutionID, fNodeID, fFixType, fOldSelector, fNewSelector, fReasoning, fStatus, fCreatedAt *string
		var fConfidence *float64
		var fAutoApplied *bool

		if err := rows.Scan(
			&result.ID, &result.ExecutionID, &result.WorkflowID, &result.Status, &result.Duration, &phasesJSON, &result.CreatedAt,
			&fID, &fWorkflowID, &fExecutionID, &fNodeID, &fFixType, &fOldSelector, &fNewSelector,
			&fReasoning, &fConfidence, &fStatus, &fAutoApplied, &fCreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan probe result: %w", err)
		}

		if len(phasesJSON) > 0 {
			json.Unmarshal(phasesJSON, &result.Phases)
		}

		// Populate AutoFix if present
		if fID != nil {
			result.AutoFix = &models.AutoFixRecord{
				ID:          *fID,
				WorkflowID:  *fWorkflowID,
				ExecutionID: *fExecutionID,
				NodeID:      *fNodeID,
				FixType:     *fFixType,
				OldSelector: *fOldSelector,
				NewSelector: *fNewSelector,
				Reasoning:   *fReasoning,
				Confidence:  *fConfidence,
				Status:      *fStatus,
				AutoApplied: *fAutoApplied,
				CreatedAt:   *fCreatedAt,
			}
		}

		results = append(results, &result)
	}

	// Get total count of all probe executions
	var total int
	r.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM probe_results").Scan(&total)

	return results, total, nil
}

// GetProbeStats returns aggregate probe statistics
func (r *probeRepository) GetProbeStats(ctx context.Context) (*ProbeStatsResult, error) {
	stats := &ProbeStatsResult{
		ByWorkflow: make(map[string]int),
	}

	// Get counts by status (from latest probe per workflow)
	query := `
		WITH latest_probes AS (
			SELECT DISTINCT ON (workflow_id) workflow_id, status
			FROM probe_results
			ORDER BY workflow_id, created_at DESC
		)
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'healthy') as healthy,
			COUNT(*) FILTER (WHERE status = 'degraded') as degraded,
			COUNT(*) FILTER (WHERE status = 'broken') as broken
		FROM latest_probes
	`

	err := r.db.Pool.QueryRow(ctx, query).Scan(&stats.Total, &stats.Healthy, &stats.Degraded, &stats.Broken)
	if err != nil {
		return nil, fmt.Errorf("failed to get probe stats: %w", err)
	}

	// Get count per workflow
	workflowQuery := `
		SELECT workflow_id, COUNT(*) 
		FROM probe_results 
		GROUP BY workflow_id
	`
	rows, err := r.db.Pool.Query(ctx, workflowQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var wid string
			var count int
			if rows.Scan(&wid, &count) == nil {
				stats.ByWorkflow[wid] = count
			}
		}
	}

	return stats, nil
}

// GetAutoFixes retrieves auto-fix records with optional status filter
func (r *probeRepository) GetAutoFixes(ctx context.Context, status string, limit int) ([]*models.AutoFixRecord, int, error) {
	var query string
	var rows interface{ Close() }
	var err error

	if status == "" {
		query = `
			SELECT id, workflow_id, execution_id, node_id, field_name, fix_type, old_selector, new_selector, 
			       reasoning, confidence, status, auto_applied, created_at
			FROM workflow_auto_fixes
			ORDER BY created_at DESC
			LIMIT $1
		`
		rows, err = r.db.Pool.Query(ctx, query, limit)
	} else {
		query = `
			SELECT id, workflow_id, execution_id, node_id, field_name, fix_type, old_selector, new_selector, 
			       reasoning, confidence, status, auto_applied, created_at
			FROM workflow_auto_fixes
			WHERE status = $1
			ORDER BY created_at DESC
			LIMIT $2
		`
		rows, err = r.db.Pool.Query(ctx, query, status, limit)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("failed to query auto-fixes: %w", err)
	}
	defer rows.Close()

	var fixes []*models.AutoFixRecord
	pgRows := rows.(interface {
		Next() bool
		Scan(...interface{}) error
	})
	for pgRows.Next() {
		var fix models.AutoFixRecord
		// Handle nullable field_name
		var fieldName *string
		if err := pgRows.Scan(&fix.ID, &fix.WorkflowID, &fix.ExecutionID, &fix.NodeID, &fieldName, &fix.FixType,
			&fix.OldSelector, &fix.NewSelector, &fix.Reasoning, &fix.Confidence,
			&fix.Status, &fix.AutoApplied, &fix.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan auto-fix: %w", err)
		}
		if fieldName != nil {
			fix.FieldName = *fieldName
		}
		fixes = append(fixes, &fix)
	}

	// Get total count
	var total int
	r.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM workflow_auto_fixes").Scan(&total)

	return fixes, total, nil
}

// UpdateAutoFixStatus updates the status of an auto-fix record
func (r *probeRepository) UpdateAutoFixStatus(ctx context.Context, fixID, status string) error {
	query := `UPDATE workflow_auto_fixes SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Pool.Exec(ctx, query, status, fixID)
	if err != nil {
		return fmt.Errorf("failed to update auto-fix status: %w", err)
	}
	return nil
}

// UpdateAutoFix updates the details of an auto-fix record (e.g. manual override)
func (r *probeRepository) UpdateAutoFix(ctx context.Context, fix *models.AutoFixRecord) error {
	query := `
		UPDATE workflow_auto_fixes 
		SET new_selector = $1, reasoning = $2, confidence = $3, auto_applied = $4, status = $5, updated_at = NOW()
		WHERE id = $6
	`
	_, err := r.db.Pool.Exec(ctx, query,
		fix.NewSelector, fix.Reasoning, fix.Confidence, fix.AutoApplied, fix.Status, fix.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update auto-fix record: %w", err)
	}
	return nil
}

// CreateAutoFix records an AI-applied workflow fix
func (r *probeRepository) CreateAutoFix(ctx context.Context, fix *models.AutoFixRecord) error {
	query := `
		INSERT INTO workflow_auto_fixes (
			id, workflow_id, execution_id, node_id, field_name, fix_type, old_selector, new_selector, 
			reasoning, confidence, status, auto_applied
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at
	`

	// Handle empty ID (should be generated by caller, but safe fallback)
	if fix.ID == "" {
		fix.ID = uuid.New().String()
	}

	err := r.db.Pool.QueryRow(ctx, query,
		fix.ID, fix.WorkflowID, fix.ExecutionID, fix.NodeID, fix.FieldName, fix.FixType, fix.OldSelector, fix.NewSelector,
		fix.Reasoning, fix.Confidence, fix.Status, fix.AutoApplied,
	).Scan(&fix.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create auto-fix record: %w", err)
	}
	return nil
}

// GetAutoFixByID retrieves a single auto-fix record by ID
func (r *probeRepository) GetAutoFixByID(ctx context.Context, fixID string) (*models.AutoFixRecord, error) {
	query := `
		SELECT id, workflow_id, execution_id, node_id, field_name, fix_type, old_selector, new_selector, 
		       reasoning, confidence, status, auto_applied, created_at
		FROM workflow_auto_fixes
		WHERE id = $1
	`

	var fix models.AutoFixRecord
	var fieldName *string
	err := r.db.Pool.QueryRow(ctx, query, fixID).Scan(
		&fix.ID, &fix.WorkflowID, &fix.ExecutionID, &fix.NodeID, &fieldName, &fix.FixType,
		&fix.OldSelector, &fix.NewSelector, &fix.Reasoning, &fix.Confidence,
		&fix.Status, &fix.AutoApplied, &fix.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get auto-fix: %w", err)
	}
	if fieldName != nil {
		fix.FieldName = *fieldName
	}
	return &fix, nil
}

// GetSnapshotPath retrieves the DOM snapshot path for a specific node execution
func (r *probeRepository) GetSnapshotPath(ctx context.Context, executionID, nodeID string) (string, error) {
	query := `SELECT phases FROM probe_results WHERE execution_id = $1`

	var phasesJSON []byte
	err := r.db.Pool.QueryRow(ctx, query, executionID).Scan(&phasesJSON)
	if err != nil {
		return "", fmt.Errorf("failed to get probe phases: %w", err)
	}

	var phases []models.PhaseProbeResult
	if err := json.Unmarshal(phasesJSON, &phases); err != nil {
		return "", fmt.Errorf("failed to unmarshal phases: %w", err)
	}

	for _, phase := range phases {
		for _, node := range phase.Nodes {
			if node.NodeID == nodeID {
				if node.Snapshot != nil && node.Snapshot.DOMPath != "" {
					return node.Snapshot.DOMPath, nil
				}
				return "", fmt.Errorf("snapshot not found for node %s", nodeID)
			}
		}
	}

	return "", fmt.Errorf("node %s not found in execution %s", nodeID, executionID)
}
