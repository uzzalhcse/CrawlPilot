package models

import (
	"encoding/json"
	"time"
)

// WorkflowVersion represents a version of a workflow configuration
type WorkflowVersion struct {
	ID           string         `json:"id" db:"id"`
	WorkflowID   string         `json:"workflow_id" db:"workflow_id"`
	Version      int            `json:"version" db:"version"`
	Config       WorkflowConfig `json:"config" db:"config"`
	ChangeReason string         `json:"change_reason" db:"change_reason"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
}

// Scan implements sql.Scanner for WorkflowVersion
// Note: WorkflowConfig already implements Scan/Value in workflow.go
func (wv *WorkflowVersion) Scan(value interface{}) error {
	// This might not be needed if we scan fields individually, but good to have if we scan the whole struct
	return nil
}

// Value implements driver.Valuer for WorkflowVersion
func (wv WorkflowVersion) Value() (interface{}, error) {
	return json.Marshal(wv)
}
