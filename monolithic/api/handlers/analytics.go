package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

type AnalyticsHandler struct {
	nodeExecRepo      *storage.NodeExecutionRepository
	extractedItemRepo *storage.ExtractedItemsRepository
	queueRepo         interface{} // URL queue repository interface
}

func NewAnalyticsHandler(
	nodeExecRepo *storage.NodeExecutionRepository,
	extractedItemRepo *storage.ExtractedItemsRepository,
	queueRepo interface{},
) *AnalyticsHandler {
	return &AnalyticsHandler{
		nodeExecRepo:      nodeExecRepo,
		extractedItemRepo: extractedItemRepo,
		queueRepo:         queueRepo,
	}
}

// RegisterRoutes registers analytics routes
func (h *AnalyticsHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/executions/:executionId/timeline", h.GetExecutionTimeline)
	router.Get("/executions/:executionId/hierarchy", h.GetURLHierarchy)
	router.Get("/executions/:executionId/node-tree", h.GetNodeTree)
	router.Get("/executions/:executionId/performance", h.GetPerformanceMetrics)
	router.Get("/executions/:executionId/items-with-hierarchy", h.GetItemsWithHierarchy)
	router.Get("/executions/:executionId/bottlenecks", h.GetBottlenecks)
}

// ExecutionTimelineResponse represents the timeline view
type ExecutionTimelineResponse struct {
	ExecutionID string           `json:"execution_id"`
	Timeline    []*TimelineEntry `json:"timeline"`
	Summary     *TimelineSummary `json:"summary"`
}

type TimelineEntry struct {
	Timestamp      time.Time `json:"timestamp"`
	NodeName       string    `json:"node_name"`
	NodeType       string    `json:"node_type"`
	Status         string    `json:"status"`
	DurationMs     *int      `json:"duration_ms,omitempty"`
	URL            string    `json:"url,omitempty"`
	URLType        string    `json:"url_type,omitempty"`
	URLsDiscovered int       `json:"urls_discovered"`
	ItemsExtracted int       `json:"items_extracted"`
	ErrorMessage   string    `json:"error_message,omitempty"`
}

type TimelineSummary struct {
	TotalNodes          int     `json:"total_nodes"`
	CompletedNodes      int     `json:"completed_nodes"`
	FailedNodes         int     `json:"failed_nodes"`
	TotalURLsDiscovered int     `json:"total_urls_discovered"`
	TotalItemsExtracted int     `json:"total_items_extracted"`
	AverageDurationMs   float64 `json:"average_duration_ms"`
}

// GetExecutionTimeline returns the complete execution timeline
func (h *AnalyticsHandler) GetExecutionTimeline(c *fiber.Ctx) error {
	executionID := c.Params("executionId")

	ctx := context.Background()

	// Get all node executions
	nodeExecs, err := h.nodeExecRepo.GetByExecutionID(ctx, executionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get node executions: " + err.Error(),
		})
	}

	// Build timeline entries
	timeline := make([]*TimelineEntry, len(nodeExecs))
	summary := &TimelineSummary{}

	var totalDuration int64
	var durationCount int

	for i, ne := range nodeExecs {
		entry := &TimelineEntry{
			Timestamp:      ne.StartedAt,
			NodeName:       ne.NodeID,
			Status:         string(ne.Status),
			DurationMs:     ne.DurationMs,
			URLsDiscovered: ne.URLsDiscovered,
			ItemsExtracted: ne.ItemsExtracted,
		}

		if ne.NodeType != nil {
			entry.NodeType = *ne.NodeType
		}

		if ne.ErrorMessage != nil {
			entry.ErrorMessage = *ne.ErrorMessage
		}

		// Accumulate summary stats
		summary.TotalNodes++
		summary.TotalURLsDiscovered += ne.URLsDiscovered
		summary.TotalItemsExtracted += ne.ItemsExtracted

		if ne.Status == models.ExecutionStatusCompleted {
			summary.CompletedNodes++
		} else if ne.Status == models.ExecutionStatusFailed {
			summary.FailedNodes++
		}

		if ne.DurationMs != nil {
			totalDuration += int64(*ne.DurationMs)
			durationCount++
		}

		timeline[i] = entry
	}

	if durationCount > 0 {
		summary.AverageDurationMs = float64(totalDuration) / float64(durationCount)
	}

	response := &ExecutionTimelineResponse{
		ExecutionID: executionID,
		Timeline:    timeline,
		Summary:     summary,
	}

	return c.JSON(response)
}

