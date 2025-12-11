package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/driver"
	"go.uber.org/zap"
)

// ScrapeHandler handles single URL scrape requests from the scrape topic
type ScrapeHandler struct {
	driverFactory   *driver.Factory
	orchestratorURL string
	httpClient      *http.Client
	timeout         time.Duration
}

// NewScrapeHandler creates a new scrape handler
func NewScrapeHandler(
	driverFactory *driver.Factory,
	orchestratorURL string,
) *ScrapeHandler {
	return &ScrapeHandler{
		driverFactory:   driverFactory,
		orchestratorURL: orchestratorURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout: 60 * time.Second,
	}
}

// StartSubscription starts listening to the scrape topic
func (h *ScrapeHandler) StartSubscription(ctx context.Context, cfg *config.GCPConfig) error {
	client, err := pubsub.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to create pubsub client: %w", err)
	}

	subName := cfg.ScrapeSubscription
	if subName == "" {
		subName = "scrape-tasks-sub"
	}

	sub := client.Subscription(subName)
	exists, err := sub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check subscription: %w", err)
	}

	if !exists {
		// Create subscription if it doesn't exist
		topicName := cfg.ScrapeTopic
		if topicName == "" {
			topicName = "scrape-tasks"
		}
		topic := client.Topic(topicName)

		sub, err = client.CreateSubscription(ctx, subName, pubsub.SubscriptionConfig{
			Topic:       topic,
			AckDeadline: 60 * time.Second,
		})
		if err != nil {
			return fmt.Errorf("failed to create subscription: %w", err)
		}
		logger.Info("Created scrape subscription", zap.String("subscription", subName))
	}

	// Configure for low latency single requests
	sub.ReceiveSettings.MaxOutstandingMessages = 5
	sub.ReceiveSettings.NumGoroutines = 2

	logger.Info("Starting scrape task subscription",
		zap.String("subscription", subName),
	)

	return sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		h.handleMessage(ctx, msg)
	})
}

// handleMessage processes a single scrape request
func (h *ScrapeHandler) handleMessage(ctx context.Context, msg *pubsub.Message) {
	startTime := time.Now()

	var req models.ScrapeRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		logger.Error("Failed to unmarshal scrape request",
			zap.String("message_id", msg.ID),
			zap.Error(err),
		)
		msg.Ack() // Ack to prevent retry of malformed message
		return
	}

	logger.Info("Processing scrape request",
		zap.String("scrape_id", req.ID),
		zap.String("url", req.URL),
		zap.String("driver", req.Driver),
		zap.String("format", req.OutputFormat),
	)

	// Execute scrape
	result := h.executeScrape(ctx, &req)
	result.Duration = time.Since(startTime).Milliseconds()

	// Send result to orchestrator
	if err := h.sendResult(ctx, result); err != nil {
		logger.Error("Failed to send scrape result",
			zap.String("scrape_id", req.ID),
			zap.Error(err),
		)
		msg.Nack() // Retry
		return
	}

	logger.Info("Scrape completed",
		zap.String("scrape_id", req.ID),
		zap.String("status", result.Status),
		zap.Int64("duration_ms", result.Duration),
	)

	msg.Ack()
}

// executeScrape performs the actual scraping
func (h *ScrapeHandler) executeScrape(ctx context.Context, req *models.ScrapeRequest) *models.ScrapeResult {
	result := &models.ScrapeResult{
		ID:     req.ID,
		URL:    req.URL,
		Status: models.ScrapeStatusRunning,
	}

	// Apply timeout
	timeout := time.Duration(req.Timeout) * time.Second
	if timeout == 0 {
		timeout = h.timeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create driver based on type
	d, err := h.createDriverByType(req.Driver)
	if err != nil {
		result.Status = models.ScrapeStatusFailed
		result.Error = fmt.Sprintf("Failed to create driver: %v", err)
		return result
	}
	defer d.Close() // Clean up driver after use

	// Create page
	page, err := d.NewPage(ctx)
	if err != nil {
		result.Status = models.ScrapeStatusFailed
		result.Error = fmt.Sprintf("Failed to create page: %v", err)
		return result
	}
	defer page.Close()

	// Navigate
	if err := page.Goto(req.URL); err != nil {
		result.Status = models.ScrapeStatusFailed
		result.Error = fmt.Sprintf("Failed to navigate: %v", err)
		return result
	}

	// Wait for selector if specified
	if req.WaitForSelector != "" {
		if err := page.WaitForSelector(req.WaitForSelector); err != nil {
			logger.Warn("Wait for selector failed",
				zap.String("selector", req.WaitForSelector),
				zap.Error(err),
			)
		}
	}

	// Get content based on output format
	switch req.OutputFormat {
	case models.OutputFormatHTML:
		content, err := page.Content()
		if err != nil {
			result.Status = models.ScrapeStatusFailed
			result.Error = fmt.Sprintf("Failed to get content: %v", err)
			return result
		}
		result.Content = content
		result.ContentType = "text/html"
		result.Status = models.ScrapeStatusCompleted

	case models.OutputFormatMarkdown:
		content, err := page.Content()
		if err != nil {
			result.Status = models.ScrapeStatusFailed
			result.Error = fmt.Sprintf("Failed to get content: %v", err)
			return result
		}
		// Convert HTML to Markdown (basic conversion)
		result.Content = htmlToMarkdown(content)
		result.ContentType = "text/markdown"
		result.Status = models.ScrapeStatusCompleted

	case models.OutputFormatScreenshot:
		screenshot, err := page.Screenshot()
		if err != nil {
			result.Status = models.ScrapeStatusFailed
			result.Error = fmt.Sprintf("Failed to capture screenshot: %v", err)
			return result
		}
		result.Screenshot = base64.StdEncoding.EncodeToString(screenshot)
		result.ContentType = "image/png"
		result.Status = models.ScrapeStatusCompleted

	default:
		result.Status = models.ScrapeStatusFailed
		result.Error = fmt.Sprintf("Unknown output format: %s", req.OutputFormat)
	}

	return result
}

// sendResult sends the scrape result to orchestrator
func (h *ScrapeHandler) sendResult(ctx context.Context, result *models.ScrapeResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/internal/scrape/result", h.orchestratorURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send result: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// htmlToMarkdown performs basic HTML to Markdown conversion
// For production, consider using a library like gomarkdown/markdown
func htmlToMarkdown(html string) string {
	// This is a simple placeholder - in production use a proper library
	// For now, just return the raw HTML with a note
	return "<!-- Converted from HTML -->\n\n" + html
}

// createDriverByType creates a driver instance based on driver type name
func (h *ScrapeHandler) createDriverByType(driverType string) (driver.Driver, error) {
	// Create a temporary profile with the driver type
	profile := &models.BrowserProfile{
		Name:       "scrape-temp", // Required by validation
		DriverType: driverType,
	}
	profile.SetDefaults() // Set screen size, browser type, etc.
	return h.driverFactory.CreateDriverFromProfile(profile)
}
