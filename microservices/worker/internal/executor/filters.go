package executor

import (
	"context"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/dedup"
	"go.uber.org/zap"
)

// URLWithMarker holds a URL and its associated marker
type URLWithMarker struct {
	URL    string
	Marker string
}

// Default batch size for pipeline dedup operations
const defaultDedupBatchSize = 1000

// passesURLFilter checks if task passes the phase's URL filter
func (e *TaskExecutor) passesURLFilter(task *models.Task) bool {
	filter := task.PhaseConfig.URLFilter
	if filter == nil {
		return true // No filter, allow all
	}

	// Check depth filter (only if explicitly set to positive value or 0 in JSON)
	// Note: If depth is not specified in filter, it will be 0 (Go default)
	// We only want to filter if depth was explicitly set in the workflow
	// A depth of 0 matches depth 0, but we need to distinguish between "not set" and "set to 0"
	// For now, treat filter.Depth >= 0 with checks only when Depth field exists in filter
	// Better approach: Use pointer *int for Depth in URLFilter struct

	// TEMPORARY: Skip depth filtering if no depth specified in JSON
	// This allows all depths through when depth filter is not explicitly set
	if filter.Depth > 0 && task.Depth != filter.Depth {
		logger.Debug("Task filtered by depth",
			zap.Int("task_depth", task.Depth),
			zap.Int("required_depth", filter.Depth),
		)
		return false
	}

	// Check marker filter
	if len(filter.Markers) > 0 {
		if task.Marker == "" {
			logger.Debug("Task filtered: no marker set")
			return false
		}

		hasMarker := false
		for _, marker := range filter.Markers {
			if task.Marker == marker {
				hasMarker = true
				break
			}
		}

		if !hasMarker {
			logger.Debug("Task filtered by marker",
				zap.String("task_marker", task.Marker),
				zap.Strings("required_markers", filter.Markers),
			)
			return false
		}
	}

	return true
}

// processDiscoveredURLs processes discovered URLs with marker propagation and max depth
// OPTIMIZED: Uses batch dedup when available to reduce N Redis round-trips to 1
func (e *TaskExecutor) processDiscoveredURLs(ctx context.Context, task *models.Task, discoveredURLs interface{}) []URLWithMarker {
	var results []URLWithMarker

	// Check if deduplicator supports batch operations
	batchDedup, hasBatch := e.deduplicator.(dedup.BatchDeduplicator)

	// discoveredURLs can be []string or []map[string]interface{}
	switch urls := discoveredURLs.(type) {
	case []string:
		// OPTIMIZED PATH: Use batch dedup for string arrays
		if hasBatch && len(urls) > 0 {
			uniqueURLs, err := batchDedup.FilterDuplicatesBatch(ctx, task.ExecutionID, task.PhaseID, urls, defaultDedupBatchSize)
			if err != nil {
				logger.Warn("Batch dedup failed, falling back to sequential", zap.Error(err))
				// Fall through to sequential processing
			} else {
				// Batch succeeded - convert to results
				for _, url := range uniqueURLs {
					results = append(results, URLWithMarker{URL: url, Marker: ""})
				}
				return results
			}
		}

		// Fallback: Sequential processing
		for _, url := range urls {
			isDup, err := e.deduplicator.IsDuplicate(ctx, task.ExecutionID, task.PhaseID, url)
			if err != nil || isDup {
				continue
			}
			results = append(results, URLWithMarker{URL: url, Marker: ""})
		}

	case []map[string]interface{}:
		// Extract URLs and markers, then batch dedup
		urlsOnly := make([]string, 0, len(urls))
		urlMarkers := make(map[string]string) // URL -> marker

		for _, urlData := range urls {
			url, ok := urlData["url"].(string)
			if !ok || url == "" {
				continue
			}
			urlsOnly = append(urlsOnly, url)
			if m, ok := urlData["marker"].(string); ok {
				urlMarkers[url] = m
			}
		}

		if hasBatch && len(urlsOnly) > 0 {
			uniqueURLs, err := batchDedup.FilterDuplicatesBatch(ctx, task.ExecutionID, task.PhaseID, urlsOnly, defaultDedupBatchSize)
			if err != nil {
				logger.Warn("Batch dedup failed, falling back to sequential", zap.Error(err))
			} else {
				for _, url := range uniqueURLs {
					results = append(results, URLWithMarker{URL: url, Marker: urlMarkers[url]})
				}
				return results
			}
		}

		// Fallback: Sequential processing
		for _, urlData := range urls {
			url, ok := urlData["url"].(string)
			if !ok || url == "" {
				continue
			}

			isDup, err := e.deduplicator.IsDuplicate(ctx, task.ExecutionID, task.PhaseID, url)
			if err != nil || isDup {
				continue
			}

			marker := ""
			if m, ok := urlData["marker"].(string); ok {
				marker = m
			}

			results = append(results, URLWithMarker{URL: url, Marker: marker})
		}

	case []interface{}:
		// Extract URLs and markers from generic interface array
		urlsOnly := make([]string, 0, len(urls))
		urlMarkers := make(map[string]string) // URL -> marker

		for _, item := range urls {
			switch v := item.(type) {
			case string:
				urlsOnly = append(urlsOnly, v)
			case map[string]interface{}:
				url, ok := v["url"].(string)
				if !ok || url == "" {
					continue
				}
				urlsOnly = append(urlsOnly, url)
				if m, ok := v["marker"].(string); ok {
					urlMarkers[url] = m
				}
			}
		}

		if hasBatch && len(urlsOnly) > 0 {
			uniqueURLs, err := batchDedup.FilterDuplicatesBatch(ctx, task.ExecutionID, task.PhaseID, urlsOnly, defaultDedupBatchSize)
			if err != nil {
				logger.Warn("Batch dedup failed, falling back to sequential", zap.Error(err))
			} else {
				for _, url := range uniqueURLs {
					results = append(results, URLWithMarker{URL: url, Marker: urlMarkers[url]})
				}
				return results
			}
		}

		// Fallback: Sequential processing
		for _, item := range urls {
			switch v := item.(type) {
			case string:
				isDup, err := e.deduplicator.IsDuplicate(ctx, task.ExecutionID, task.PhaseID, v)
				if err != nil || isDup {
					continue
				}
				results = append(results, URLWithMarker{URL: v, Marker: ""})

			case map[string]interface{}:
				url, ok := v["url"].(string)
				if !ok || url == "" {
					continue
				}

				isDup, err := e.deduplicator.IsDuplicate(ctx, task.ExecutionID, task.PhaseID, url)
				if err != nil || isDup {
					continue
				}

				marker := ""
				if m, ok := v["marker"].(string); ok {
					marker = m
				}

				results = append(results, URLWithMarker{URL: url, Marker: marker})
			}
		}
	}

	return results
}
