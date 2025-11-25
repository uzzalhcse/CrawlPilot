package workflow

import (
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// getParentNodeExecID determines the parent node execution ID for SSE events
// This is the same logic used when creating node executions
func getParentNodeExecID(execCtx *models.ExecutionContext, item *models.URLQueueItem) *string {
	// Priority 1: If this is the first node processing a discovered URL, use the discovering node's execution ID from the URL queue
	if item.ParentNodeExecutionID != nil && *item.ParentNodeExecutionID != "" {
		if _, ok := execCtx.Get("_last_node_exec_id"); !ok {
			// First node for this URL - parent is the discovering node
			return item.ParentNodeExecutionID
		}
	}

	// Priority 2: Use the last executed node's ID from context (sequential nodes)
	if lastNodeID, ok := execCtx.Get("_last_node_exec_id"); ok {
		if lastNodeIDStr, ok := lastNodeID.(string); ok {
			return &lastNodeIDStr
		}
	}

	return nil
}
