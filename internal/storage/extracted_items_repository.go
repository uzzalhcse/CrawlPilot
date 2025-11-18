package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/uzzalhcse/crawlify/pkg/models"
)

// ExtractedItemsRepository handles database operations for extracted items
type ExtractedItemsRepository struct {
	db *PostgresDB
}

// NewExtractedItemsRepository creates a new repository instance
func NewExtractedItemsRepository(db *PostgresDB) *ExtractedItemsRepository {
	return &ExtractedItemsRepository{db: db}
}

// Create inserts a new extracted item
func (r *ExtractedItemsRepository) Create(ctx context.Context, item *models.ExtractedItem) error {
	attributesJSON, err := json.Marshal(item.Attributes)
	if err != nil {
		return fmt.Errorf("failed to marshal attributes: %w", err)
	}

	query := `
		INSERT INTO extracted_items (
			execution_id, url_id, node_execution_id, item_type, schema_name,
			title, price, currency, availability, rating, review_count, attributes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, extracted_at
	`

	err = r.db.Pool.QueryRow(ctx, query,
		item.ExecutionID, item.URLID, item.NodeExecutionID, item.ItemType, item.SchemaName,
		item.Title, item.Price, item.Currency, item.Availability, item.Rating, item.ReviewCount, string(attributesJSON),
	).Scan(&item.ID, &item.ExtractedAt)

	if err != nil {
		return fmt.Errorf("failed to create extracted item: %w", err)
	}

	return nil
}

