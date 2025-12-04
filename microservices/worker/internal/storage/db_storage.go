package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/uzzalhcse/crawlify/microservices/shared/database"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// DBStorage handles storing extracted items in PostgreSQL
type DBStorage struct {
	db *database.DB
}

// NewDBStorage creates a new database storage client
func NewDBStorage(db *database.DB) *DBStorage {
	logger.Info("Database storage initialized for extracted items")
	return &DBStorage{db: db}
}

// SaveExtractedItems saves extracted items directly to database
func (s *DBStorage) SaveExtractedItems(
	ctx context.Context,
	executionID string,
	workflowID string,
	taskID string,
	url string,
	items []map[string]interface{},
) error {
	if len(items) == 0 {
		return nil
	}

	// Insert each item into database
	query := `
		INSERT INTO extracted_items (execution_id, workflow_id, task_id, url, data)
		VALUES ($1, $2, $3, $4, $5)
	`

	for _, item := range items {
		dataBytes, err := json.Marshal(item)
		if err != nil {
			logger.Warn("Failed to marshal item", zap.Error(err))
			continue
		}
		// Convert to string for PgBouncer simple protocol compatibility
		dataJSON := string(dataBytes)

		_, err = s.db.Pool.Exec(ctx, query, executionID, workflowID, taskID, url, dataJSON)
		if err != nil {
			return fmt.Errorf("failed to save extracted item: %w", err)
		}
	}

	logger.Info("Extracted items saved to database",
		zap.String("execution_id", executionID),
		zap.String("task_id", taskID),
		zap.Int("item_count", len(items)),
	)

	return nil
}

// GetExtractedItems retrieves extracted items from database
func (s *DBStorage) GetExtractedItems(
	ctx context.Context,
	executionID string,
) ([]map[string]interface{}, error) {
	query := `
		SELECT data 
		FROM extracted_items 
		WHERE execution_id = $1 
		ORDER BY extracted_at ASC
	`

	rows, err := s.db.Pool.Query(ctx, query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query extracted items: %w", err)
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var dataJSON []byte
		if err := rows.Scan(&dataJSON); err != nil {
			logger.Warn("Failed to scan row", zap.Error(err))
			continue
		}

		var item map[string]interface{}
		if err := json.Unmarshal(dataJSON, &item); err != nil {
			logger.Warn("Failed to unmarshal item", zap.Error(err))
			continue
		}

		items = append(items, item)
	}

	return items, nil
}