// URLHierarchyNode represents a node in the URL tree
type URLHierarchyNode struct {
	ID               string              `json:"id"`
	URL              string              `json:"url"`
	URLType          string              `json:"url_type"`
	Depth            int                 `json:"depth"`
	Status           string              `json:"status"`
	DiscoveredByNode *string             `json:"discovered_by_node,omitempty"`
	Children         []*URLHierarchyNode `json:"children,omitempty"`
	ItemsExtracted   int                 `json:"items_extracted"`
}

type URLHierarchyResponse struct {
	ExecutionID string              `json:"execution_id"`
	Tree        []*URLHierarchyNode `json:"tree"`
	Stats       *HierarchyStats     `json:"stats"`
}

type HierarchyStats struct {
	TotalURLs    int            `json:"total_urls"`
	MaxDepth     int            `json:"max_depth"`
	URLsByType   map[string]int `json:"urls_by_type"`
	URLsByStatus map[string]int `json:"urls_by_status"`
}

// GetURLHierarchy returns the URL hierarchy tree
func (h *AnalyticsHandler) GetURLHierarchy(c *fiber.Ctx) error {
	executionID := c.Params("executionId")

	// Query to build URL hierarchy
	query := `
		WITH RECURSIVE url_tree AS (
			SELECT id, url, parent_url_id, url_type, depth, status, discovered_by_node,
				   0 as level, ARRAY[id::text] as path
			FROM url_queue 
			WHERE execution_id = $1 AND parent_url_id IS NULL
			
			UNION ALL
			
			SELECT uq.id, uq.url, uq.parent_url_id, uq.url_type, uq.depth, uq.status, 
				   uq.discovered_by_node, ut.level + 1, ut.path || uq.id::text
			FROM url_queue uq
			INNER JOIN url_tree ut ON uq.parent_url_id = ut.id::uuid
		)
		SELECT id, url, parent_url_id, url_type, depth, status, discovered_by_node, level
		FROM url_tree
		ORDER BY path
	`

	// This is a simplified version - actual implementation would need the DB connection
	// For now, return a placeholder response
	response := &URLHierarchyResponse{
		ExecutionID: executionID,
		Tree:        []*URLHierarchyNode{},
		Stats: &HierarchyStats{
			TotalURLs:    0,
			MaxDepth:     0,
			URLsByType:   make(map[string]int),
			URLsByStatus: make(map[string]int),
		},
	}

	_ = query // Placeholder to use query variable

	return c.JSON(response)
}

// PerformanceMetrics represents performance statistics
type PerformanceMetrics struct {
	ExecutionID       string                   `json:"execution_id"`
	NodeMetrics       []*NodePerformanceMetric `json:"node_metrics"`
	TotalDurationMs   int64                    `json:"total_duration_ms"`
	URLProcessingRate float64                  `json:"url_processing_rate"` // URLs per second
}

type NodePerformanceMetric struct {
	NodeName            string  `json:"node_name"`
	NodeType            string  `json:"node_type"`
	Executions          int     `json:"executions"`
	AvgDurationMs       float64 `json:"avg_duration_ms"`
	TotalURLsDiscovered int     `json:"total_urls_discovered"`
	TotalItemsExtracted int     `json:"total_items_extracted"`
	Failures            int     `json:"failures"`
	SuccessRate         float64 `json:"success_rate"`
}