// CreateBatch inserts multiple extracted items in a single transaction
func (r *ExtractedItemsRepository) CreateBatch(ctx context.Context, items []*models.ExtractedItem) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO extracted_items (
			execution_id, url_id, node_execution_id, item_type, schema_name,
			title, price, currency, availability, rating, review_count, attributes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, extracted_at
	`

	for _, item := range items {
		attributesJSON, err := json.Marshal(item.Attributes)
		if err != nil {
			return fmt.Errorf("failed to marshal attributes: %w", err)
		}

		err = tx.QueryRow(ctx, query,
			item.ExecutionID, item.URLID, item.NodeExecutionID, item.ItemType, item.SchemaName,
			item.Title, item.Price, item.Currency, item.Availability, item.Rating, item.ReviewCount, string(attributesJSON),
		).Scan(&item.ID, &item.ExtractedAt)

		if err != nil {
			return fmt.Errorf("failed to insert item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetByExecutionID retrieves all extracted items for an execution
func (r *ExtractedItemsRepository) GetByExecutionID(ctx context.Context, executionID string) ([]*models.ExtractedItem, error) {
	query := `
		SELECT id, execution_id, url_id, node_execution_id, item_type, schema_name,
			   title, price, currency, availability, rating, review_count, attributes, extracted_at
		FROM extracted_items
		WHERE execution_id = $1
		ORDER BY extracted_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query extracted items: %w", err)
	}
	defer rows.Close()

	var items []*models.ExtractedItem
	for rows.Next() {
		item, err := r.scanPgxRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

// GetByURL retrieves all extracted items for a specific URL
func (r *ExtractedItemsRepository) GetByURL(ctx context.Context, executionID, urlID string) ([]*models.ExtractedItem, error) {
	query := `
		SELECT id, execution_id, url_id, node_execution_id, item_type, schema_name,
			   title, price, currency, availability, rating, review_count, attributes, extracted_at
		FROM extracted_items
		WHERE execution_id = $1 AND url_id = $2
		ORDER BY extracted_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, executionID, urlID)
	if err != nil {
		return nil, fmt.Errorf("failed to query extracted items: %w", err)
	}
	defer rows.Close()

	var items []*models.ExtractedItem
	for rows.Next() {
		item, err := r.scanPgxRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

// GetByItemType retrieves all extracted items of a specific type
func (r *ExtractedItemsRepository) GetByItemType(ctx context.Context, executionID, itemType string) ([]*models.ExtractedItem, error) {
	query := `
		SELECT id, execution_id, url_id, node_execution_id, item_type, schema_name,
			   title, price, currency, availability, rating, review_count, attributes, extracted_at
		FROM extracted_items
		WHERE execution_id = $1 AND item_type = $2
		ORDER BY extracted_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, executionID, itemType)
	if err != nil {
		return nil, fmt.Errorf("failed to query extracted items: %w", err)
	}
	defer rows.Close()

	var items []*models.ExtractedItem
	for rows.Next() {
		item, err := r.scanPgxRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

// GetWithPriceRange retrieves items within a price range
func (r *ExtractedItemsRepository) GetWithPriceRange(ctx context.Context, executionID string, minPrice, maxPrice float64) ([]*models.ExtractedItem, error) {
	query := `
		SELECT id, execution_id, url_id, node_execution_id, item_type, schema_name,
			   title, price, currency, availability, rating, review_count, attributes, extracted_at
		FROM extracted_items
		WHERE execution_id = $1 AND price >= $2 AND price <= $3
		ORDER BY price ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, executionID, minPrice, maxPrice)
	if err != nil {
		return nil, fmt.Errorf("failed to query extracted items: %w", err)
	}
	defer rows.Close()

	var items []*models.ExtractedItem
	for rows.Next() {
		item, err := r.scanPgxRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

// SearchByTitle performs full-text search on item titles
func (r *ExtractedItemsRepository) SearchByTitle(ctx context.Context, executionID, searchTerm string) ([]*models.ExtractedItem, error) {
	query := `
		SELECT id, execution_id, url_id, node_execution_id, item_type, schema_name,
			   title, price, currency, availability, rating, review_count, attributes, extracted_at
		FROM extracted_items
		WHERE execution_id = $1 AND to_tsvector('english', title) @@ plainto_tsquery('english', $2)
		ORDER BY extracted_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, executionID, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("failed to search extracted items: %w", err)
	}
	defer rows.Close()

	var items []*models.ExtractedItem
	for rows.Next() {
		item, err := r.scanPgxRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

// GetCount returns the total count of extracted items for an execution
func (r *ExtractedItemsRepository) GetCount(ctx context.Context, executionID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM extracted_items WHERE execution_id = $1`
	err := r.db.Pool.QueryRow(ctx, query, executionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count extracted items: %w", err)
	}
	return count, nil
}

// GetCountByType returns counts grouped by item type
func (r *ExtractedItemsRepository) GetCountByType(ctx context.Context, executionID string) (map[string]int, error) {
	query := `
		SELECT item_type, COUNT(*) as count
		FROM extracted_items
		WHERE execution_id = $1
		GROUP BY item_type
	`

	rows, err := r.db.Pool.Query(ctx, query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query item counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var itemType string
		var count int
		if err := rows.Scan(&itemType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[itemType] = count
	}

	return counts, nil
}

// DeleteByExecutionID deletes all extracted items for an execution
func (r *ExtractedItemsRepository) DeleteByExecutionID(ctx context.Context, executionID string) error {
	query := `DELETE FROM extracted_items WHERE execution_id = $1`
	_, err := r.db.Pool.Exec(ctx, query, executionID)
	if err != nil {
		return fmt.Errorf("failed to delete extracted items: %w", err)
	}
	return nil
}

// scanPgxRow scans a pgx row into an ExtractedItem model
func (r *ExtractedItemsRepository) scanPgxRow(rows interface {
	Scan(dest ...interface{}) error
}) (*models.ExtractedItem, error) {
	var item models.ExtractedItem
	var attributesJSON sql.NullString

	err := rows.Scan(
		&item.ID, &item.ExecutionID, &item.URLID, &item.NodeExecutionID, &item.ItemType, &item.SchemaName,
		&item.Title, &item.Price, &item.Currency, &item.Availability, &item.Rating, &item.ReviewCount,
		&attributesJSON, &item.ExtractedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan item: %w", err)
	}

	if attributesJSON.Valid {
		if err := json.Unmarshal([]byte(attributesJSON.String), &item.Attributes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
		}
	}

	return &item, nil
}

// GetWithHierarchy retrieves items with their URL hierarchy
func (r *ExtractedItemsRepository) GetWithHierarchy(ctx context.Context, executionID string) ([]*ExtractedItemWithHierarchy, error) {
	query := `
		WITH RECURSIVE url_tree AS (
			SELECT id, url, parent_url_id, url_type, 0 as level, 
				   ARRAY[id::text] as path
			FROM url_queue 
			WHERE execution_id = $1 AND parent_url_id IS NULL
			
			UNION ALL
			
			SELECT uq.id, uq.url, uq.parent_url_id, uq.url_type, ut.level + 1,
				   ut.path || uq.id::text
			FROM url_queue uq
			INNER JOIN url_tree ut ON uq.parent_url_id = ut.id::uuid
		)
		SELECT 
			ei.id, ei.execution_id, ei.url_id, ei.item_type, ei.schema_name,
			ei.title, ei.price, ei.currency, ei.availability, ei.rating, ei.review_count,
			ei.attributes, ei.extracted_at,
			ut.url, ut.url_type, ut.level,
			parent_uq.url as parent_url, parent_uq.url_type as parent_url_type
		FROM extracted_items ei
		JOIN url_tree ut ON ei.url_id = ut.id::uuid
		LEFT JOIN url_queue parent_uq ON ut.parent_url_id = parent_uq.id
		WHERE ei.execution_id = $1
		ORDER BY ut.path, ei.extracted_at
	`

	rows, err := r.db.Pool.Query(ctx, query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query items with hierarchy: %w", err)
	}
	defer rows.Close()

	var items []*ExtractedItemWithHierarchy
	for rows.Next() {
		var item ExtractedItemWithHierarchy
		var attributesJSON sql.NullString
		var parentURL, parentURLType sql.NullString

		err := rows.Scan(
			&item.ID, &item.ExecutionID, &item.URLID, &item.ItemType, &item.SchemaName,
			&item.Title, &item.Price, &item.Currency, &item.Availability, &item.Rating, &item.ReviewCount,
			&attributesJSON, &item.ExtractedAt,
			&item.URL, &item.URLType, &item.URLLevel,
			&parentURL, &parentURLType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item with hierarchy: %w", err)
		}

		if attributesJSON.Valid {
			if err := json.Unmarshal([]byte(attributesJSON.String), &item.Attributes); err != nil {
				return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
			}
		}

		if parentURL.Valid {
			item.ParentURL = &parentURL.String
		}
		if parentURLType.Valid {
			item.ParentURLType = &parentURLType.String
		}

		items = append(items, &item)
	}

	return items, nil
}

// ExtractedItemWithHierarchy includes URL hierarchy information
type ExtractedItemWithHierarchy struct {
	models.ExtractedItem
	URL           string  `json:"url"`
	URLType       string  `json:"url_type"`
	URLLevel      int     `json:"url_level"`
	ParentURL     *string `json:"parent_url,omitempty"`
	ParentURLType *string `json:"parent_url_type,omitempty"`
}
