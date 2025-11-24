package storage

import (
	"context"
	"encoding/json"

	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// HealthCheckRepository manages health check report persistence
type HealthCheckRepository struct {
	db *PostgresDB
}

// NewHealthCheckRepository creates a new health check repository
func NewHealthCheckRepository(db *PostgresDB) *HealthCheckRepository {
	return &HealthCheckRepository{db: db}
}

// Create saves a new health check report
func (r *HealthCheckRepository) Create(ctx context.Context, report *models.HealthCheckReport) error {
	// Marshal JSON fields
	resultsJSON, _ := json.Marshal(report.Results)
	summaryJSON, _ := json.Marshal(report.Summary)
	configJSON, _ := json.Marshal(report.Config)

	query := `
		INSERT INTO health_check_reports
		(id, workflow_id, execution_id, status, started_at, completed_at, duration_ms, results, summary, config)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		report.ID,
		report.WorkflowID,
		report.ExecutionID,
		report.Status,
		report.StartedAt,
		report.CompletedAt,
		report.Duration,
		resultsJSON,
		summaryJSON,
		configJSON,
	)

	if err != nil {
		logger.Error("Failed to create health check report", zap.Error(err))
		return err
	}

	return nil
}

// Update updates an existing health check report
func (r *HealthCheckRepository) Update(ctx context.Context, report *models.HealthCheckReport) error {
	resultsJSON, err := json.Marshal(report.Results)
	if err != nil {
		return err
	}

	summaryJSON, err := json.Marshal(report.Summary)
	if err != nil {
		return err
	}

	configJSON, err := json.Marshal(report.Config)
	if err != nil {
		return err
	}

	query := `
		UPDATE health_check_reports 
		SET status = $1, completed_at = $2, duration_ms = $3, results = $4, summary = $5, config = $6
		WHERE id = $7
	`

	_, err = r.db.Pool.Exec(ctx, query,
		report.Status,
		report.CompletedAt,
		report.Duration,
		resultsJSON,
		summaryJSON,
		configJSON,
		report.ID,
	)

	if err != nil {
		logger.Error("Failed to update health check report", zap.Error(err))
		return err
	}

	return nil
}

// GetByID retrieves a health check report by ID
func (r *HealthCheckRepository) GetByID(ctx context.Context, id string) (*models.HealthCheckReport, error) {
	query := `
		SELECT 
			hc.id, hc.workflow_id, w.name as workflow_name, hc.execution_id, 
			hc.status, hc.started_at, hc.completed_at, hc.duration_ms, 
			hc.results, hc.summary, hc.config
		FROM health_check_reports hc
		LEFT JOIN workflows w ON hc.workflow_id = w.id
		WHERE hc.id = $1
	`

	report := &models.HealthCheckReport{}
	var resultsJSON, summaryJSON, configJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&report.ID,
		&report.WorkflowID,
		&report.WorkflowName,
		&report.ExecutionID,
		&report.Status,
		&report.StartedAt,
		&report.CompletedAt,
		&report.Duration,
		&resultsJSON,
		&summaryJSON,
		&configJSON,
	)

	if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	json.Unmarshal(resultsJSON, &report.Results)
	json.Unmarshal(summaryJSON, &report.Summary)
	json.Unmarshal(configJSON, &report.Config)

	return report, nil
}

// ListByWorkflow lists health check reports for a workflow
func (r *HealthCheckRepository) ListByWorkflow(ctx context.Context, workflowID string, limit int) ([]*models.HealthCheckReport, error) {
	query := `
		SELECT 
			hc.id, hc.workflow_id, hc.status, hc.started_at, 
			hc.completed_at, hc.duration_ms, hc.summary
		FROM health_check_reports hc
		WHERE hc.workflow_id = $1
		ORDER BY hc.started_at DESC
		LIMIT $2
	`

	rows, err := r.db.Pool.Query(ctx, query, workflowID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reports := []*models.HealthCheckReport{}
	for rows.Next() {
		report := &models.HealthCheckReport{}
		var summaryJSON []byte

		err := rows.Scan(
			&report.ID,
			&report.WorkflowID,
			&report.Status,
			&report.StartedAt,
			&report.CompletedAt,
			&report.Duration,
			&summaryJSON,
		)

		if err != nil {
			continue
		}

		json.Unmarshal(summaryJSON, &report.Summary)
		reports = append(reports, report)
	}

	return reports, nil
}

// SetAsBaseline marks a report as the baseline for its workflow
func (r *HealthCheckRepository) SetAsBaseline(ctx context.Context, reportID string) error {
	query := `UPDATE health_check_reports SET is_baseline = true WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, reportID)
	return err
}

// UnsetBaseline removes baseline status from all reports in a workflow
func (r *HealthCheckRepository) UnsetBaseline(ctx context.Context, workflowID string) error {
	query := `UPDATE health_check_reports SET is_baseline = false WHERE workflow_id = $1`
	_, err := r.db.Pool.Exec(ctx, query, workflowID)
	return err
}

// GetBaseline retrieves the baseline report for a workflow
func (r *HealthCheckRepository) GetBaseline(ctx context.Context, workflowID string) (*models.HealthCheckReport, error) {
	query := `
		SELECT 
			id, workflow_id, status, started_at, completed_at, 
			duration_ms, results, summary, config
		FROM health_check_reports
		WHERE workflow_id = $1 AND is_baseline = true
		LIMIT 1
	`

	report := &models.HealthCheckReport{}
	var resultsJSON, summaryJSON, configJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, workflowID).Scan(
		&report.ID,
		&report.WorkflowID,
		&report.Status,
		&report.StartedAt,
		&report.CompletedAt,
		&report.Duration,
		&resultsJSON,
		&summaryJSON,
		&configJSON,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(resultsJSON, &report.Results)
	json.Unmarshal(summaryJSON, &report.Summary)
	json.Unmarshal(configJSON, &report.Config)

	return report, nil
}
