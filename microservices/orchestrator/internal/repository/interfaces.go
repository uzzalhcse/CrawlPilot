package repository

import (
	"context"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/models"
)

// WorkflowRepository defines the interface for workflow data access
type WorkflowRepository interface {
	// Create creates a new workflow
	Create(ctx context.Context, workflow *models.Workflow) error

	// Get retrieves a workflow by ID
	Get(ctx context.Context, id string) (*models.Workflow, error)

	// List retrieves all workflows with optional filters
	List(ctx context.Context, filters ListFilters) ([]*models.Workflow, error)

	// Update updates an existing workflow
	Update(ctx context.Context, workflow *models.Workflow) error

	// Delete soft-deletes a workflow
	Delete(ctx context.Context, id string) error

	// UpdateStatus updates workflow status
	UpdateStatus(ctx context.Context, id string, status string) error
}

// ExecutionRepository defines the interface for execution data access
type ExecutionRepository interface {
	// Create creates a new execution
	Create(ctx context.Context, execution *models.Execution) error

	// Get retrieves an execution by ID
	Get(ctx context.Context, id string) (*models.Execution, error)

	// List retrieves executions for a workflow
	List(ctx context.Context, workflowID string, filters ListFilters) ([]*models.Execution, error)

	// UpdateStatus updates execution status
	UpdateStatus(ctx context.Context, id string, status string) error

	// UpdateStats updates execution statistics (single execution)
	UpdateStats(ctx context.Context, id string, stats ExecutionStats) error

	// BatchUpdateStats updates multiple execution statistics in a single operation
	// Critical for high-throughput scenarios (10k+ URLs/sec)
	BatchUpdateStats(ctx context.Context, updates []BatchExecutionStats) error

	// Complete marks an execution as completed
	Complete(ctx context.Context, id string, status string) error

	// GetErrors retrieves error logs for an execution
	GetErrors(ctx context.Context, executionID string, limit int, offset int) ([]*models.ExecutionError, error)

	// BatchInsertErrors inserts multiple errors in a single operation
	BatchInsertErrors(ctx context.Context, errors []models.ExecutionError) error

	// UpdatePhaseStats updates phase-level statistics
	UpdatePhaseStats(ctx context.Context, id string, phaseStats map[string]models.PhaseStatEntry) error
}

// ListFilters defines common list query filters
type ListFilters struct {
	Limit  int
	Offset int
	Status string
}

// ExecutionStats holds execution statistics
type ExecutionStats struct {
	URLsProcessed  int
	URLsDiscovered int
	ItemsExtracted int
	Errors         int
	UpdatedAt      time.Time
}

// BatchExecutionStats holds stats for a single execution in a batch update
type BatchExecutionStats struct {
	ExecutionID    string
	URLsProcessed  int
	URLsDiscovered int
	ItemsExtracted int
	Errors         int
}

// BrowserProfileRepository defines the interface for browser profile data access
type BrowserProfileRepository interface {
	// Create creates a new browser profile
	Create(ctx context.Context, profile *models.BrowserProfile) error

	// Get retrieves a browser profile by ID
	Get(ctx context.Context, id string) (*models.BrowserProfile, error)

	// List retrieves browser profiles with optional filters
	List(ctx context.Context, filters BrowserProfileFilters) ([]*models.BrowserProfile, error)

	// Update updates an existing browser profile
	Update(ctx context.Context, profile *models.BrowserProfile) error

	// Delete soft-deletes a browser profile
	Delete(ctx context.Context, id string) error

	// Duplicate creates a copy of a browser profile with a new ID
	Duplicate(ctx context.Context, id string) (*models.BrowserProfile, error)

	// UpdateUsage increments usage count and updates last_used_at
	UpdateUsage(ctx context.Context, id string) error

	// GetFolders returns all unique folder names
	GetFolders(ctx context.Context) ([]string, error)
}

// BrowserProfileFilters defines browser profile query filters
type BrowserProfileFilters struct {
	Limit      int
	Offset     int
	Status     string
	Folder     string
	DriverType string
}
