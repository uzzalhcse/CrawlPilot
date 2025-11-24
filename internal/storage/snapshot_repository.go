package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/uzzalhcse/crawlify/pkg/models"
)

// SnapshotRepository handles database operations for health check snapshots
type SnapshotRepository struct {
	db *PostgresDB
}

// NewSnapshotRepository creates a new snapshot repository
func NewSnapshotRepository(db *PostgresDB) *SnapshotRepository {
	return &SnapshotRepository{db: db}
}

// Create saves a new snapshot to the database
func (r *SnapshotRepository) Create(ctx context.Context, snapshot *models.HealthCheckSnapshot) error {
	// Marshal console logs
	var consoleLogsJSON []byte
	var err error
	if snapshot.ConsoleLogsData != nil {
		consoleLogsJSON, err = json.Marshal(snapshot.ConsoleLogsData)
		if err != nil {
			return fmt.Errorf("failed to marshal console logs: %w", err)
		}
	}
	snapshot.ConsoleLogs = consoleLogsJSON

	// Marshal metadata
	var metadataJSON []byte
	if snapshot.MetadataData != nil {
		metadataJSON, err = json.Marshal(snapshot.MetadataData)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}
	snapshot.Metadata = metadataJSON

	query := `
		INSERT INTO health_check_snapshots (
			report_id, node_id, phase_name, url, page_title, status_code,
			screenshot_path, dom_snapshot_path, console_logs,
			selector_type, selector_value, elements_found, error_message, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		) RETURNING id, created_at
	`

	err = r.db.Pool.QueryRow(
		ctx, query,
		snapshot.ReportID, snapshot.NodeID, snapshot.PhaseName, snapshot.URL,
		snapshot.PageTitle, snapshot.StatusCode, snapshot.ScreenshotPath,
		snapshot.DOMSnapshotPath, snapshot.ConsoleLogs, snapshot.SelectorType,
		snapshot.SelectorValue, snapshot.ElementsFound, snapshot.ErrorMessage,
		snapshot.Metadata,
	).Scan(&snapshot.ID, &snapshot.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	return nil
}

// GetByID retrieves a snapshot by ID
func (r *SnapshotRepository) GetByID(ctx context.Context, id string) (*models.HealthCheckSnapshot, error) {
	snapshot := &models.HealthCheckSnapshot{}
	query := `
		SELECT id, report_id, node_id, phase_name, created_at, url, page_title,
		       status_code, screenshot_path, dom_snapshot_path, console_logs,
		       selector_type, selector_value, elements_found, error_message, metadata
		FROM health_check_snapshots
		WHERE id = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&snapshot.ID, &snapshot.ReportID, &snapshot.NodeID, &snapshot.PhaseName,
		&snapshot.CreatedAt, &snapshot.URL, &snapshot.PageTitle, &snapshot.StatusCode,
		&snapshot.ScreenshotPath, &snapshot.DOMSnapshotPath, &snapshot.ConsoleLogs,
		&snapshot.SelectorType, &snapshot.SelectorValue, &snapshot.ElementsFound,
		&snapshot.ErrorMessage, &snapshot.Metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	// Unmarshal console logs
	if len(snapshot.ConsoleLogs) > 0 {
		if err := json.Unmarshal(snapshot.ConsoleLogs, &snapshot.ConsoleLogsData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal console logs: %w", err)
		}
	}

	// Unmarshal metadata
	if len(snapshot.Metadata) > 0 {
		if err := json.Unmarshal(snapshot.Metadata, &snapshot.MetadataData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return snapshot, nil
}

// GetByReportID retrieves all snapshots for a specific health check report
func (r *SnapshotRepository) GetByReportID(ctx context.Context, reportID string) ([]*models.HealthCheckSnapshot, error) {
	query := `
		SELECT id, report_id, node_id, phase_name, created_at, url, page_title,
		       status_code, screenshot_path, dom_snapshot_path, console_logs,
		       selector_type, selector_value, elements_found, error_message, metadata
		FROM health_check_snapshots
		WHERE report_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, reportID)
	if err != nil {
		return nil, fmt.Errorf("failed to query snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []*models.HealthCheckSnapshot
	for rows.Next() {
		snapshot := &models.HealthCheckSnapshot{}
		err := rows.Scan(
			&snapshot.ID, &snapshot.ReportID, &snapshot.NodeID, &snapshot.PhaseName,
			&snapshot.CreatedAt, &snapshot.URL, &snapshot.PageTitle, &snapshot.StatusCode,
			&snapshot.ScreenshotPath, &snapshot.DOMSnapshotPath, &snapshot.ConsoleLogs,
			&snapshot.SelectorType, &snapshot.SelectorValue, &snapshot.ElementsFound,
			&snapshot.ErrorMessage, &snapshot.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}

		// Unmarshal JSONB fields
		if len(snapshot.ConsoleLogs) > 0 {
			if err := json.Unmarshal(snapshot.ConsoleLogs, &snapshot.ConsoleLogsData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal console logs: %w", err)
			}
		}
		if len(snapshot.Metadata) > 0 {
			if err := json.Unmarshal(snapshot.Metadata, &snapshot.MetadataData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, rows.Err()
}

// GetByNodeID retrieves all snapshots for a specific node across all reports
func (r *SnapshotRepository) GetByNodeID(ctx context.Context, nodeID string, limit int) ([]*models.HealthCheckSnapshot, error) {
	query := `
		SELECT id, report_id, node_id, phase_name, created_at, url, page_title,
		       status_code, screenshot_path, dom_snapshot_path, console_logs,
		       selector_type, selector_value, elements_found, error_message, metadata
		FROM health_check_snapshots
		WHERE node_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Pool.Query(ctx, query, nodeID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []*models.HealthCheckSnapshot
	for rows.Next() {
		snapshot := &models.HealthCheckSnapshot{}
		err := rows.Scan(
			&snapshot.ID, &snapshot.ReportID, &snapshot.NodeID, &snapshot.PhaseName,
			&snapshot.CreatedAt, &snapshot.URL, &snapshot.PageTitle, &snapshot.StatusCode,
			&snapshot.ScreenshotPath, &snapshot.DOMSnapshotPath, &snapshot.ConsoleLogs,
			&snapshot.SelectorType, &snapshot.SelectorValue, &snapshot.ElementsFound,
			&snapshot.ErrorMessage, &snapshot.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}

		// Unmarshal JSONB fields
		if len(snapshot.ConsoleLogs) > 0 {
			if err := json.Unmarshal(snapshot.ConsoleLogs, &snapshot.ConsoleLogsData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal console logs: %w", err)
			}
		}
		if len(snapshot.Metadata) > 0 {
			if err := json.Unmarshal(snapshot.Metadata, &snapshot.MetadataData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, rows.Err()
}

// Delete removes a snapshot from the database (note: files must be deleted separately)
func (r *SnapshotRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM health_check_snapshots WHERE id = $1`
	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("snapshot not found")
	}

	return nil
}

// DeleteByReportID removes all snapshots for a report
func (r *SnapshotRepository) DeleteByReportID(ctx context.Context, reportID string) error {
	query := `DELETE FROM health_check_snapshots WHERE report_id = $1`
	_, err := r.db.Pool.Exec(ctx, query, reportID)
	if err != nil {
		return fmt.Errorf("failed to delete snapshots by report: %w", err)
	}
	return nil
}
