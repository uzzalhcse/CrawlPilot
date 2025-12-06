package reporter

import "time"

// Note: Old synchronous StatsReporter removed - use BatchedStatsReporter instead
// See batched_stats.go for the high-throughput implementation

// TaskStats holds statistics for a single task
type TaskStats struct {
	URLsProcessed  int
	URLsDiscovered int
	ItemsExtracted int
	Errors         int
	PhaseID        string        // Phase that processed this task
	Duration       time.Duration // How long the phase took
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

// SetPhase sets the phase context for this stats
func (s *TaskStats) SetPhase(phaseID string, duration time.Duration) {
	s.PhaseID = phaseID
	s.Duration = duration
}
