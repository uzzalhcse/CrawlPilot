package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/shared/queue"
	"go.uber.org/zap"
)

const (
	outstandingKeyPrefix = "crawlify:outstanding:"
	completionTTL        = 24 * time.Hour
)

// ExecutionService handles execution orchestration
type ExecutionService struct {
	workflowRepo  repository.WorkflowRepository
	executionRepo repository.ExecutionRepository
	profileRepo   repository.BrowserProfileRepository // For node-level profile resolution
	workflowSvc   *WorkflowService
	pubsubClient  *queue.PubSubClient
	redisCache    *cache.Cache // For completion tracking
}

// NewExecutionService creates a new execution service
func NewExecutionService(
	workflowRepo repository.WorkflowRepository,
	executionRepo repository.ExecutionRepository,
	profileRepo repository.BrowserProfileRepository,
	workflowSvc *WorkflowService,
	pubsubClient *queue.PubSubClient,
	redisCache *cache.Cache,
) *ExecutionService {
	return &ExecutionService{
		workflowRepo:  workflowRepo,
		executionRepo: executionRepo,
		profileRepo:   profileRepo,
		workflowSvc:   workflowSvc,
		pubsubClient:  pubsubClient,
		redisCache:    redisCache,
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

// ListAllExecutions retrieves all executions across all workflows with optional filters
func (s *ExecutionService) ListAllExecutions(ctx context.Context, filters repository.ListFilters) ([]*models.Execution, error) {
	return s.executionRepo.ListAll(ctx, filters)
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

	// Resolve all node-level browser profiles upfront (zero DB calls during execution)
	nodeProfiles := s.resolveNodeProfiles(ctx, workflow.Config.Phases)

	// Prepare metadata with workflow config for phase transitions
	metadata := map[string]interface{}{
		"max_depth":        workflow.Config.MaxDepth,
		"rate_limit_delay": workflow.Config.RateLimitDelay,
		"phases":           workflow.Config.Phases, // Include phases for transitions
	}

	// Embed resolved profiles in metadata (workers read from here, no API calls)
	if len(nodeProfiles) > 0 {
		metadata["node_profiles"] = nodeProfiles
		logger.Info("Embedded node profiles in task metadata",
			zap.Int("profile_count", len(nodeProfiles)),
		)
	}

	tasks := make([]*models.Task, 0, len(workflow.Config.StartURLs))

	for _, startURL := range workflow.Config.StartURLs {
		task := &models.Task{
			TaskID:           uuid.New().String(),
			ExecutionID:      execution.ID,
			WorkflowID:       workflow.ID,
			URL:              startURL,
			Depth:            0,
			ParentURLID:      nil,
			PhaseID:          firstPhase.ID,
			PhaseConfig:      firstPhase,
			WorkflowConfig:   &workflow.Config, // Pass workflow config so worker can access defaults
			Metadata:         metadata,
			RetryCount:       0,
			BrowserProfileID: workflow.BrowserProfileID, // Pass browser profile to task
		}

		tasks = append(tasks, task)
	}

	// Publish tasks to Pub/Sub
	if err := s.pubsubClient.PublishBatch(ctx, tasks); err != nil {
		return fmt.Errorf("failed to publish tasks: %w", err)
	}

	// Initialize outstanding task count in Redis for completion tracking
	if s.redisCache != nil && len(tasks) > 0 {
		key := outstandingKeyPrefix + execution.ID
		_, err := s.redisCache.IncrBy(ctx, key, int64(len(tasks)))
		if err != nil {
			logger.Warn("Failed to initialize outstanding task count",
				zap.String("execution_id", execution.ID),
				zap.Error(err),
			)
		} else {
			s.redisCache.Expire(ctx, key, completionTTL)
			logger.Info("Initialized outstanding task count",
				zap.String("execution_id", execution.ID),
				zap.Int("count", len(tasks)),
			)
		}
	}

	return nil
}

// resolveNodeProfiles collects and fetches all browser profiles referenced in workflow nodes
// This is called ONCE at execution start - profiles are embedded in task metadata for workers
func (s *ExecutionService) resolveNodeProfiles(ctx context.Context, phases []models.WorkflowPhase) map[string]interface{} {
	profileIDs := make(map[string]bool)

	// Collect unique profile IDs from all nodes
	for _, phase := range phases {
		for _, node := range phase.Nodes {
			if profileID, ok := node.Params["browser_profile_id"].(string); ok && profileID != "" {
				profileIDs[profileID] = true
			}
		}
	}

	if len(profileIDs) == 0 {
		return nil
	}

	// Fetch profiles (one DB query per profile, done once at start)
	profiles := make(map[string]interface{})
	for profileID := range profileIDs {
		profile, err := s.profileRepo.Get(ctx, profileID)
		if err != nil {
			logger.Warn("Failed to fetch node profile, will be skipped",
				zap.String("profile_id", profileID),
				zap.Error(err),
			)
			continue
		}
		profiles[profileID] = profile
	}

	return profiles
}

// UpdateExecutionStats updates execution statistics (called by workers)
func (s *ExecutionService) UpdateExecutionStats(ctx context.Context, executionID string, stats repository.ExecutionStats) error {
	return s.executionRepo.UpdateStats(ctx, executionID, stats)
}

// GetExecutionErrors retrieves error logs for an execution
func (s *ExecutionService) GetExecutionErrors(ctx context.Context, executionID string, limit, offset int) ([]*models.ExecutionError, error) {
	return s.executionRepo.GetErrors(ctx, executionID, limit, offset)
}

// BatchInsertErrors inserts multiple errors (called by internal stats batch endpoint)
func (s *ExecutionService) BatchInsertErrors(ctx context.Context, errors []models.ExecutionError) error {
	return s.executionRepo.BatchInsertErrors(ctx, errors)
}