// GetPerformanceMetrics returns performance analysis
func (h *AnalyticsHandler) GetPerformanceMetrics(c *fiber.Ctx) error {
	executionID := c.Params("executionId")

	ctx := context.Background()

	nodeExecs, err := h.nodeExecRepo.GetByExecutionID(ctx, executionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get node executions: " + err.Error(),
		})
	}

	// Aggregate by node name
	nodeStats := make(map[string]*NodePerformanceMetric)
	var totalDuration int64

	for _, ne := range nodeExecs {
		if _, exists := nodeStats[ne.NodeID]; !exists {
			nodeType := ""
			if ne.NodeType != nil {
				nodeType = *ne.NodeType
			}
			nodeStats[ne.NodeID] = &NodePerformanceMetric{
				NodeName: ne.NodeID,
				NodeType: nodeType,
			}
		}

		metric := nodeStats[ne.NodeID]
		metric.Executions++
		metric.TotalURLsDiscovered += ne.URLsDiscovered
		metric.TotalItemsExtracted += ne.ItemsExtracted

		if ne.Status == models.ExecutionStatusFailed {
			metric.Failures++
		}

		if ne.DurationMs != nil {
			totalDuration += int64(*ne.DurationMs)
			metric.AvgDurationMs = (metric.AvgDurationMs*float64(metric.Executions-1) + float64(*ne.DurationMs)) / float64(metric.Executions)
		}
	}

	// Calculate success rates
	nodeMetrics := make([]*NodePerformanceMetric, 0, len(nodeStats))
	for _, metric := range nodeStats {
		if metric.Executions > 0 {
			metric.SuccessRate = float64(metric.Executions-metric.Failures) / float64(metric.Executions) * 100
		}
		nodeMetrics = append(nodeMetrics, metric)
	}

	response := &PerformanceMetrics{
		ExecutionID:     executionID,
		NodeMetrics:     nodeMetrics,
		TotalDurationMs: totalDuration,
	}

	return c.JSON(response)
}

// GetItemsWithHierarchy returns extracted items with their URL hierarchy
func (h *AnalyticsHandler) GetItemsWithHierarchy(c *fiber.Ctx) error {
	executionID := c.Params("executionId")

	ctx := context.Background()

	items, err := h.extractedItemRepo.GetWithHierarchy(ctx, executionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get items with hierarchy: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"execution_id": executionID,
		"items":        items,
		"count":        len(items),
	})
}

// BottleneckInfo represents a performance bottleneck
type BottleneckInfo struct {
	NodeExecutionID string `json:"node_execution_id"`
	NodeName        string `json:"node_name"`
	NodeType        string `json:"node_type"`
	URL             string `json:"url"`
	DurationMs      int    `json:"duration_ms"`
	Status          string `json:"status"`
	ErrorMessage    string `json:"error_message,omitempty"`
}

// GetBottlenecks identifies slow operations
func (h *AnalyticsHandler) GetBottlenecks(c *fiber.Ctx) error {
	executionID := c.Params("executionId")

	ctx := context.Background()

	nodeExecs, err := h.nodeExecRepo.GetByExecutionID(ctx, executionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get node executions: " + err.Error(),
		})
	}

	// Find slowest operations
	bottlenecks := make([]*BottleneckInfo, 0)

	for _, ne := range nodeExecs {
		if ne.DurationMs != nil && *ne.DurationMs > 5000 { // Threshold: 5 seconds
			nodeType := ""
			if ne.NodeType != nil {
				nodeType = *ne.NodeType
			}

			errorMsg := ""
			if ne.ErrorMessage != nil {
				errorMsg = *ne.ErrorMessage
			}

			bottleneck := &BottleneckInfo{
				NodeExecutionID: ne.ID,
				NodeName:        ne.NodeID,
				NodeType:        nodeType,
				DurationMs:      *ne.DurationMs,
				Status:          string(ne.Status),
				ErrorMessage:    errorMsg,
			}

			bottlenecks = append(bottlenecks, bottleneck)
		}
	}

	return c.JSON(fiber.Map{
		"execution_id": executionID,
		"bottlenecks":  bottlenecks,
		"count":        len(bottlenecks),
	})
}
