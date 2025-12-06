package reporter

// Note: Old synchronous StatsReporter removed - use BatchedStatsReporter instead
// See batched_stats.go for the high-throughput implementation

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
