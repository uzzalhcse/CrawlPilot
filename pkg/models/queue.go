package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// URLQueueItem represents an item in the URL queue
type URLQueueItem struct {
	ID               string          `json:"id" db:"id"`
	ExecutionID      string          `json:"execution_id" db:"execution_id"`
	URL              string          `json:"url" db:"url"`
	URLHash          string          `json:"url_hash" db:"url_hash"`
	Depth            int             `json:"depth" db:"depth"`
	Priority         int             `json:"priority" db:"priority"`
	Status           QueueItemStatus `json:"status" db:"status"`
	ParentURLID      *string         `json:"parent_url_id,omitempty" db:"parent_url_id"`
	DiscoveredByNode *string         `json:"discovered_by_node,omitempty" db:"discovered_by_node"`
	URLType          string          `json:"url_type" db:"url_type"` // Deprecated, use Marker
	Marker           string          `json:"marker" db:"marker"`     // NEW: Phase marker for URL filtering
	PhaseID          string          `json:"phase_id" db:"phase_id"` // NEW: Which phase should process this URL
	RetryCount       int             `json:"retry_count" db:"retry_count"`
	Error            string          `json:"error,omitempty" db:"error"`
	Metadata         string          `json:"metadata,omitempty" db:"metadata"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	ProcessedAt      *time.Time      `json:"processed_at,omitempty" db:"processed_at"`
	LockedAt         *time.Time      `json:"locked_at,omitempty" db:"locked_at"`
	LockedBy         string          `json:"locked_by,omitempty" db:"locked_by"`
}

// QueueItemStatus represents the status of a queue item
type QueueItemStatus string

const (
	QueueItemStatusPending    QueueItemStatus = "pending"
	QueueItemStatusProcessing QueueItemStatus = "processing"
	QueueItemStatusCompleted  QueueItemStatus = "completed"
	QueueItemStatusFailed     QueueItemStatus = "failed"
	QueueItemStatusSkipped    QueueItemStatus = "skipped"
)

// JSONMap is a custom type for JSON data stored in database
type JSONMap map[string]interface{}

// Scan implements sql.Scanner for JSONMap
func (jm *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*jm = make(map[string]interface{})
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, jm)
}

// Value implements driver.Valuer for JSONMap
func (jm JSONMap) Value() (driver.Value, error) {
	if jm == nil {
		return json.Marshal(map[string]interface{}{})
	}
	return json.Marshal(jm)
}

// ExtractedData represents data extracted from a page (DEPRECATED - use ExtractedItem)
type ExtractedData struct {
	ID          string    `json:"id" db:"id"`
	ExecutionID string    `json:"execution_id" db:"execution_id"`
	URL         string    `json:"url" db:"url"`
	Data        JSONMap   `json:"data" db:"data"`
	Schema      string    `json:"schema,omitempty" db:"schema"`
	ExtractedAt time.Time `json:"extracted_at" db:"extracted_at"`
}

// ExtractedItem represents structured extracted data (replaces ExtractedData)
type ExtractedItem struct {
	ID              string    `json:"id" db:"id"`
	ExecutionID     string    `json:"execution_id" db:"execution_id"`
	URLID           string    `json:"url_id" db:"url_id"`
	NodeExecutionID *string   `json:"node_execution_id,omitempty" db:"node_execution_id"`
	ItemType        string    `json:"item_type" db:"item_type"`
	SchemaName      *string   `json:"schema_name,omitempty" db:"schema_name"`
	Title           *string   `json:"title,omitempty" db:"title"`
	Price           *float64  `json:"price,omitempty" db:"price"`
	Currency        *string   `json:"currency,omitempty" db:"currency"`
	Availability    *string   `json:"availability,omitempty" db:"availability"`
	Rating          *float64  `json:"rating,omitempty" db:"rating"`
	ReviewCount     *int      `json:"review_count,omitempty" db:"review_count"`
	Attributes      JSONMap   `json:"attributes,omitempty" db:"attributes"`
	ExtractedAt     time.Time `json:"extracted_at" db:"extracted_at"`
}
