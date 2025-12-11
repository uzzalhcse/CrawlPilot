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
type UniversalScraperService struct {
	scrapeTopic *pubsub.Topic
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
	// Create Pub/Sub client for scrape topic
	client, err := pubsub.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	// Get or create scrape topic
	topicName := cfg.ScrapeTopic
	if topicName == "" {
		topicName = "scrape-tasks"
	}

	topic := client.Topic(topicName)
	exists, err := topic.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check topic existence: %w", err)
	}

	if !exists {
		// Create the topic if it doesn't exist
		topic, err = client.CreateTopic(ctx, topicName)
		if err != nil {
			return nil, fmt.Errorf("failed to create scrape topic: %w", err)
		}
		logger.Info("Created scrape topic", zap.String("topic", topicName))
	}

	logger.Info("Universal scraper service initialized",
		zap.String("topic", topicName),
	)

	return &UniversalScraperService{
		scrapeTopic: topic,
		profileRepo: profileRepo,
		cache:       cache,
	}, nil
}

// SubmitScrape publishes a scrape request to the queue and stores initial status in cache
func (s *UniversalScraperService) SubmitScrape(ctx context.Context, req *models.ScrapeRequest) error {
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

	// Serialize request
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal scrape request: %w", err)
	}

	// Publish to scrape topic
	result2 := s.scrapeTopic.Publish(ctx, &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			"scrape_id": req.ID,
			"driver":    req.Driver,
			"format":    req.OutputFormat,
		},
	})

	// Wait for publish to complete
	_, err = result2.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish scrape request: %w", err)
	}

	logger.Info("Scrape request published",
		zap.String("scrape_id", req.ID),
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
	if s.scrapeTopic != nil {
		s.scrapeTopic.Stop()
	}
}
