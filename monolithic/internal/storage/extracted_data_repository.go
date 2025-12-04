package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

type ExtractedDataRepository struct {
	db *PostgresDB
}

func NewExtractedDataRepository(db *PostgresDB) *ExtractedDataRepository {
	return &ExtractedDataRepository{db: db}
}

func (r *ExtractedDataRepository) Create(ctx context.Context, data *models.ExtractedData) error {
	if data.ID == "" {
		data.ID = uuid.New().String()
	}

	if data.ExtractedAt.IsZero() {
		data.ExtractedAt = time.Now()
	}

	query := `
		INSERT INTO extracted_data (id, execution_id, url, data, schema, extracted_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING extracted_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		data.ID,
		data.ExecutionID,
		data.URL,
		data.Data,
		data.Schema,
		data.ExtractedAt,
	).Scan(&data.ExtractedAt)

	if err != nil {
		return fmt.Errorf("failed to save extracted data: %w", err)
	}

	return nil
}

func (r *ExtractedDataRepository) GetByExecutionID(ctx context.Context, executionID string, limit, offset int) ([]*models.ExtractedData, error) {
	query := `
		SELECT id, execution_id, url, data, schema, extracted_at
		FROM extracted_data
		WHERE execution_id = $1
		ORDER BY extracted_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Pool.Query(ctx, query, executionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get extracted data: %w", err)
	}
	defer rows.Close()

	var results []*models.ExtractedData
	for rows.Next() {
		var data models.ExtractedData
		err := rows.Scan(
			&data.ID,
			&data.ExecutionID,
			&data.URL,
			&data.Data,
			&data.Schema,
			&data.ExtractedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan extracted data: %w", err)
		}
		results = append(results, &data)
	}

	return results, nil
}

func (r *ExtractedDataRepository) Count(ctx context.Context, executionID string) (int, error) {
	query := `SELECT COUNT(*) FROM extracted_data WHERE execution_id = $1`

	var count int
	err := r.db.Pool.QueryRow(ctx, query, executionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count extracted data: %w", err)
	}

	return count, nil
}
