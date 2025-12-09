package service

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// ScheduleService handles schedule business logic
type ScheduleService struct {
	repo       repository.ScheduleRepository
	cronParser cron.Parser
}

// NewScheduleService creates a new schedule service
func NewScheduleService(repo repository.ScheduleRepository) *ScheduleService {
	// Standard cron parser: minute hour day-of-month month day-of-week
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	return &ScheduleService{
		repo:       repo,
		cronParser: parser,
	}
}

// CreateSchedule creates a new schedule with validation
func (s *ScheduleService) CreateSchedule(ctx context.Context, schedule *models.Schedule) error {
	// Validate cron expression
	if err := s.validateCronExpression(schedule.CronExpression); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Validate timezone
	if schedule.Timezone != "" {
		if _, err := time.LoadLocation(schedule.Timezone); err != nil {
			return fmt.Errorf("invalid timezone: %w", err)
		}
	}

	// Calculate next run time
	nextRun, err := s.CalculateNextRun(schedule.CronExpression, schedule.Timezone, time.Now())
	if err != nil {
		return fmt.Errorf("failed to calculate next run: %w", err)
	}
	schedule.NextRunAt = &nextRun

	// Default to enabled
	schedule.IsEnabled = true

	if err := s.repo.Create(ctx, schedule); err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	logger.Info("Schedule created",
		zap.String("schedule_id", schedule.ID),
		zap.String("name", schedule.Name),
		zap.String("cron", schedule.CronExpression),
		zap.Time("next_run", *schedule.NextRunAt),
	)

	return nil
}

// GetSchedule retrieves a schedule by ID
func (s *ScheduleService) GetSchedule(ctx context.Context, id string) (*models.Schedule, error) {
	return s.repo.Get(ctx, id)
}

// ListSchedules retrieves schedules with filters
func (s *ScheduleService) ListSchedules(ctx context.Context, filters repository.ScheduleFilters) ([]*models.Schedule, error) {
	return s.repo.List(ctx, filters)
}

// UpdateSchedule updates a schedule
func (s *ScheduleService) UpdateSchedule(ctx context.Context, schedule *models.Schedule) error {
	// Validate cron expression if provided
	if schedule.CronExpression != "" {
		if err := s.validateCronExpression(schedule.CronExpression); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
	}

	// Validate timezone if provided
	if schedule.Timezone != "" {
		if _, err := time.LoadLocation(schedule.Timezone); err != nil {
			return fmt.Errorf("invalid timezone: %w", err)
		}
	}

	// Recalculate next run time if schedule is enabled and cron changed
	if schedule.IsEnabled && schedule.CronExpression != "" {
		nextRun, err := s.CalculateNextRun(schedule.CronExpression, schedule.Timezone, time.Now())
		if err != nil {
			return fmt.Errorf("failed to calculate next run: %w", err)
		}
		schedule.NextRunAt = &nextRun
	}

	if err := s.repo.Update(ctx, schedule); err != nil {
		return err
	}

	logger.Info("Schedule updated",
		zap.String("schedule_id", schedule.ID),
	)

	return nil
}

// DeleteSchedule deletes a schedule
func (s *ScheduleService) DeleteSchedule(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	logger.Info("Schedule deleted", zap.String("schedule_id", id))
	return nil
}

// ToggleSchedule enables or disables a schedule
func (s *ScheduleService) ToggleSchedule(ctx context.Context, id string) (*models.Schedule, error) {
	if err := s.repo.Toggle(ctx, id); err != nil {
		return nil, err
	}

	// Get updated schedule
	schedule, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// If enabled, recalculate next run
	if schedule.IsEnabled {
		nextRun, err := s.CalculateNextRun(schedule.CronExpression, schedule.Timezone, time.Now())
		if err == nil {
			schedule.NextRunAt = &nextRun
			_ = s.repo.Update(ctx, schedule)
		}
	}

	logger.Info("Schedule toggled",
		zap.String("schedule_id", id),
		zap.Bool("is_enabled", schedule.IsEnabled),
	)

	return schedule, nil
}

// GetDueSchedules retrieves all enabled schedules that are due to run
func (s *ScheduleService) GetDueSchedules(ctx context.Context, now time.Time) ([]*models.Schedule, error) {
	return s.repo.GetDueSchedules(ctx, now)
}

// UpdateAfterRun updates the schedule after execution
func (s *ScheduleService) UpdateAfterRun(ctx context.Context, schedule *models.Schedule, executionID string) error {
	now := time.Now()

	// Calculate next run time
	nextRun, err := s.CalculateNextRun(schedule.CronExpression, schedule.Timezone, now)
	if err != nil {
		return fmt.Errorf("failed to calculate next run: %w", err)
	}

	return s.repo.UpdateAfterRun(ctx, schedule.ID, now, nextRun, executionID)
}

// CalculateNextRun calculates the next run time based on cron expression and timezone
func (s *ScheduleService) CalculateNextRun(cronExpr string, timezone string, from time.Time) (time.Time, error) {
	schedule, err := s.cronParser.Parse(cronExpr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse cron expression: %w", err)
	}

	// Load timezone
	loc := time.UTC
	if timezone != "" {
		loc, err = time.LoadLocation(timezone)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to load timezone: %w", err)
		}
	}

	// Convert from time to the schedule's timezone, calculate next, then convert back to UTC
	fromInTz := from.In(loc)
	nextInTz := schedule.Next(fromInTz)
	nextUTC := nextInTz.UTC()

	return nextUTC, nil
}

// validateCronExpression validates a cron expression
func (s *ScheduleService) validateCronExpression(cronExpr string) error {
	_, err := s.cronParser.Parse(cronExpr)
	return err
}

// GetCronPresets returns common cron expression presets for the UI
func (s *ScheduleService) GetCronPresets() []CronPreset {
	return []CronPreset{
		{Name: "Every minute", Expression: "* * * * *", Description: "Runs every minute"},
		{Name: "Every 12 hours", Expression: "0 */12 * * *", Description: "Runs at midnight and noon"},
		{Name: "Daily", Expression: "0 0 * * *", Description: "Runs once a day at midnight"},
		{Name: "Weekly", Expression: "0 0 * * 0", Description: "Runs every Sunday at midnight"},
		{Name: "Bi-weekly", Expression: "0 0 1,15 * *", Description: "Runs on the 1st and 15th of each month"},
		{Name: "Monthly", Expression: "0 0 1 * *", Description: "Runs on the 1st of every month"},
		{Name: "10th & 25th", Expression: "0 0 10,25 * *", Description: "Runs on the 10th and 25th of every month"},
	}
}

// CronPreset represents a common cron expression preset
type CronPreset struct {
	Name        string `json:"name"`
	Expression  string `json:"expression"`
	Description string `json:"description"`
}
