package storage

import (
	"context"
	"encoding/json"

	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// MonitoringRepository manages monitoring report persistence
type MonitoringRepository struct {
	db *PostgresDB
}

// NewMonitoringRepository creates a new monitoring repository
func NewMonitoringRepository(db *PostgresDB) *MonitoringRepository {
	return &MonitoringRepository{db: db}
}

// Create saves a new monitoring report
func (r *MonitoringRepository) Create(ctx context.Context, report *models.MonitoringReport) error {
	// Marshal JSON fields
	resultsJSON, _ := json.Marshal(report.Results)
	summaryJSON, _ := json.Marshal(report.Summary)
	configJSON, _ := json.Marshal(report.Config)

	query := `
		INSERT INTO monitoring_reports
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
		logger.Error("Failed to create monitoring report", zap.Error(err))
		return err
	}

	return nil
}

// Update updates an existing monitoring report
func (r *MonitoringRepository) Update(ctx context.Context, report *models.MonitoringReport) error {
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
		UPDATE monitoring_reports 
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
		logger.Error("Failed to update monitoring report", zap.Error(err))
		return err
	}

	return nil
}

// GetByID retrieves a monitoring report by ID
func (r *MonitoringRepository) GetByID(ctx context.Context, id string) (*models.MonitoringReport, error) {
	query := `
		SELECT 
			hc.id, hc.workflow_id, w.name as workflow_name, hc.execution_id, 
			hc.status, hc.started_at, hc.completed_at, hc.duration_ms, 
			hc.results, hc.summary, hc.config
		FROM monitoring_reports hc
		LEFT JOIN workflows w ON hc.workflow_id = w.id
		WHERE hc.id = $1
	`

	report := &models.MonitoringReport{}
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

// ListByWorkflow lists monitoring reports for a workflow
func (r *MonitoringRepository) ListByWorkflow(ctx context.Context, workflowID string, limit int) ([]*models.MonitoringReport, error) {
	query := `
		SELECT 
			hc.id, hc.workflow_id, hc.status, hc.started_at, 
			hc.completed_at, hc.duration_ms, hc.summary
		FROM monitoring_reports hc
		WHERE hc.workflow_id = $1
		ORDER BY hc.started_at DESC
		LIMIT $2
	`

	rows, err := r.db.Pool.Query(ctx, query, workflowID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reports := []*models.MonitoringReport{}
	for rows.Next() {
		report := &models.MonitoringReport{}
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
func (r *MonitoringRepository) SetAsBaseline(ctx context.Context, reportID string) error {
	query := `UPDATE monitoring_reports SET is_baseline = true WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, reportID)
	return err
}

// UnsetBaseline removes baseline status from all reports in a workflow
func (r *MonitoringRepository) UnsetBaseline(ctx context.Context, workflowID string) error {
	query := `UPDATE monitoring_reports SET is_baseline = false WHERE workflow_id = $1`
	_, err := r.db.Pool.Exec(ctx, query, workflowID)
	return err
}

// GetBaseline retrieves the baseline report for a workflow
func (r *MonitoringRepository) GetBaseline(ctx context.Context, workflowID string) (*models.MonitoringReport, error) {
	query := `
		SELECT 
			id, workflow_id, status, started_at, completed_at, 
			duration_ms, results, summary, config
		FROM monitoring_reports
		WHERE workflow_id = $1 AND is_baseline = true
		LIMIT 1
	`

	report := &models.MonitoringReport{}
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
