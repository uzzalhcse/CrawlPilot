package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
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
	var dataJSON string
	if item.Data != "" {
		dataJSON = item.Data
	} else {
		dataJSON = "{}"
	}

	const query = `
		INSERT INTO extracted_items (
			id, execution_id, url_id, node_execution_id, schema_name, data, extracted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (execution_id, url_id, schema_name) 
		DO UPDATE SET data = EXCLUDED.data, extracted_at = EXCLUDED.extracted_at
		RETURNING id
	`

	err := r.db.Pool.QueryRow(ctx, query,
		item.ID, item.ExecutionID, item.URLID, item.NodeExecutionID,
		item.SchemaName, dataJSON, item.ExtractedAt,
	).Scan(&item.ID)

	if err != nil {
		return fmt.Errorf("failed to create extracted item: %w", err)
	}

	return nil
}

// CreateBatch inserts multiple extracted items in a single batch
func (r *ExtractedItemsRepository) CreateBatch(ctx context.Context, items []*models.ExtractedItem) error {
	if len(items) == 0 {
		return nil
	}

	batch := &pgx.Batch{}

	for _, item := range items {
		var dataJSON string
		if item.Data != "" {
			dataJSON = item.Data
		} else {
			dataJSON = "{}"
		}

		batch.Queue(`
			INSERT INTO extracted_items (
				id, execution_id, url_id, node_execution_id, schema_name, data, extracted_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (execution_id, url_id, schema_name) 
			DO UPDATE SET data = EXCLUDED.data, extracted_at = EXCLUDED.extracted_at
		`,
			item.ID, item.ExecutionID, item.URLID, item.NodeExecutionID,
			item.SchemaName, dataJSON, item.ExtractedAt,
		)
	}

	br := r.db.Pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(items); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to insert item at index %d: %w", i, err)
		}
	}

	return nil
}

// GetByExecutionID retrieves all extracted items for an execution
func (r *ExtractedItemsRepository) GetByExecutionID(ctx context.Context, executionID string) ([]*models.ExtractedItem, error) {
	query := `
		SELECT id, execution_id, url_id, node_execution_id, schema_name,
			   data, extracted_at
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
		item, err := r.scanRow(rows)
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
		SELECT id, execution_id, url_id, node_execution_id, schema_name,
			   data, extracted_at
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
		item, err := r.scanRow(rows)
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

// DeleteByExecutionID deletes all extracted items for an execution
func (r *ExtractedItemsRepository) DeleteByExecutionID(ctx context.Context, executionID string) error {
	query := `DELETE FROM extracted_items WHERE execution_id = $1`
	_, err := r.db.Pool.Exec(ctx, query, executionID)
	if err != nil {
		return fmt.Errorf("failed to delete extracted items: %w", err)
	}
	return nil
}

// scanRow scans a database row into an ExtractedItem model
func (r *ExtractedItemsRepository) scanRow(rows interface {
	Scan(dest ...interface{}) error
}) (*models.ExtractedItem, error) {
	var item models.ExtractedItem
	var dataJSON sql.NullString

	err := rows.Scan(
		&item.ID, &item.ExecutionID, &item.URLID, &item.NodeExecutionID, &item.SchemaName,
		&dataJSON, &item.ExtractedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan item: %w", err)
	}

	if dataJSON.Valid {
		item.Data = dataJSON.String
	} else {
		item.Data = "{}"
	}

	return &item, nil
}

// GetWithHierarchy retrieves items with their URL hierarchy
func (r *ExtractedItemsRepository) GetWithHierarchy(ctx context.Context, executionID string) ([]*ExtractedItemWithHierarchy, error) {
	query := `
		WITH RECURSIVE url_tree AS (
			SELECT id, url, parent_url_id, url_type, marker, 0 as level, 
				   ARRAY[id::text] as path
			FROM url_queue 
			WHERE execution_id = $1 AND parent_url_id IS NULL
			
			UNION ALL
			
			SELECT uq.id, uq.url, uq.parent_url_id, uq.url_type, uq.marker, ut.level + 1,
				   ut.path || uq.id::text
			FROM url_queue uq
			INNER JOIN url_tree ut ON uq.parent_url_id = ut.id::uuid
		)
		SELECT 
			ei.id, ei.execution_id, ei.url_id, ei.schema_name,
			ei.data, ei.extracted_at,
			ut.url, ut.url_type, ut.marker, ut.level,
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
		var dataJSON sql.NullString
		var parentURL, parentURLType sql.NullString

		err := rows.Scan(
			&item.ID, &item.ExecutionID, &item.URLID, &item.SchemaName,
			&dataJSON, &item.ExtractedAt,
			&item.URL, &item.URLType, &item.Marker, &item.URLLevel,
			&parentURL, &parentURLType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item with hierarchy: %w", err)
		}

		if dataJSON.Valid {
			item.Data = dataJSON.String
		} else {
			item.Data = "{}"
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
	Marker        string  `json:"marker"`
	URLLevel      int     `json:"url_level"`
	ParentURL     *string `json:"parent_url,omitempty"`
	ParentURLType *string `json:"parent_url_type,omitempty"`
}
