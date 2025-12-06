package service

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// WorkflowService handles workflow business logic
type WorkflowService struct {
	repo  repository.WorkflowRepository
	cache *cache.Cache
}

// NewWorkflowService creates a new workflow service
func NewWorkflowService(repo repository.WorkflowRepository, cache *cache.Cache) *WorkflowService {
	return &WorkflowService{
		repo:  repo,
		cache: cache,
	}
}

// CreateWorkflow creates a new workflow with validation
func (s *WorkflowService) CreateWorkflow(ctx context.Context, workflow *models.Workflow) error {
	// Validate workflow
	if err := s.validateWorkflow(workflow); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Set initial status (only 'active' and 'inactive' are valid in database)
	if workflow.Status == "" || workflow.Status == "draft" {
		workflow.Status = "active"
	}

	// Create in database
	if err := s.repo.Create(ctx, workflow); err != nil {
		return fmt.Errorf("failed to create workflow: %w", err)
	}

	logger.Info("Workflow created",
		zap.String("workflow_id", workflow.ID),
		zap.String("name", workflow.Name),
	)

	return nil
}

// GetWorkflow retrieves a workflow, using cache if available
func (s *WorkflowService) GetWorkflow(ctx context.Context, id string) (*models.Workflow, error) {
	// Try cache first
	if s.cache != nil {
		cachedWorkflow, err := s.getFromCache(ctx, id)
		if err == nil && cachedWorkflow != nil {
			logger.Debug("Workflow cache hit", zap.String("workflow_id", id))
			return cachedWorkflow, nil
		}
	}

	// Cache miss - get from database
	workflow, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store in cache
	if s.cache != nil {
		if err := s.putInCache(ctx, workflow); err != nil {
			logger.Warn("Failed to cache workflow", zap.Error(err))
		}
	}

	return workflow, nil
}

// ListWorkflows retrieves workflows with filters
func (s *WorkflowService) ListWorkflows(ctx context.Context, filters repository.ListFilters) ([]*models.Workflow, error) {
	return s.repo.List(ctx, filters)
}

// UpdateWorkflow updates a workflow and invalidates cache
func (s *WorkflowService) UpdateWorkflow(ctx context.Context, workflow *models.Workflow) error {
	// Validate workflow
	if err := s.validateWorkflow(workflow); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Update in database
	if err := s.repo.Update(ctx, workflow); err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		s.invalidateCache(ctx, workflow.ID)
	}

	logger.Info("Workflow updated",
		zap.String("workflow_id", workflow.ID),
	)

	return nil
}

// DeleteWorkflow soft-deletes a workflow
func (s *WorkflowService) DeleteWorkflow(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		s.invalidateCache(ctx, id)
	}

	logger.Info("Workflow deleted", zap.String("workflow_id", id))

	return nil
}

// validateWorkflow validates workflow configuration
func (s *WorkflowService) validateWorkflow(workflow *models.Workflow) error {
	if workflow.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	if len(workflow.Config.StartURLs) == 0 {
		return fmt.Errorf("at least one start URL is required")
	}

	if len(workflow.Config.Phases) == 0 {
		return fmt.Errorf("at least one phase is required")
	}

	// Validate each phase has nodes
	for _, phase := range workflow.Config.Phases {
		if len(phase.Nodes) == 0 {
			return fmt.Errorf("phase %s has no nodes", phase.ID)
		}
	}

	return nil
}

// Cache helpers
func (s *WorkflowService) getFromCache(ctx context.Context, id string) (*models.Workflow, error) {
	key := fmt.Sprintf("workflow:%s", id)
	var workflow models.Workflow

	if err := s.cache.GetJSON(ctx, key, &workflow); err != nil {
		return nil, err
	}

	return &workflow, nil
}

func (s *WorkflowService) putInCache(ctx context.Context, workflow *models.Workflow) error {
	key := fmt.Sprintf("workflow:%s", workflow.ID)
	return s.cache.SetJSON(ctx, key, workflow, 3600) // 1 hour TTL
}

func (s *WorkflowService) invalidateCache(ctx context.Context, id string) {
	key := fmt.Sprintf("workflow:%s", id)
	if err := s.cache.Delete(ctx, key); err != nil {
		logger.Warn("Failed to invalidate cache", zap.Error(err))
	}
}
