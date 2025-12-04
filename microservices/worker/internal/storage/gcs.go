package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/storage"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

// GCSClient wraps Google Cloud Storage operations
type GCSClient struct {
	client *storage.Client
	bucket string
}

// NewGCSClient creates a new Cloud Storage client
func NewGCSClient(ctx context.Context, cfg *config.GCPConfig, opts ...option.ClientOption) (*GCSClient, error) {
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	logger.Info("Cloud Storage client initialized",
		zap.String("bucket", cfg.StorageBucket),
	)

	return &GCSClient{
		client: client,
		bucket: cfg.StorageBucket,
	}, nil
}

// Close closes the storage client
func (c *GCSClient) Close() error {
	return c.client.Close()
}

// UploadExtractedItems uploads extracted items as JSONL
func (c *GCSClient) UploadExtractedItems(ctx context.Context, executionID string, items []map[string]interface{}) (string, error) {
	if len(items) == 0 {
		return "", nil
	}

	// Generate object path
	timestamp := time.Now().Format("20060102-150405")
	objectPath := fmt.Sprintf("extractions/%s/%s.jsonl", executionID, timestamp)

	// Create JSONL content
	var jsonlContent []byte
	for _, item := range items {
		data, err := json.Marshal(item)
		if err != nil {
			logger.Warn("Failed to marshal item", zap.Error(err))
			continue
		}
		jsonlContent = append(jsonlContent, data...)
		jsonlContent = append(jsonlContent, '\n')
	}

	// Upload to GCS
	obj := c.client.Bucket(c.bucket).Object(objectPath)
	w := obj.NewWriter(ctx)
	w.ContentType = "application/x-ndjson"
	w.Metadata = map[string]string{
		"execution_id": executionID,
		"item_count":   fmt.Sprintf("%d", len(items)),
		"uploaded_at":  time.Now().UTC().Format(time.RFC3339),
	}

	if _, err := w.Write(jsonlContent); err != nil {
		w.Close()
		return "", fmt.Errorf("failed to write to GCS: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("failed to close GCS writer: %w", err)
	}

	gcsPath := fmt.Sprintf("gs://%s/%s", c.bucket, objectPath)

	logger.Info("Extracted items uploaded to GCS",
		zap.String("path", gcsPath),
		zap.Int("item_count", len(items)),
	)

	return gcsPath, nil
}

// DownloadExtractedItems downloads and parses JSONL file
func (c *GCSClient) DownloadExtractedItems(ctx context.Context, gcsPath string) ([]map[string]interface{}, error) {
	// Parse GCS path
	// Format: gs://bucket-name/path/to/file.jsonl
	if len(gcsPath) < 6 || gcsPath[:5] != "gs://" {
		return nil, fmt.Errorf("invalid GCS path: %s", gcsPath)
	}

	// Extract bucket and object path
	pathParts := gcsPath[5:] // Remove "gs://"
	bucket := c.bucket
	objectPath := pathParts[len(bucket)+1:] // Remove bucket name and "/"

	// Download from GCS
	obj := c.client.Bucket(bucket).Object(objectPath)
	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read from GCS: %w", err)
	}
	defer r.Close()

	// Parse JSONL
	items := make([]map[string]interface{}, 0)
	decoder := json.NewDecoder(r)

	for decoder.More() {
		var item map[string]interface{}
		if err := decoder.Decode(&item); err != nil {
			logger.Warn("Failed to decode item", zap.Error(err))
			continue
		}
		items = append(items, item)
	}

	return items, nil
}
