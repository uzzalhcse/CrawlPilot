package monitoring

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/monitoring/notifications"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// SchedulerService manages scheduled monitoring execution
type SchedulerService struct {
	scheduleRepo   *storage.MonitoringScheduleRepository
	monitoringRepo *storage.MonitoringRepository
	workflowRepo   *storage.WorkflowRepository
	orchestrator   *Orchestrator
	slackNotifier  *notifications.SlackNotifier
	stopChan       chan struct{}
	running        bool
}

// NewSchedulerService creates a new scheduler service
func NewSchedulerService(
	scheduleRepo *storage.MonitoringScheduleRepository,
	monitoringRepo *storage.MonitoringRepository,
	workflowRepo *storage.WorkflowRepository,
	orchestrator *Orchestrator,
) *SchedulerService {
	return &SchedulerService{
		scheduleRepo:   scheduleRepo,
		monitoringRepo: monitoringRepo,
		workflowRepo:   workflowRepo,
		orchestrator:   orchestrator,
		slackNotifier:  notifications.NewSlackNotifier(),
		stopChan:       make(chan struct{}),
	}
}

// Start begins the scheduler's main loop
func (s *SchedulerService) Start() {
	if s.running {
		return
	}

	s.running = true
	logger.Info("Monitoring scheduler started")

	// Run schedule check every minute
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkAndRunSchedules()
		case <-s.stopChan:
			logger.Info("Monitoring scheduler stopped")
			return
		}
	}
}

// Stop stops the scheduler
func (s *SchedulerService) Stop() {
	if !s.running {
		return
	}
	close(s.stopChan)
	s.running = false
}

// checkAndRunSchedules checks for due schedules and executes them
func (s *SchedulerService) checkAndRunSchedules() {
	ctx := context.Background()

	schedules, err := s.scheduleRepo.GetDueSchedules(ctx)
	if err != nil {
		logger.Error("Failed to get due schedules", zap.Error(err))
		return
	}

	for _, schedule := range schedules {
		go s.executeScheduledHealthCheck(schedule)
	}
}

// executeScheduledHealthCheck runs a monitoring for a schedule
func (s *SchedulerService) executeScheduledHealthCheck(schedule *models.MonitoringSchedule) {
	ctx := context.Background()

	logger.Info("Executing scheduled monitoring",
		zap.String("schedule_id", schedule.ID),
		zap.String("workflow_id", schedule.WorkflowID))

	// Get workflow
	workflow, err := s.workflowRepo.GetByID(ctx, schedule.WorkflowID)
	if err != nil {
		logger.Error("Failed to get workflow for scheduled monitoring",
			zap.String("workflow_id", schedule.WorkflowID),
			zap.Error(err))
		return
	}

	// Run monitoring
	report, err := s.orchestrator.RunMonitoring(ctx, workflow)
	if err != nil {
		logger.Error("Scheduled monitoring failed",
			zap.String("workflow_id", schedule.WorkflowID),
			zap.Error(err))
		return
	}

	// Save report
	err = s.monitoringRepo.Create(ctx, report)
	if err != nil {
		logger.Error("Failed to save monitoring report",
			zap.String("report_id", report.ID),
			zap.Error(err))
	}

	// Update schedule last run time
	now := time.Now()
	schedule.LastRunAt = &now
	schedule.NextRunAt = calculateNextRun(schedule.Schedule, now)

	err = s.scheduleRepo.Update(ctx, schedule)
	if err != nil {
		logger.Error("Failed to update schedule",
			zap.String("schedule_id", schedule.ID),
			zap.Error(err))
	}

	// Send notifications if configured
	if schedule.NotificationConfig != nil {
		s.sendNotifications(schedule.NotificationConfig, report, workflow.Name)
	}

	logger.Info("Scheduled monitoring completed",
		zap.String("schedule_id", schedule.ID),
		zap.String("report_id", report.ID),
		zap.String("status", string(report.Status)))
}

// sendNotifications sends notifications based on config
func (s *SchedulerService) sendNotifications(config *models.NotificationConfig, report *models.MonitoringReport, workflowName string) {
	// Check if we should notify based on config
	if config.OnlyOnFailure && report.Status == models.MonitoringStatusHealthy {
		return
	}

	// Send Slack notification
	if config.Slack != nil {
		err := s.slackNotifier.Send(config.Slack, report, workflowName)
		if err != nil {
			logger.Error("Failed to send Slack notification",
				zap.String("report_id", report.ID),
				zap.Error(err))
		} else {
			logger.Info("Slack notification sent",
				zap.String("report_id", report.ID),
				zap.String("status", string(report.Status)))
		}
	}
}

// calculateNextRun calculates the next run time based on cron schedule
func calculateNextRun(cronSchedule string, fromTime time.Time) *time.Time {
	// Simple implementation - for production use a proper cron parser library
	// For now, support basic intervals like "*/6 * * * *" (every 6 hours)

	// This is a placeholder - in production use github.com/robfig/cron
	// For MVP, we'll add a fixed interval
	next := fromTime.Add(6 * time.Hour)
	return &next
}

// CreateSchedule creates a new monitoring schedule
func (s *SchedulerService) CreateSchedule(ctx context.Context, workflowID, cronSchedule string, config *models.NotificationConfig) (*models.MonitoringSchedule, error) {
	schedule := &models.MonitoringSchedule{
		ID:                 uuid.New().String(),
		WorkflowID:         workflowID,
		Schedule:           cronSchedule,
		Enabled:            true,
		NotificationConfig: config,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	now := time.Now()
	schedule.NextRunAt = calculateNextRun(cronSchedule, now)

	err := s.scheduleRepo.Create(ctx, schedule)
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	return schedule, nil
}
