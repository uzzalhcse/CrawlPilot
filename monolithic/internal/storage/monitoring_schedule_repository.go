package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/uzzalhcse/crawlify/pkg/models"
)

// MonitoringScheduleRepository handles schedule persistence
type MonitoringScheduleRepository struct {
	db *PostgresDB
}

// NewMonitoringScheduleRepository creates a new schedule repository
func NewMonitoringScheduleRepository(db *PostgresDB) *MonitoringScheduleRepository {
	return &MonitoringScheduleRepository{db: db}
}

// Create creates a new monitoring schedule
func (r *MonitoringScheduleRepository) Create(ctx context.Context, schedule *models.MonitoringSchedule) error {
	notificationJSON, _ := json.Marshal(schedule.NotificationConfig)

	query := `
		INSERT INTO monitoring_schedules
		(id, workflow_id, schedule, enabled, last_run_at, next_run_at, notification_config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		schedule.ID,
		schedule.WorkflowID,
		schedule.Schedule,
		schedule.Enabled,
		schedule.LastRunAt,
		schedule.NextRunAt,
		notificationJSON,
		schedule.CreatedAt,
		schedule.UpdatedAt,
	)

	return err
}

// GetByWorkflowID gets the schedule for a workflow
func (r *MonitoringScheduleRepository) GetByWorkflowID(ctx context.Context, workflowID string) (*models.MonitoringSchedule, error) {
	query := `
		SELECT id, workflow_id, schedule, enabled, last_run_at, next_run_at, notification_config, created_at, updated_at
		FROM monitoring_schedules
		WHERE workflow_id = $1
	`

	schedule := &models.MonitoringSchedule{}
	var notificationJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, workflowID).Scan(
		&schedule.ID,
		&schedule.WorkflowID,
		&schedule.Schedule,
		&schedule.Enabled,
		&schedule.LastRunAt,
		&schedule.NextRunAt,
		&notificationJSON,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if len(notificationJSON) > 0 {
		json.Unmarshal(notificationJSON, &schedule.NotificationConfig)
	}

	return schedule, nil
}

// Update updates an existing schedule
func (r *MonitoringScheduleRepository) Update(ctx context.Context, schedule *models.MonitoringSchedule) error {
	notificationJSON, _ := json.Marshal(schedule.NotificationConfig)

	query := `
		UPDATE monitoring_schedules
		SET schedule = $1, enabled = $2, last_run_at = $3, next_run_at = $4, 
		    notification_config = $5, updated_at = $6
		WHERE id = $7
	`

	_, err := r.db.Pool.Exec(ctx, query,
		schedule.Schedule,
		schedule.Enabled,
		schedule.LastRunAt,
		schedule.NextRunAt,
		notificationJSON,
		time.Now(),
		schedule.ID,
	)

	return err
}

// Delete deletes a schedule
func (r *MonitoringScheduleRepository) Delete(ctx context.Context, workflowID string) error {
	query := `DELETE FROM monitoring_schedules WHERE workflow_id = $1`
	_, err := r.db.Pool.Exec(ctx, query, workflowID)
	return err
}

// GetDueSchedules gets all enabled schedules that are due to run
func (r *MonitoringScheduleRepository) GetDueSchedules(ctx context.Context) ([]*models.MonitoringSchedule, error) {
	query := `
		SELECT id, workflow_id, schedule, enabled, last_run_at, next_run_at, notification_config, created_at, updated_at
		FROM monitoring_schedules
		WHERE enabled = true AND (next_run_at IS NULL OR next_run_at <= $1)
		ORDER BY next_run_at ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []*models.MonitoringSchedule
	for rows.Next() {
		schedule := &models.MonitoringSchedule{}
		var notificationJSON []byte

		err := rows.Scan(
			&schedule.ID,
			&schedule.WorkflowID,
			&schedule.Schedule,
			&schedule.Enabled,
			&schedule.LastRunAt,
			&schedule.NextRunAt,
			&notificationJSON,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)

		if err != nil {
			continue
		}

		if len(notificationJSON) > 0 {
			json.Unmarshal(notificationJSON, &schedule.NotificationConfig)
		}

		schedules = append(schedules, schedule)
	}

	return schedules, nil
}
