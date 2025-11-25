package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

// NodeTreeNode represents a node in the execution tree
type NodeTreeNode struct {
	ID                    string          `json:"id"`
	NodeID                string          `json:"node_id"`
	NodeType              string          `json:"node_type"`
	Status                string          `json:"status"`
	StartedAt             string          `json:"started_at"`
	CompletedAt           *string         `json:"completed_at,omitempty"`
	DurationMs            *int            `json:"duration_ms,omitempty"`
	URLsDiscovered        int             `json:"urls_discovered"`
	ItemsExtracted        int             `json:"items_extracted"`
	ErrorMessage          *string         `json:"error_message,omitempty"`
	ParentNodeExecutionID *string         `json:"parent_node_execution_id,omitempty"`
	Children              []*NodeTreeNode `json:"children,omitempty"`
}

type NodeTreeResponse struct {
	ExecutionID string          `json:"execution_id"`
	Tree        []*NodeTreeNode `json:"tree"`
	Stats       *NodeTreeStats  `json:"stats"`
}

type NodeTreeStats struct {
	TotalNodes      int `json:"total_nodes"`
	CompletedNodes  int `json:"completed_nodes"`
	FailedNodes     int `json:"failed_nodes"`
	MaxDepth        int `json:"max_depth"`
	TotalURLsFound  int `json:"total_urls_found"`
	TotalItemsFound int `json:"total_items_found"`
}

// GetNodeTree returns the hierarchical node execution tree
func (h *AnalyticsHandler) GetNodeTree(c *fiber.Ctx) error {
	executionID := c.Params("executionId")

	ctx := context.Background()

	// Get all node executions
	nodeExecs, err := h.nodeExecRepo.GetByExecutionID(ctx, executionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get node executions: " + err.Error(),
		})
	}

	// Build tree structure
	nodeMap := make(map[string]*NodeTreeNode)
	var rootNodes []*NodeTreeNode
	stats := &NodeTreeStats{}

	// First pass: create all nodes
	for _, ne := range nodeExecs {
		nodeType := ""
		if ne.NodeType != nil {
			nodeType = *ne.NodeType
		}

		var completedAt *string
		if ne.CompletedAt != nil {
			completedAtStr := ne.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
			completedAt = &completedAtStr
		}

		node := &NodeTreeNode{
			ID:                    ne.ID,
			NodeID:                ne.NodeID,
			NodeType:              nodeType,
			Status:                string(ne.Status),
			StartedAt:             ne.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
			CompletedAt:           completedAt,
			DurationMs:            ne.DurationMs,
			URLsDiscovered:        ne.URLsDiscovered,
			ItemsExtracted:        ne.ItemsExtracted,
			ErrorMessage:          ne.ErrorMessage,
			ParentNodeExecutionID: ne.ParentNodeExecutionID,
			Children:              []*NodeTreeNode{},
		}

		nodeMap[ne.ID] = node

		// Update stats
		stats.TotalNodes++
		stats.TotalURLsFound += ne.URLsDiscovered
		stats.TotalItemsFound += ne.ItemsExtracted

		if ne.Status == "completed" {
			stats.CompletedNodes++
		} else if ne.Status == "failed" {
			stats.FailedNodes++
		}
	}

	// Second pass: build parent-child relationships
	for _, node := range nodeMap {
		if node.ParentNodeExecutionID != nil && *node.ParentNodeExecutionID != "" {
			// This node has a parent
			if parent, exists := nodeMap[*node.ParentNodeExecutionID]; exists {
				parent.Children = append(parent.Children, node)
			} else {
				// Parent not found in this execution - treat as root
				rootNodes = append(rootNodes, node)
			}
		} else {
			// No parent - this is a root node
			rootNodes = append(rootNodes, node)
		}
	}

	// Calculate max depth
	for _, root := range rootNodes {
		depth := calculateDepth(root, 1)
		if depth > stats.MaxDepth {
			stats.MaxDepth = depth
		}
	}

	response := &NodeTreeResponse{
		ExecutionID: executionID,
		Tree:        rootNodes,
		Stats:       stats,
	}

	return c.JSON(response)
}

// calculateDepth recursively calculates the maximum depth of a tree
func calculateDepth(node *NodeTreeNode, currentDepth int) int {
	if len(node.Children) == 0 {
		return currentDepth
	}

	maxChildDepth := currentDepth
	for _, child := range node.Children {
		childDepth := calculateDepth(child, currentDepth+1)
		if childDepth > maxChildDepth {
			maxChildDepth = childDepth
		}
	}

	return maxChildDepth
}
