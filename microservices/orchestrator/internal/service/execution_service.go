package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/shared/queue"
	"go.uber.org/zap"
)

// ExecutionService handles execution orchestration
type ExecutionService struct {
	workflowRepo  repository.WorkflowRepository
	executionRepo repository.ExecutionRepository
	workflowSvc   *WorkflowService
	pubsubClient  *queue.PubSubClient
}

// NewExecutionService creates a new execution service
func NewExecutionService(
	workflowRepo repository.WorkflowRepository,
	executionRepo repository.ExecutionRepository,
	workflowSvc *WorkflowService,
	pubsubClient *queue.PubSubClient,
) *ExecutionService {
	return &ExecutionService{
		workflowRepo:  workflowRepo,
		executionRepo: executionRepo,
		workflowSvc:   workflowSvc,
		pubsubClient:  pubsubClient,
	}
}

// StartExecution starts a new workflow execution
func (s *ExecutionService) StartExecution(ctx context.Context, workflowID string) (*models.Execution, error) {
	// Get workflow (from cache if available)
	workflow, err := s.workflowSvc.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Validate workflow is active
	if workflow.Status != "active" {
		return nil, fmt.Errorf("workflow is not active: %s", workflow.Status)
	}

	// Create execution record
	execution := &models.Execution{
		WorkflowID: workflowID,
		Metadata:   make(map[string]interface{}),
	}

	if err := s.executionRepo.Create(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	logger.Info("Execution created",
		zap.String("execution_id", execution.ID),
		zap.String("workflow_id", workflowID),
	)

	// Enqueue start URLs as tasks
	if err := s.enqueueStartURLs(ctx, workflow, execution); err != nil {
		// Mark execution as failed
		s.executionRepo.Complete(ctx, execution.ID, "failed")
		return nil, fmt.Errorf("failed to enqueue start URLs: %w", err)
	}

	logger.Info("Start URLs enqueued",
		zap.String("execution_id", execution.ID),
		zap.Int("url_count", len(workflow.Config.StartURLs)),
	)

	return execution, nil
}

// GetExecution retrieves an execution by ID
func (s *ExecutionService) GetExecution(ctx context.Context, id string) (*models.Execution, error) {
	return s.executionRepo.Get(ctx, id)
}

// ListExecutions retrieves executions for a workflow
func (s *ExecutionService) ListExecutions(ctx context.Context, workflowID string, filters repository.ListFilters) ([]*models.Execution, error) {
	return s.executionRepo.List(ctx, workflowID, filters)
}

// StopExecution stops a running execution
func (s *ExecutionService) StopExecution(ctx context.Context, id string) error {
	execution, err := s.executionRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	if execution.Status != "running" {
		return fmt.Errorf("execution is not running: %s", execution.Status)
	}

	if err := s.executionRepo.Complete(ctx, id, "stopped"); err != nil {
		return err
	}

	logger.Info("Execution stopped", zap.String("execution_id", id))

	return nil
}

// enqueueStartURLs creates tasks for all start URLs
func (s *ExecutionService) enqueueStartURLs(ctx context.Context, workflow *models.Workflow, execution *models.Execution) error {
	if len(workflow.Config.Phases) == 0 {
		return fmt.Errorf("no phases defined in workflow")
	}

	// Use first phase for start URLs
	firstPhase := workflow.Config.Phases[0]

	// Prepare metadata with workflow config for phase transitions
	metadata := map[string]interface{}{
		"max_depth":        workflow.Config.MaxDepth,
		"rate_limit_delay": workflow.Config.RateLimitDelay,
		"phases":           workflow.Config.Phases, // Include phases for transitions
	}

	tasks := make([]*models.Task, 0, len(workflow.Config.StartURLs))

	for _, startURL := range workflow.Config.StartURLs {
		task := &models.Task{
			TaskID:      uuid.New().String(),
			ExecutionID: execution.ID,
			WorkflowID:  workflow.ID,
			URL:         startURL,
			Depth:       0,
			ParentURLID: nil,
			PhaseID:     firstPhase.ID,
			PhaseConfig: firstPhase,
			Metadata:    metadata,
			RetryCount:  0,
		}

		tasks = append(tasks, task)
	}

	// Publish tasks to Pub/Sub
	if err := s.pubsubClient.PublishBatch(ctx, tasks); err != nil {
		return fmt.Errorf("failed to publish tasks: %w", err)
	}

	return nil
}

// UpdateExecutionStats updates execution statistics (called by workers)
func (s *ExecutionService) UpdateExecutionStats(ctx context.Context, executionID string, stats repository.ExecutionStats) error {
	return s.executionRepo.UpdateStats(ctx, executionID, stats)
}
