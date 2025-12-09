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

// postgresScheduleRepo implements ScheduleRepository using PostgreSQL
type postgresScheduleRepo struct {
	db *database.DB
}

// NewScheduleRepository creates a new PostgreSQL schedule repository
func NewScheduleRepository(db *database.DB) ScheduleRepository {
	return &postgresScheduleRepo{db: db}
}

func (r *postgresScheduleRepo) Create(ctx context.Context, schedule *models.Schedule) error {
	if schedule.ID == "" {
		schedule.ID = uuid.New().String()
	}

	if schedule.Timezone == "" {
		schedule.Timezone = "UTC"
	}

	now := time.Now()
	schedule.CreatedAt = now
	schedule.UpdatedAt = now

	query := `
		INSERT INTO schedules (id, workflow_id, name, cron_expression, timezone, is_enabled, next_run_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		schedule.ID,
		schedule.WorkflowID,
		schedule.Name,
		schedule.CronExpression,
		schedule.Timezone,
		schedule.IsEnabled,
		schedule.NextRunAt,
		schedule.CreatedAt,
		schedule.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	return nil
}

func (r *postgresScheduleRepo) Get(ctx context.Context, id string) (*models.Schedule, error) {
	query := `
		SELECT s.id, s.workflow_id, s.name, s.cron_expression, s.timezone, s.is_enabled, 
		       s.next_run_at, s.last_run_at, s.last_execution_id, s.created_at, s.updated_at,
		       w.name as workflow_name
		FROM schedules s
		LEFT JOIN workflows w ON s.workflow_id = w.id
		WHERE s.id = $1
	`

	var schedule models.Schedule
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&schedule.ID,
		&schedule.WorkflowID,
		&schedule.Name,
		&schedule.CronExpression,
		&schedule.Timezone,
		&schedule.IsEnabled,
		&schedule.NextRunAt,
		&schedule.LastRunAt,
		&schedule.LastExecutionID,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
		&schedule.WorkflowName,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("schedule not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	return &schedule, nil
}

func (r *postgresScheduleRepo) List(ctx context.Context, filters ScheduleFilters) ([]*models.Schedule, error) {
	query := `
		SELECT s.id, s.workflow_id, s.name, s.cron_expression, s.timezone, s.is_enabled,
		       s.next_run_at, s.last_run_at, s.last_execution_id, s.created_at, s.updated_at,
		       w.name as workflow_name
		FROM schedules s
		LEFT JOIN workflows w ON s.workflow_id = w.id
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	if filters.WorkflowID != "" {
		query += fmt.Sprintf(" AND s.workflow_id = $%d", argPos)
		args = append(args, filters.WorkflowID)
		argPos++
	}

	if filters.IsEnabled != nil {
		query += fmt.Sprintf(" AND s.is_enabled = $%d", argPos)
		args = append(args, *filters.IsEnabled)
		argPos++
	}

	query += " ORDER BY s.created_at DESC"

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
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}
	defer rows.Close()

	schedules := make([]*models.Schedule, 0)
	for rows.Next() {
		var schedule models.Schedule
		err := rows.Scan(
			&schedule.ID,
			&schedule.WorkflowID,
			&schedule.Name,
			&schedule.CronExpression,
			&schedule.Timezone,
			&schedule.IsEnabled,
			&schedule.NextRunAt,
			&schedule.LastRunAt,
			&schedule.LastExecutionID,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
			&schedule.WorkflowName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}
		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}

func (r *postgresScheduleRepo) Update(ctx context.Context, schedule *models.Schedule) error {
	schedule.UpdatedAt = time.Now()

	query := `
		UPDATE schedules
		SET name = $2, cron_expression = $3, timezone = $4, is_enabled = $5, next_run_at = $6, updated_at = $7
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query,
		schedule.ID,
		schedule.Name,
		schedule.CronExpression,
		schedule.Timezone,
		schedule.IsEnabled,
		schedule.NextRunAt,
		schedule.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("schedule not found: %s", schedule.ID)
	}

	return nil
}

func (r *postgresScheduleRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM schedules WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("schedule not found: %s", id)
	}

	return nil
}

func (r *postgresScheduleRepo) Toggle(ctx context.Context, id string) error {
	query := `
		UPDATE schedules
		SET is_enabled = NOT is_enabled, updated_at = $2
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to toggle schedule: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("schedule not found: %s", id)
	}

	return nil
}

func (r *postgresScheduleRepo) GetDueSchedules(ctx context.Context, now time.Time) ([]*models.Schedule, error) {
	query := `
		SELECT s.id, s.workflow_id, s.name, s.cron_expression, s.timezone, s.is_enabled,
		       s.next_run_at, s.last_run_at, s.last_execution_id, s.created_at, s.updated_at,
		       w.name as workflow_name
		FROM schedules s
		LEFT JOIN workflows w ON s.workflow_id = w.id
		WHERE s.is_enabled = true 
		  AND s.next_run_at IS NOT NULL 
		  AND s.next_run_at <= $1
		ORDER BY s.next_run_at ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get due schedules: %w", err)
	}
	defer rows.Close()

	schedules := make([]*models.Schedule, 0)
	for rows.Next() {
		var schedule models.Schedule
		err := rows.Scan(
			&schedule.ID,
			&schedule.WorkflowID,
			&schedule.Name,
			&schedule.CronExpression,
			&schedule.Timezone,
			&schedule.IsEnabled,
			&schedule.NextRunAt,
			&schedule.LastRunAt,
			&schedule.LastExecutionID,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
			&schedule.WorkflowName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan due schedule: %w", err)
		}
		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}

func (r *postgresScheduleRepo) UpdateAfterRun(ctx context.Context, id string, lastRunAt time.Time, nextRunAt time.Time, executionID string) error {
	query := `
		UPDATE schedules
		SET last_run_at = $2, next_run_at = $3, last_execution_id = $4, updated_at = $5
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query, id, lastRunAt, nextRunAt, executionID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update schedule after run: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("schedule not found: %s", id)
	}

	return nil
}
