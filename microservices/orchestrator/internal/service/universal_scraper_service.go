package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

const (
	scrapeResultKeyPrefix = "scrape:"
	scrapeResultTTL       = 10 * time.Minute
)

// UniversalScraperService handles universal scraper operations
// Uses the regular task queue for processing, enabling reuse of recovery infrastructure
type UniversalScraperService struct {
	taskTopic   *pubsub.Topic // Regular workflow-tasks topic
	profileRepo repository.BrowserProfileRepository
	cache       *cache.Cache
}

// NewUniversalScraperService creates a new universal scraper service
func NewUniversalScraperService(
	ctx context.Context,
	cfg *config.GCPConfig,
	profileRepo repository.BrowserProfileRepository,
	cache *cache.Cache,
) (*UniversalScraperService, error) {
	// Create Pub/Sub client
	client, err := pubsub.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	// Use regular workflow-tasks topic (same as workflows)
	// This enables reuse of recovery, proxy rotation, tier escalation
	topicName := cfg.PubSubTopic
	if topicName == "" {
		topicName = "workflow-tasks"
	}

	topic := client.Topic(topicName)
	exists, err := topic.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check topic existence: %w", err)
	}

	if !exists {
		return nil, fmt.Errorf("task topic %s does not exist", topicName)
	}

	logger.Info("Universal scraper service initialized (using workflow task queue)",
		zap.String("topic", topicName),
	)

	return &UniversalScraperService{
		taskTopic:   topic,
		profileRepo: profileRepo,
		cache:       cache,
	}, nil
}

// SubmitScrape converts scrape request to a synthetic Task and publishes to task queue
// This allows Universal Scraper to reuse all workflow infrastructure (recovery, proxy rotation, etc.)
func (s *UniversalScraperService) SubmitScrape(ctx context.Context, req *models.ScrapeRequest) error {
	// If profile ID is provided, fetch and embed the profile
	if req.ProfileID != "" && s.profileRepo != nil {
		profile, err := s.profileRepo.Get(ctx, req.ProfileID)
		if err != nil {
			logger.Warn("Failed to fetch profile, using default settings",
				zap.String("profile_id", req.ProfileID),
				zap.Error(err),
			)
		} else {
			req.Profile = profile
			// Use profile's driver type if not explicitly specified
			if req.Driver == "" || req.Driver == "http" {
				req.Driver = profile.DriverType
			}
			logger.Info("Embedded profile in scrape request",
				zap.String("profile_id", req.ProfileID),
				zap.String("profile_name", profile.Name),
			)
		}
	}

	// Store initial pending status in Redis
	result := &models.ScrapeResult{
		ID:     req.ID,
		Status: models.ScrapeStatusPending,
		URL:    req.URL,
	}

	if err := s.storeResult(ctx, result); err != nil {
		logger.Warn("Failed to store initial scrape status",
			zap.String("scrape_id", req.ID),
			zap.Error(err),
		)
	}

	// Convert ScrapeRequest to synthetic Task
	// This is the key change - reuses workflow infrastructure
	task := req.ToTask()

	// Serialize task (not raw request)
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// Publish to regular task topic (same as workflows!)
	pubResult := s.taskTopic.Publish(ctx, &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			"execution_id": task.ExecutionID,
			"workflow_id":  task.WorkflowID,
			"scrape_id":    req.ID,
		},
	})

	// Wait for publish to complete
	_, err = pubResult.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish task: %w", err)
	}

	logger.Info("Scrape request submitted as synthetic task",
		zap.String("scrape_id", req.ID),
		zap.String("task_id", task.TaskID),
		zap.String("url", req.URL),
		zap.String("driver", req.Driver),
	)

	return nil
}

// GetResult retrieves a scrape result from cache
func (s *UniversalScraperService) GetResult(ctx context.Context, id string) (*models.ScrapeResult, error) {
	if s.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}

	key := scrapeResultKeyPrefix + id
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("result not found: %w", err)
	}

	var result models.ScrapeResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// StoreResult stores a scrape result in cache (called by worker via internal API)
func (s *UniversalScraperService) StoreResult(ctx context.Context, result *models.ScrapeResult) error {
	return s.storeResult(ctx, result)
}

// storeResult internal method to store result
func (s *UniversalScraperService) storeResult(ctx context.Context, result *models.ScrapeResult) error {
	if s.cache == nil {
		return fmt.Errorf("cache not available")
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	key := scrapeResultKeyPrefix + result.ID
	return s.cache.Set(ctx, key, string(data), scrapeResultTTL)
}

// ListProfiles returns browser profiles for dropdown
func (s *UniversalScraperService) ListProfiles(ctx context.Context) ([]map[string]interface{}, error) {
	profiles, err := s.profileRepo.List(ctx, repository.BrowserProfileFilters{
		Limit:  100,
		Offset: 0,
	})
	if err != nil {
		return nil, err
	}

	// Convert to lightweight response
	result := make([]map[string]interface{}, 0, len(profiles)+1)

	// Add "No Profile" option
	result = append(result, map[string]interface{}{
		"id":   "",
		"name": "No Profile",
	})

	for _, p := range profiles {
		result = append(result, map[string]interface{}{
			"id":   p.ID,
			"name": p.Name,
		})
	}

	return result, nil
}

// Close cleans up resources
func (s *UniversalScraperService) Close() {
	if s.taskTopic != nil {
		s.taskTopic.Stop()
	}
}
