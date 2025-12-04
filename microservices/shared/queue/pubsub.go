package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// PubSubClient wraps Google Cloud Pub/Sub
type PubSubClient struct {
	client *pubsub.Client
	topic  *pubsub.Topic
	cfg    *config.GCPConfig
}

// NewPubSubClient creates a new Pub/Sub client
func NewPubSubClient(ctx context.Context, cfg *config.GCPConfig) (*PubSubClient, error) {
	if !cfg.PubSubEnabled {
		return nil, fmt.Errorf("pub/sub is not enabled")
	}

	client, err := pubsub.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pub/sub client: %w", err)
	}

	topic := client.Topic(cfg.PubSubTopic)
	exists, err := topic.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check topic existence: %w", err)
	}

	if !exists {
		return nil, fmt.Errorf("topic %s does not exist", cfg.PubSubTopic)
	}

	logger.Info("Pub/Sub client initialized",
		zap.String("project", cfg.ProjectID),
		zap.String("topic", cfg.PubSubTopic),
	)

	return &PubSubClient{
		client: client,
		topic:  topic,
		cfg:    cfg,
	}, nil
}

// Close closes the Pub/Sub client
func (c *PubSubClient) Close() error {
	c.topic.Stop()
	return c.client.Close()
}

// PublishTask publishes a single task to the queue
func (c *PubSubClient) PublishTask(ctx context.Context, task *models.Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	result := c.topic.Publish(ctx, &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			"execution_id": task.ExecutionID,
			"workflow_id":  task.WorkflowID,
		},
	})

	// Wait for publish to complete
	_, err = result.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish task: %w", err)
	}

	return nil
}

// PublishBatch publishes multiple tasks in batch
func (c *PubSubClient) PublishBatch(ctx context.Context, tasks []*models.Task) error {
	results := make([]*pubsub.PublishResult, 0, len(tasks))

	// Publish all tasks asynchronously
	for _, task := range tasks {
		data, err := json.Marshal(task)
		if err != nil {
			return fmt.Errorf("failed to marshal task: %w", err)
		}

		result := c.topic.Publish(ctx, &pubsub.Message{
			Data: data,
			Attributes: map[string]string{
				"execution_id": task.ExecutionID,
				"workflow_id":  task.WorkflowID,
			},
		})

		results = append(results, result)
	}

	// Wait for all publishes to complete
	for i, result := range results {
		if _, err := result.Get(ctx); err != nil {
			return fmt.Errorf("failed to publish task %d: %w", i, err)
		}
	}

	logger.Info("Published batch of tasks",
		zap.Int("count", len(tasks)),
	)

	return nil
}

// Subscribe creates a pull subscription and processes messages
func (c *PubSubClient) Subscribe(ctx context.Context, handler func(context.Context, *models.Task) error) error {
	sub := c.client.Subscription(c.cfg.PubSubSubscription)

	exists, err := sub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check subscription: %w", err)
	}

	if !exists {
		return fmt.Errorf("subscription %s does not exist", c.cfg.PubSubSubscription)
	}

	// Configure subscription settings
	sub.ReceiveSettings.MaxOutstandingMessages = 100
	sub.ReceiveSettings.NumGoroutines = 10

	logger.Info("Starting to receive messages from subscription",
		zap.String("subscription", c.cfg.PubSubSubscription),
	)

	logger.Debug("Subscription configuration",
		zap.Int("max_outstanding", sub.ReceiveSettings.MaxOutstandingMessages),
		zap.Int("num_goroutines", sub.ReceiveSettings.NumGoroutines),
	)

	return sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		logger.Debug("📨 Message received from Pub/Sub",
			zap.String("message_id", msg.ID),
			zap.Int("data_size", len(msg.Data)),
			zap.Any("attributes", msg.Attributes),
		)

		var task models.Task
		if err := json.Unmarshal(msg.Data, &task); err != nil {
			logger.Error("Failed to unmarshal task", zap.Error(err))
			msg.Nack()
			return
		}

		logger.Info("Processing task",
			zap.String("task_id", task.TaskID),
			zap.String("execution_id", task.ExecutionID),
			zap.String("url", task.URL),
		)

		// Process task
		if err := handler(ctx, &task); err != nil {
			logger.Error("Failed to process task",
				zap.String("task_id", task.TaskID),
				zap.Error(err),
			)
			msg.Nack() // Requeue
		} else {
			logger.Info("Task completed successfully",
				zap.String("task_id", task.TaskID),
			)
			msg.Ack() // Success
		}
	})
}
