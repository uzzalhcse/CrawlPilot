package service

import (
	"context"
	"sync"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// Scheduler is a background service that executes scheduled workflows
type Scheduler struct {
	scheduleSvc  *ScheduleService
	executionSvc *ExecutionService
	stopCh       chan struct{}
	wg           sync.WaitGroup
	interval     time.Duration
}

// NewScheduler creates a new scheduler
func NewScheduler(scheduleSvc *ScheduleService, executionSvc *ExecutionService) *Scheduler {
	return &Scheduler{
		scheduleSvc:  scheduleSvc,
		executionSvc: executionSvc,
		stopCh:       make(chan struct{}),
		interval:     1 * time.Minute, // Check every minute
	}
}

// Start starts the scheduler background goroutine
func (s *Scheduler) Start() {
	s.wg.Add(1)
	go s.run()
	logger.Info("Scheduler started", zap.Duration("interval", s.interval))
}

// Stop stops the scheduler gracefully
func (s *Scheduler) Stop() {
	logger.Info("Stopping scheduler...")
	close(s.stopCh)
	s.wg.Wait()
	logger.Info("Scheduler stopped")
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	defer s.wg.Done()

	// Initial check on startup
	s.checkAndExecuteDueSchedules()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkAndExecuteDueSchedules()
		case <-s.stopCh:
			return
		}
	}
}

// checkAndExecuteDueSchedules checks for due schedules and executes them
func (s *Scheduler) checkAndExecuteDueSchedules() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()

	// Get all due schedules
	schedules, err := s.scheduleSvc.GetDueSchedules(ctx, now)
	if err != nil {
		logger.Error("Failed to get due schedules", zap.Error(err))
		return
	}

	if len(schedules) == 0 {
		return
	}

	logger.Info("Found due schedules", zap.Int("count", len(schedules)))

	// Execute each due schedule
	for _, schedule := range schedules {
		s.executeSchedule(ctx, schedule)
	}
}

// executeSchedule executes a single scheduled workflow
func (s *Scheduler) executeSchedule(ctx context.Context, schedule *models.Schedule) {
	logger.Info("Executing scheduled workflow",
		zap.String("schedule_id", schedule.ID),
		zap.String("schedule_name", schedule.Name),
		zap.String("workflow_id", schedule.WorkflowID),
	)

	// Start the workflow execution
	executionID, err := s.executionSvc.StartScheduledExecution(ctx, schedule.WorkflowID, schedule.ID)
	if err != nil {
		logger.Error("Failed to execute scheduled workflow",
			zap.String("schedule_id", schedule.ID),
			zap.String("workflow_id", schedule.WorkflowID),
			zap.Error(err),
		)
		// Still update next_run_at even on failure to prevent infinite retries
		if updateErr := s.scheduleSvc.UpdateAfterRun(ctx, schedule, ""); updateErr != nil {
			logger.Error("Failed to update schedule after failed run",
				zap.String("schedule_id", schedule.ID),
				zap.Error(updateErr),
			)
		}
		return
	}

	logger.Info("Scheduled workflow execution started",
		zap.String("schedule_id", schedule.ID),
		zap.String("execution_id", executionID),
	)

	// Update schedule with new last_run_at and next_run_at
	if err := s.scheduleSvc.UpdateAfterRun(ctx, schedule, executionID); err != nil {
		logger.Error("Failed to update schedule after run",
			zap.String("schedule_id", schedule.ID),
			zap.Error(err),
		)
	}
}
