package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// StatsReporter reports execution statistics to the orchestrator
type StatsReporter struct {
	orchestratorURL string
	httpClient      *http.Client
}

// NewStatsReporter creates a new stats reporter
func NewStatsReporter(orchestratorURL string) *StatsReporter {
	return &StatsReporter{
		orchestratorURL: orchestratorURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ExecutionStats holds execution statistics
type ExecutionStats struct {
	ExecutionID    string    `json:"execution_id"`
	URLsProcessed  int       `json:"urls_processed"`
	URLsDiscovered int       `json:"urls_discovered"`
	ItemsExtracted int       `json:"items_extracted"`
	Errors         int       `json:"errors"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ReportStats sends statistics to the orchestrator
func (r *StatsReporter) ReportStats(ctx context.Context, stats *ExecutionStats) error {
	if r.orchestratorURL == "" {
		logger.Debug("Orchestrator URL not configured, skipping stats report")
		return nil
	}

	stats.UpdatedAt = time.Now()

	// Marshal stats
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	// Send to orchestrator
	url := fmt.Sprintf("%s/api/v1/internal/executions/%s/stats", r.orchestratorURL, stats.ExecutionID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		logger.Warn("Failed to report stats to orchestrator",
			zap.String("execution_id", stats.ExecutionID),
			zap.Error(err),
		)
		// Don't fail the task if stats reporting fails
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		logger.Warn("Unexpected status code from orchestrator",
			zap.String("execution_id", stats.ExecutionID),
			zap.Int("status_code", resp.StatusCode),
		)
	}

	logger.Debug("Stats reported to orchestrator",
		zap.String("execution_id", stats.ExecutionID),
		zap.Int("urls_processed", stats.URLsProcessed),
	)

	return nil
}

// TaskStats holds statistics for a single task
type TaskStats struct {
	URLsProcessed  int
	URLsDiscovered int
	ItemsExtracted int
	Errors         int
}

// NewTaskStats creates a new task stats tracker
func NewTaskStats() *TaskStats {
	return &TaskStats{}
}

// Record records stats from a task result
func (s *TaskStats) Record(itemsExtracted int, urlsDiscovered int, errorCount int) {
	s.URLsProcessed++
	s.ItemsExtracted += itemsExtracted
	s.URLsDiscovered += urlsDiscovered
	s.Errors += errorCount
}

// ToExecutionStats converts task stats to execution stats
func (s *TaskStats) ToExecutionStats(executionID string) *ExecutionStats {
	return &ExecutionStats{
		ExecutionID:    executionID,
		URLsProcessed:  s.URLsProcessed,
		URLsDiscovered: s.URLsDiscovered,
		ItemsExtracted: s.ItemsExtracted,
		Errors:         s.Errors,
	}
}
