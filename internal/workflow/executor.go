package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/extraction"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/queue"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// ErrURLRequeued is returned when a URL is requeued for later processing
var ErrURLRequeued = errors.New("url requeued for later processing")

type Executor struct {
	browserPool        *browser.BrowserPool
	urlQueue           *queue.URLQueue
	parser             *Parser
	extractedItemsRepo *storage.ExtractedItemsRepository
	nodeExecRepo       *storage.NodeExecutionRepository
	executionRepo      *storage.ExecutionRepository
}

func NewExecutor(browserPool *browser.BrowserPool, urlQueue *queue.URLQueue, extractedItemsRepo *storage.ExtractedItemsRepository, nodeExecRepo *storage.NodeExecutionRepository, executionRepo *storage.ExecutionRepository) *Executor {
	return &Executor{
		browserPool:        browserPool,
		urlQueue:           urlQueue,
		parser:             NewParser(),
		extractedItemsRepo: extractedItemsRepo,
		nodeExecRepo:       nodeExecRepo,
		executionRepo:      executionRepo,
	}
}

// ExecuteWorkflow executes a complete workflow
func (e *Executor) ExecuteWorkflow(ctx context.Context, workflow *models.Workflow, executionID string) error {
	logger.Info("Starting workflow execution",
		zap.String("workflow_id", workflow.ID),
		zap.String("execution_id", executionID),
		zap.Int("start_url_count", len(workflow.Config.StartURLs)),
		zap.Any("start_urls", workflow.Config.StartURLs),
	)

	// Initialize execution stats
	startTime := time.Now()
	stats := models.ExecutionStats{
		URLsDiscovered:  0,
		URLsProcessed:   0,
		URLsFailed:      0,
		ItemsExtracted:  0,
		BytesDownloaded: 0,
		Duration:        0,
		NodesExecuted:   0,
		NodesFailed:     0,
		LastUpdate:      time.Now(),
	}

	// Enqueue start URLs
	for _, startURL := range workflow.Config.StartURLs {
		item := &models.URLQueueItem{
			ExecutionID: executionID,
			URL:         startURL,
			Depth:       0,
			Priority:    100,
			URLType:     "start", // Mark as start URL for proper phase detection
		}
		if err := e.urlQueue.Enqueue(ctx, item); err != nil {
			logger.Error("Failed to enqueue start URL", zap.Error(err), zap.String("url", startURL))
		} else {
			stats.URLsDiscovered++
		}
	}

	// Update stats periodically
	updateTicker := time.NewTicker(5 * time.Second)
	defer updateTicker.Stop()

	// Process URLs from queue
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-updateTicker.C:
			// Update execution stats
			stats.Duration = time.Since(startTime).Milliseconds()
			stats.LastUpdate = time.Now()

			// Get node execution stats from database
			if e.nodeExecRepo != nil {
				nodeStats, err := e.nodeExecRepo.GetStatsByExecutionID(ctx, executionID)
				if err == nil {
					stats.NodesExecuted = nodeStats["completed"]
					stats.NodesFailed = nodeStats["failed"]
				}
			}

			// Get extracted data count from database
			if e.extractedItemsRepo != nil {
				count, err := e.extractedItemsRepo.GetCount(ctx, executionID)
				if err == nil {
					stats.ItemsExtracted = count
				}
			}

			if e.executionRepo != nil {
				if err := e.executionRepo.UpdateStats(ctx, executionID, stats); err != nil {
					logger.Error("Failed to update execution stats", zap.Error(err))
				}
			}
		default:
			// Dequeue next URL
			item, err := e.urlQueue.Dequeue(ctx, executionID)
			if err != nil {
				logger.Error("Failed to dequeue URL", zap.Error(err))
				time.Sleep(1 * time.Second)
				continue
			}

			if item == nil {
				// No more URLs available right now - check if there might be requeued items
				// Wait a bit and check again before declaring completion
				time.Sleep(2 * time.Second)

				// Double-check if there are any pending items
				pendingCount, err := e.urlQueue.GetPendingCount(ctx, executionID)
				if err == nil && pendingCount > 0 {
					// There are still pending items (likely requeued), continue processing
					logger.Debug("Pending URLs found, continuing processing", zap.Int("pending_count", pendingCount))
					continue
				}

				// No more URLs in queue - update final stats
				stats.Duration = time.Since(startTime).Milliseconds()
				stats.LastUpdate = time.Now()

				// Get final node execution stats
				if e.nodeExecRepo != nil {
					nodeStats, err := e.nodeExecRepo.GetStatsByExecutionID(ctx, executionID)
					if err == nil {
						stats.NodesExecuted = nodeStats["completed"]
						stats.NodesFailed = nodeStats["failed"]
					}
				}

				// Get final extracted data count
				if e.extractedItemsRepo != nil {
					count, err := e.extractedItemsRepo.GetCount(ctx, executionID)
					if err == nil {
						stats.ItemsExtracted = count
					}
				}

				if e.executionRepo != nil {
					e.executionRepo.UpdateStats(ctx, executionID, stats)
					e.executionRepo.UpdateStatus(ctx, executionID, models.ExecutionStatusCompleted, "")
				}
				logger.Info("No more URLs to process",
					zap.Int("urls_processed", stats.URLsProcessed),
					zap.Int("items_extracted", stats.ItemsExtracted),
					zap.Int("nodes_executed", stats.NodesExecuted),
				)
				return nil
			}

			// Process the URL
			if err := e.processURL(ctx, workflow, executionID, item); err != nil {
				// Check if URL was requeued
				if errors.Is(err, ErrURLRequeued) {
					// URL was requeued for later, not a failure
					logger.Debug("URL requeued successfully", zap.String("url", item.URL))
					continue
				}

				logger.Error("Failed to process URL",
					zap.Error(err),
					zap.String("url", item.URL),
				)
				e.urlQueue.MarkFailed(ctx, item.ID, err.Error(), item.RetryCount < 3)
				stats.URLsFailed++
				continue
			}

			// Mark as completed
			if err := e.urlQueue.MarkCompleted(ctx, item.ID); err != nil {
				logger.Error("Failed to mark URL as completed", zap.Error(err))
			}
			stats.URLsProcessed++
		}
	}
}

// processURL processes a single URL
func (e *Executor) processURL(ctx context.Context, workflow *models.Workflow, executionID string, item *models.URLQueueItem) error {
	logger.Info("Processing URL", zap.String("url", item.URL), zap.Int("depth", item.Depth), zap.String("url_type", item.URLType))

	// Determine which nodes to execute based on URL type and discovery status
	isDiscoveryPhase := e.isURLDiscoveryPhase(item, workflow)
	isDataExtractionPhase := e.isDataExtractionPhase(item, workflow)

	// Early check: If this is an extraction URL and discovery is not complete, requeue immediately
	// This avoids unnecessary browser navigation and resource usage
	if isDataExtractionPhase && len(workflow.Config.DataExtraction) > 0 {
		if !e.shouldExecuteDataExtraction(ctx, item, workflow, executionID) {
			logger.Debug("Discovery not complete, re-queueing extraction URL (early check)",
				zap.String("url", item.URL),
				zap.String("url_type", item.URLType))

			if err := e.urlQueue.RequeueForLater(ctx, item.ID); err != nil {
				logger.Error("Failed to requeue URL", zap.Error(err))
				return err
			}
			return ErrURLRequeued
		}
	}

	// Acquire browser context
	browserCtx, err := e.browserPool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire browser context: %w", err)
	}
	defer e.browserPool.Release(browserCtx)

	// Set headers and cookies if configured
	if len(workflow.Config.Headers) > 0 {
		browserCtx.SetHeaders(workflow.Config.Headers)
	}

	// Navigate to URL
	_, err = browserCtx.Navigate(item.URL)
	if err != nil {
		return fmt.Errorf("failed to navigate to URL: %w", err)
	}

	// Apply rate limiting
	if workflow.Config.RateLimitDelay > 0 {
		time.Sleep(time.Duration(workflow.Config.RateLimitDelay) * time.Millisecond)
	}

	// Create execution context
	execCtx := models.NewExecutionContext()
	execCtx.Set("url", item.URL)
	execCtx.Set("depth", item.Depth)

	// Execute URL discovery nodes if in discovery phase
	if isDiscoveryPhase && len(workflow.Config.URLDiscovery) > 0 && item.Depth < workflow.Config.MaxDepth {
		// Get the specific node to execute based on dependencies
		nodesToExecute := e.getExecutableDiscoveryNodes(ctx, workflow.Config.URLDiscovery, item, executionID)

		if len(nodesToExecute) > 0 {
			logger.Info("Executing URL discovery nodes",
				zap.String("url", item.URL),
				zap.Int("node_count", len(nodesToExecute)))

			if err := e.executeSpecificNodes(ctx, nodesToExecute, browserCtx, &execCtx, executionID, item); err != nil {
				logger.Error("URL discovery failed", zap.Error(err))
			}
		}
	}

	// Execute data extraction nodes (discovery is already confirmed to be complete in early check)
	if isDataExtractionPhase && len(workflow.Config.DataExtraction) > 0 {
		logger.Info("Executing data extraction nodes",
			zap.String("url", item.URL),
			zap.String("url_type", item.URLType))

		var lastNodeExecID string
		if err := e.executeNodeGroup(ctx, workflow.Config.DataExtraction, browserCtx, &execCtx, executionID, item); err != nil {
			logger.Error("Data extraction failed", zap.Error(err))
		}

		// Save extracted data to database
		if e.extractedItemsRepo != nil {
			extractedData := e.collectExtractedData(&execCtx)
			if len(extractedData) > 0 {
				if err := e.saveExtractedItem(ctx, executionID, item.ID, item.URL, extractedData, lastNodeExecID); err != nil {
					logger.Error("Failed to save extracted data", zap.Error(err), zap.String("url", item.URL))
				} else {
					logger.Info("Saved extracted data", zap.String("url", item.URL), zap.Int("fields", len(extractedData)))
				}
			}
		}
	}

	return nil
}

// updateExecutionStatsForURL updates stats based on URL processing results
func (e *Executor) updateExecutionStatsForURL(ctx context.Context, executionID string, extracted bool, nodesExecuted int, nodesFailed int) {
	// This can be called periodically or after each URL
	// For now, we'll rely on the periodic updates in ExecuteWorkflow
}

// executeNodeGroup executes a group of nodes in DAG order
func (e *Executor) executeNodeGroup(ctx context.Context, nodes []models.Node, browserCtx *browser.BrowserContext, execCtx *models.ExecutionContext, executionID string, item *models.URLQueueItem) error {
	// Build DAG
	dag, err := e.parser.BuildDAG(nodes)
	if err != nil {
		return fmt.Errorf("failed to build DAG: %w", err)
	}

	// Get topologically sorted nodes
	sortedNodes, err := dag.TopologicalSort()
	if err != nil {
		return fmt.Errorf("failed to sort nodes: %w", err)
	}

	// Execute nodes in order
	for _, node := range sortedNodes {
		if err := e.executeNode(ctx, node, browserCtx, execCtx, executionID, item); err != nil {
			if !node.Optional {
				return fmt.Errorf("node '%s' failed: %w", node.ID, err)
			}
			logger.Warn("Optional node failed", zap.String("node_id", node.ID), zap.Error(err))
		}
	}

	return nil
}

// executeNode executes a single node
func (e *Executor) executeNode(ctx context.Context, node *models.Node, browserCtx *browser.BrowserContext, execCtx *models.ExecutionContext, executionID string, item *models.URLQueueItem) error {
	logger.Debug("Executing node", zap.String("node_id", node.ID), zap.String("type", string(node.Type)))

	// Create node execution record
	var nodeExecID string
	if e.nodeExecRepo != nil {
		inputData, _ := json.Marshal(node.Params)

		// Determine node type from node.Type
		nodeType := string(node.Type)

		nodeExec := &models.NodeExecution{
			ExecutionID: executionID,
			NodeID:      node.ID,
			Status:      models.ExecutionStatusRunning,
			URLID:       &item.ID, // Link to the URL being processed
			NodeType:    &nodeType,
			StartedAt:   time.Now(),
			Input:       inputData,
			RetryCount:  0,
		}

		if err := e.nodeExecRepo.Create(ctx, nodeExec); err != nil {
			logger.Error("Failed to create node execution record", zap.Error(err))
		} else {
			nodeExecID = nodeExec.ID
		}
	}

	interactionEngine := browser.NewInteractionEngine(browserCtx)
	extractionEngine := extraction.NewExtractionEngine(browserCtx.Page)

	var result interface{}
	var err error

	switch node.Type {
	case models.NodeTypeClick:
		selector := getStringParam(node.Params, "selector")
		err = interactionEngine.Click(selector)

	case models.NodeTypeScroll:
		x := getIntParam(node.Params, "x")
		y := getIntParam(node.Params, "y")
		err = interactionEngine.Scroll(x, y)

	case models.NodeTypeType:
		selector := getStringParam(node.Params, "selector")
		text := getStringParam(node.Params, "text")
		delay := time.Duration(getIntParam(node.Params, "delay")) * time.Millisecond
		err = interactionEngine.Type(selector, text, delay)

	case models.NodeTypeWait:
		duration := time.Duration(getIntParam(node.Params, "duration")) * time.Millisecond
		err = interactionEngine.Wait(duration)

	case models.NodeTypeWaitFor:
		selector := getStringParam(node.Params, "selector")
		timeout := time.Duration(getIntParam(node.Params, "timeout")) * time.Millisecond
		state := getStringParam(node.Params, "state")
		err = interactionEngine.WaitForSelector(selector, timeout, state)

	case models.NodeTypeExtract:
		var config extraction.ExtractConfig
		configBytes, _ := json.Marshal(node.Params)
		json.Unmarshal(configBytes, &config)

		// Handle field-based extraction (when fields are defined but no top-level selector)
		if len(config.Fields) > 0 && config.Selector == "" {
			// For field-based extraction, we extract individual fields and combine results
			fieldResults := make(map[string]interface{})

			for fieldName, fieldConfigRaw := range config.Fields {
				var fieldConfig extraction.ExtractConfig

				// Convert field config to ExtractConfig
				fieldConfigBytes, _ := json.Marshal(fieldConfigRaw)
				if err := json.Unmarshal(fieldConfigBytes, &fieldConfig); err != nil {
					logger.Warn("Failed to parse field config", zap.String("field", fieldName), zap.Error(err))
					continue
				}

				// Extract field value
				fieldValue, fieldErr := extractionEngine.Extract(fieldConfig)
				if fieldErr != nil {
					logger.Debug("Field extraction failed", zap.String("field", fieldName), zap.Error(fieldErr))
					// Use default value if extraction fails and it's defined
					if fieldConfig.DefaultValue != nil {
						fieldResults[fieldName] = fieldConfig.DefaultValue
					} else if defaultVal, ok := fieldConfigRaw.(map[string]interface{})["default"]; ok {
						// Handle "default" key for backward compatibility
						fieldResults[fieldName] = defaultVal
					}
					// Note: We still store the field with default value, not skip it
				} else {
					fieldResults[fieldName] = fieldValue
				}
			}

			result = fieldResults
			err = nil

			// Always store field-based extraction results in context for database saving
			for fieldName, fieldValue := range fieldResults {
				execCtx.Set(fieldName, fieldValue)
			}
		} else {
			// Normal extraction with top-level selector
			result, err = extractionEngine.Extract(config)
		}

		// Store schema name if present for later use
		if schema, ok := node.Params["schema"].(string); ok && schema != "" {
			execCtx.Set("_schema", schema)
		}

	case models.NodeTypeExtractLinks:
		selector := getStringParam(node.Params, "selector")
		if selector == "" {
			selector = "a"
		}
		// Get limit parameter if specified (0 means no limit)
		limit := getIntParam(node.Params, "limit")

		links, linkErr := extractionEngine.ExtractLinks(selector, limit)
		if linkErr == nil {
			result = links
			// Enqueue discovered URLs with hierarchy tracking
			if err := e.enqueueLinks(ctx, executionID, item, links, node.Params, node.ID, nodeExecID); err != nil {
				logger.Error("Failed to enqueue links", zap.Error(err))
			} else if e.nodeExecRepo != nil && nodeExecID != "" {
				// Update node execution with URLs discovered count
				if nodeExec, getErr := e.nodeExecRepo.GetByID(ctx, nodeExecID); getErr == nil {
					nodeExec.URLsDiscovered = len(links)
					e.nodeExecRepo.Update(ctx, nodeExec)
				}
			}
		}
		err = linkErr

	case models.NodeTypeNavigate:
		targetURL := getStringParam(node.Params, "url")
		_, err = browserCtx.Navigate(targetURL)

	case models.NodeTypeHover:
		selector := getStringParam(node.Params, "selector")
		err = interactionEngine.Hover(selector)

	case models.NodeTypePaginate:
		// Pagination node - handles both click-based and link-based pagination
		result, err = e.executePagination(ctx, node, browserCtx, extractionEngine, executionID, item)

	default:
		logger.Warn("Unknown node type", zap.String("type", string(node.Type)))
	}

	// Store result in context
	if result != nil {
		if node.OutputKey != "" {
			// Use specified output key
			execCtx.Set(node.OutputKey, result)
		} else if node.Type == models.NodeTypeExtract {
			// For extraction nodes without output key, store result with node ID
			execCtx.Set(node.ID, result)
		}
	}

	// Update node execution status
	if e.nodeExecRepo != nil && nodeExecID != "" {
		if err != nil {
			if updateErr := e.nodeExecRepo.MarkFailed(ctx, nodeExecID, err.Error()); updateErr != nil {
				logger.Error("Failed to mark node execution as failed", zap.Error(updateErr))
			}
		} else {
			if updateErr := e.nodeExecRepo.MarkCompleted(ctx, nodeExecID, result); updateErr != nil {
				logger.Error("Failed to mark node execution as completed", zap.Error(updateErr))
			}
		}
	}

	if err != nil && node.Retry.MaxRetries > 0 {
		// Retry logic
		for retry := 0; retry < node.Retry.MaxRetries; retry++ {
			time.Sleep(time.Duration(node.Retry.Delay) * time.Millisecond)
			logger.Info("Retrying node", zap.String("node_id", node.ID), zap.Int("attempt", retry+1))
			// Re-execute the node (simplified - should recursively call executeNode)
			break
		}
	}

	return err
}

// enqueueLinks enqueues discovered links with hierarchy tracking
func (e *Executor) enqueueLinks(ctx context.Context, executionID string, parentItem *models.URLQueueItem, links []string, params map[string]interface{}, nodeID, nodeExecID string) error {
	baseURL, err := url.Parse(parentItem.URL)
	if err != nil {
		return err
	}

	var items []*models.URLQueueItem

	// Determine URL type from params or default
	urlType := getStringParam(params, "url_type")
	if urlType == "" {
		urlType = "page" // default
	}

	for _, link := range links {
		// Resolve relative URLs
		linkURL, err := url.Parse(link)
		if err != nil {
			continue
		}

		absoluteURL := baseURL.ResolveReference(linkURL).String()

		// Apply URL filters if specified
		if shouldSkipURL(absoluteURL, params) {
			continue
		}

		item := &models.URLQueueItem{
			ExecutionID:      executionID,
			URL:              absoluteURL,
			Depth:            parentItem.Depth + 1,
			Priority:         parentItem.Priority - 10,
			ParentURLID:      &parentItem.ID, // Track parent URL
			DiscoveredByNode: &nodeID,        // Track which node discovered this
			URLType:          urlType,        // Set URL type
		}

		items = append(items, item)
	}

	return e.urlQueue.EnqueueBatch(ctx, items)
}

// Helper functions
func getStringParam(params map[string]interface{}, key string) string {
	if val, ok := params[key].(string); ok {
		return val
	}
	return ""
}

func getIntParam(params map[string]interface{}, key string) int {
	if val, ok := params[key].(float64); ok {
		return int(val)
	}
	if val, ok := params[key].(int); ok {
		return val
	}
	return 0
}

func shouldSkipURL(url string, params map[string]interface{}) bool {
	// Implement URL filtering logic based on params
	// For example: pattern matching, domain filtering, etc.
	return false
}

// isURLDiscoveryPhase checks if this URL should be in the discovery phase
func (e *Executor) isURLDiscoveryPhase(item *models.URLQueueItem, workflow *models.Workflow) bool {
	// Start URLs and URLs discovered during discovery should go through discovery
	// URLs with empty URLType are considered start URLs
	if item.URLType == "" || item.URLType == "start" {
		return true
	}

	// Check if this URL type is from a discovery node that is NOT a leaf node
	// (i.e., it has dependent nodes)
	return e.isDiscoveryURLType(item.URLType, workflow.Config.URLDiscovery) &&
		!e.isLastDiscoveryURLType(item.URLType, workflow.Config.URLDiscovery)
}

// isLastDiscoveryURLType checks if the URL type is from a leaf discovery node
func (e *Executor) isLastDiscoveryURLType(urlType string, discoveryNodes []models.Node) bool {
	lastTypes := e.getLastDiscoveryNodeURLTypes(discoveryNodes)
	for _, lastType := range lastTypes {
		if urlType == lastType {
			return true
		}
	}
	return false
}

// isDataExtractionPhase checks if this URL should be in the data extraction phase
func (e *Executor) isDataExtractionPhase(item *models.URLQueueItem, workflow *models.Workflow) bool {
	// Only process data extraction if URLType indicates it's from final discovery node
	// or if it's explicitly marked as needing extraction
	return e.isExtractionURLType(item.URLType, workflow.Config.URLDiscovery, workflow.Config.DataExtraction)
}

// isDiscoveryURLType checks if the URL type is produced by discovery nodes
func (e *Executor) isDiscoveryURLType(urlType string, discoveryNodes []models.Node) bool {
	if urlType == "" || urlType == "start" {
		return true
	}

	for _, node := range discoveryNodes {
		if nodeURLType := getStringParam(node.Params, "url_type"); nodeURLType == urlType {
			return true
		}
	}
	return false
}

// isExtractionURLType checks if URL type is suitable for data extraction
func (e *Executor) isExtractionURLType(urlType string, discoveryNodes []models.Node, extractionNodes []models.Node) bool {
	// Get the URL type from the last discovery node (nodes with no dependents)
	lastDiscoveryNodeURLTypes := e.getLastDiscoveryNodeURLTypes(discoveryNodes)

	// Check if this URL type matches the last discovery nodes
	for _, lastURLType := range lastDiscoveryNodeURLTypes {
		if urlType == lastURLType {
			return true
		}
	}

	// Also check if extraction nodes specify a url_type_filter
	for _, node := range extractionNodes {
		if urlTypeFilter := getStringParam(node.Params, "url_type_filter"); urlTypeFilter != "" {
			if urlType == urlTypeFilter {
				return true
			}
		}
	}

	return false
}

// getLastDiscoveryNodeURLTypes returns URL types from nodes with no dependents
func (e *Executor) getLastDiscoveryNodeURLTypes(discoveryNodes []models.Node) []string {
	if len(discoveryNodes) == 0 {
		return []string{}
	}

	// Build a map of nodes that have dependents
	hasDependents := make(map[string]bool)
	for _, node := range discoveryNodes {
		for _, depID := range node.Dependencies {
			hasDependents[depID] = true
		}
	}

	// Find nodes without dependents (leaf nodes)
	var urlTypes []string
	for _, node := range discoveryNodes {
		if !hasDependents[node.ID] {
			if urlType := getStringParam(node.Params, "url_type"); urlType != "" {
				urlTypes = append(urlTypes, urlType)
			}
		}
	}

	return urlTypes
}

// getExecutableDiscoveryNodes returns discovery nodes that should execute for this URL
func (e *Executor) getExecutableDiscoveryNodes(ctx context.Context, discoveryNodes []models.Node, item *models.URLQueueItem, executionID string) []models.Node {
	var executableNodes []models.Node

	// For start URLs (depth 0 or empty URLType), execute root nodes only
	if item.Depth == 0 || item.URLType == "" || item.URLType == "start" {
		// Get root nodes (nodes with no dependencies)
		for _, node := range discoveryNodes {
			if len(node.Dependencies) == 0 {
				executableNodes = append(executableNodes, node)
			}
		}
		return executableNodes
	}

	// For discovered URLs, find nodes that should execute based on the discovered URL type
	// We need to find nodes whose dependencies have produced this URL type
	if item.DiscoveredByNode != nil && *item.DiscoveredByNode != "" {
		// Find nodes that depend on the node that discovered this URL
		for _, node := range discoveryNodes {
			for _, depID := range node.Dependencies {
				if depID == *item.DiscoveredByNode {
					executableNodes = append(executableNodes, node)
					break
				}
			}
		}
	}

	return executableNodes
}

// executeSpecificNodes executes specific nodes without DAG sorting
func (e *Executor) executeSpecificNodes(ctx context.Context, nodes []models.Node, browserCtx *browser.BrowserContext, execCtx *models.ExecutionContext, executionID string, item *models.URLQueueItem) error {
	// Execute nodes in the order provided
	for _, node := range nodes {
		if err := e.executeNode(ctx, &node, browserCtx, execCtx, executionID, item); err != nil {
			if !node.Optional {
				return fmt.Errorf("node '%s' failed: %w", node.ID, err)
			}
			logger.Warn("Optional node failed", zap.String("node_id", node.ID), zap.Error(err))
		}
	}
	return nil
}

// shouldExecuteDataExtraction determines if data extraction should run for this URL
func (e *Executor) shouldExecuteDataExtraction(ctx context.Context, item *models.URLQueueItem, workflow *models.Workflow, executionID string) bool {
	// Check if all URL discovery is complete for this execution
	if !e.isAllURLDiscoveryComplete(ctx, workflow, executionID) {
		logger.Debug("URL discovery not complete yet",
			zap.String("execution_id", executionID),
			zap.String("url", item.URL))
		return false
	}

	// Check if URL type is appropriate for extraction
	return e.isExtractionURLType(item.URLType, workflow.Config.URLDiscovery, workflow.Config.DataExtraction)
}

// isAllURLDiscoveryComplete checks if all URL discovery nodes have completed
func (e *Executor) isAllURLDiscoveryComplete(ctx context.Context, workflow *models.Workflow, executionID string) bool {
	// Check if there are any pending discovery URLs in the queue
	if e.urlQueue != nil {
		hasPendingDiscovery, err := e.urlQueue.HasPendingDiscoveryURLs(ctx, executionID, workflow.Config.URLDiscovery)
		if err != nil {
			logger.Error("Failed to check pending discovery URLs", zap.Error(err))
			// On error, assume discovery is not complete to be safe
			return false
		}
		return !hasPendingDiscovery
	}

	// If we can't check, assume it's complete (fallback)
	return true
}

// matchesPattern checks if URL matches a glob-style pattern
func matchesPattern(url string, pattern string) bool {
	// Simple pattern matching - supports wildcards
	// Pattern examples:
	// - "*/dp/*" matches any URL with /dp/ in it
	// - "https://www.amazon.com/dp/*" matches Amazon product pages
	// - "*product*" matches any URL containing "product"

	// Convert glob pattern to simple matching
	if pattern == "*" {
		return true
	}

	// Check if pattern contains the URL or vice versa
	if len(pattern) > 0 && pattern[0] == '*' && pattern[len(pattern)-1] == '*' {
		// *pattern* - contains
		substr := pattern[1 : len(pattern)-1]
		return len(substr) == 0 || containsString(url, substr)
	} else if len(pattern) > 0 && pattern[0] == '*' {
		// *pattern - ends with
		suffix := pattern[1:]
		return len(url) >= len(suffix) && url[len(url)-len(suffix):] == suffix
	} else if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		// pattern* - starts with
		prefix := pattern[:len(pattern)-1]
		return len(url) >= len(prefix) && url[:len(prefix)] == prefix
	}

	// Exact match
	return url == pattern
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) && (s[:len(substr)] == substr ||
				containsString(s[1:], substr))))
}

// collectExtractedData collects all extracted data from execution context
func (e *Executor) collectExtractedData(execCtx *models.ExecutionContext) map[string]interface{} {
	data := make(map[string]interface{})

	// Get all values from execution context, excluding internal fields
	contextData := execCtx.GetAll()
	for key, value := range contextData {
		// Skip internal fields like url, depth, but keep _schema
		if key != "url" && key != "depth" {
			data[key] = value
		}
	}

	return data
}

// saveExtractedItem saves extracted data to database as ExtractedItem
func (e *Executor) saveExtractedItem(ctx context.Context, executionID, urlID, url string, data map[string]interface{}, nodeExecID string) error {
	if len(data) == 0 {
		return nil
	}

	// Extract common fields from data
	var title *string
	var price *float64
	var currency *string
	var availability *string
	var rating *float64
	var reviewCount *int
	var schemaName *string

	// Set default item type
	itemType := "item"

	// Try to extract schema name from data
	if schema, ok := data["_schema"].(string); ok {
		schemaName = &schema
		itemType = schema
		delete(data, "_schema")
	} else if schema, ok := data["schema"].(string); ok {
		schemaName = &schema
		itemType = schema
		delete(data, "schema")
	}

	// Extract title
	if t, ok := data["title"].(string); ok {
		title = &t
	}

	// Extract price
	if p, ok := data["price"].(string); ok {
		// Try to parse as float
		if pf, err := parsePrice(p); err == nil {
			price = &pf
		}
	} else if p, ok := data["price"].(float64); ok {
		price = &p
	} else if p, ok := data["price"].(int); ok {
		pf := float64(p)
		price = &pf
	}

	// Extract currency
	if c, ok := data["currency"].(string); ok {
		currency = &c
	} else {
		defaultCurrency := "USD"
		currency = &defaultCurrency
	}

	// Extract availability
	if a, ok := data["availability"].(string); ok {
		availability = &a
	}

	// Extract rating
	if r, ok := data["rating"].(string); ok {
		if rf, err := parseRating(r); err == nil {
			rating = &rf
		}
	} else if r, ok := data["rating"].(float64); ok {
		rating = &r
	}

	// Extract review count
	if rc, ok := data["review_count"].(string); ok {
		if rci, err := parseInt(rc); err == nil {
			reviewCount = &rci
		}
	} else if rc, ok := data["review_count"].(int); ok {
		reviewCount = &rc
	} else if rc, ok := data["review_count"].(float64); ok {
		rci := int(rc)
		reviewCount = &rci
	}

	// Store remaining data in attributes
	attributes := make(models.JSONMap)
	for k, v := range data {
		// Skip already extracted fields
		if k != "title" && k != "price" && k != "currency" && k != "availability" && k != "rating" && k != "review_count" {
			attributes[k] = v
		}
	}

	var nodeExecIDPtr *string
	if nodeExecID != "" {
		nodeExecIDPtr = &nodeExecID
	}

	item := &models.ExtractedItem{
		ExecutionID:     executionID,
		URLID:           urlID,
		NodeExecutionID: nodeExecIDPtr,
		ItemType:        itemType,
		SchemaName:      schemaName,
		Title:           title,
		Price:           price,
		Currency:        currency,
		Availability:    availability,
		Rating:          rating,
		ReviewCount:     reviewCount,
		Attributes:      attributes,
	}

	return e.extractedItemsRepo.Create(ctx, item)
}

// Helper functions for parsing extracted data
func parsePrice(s string) (float64, error) {
	// Remove currency symbols and whitespace
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, "€", "")
	s = strings.ReplaceAll(s, "£", "")
	s = strings.ReplaceAll(s, "¥", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)

	if s == "" {
		return 0, fmt.Errorf("empty price")
	}

	return strconv.ParseFloat(s, 64)
}

func parseRating(s string) (float64, error) {
	// Extract numeric rating from strings like "4.5 out of 5 stars"
	s = strings.TrimSpace(s)

	// Try direct parsing first
	if r, err := strconv.ParseFloat(s, 64); err == nil {
		return r, nil
	}

	// Try to extract first number
	re := regexp.MustCompile(`(\d+\.?\d*)`)
	matches := re.FindStringSubmatch(s)
	if len(matches) > 1 {
		return strconv.ParseFloat(matches[1], 64)
	}

	return 0, fmt.Errorf("could not parse rating")
}

func parseInt(s string) (int, error) {
	// Remove commas and whitespace
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")

	// Extract first number
	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindStringSubmatch(s)
	if len(matches) > 1 {
		return strconv.Atoi(matches[1])
	}

	return strconv.Atoi(s)
}

// executePagination handles pagination logic - both click-based and link-based
func (e *Executor) executePagination(ctx context.Context, node *models.Node, browserCtx *browser.BrowserContext, extractionEngine *extraction.ExtractionEngine, executionID string, item *models.URLQueueItem) (interface{}, error) {
	// Pagination parameters
	nextSelector := getStringParam(node.Params, "next_selector")       // Selector for "Next" button/link
	paginationSelector := getStringParam(node.Params, "link_selector") // Selector for pagination links (1,2,3...)
	maxPages := getIntParam(node.Params, "max_pages")                  // Maximum pages to paginate (0 = unlimited)
	paginationType := getStringParam(node.Params, "type")              // "click" or "link" (default: auto-detect)
	waitAfterClick := getIntParam(node.Params, "wait_after")           // Wait time after click/navigation (ms)
	itemSelector := getStringParam(node.Params, "item_selector")       // Items to extract from each page

	// Validate parameters
	if nextSelector == "" && paginationSelector == "" {
		return nil, fmt.Errorf("pagination requires either next_selector or link_selector")
	}

	// Default values
	if maxPages == 0 {
		maxPages = 100 // Default max pages to prevent infinite loops
	}
	if waitAfterClick == 0 {
		waitAfterClick = 2000 // Default 2 seconds wait
	}

	logger.Info("Starting pagination",
		zap.String("url", item.URL),
		zap.String("next_selector", nextSelector),
		zap.String("link_selector", paginationSelector),
		zap.Int("max_pages", maxPages),
		zap.String("type", paginationType))

	var allLinks []string
	currentPage := 1

	// Pagination loop
	for currentPage <= maxPages {
		logger.Debug("Processing pagination page",
			zap.Int("page", currentPage),
			zap.String("url", item.URL))

		// Extract items/links from current page if item_selector is provided
		if itemSelector != "" {
			links, err := extractionEngine.ExtractLinks(itemSelector, 0)
			if err == nil && len(links) > 0 {
				// Resolve relative URLs
				baseURL, _ := url.Parse(item.URL)
				for _, link := range links {
					linkURL, err := url.Parse(link)
					if err == nil {
						absoluteURL := baseURL.ResolveReference(linkURL).String()
						allLinks = append(allLinks, absoluteURL)
					}
				}
				logger.Debug("Extracted items from page",
					zap.Int("page", currentPage),
					zap.Int("items", len(links)))
			}
		}

		// Check if we've reached max pages
		if currentPage >= maxPages {
			logger.Info("Reached max pages limit", zap.Int("max_pages", maxPages))
			break
		}

		// Try to navigate to next page
		navigated := false

		// Strategy 1: Try next button/link (click-based or href-based)
		if nextSelector != "" {
			nextNavigated, err := e.tryNavigateNext(browserCtx, nextSelector, paginationType, waitAfterClick)
			if err != nil {
				logger.Debug("Next navigation failed", zap.Error(err))
			} else if nextNavigated {
				navigated = true
				item.URL = browserCtx.Page.URL() // Update current URL
			}
		}

		// Strategy 2: Try pagination links if next button didn't work
		if !navigated && paginationSelector != "" {
			linkNavigated, err := e.tryNavigatePaginationLink(browserCtx, paginationSelector, currentPage+1, paginationType, waitAfterClick)
			if err != nil {
				logger.Debug("Pagination link navigation failed", zap.Error(err))
			} else if linkNavigated {
				navigated = true
				item.URL = browserCtx.Page.URL() // Update current URL
			}
		}

		// If no navigation succeeded, we've reached the end
		if !navigated {
			logger.Info("No more pages available", zap.Int("pages_processed", currentPage))
			break
		}

		currentPage++
	}

	logger.Info("Pagination completed",
		zap.Int("pages_processed", currentPage),
		zap.Int("total_items", len(allLinks)))

	// Enqueue all discovered links if we have them
	if len(allLinks) > 0 {
		if err := e.enqueueLinks(ctx, executionID, item, allLinks, node.Params, node.ID, ""); err != nil {
			logger.Error("Failed to enqueue paginated links", zap.Error(err))
			return nil, err
		}
	}

	return map[string]interface{}{
		"pages_processed": currentPage,
		"items_found":     len(allLinks),
		"links":           allLinks,
	}, nil
}

// tryNavigateNext attempts to navigate using the "next" button/link
func (e *Executor) tryNavigateNext(browserCtx *browser.BrowserContext, nextSelector string, paginationType string, waitAfter int) (bool, error) {
	page := browserCtx.Page

	// Check if next button exists and is visible
	locator := page.Locator(nextSelector)
	count, err := locator.Count()
	if err != nil || count == 0 {
		return false, fmt.Errorf("next element not found")
	}

	// Check if element is disabled or hidden
	isVisible, err := locator.First().IsVisible()
	if err != nil || !isVisible {
		return false, fmt.Errorf("next element not visible")
	}

	// Check if it's disabled (common for pagination)
	isDisabled, err := locator.First().IsDisabled()
	if err == nil && isDisabled {
		return false, fmt.Errorf("next element is disabled")
	}

	// Scroll the element into view before interacting
	err = locator.First().ScrollIntoViewIfNeeded()
	if err != nil {
		logger.Warn("Failed to scroll next button into view", zap.Error(err))
		// Continue anyway - might still work
	}

	// Small delay after scroll to ensure element is ready
	time.Sleep(500 * time.Millisecond)

	// Auto-detect type or use specified type
	if paginationType == "" || paginationType == "auto" {
		// Try to get href attribute
		href, err := locator.First().GetAttribute("href")
		if err == nil && href != "" && href != "#" && href != "javascript:void(0)" {
			paginationType = "link"
		} else {
			paginationType = "click"
		}
	}

	currentURL := page.URL()

	// Navigate based on type
	if paginationType == "link" {
		// Extract href and navigate
		href, err := locator.First().GetAttribute("href")
		if err != nil || href == "" || href == "#" {
			return false, fmt.Errorf("invalid href")
		}

		// Resolve relative URL
		baseURL, _ := url.Parse(currentURL)
		linkURL, err := url.Parse(href)
		if err != nil {
			return false, fmt.Errorf("invalid URL: %w", err)
		}
		absoluteURL := baseURL.ResolveReference(linkURL).String()

		// Navigate to the URL
		_, err = browserCtx.Navigate(absoluteURL)
		if err != nil {
			return false, fmt.Errorf("navigation failed: %w", err)
		}
	} else {
		// Click-based navigation
		err := locator.First().Click()
		if err != nil {
			return false, fmt.Errorf("click failed: %w", err)
		}
	}

	// Wait after navigation
	time.Sleep(time.Duration(waitAfter) * time.Millisecond)

	// Verify URL changed or content loaded
	newURL := page.URL()
	if newURL == currentURL && paginationType == "link" {
		return false, fmt.Errorf("URL did not change after navigation")
	}

	return true, nil
}

// tryNavigatePaginationLink attempts to navigate using pagination number links
func (e *Executor) tryNavigatePaginationLink(browserCtx *browser.BrowserContext, linkSelector string, pageNumber int, paginationType string, waitAfter int) (bool, error) {
	page := browserCtx.Page

	// Find all pagination links
	locator := page.Locator(linkSelector)
	count, err := locator.Count()
	if err != nil || count == 0 {
		return false, fmt.Errorf("pagination links not found")
	}

	// Try to find link with matching page number
	var targetLocator interface{} = nil
	for i := 0; i < count; i++ {
		element := locator.Nth(i)
		text, err := element.InnerText()
		if err == nil {
			text = strings.TrimSpace(text)
			if text == strconv.Itoa(pageNumber) {
				targetLocator = element
				break
			}
		}
	}

	if targetLocator == nil {
		return false, fmt.Errorf("page number %d not found", pageNumber)
	}

	// Scroll the pagination link into view before interacting
	err = targetLocator.(interface{ ScrollIntoViewIfNeeded() error }).ScrollIntoViewIfNeeded()
	if err != nil {
		logger.Warn("Failed to scroll pagination link into view", zap.Error(err))
		// Continue anyway - might still work
	}

	// Small delay after scroll to ensure element is ready
	time.Sleep(500 * time.Millisecond)

	// Navigate using the found link
	currentURL := page.URL()

	if paginationType == "link" || paginationType == "" {
		// Try href first
		href, err := targetLocator.(interface{ GetAttribute(string) (string, error) }).GetAttribute("href")
		if err == nil && href != "" && href != "#" {
			baseURL, _ := url.Parse(currentURL)
			linkURL, err := url.Parse(href)
			if err == nil {
				absoluteURL := baseURL.ResolveReference(linkURL).String()
				_, err = browserCtx.Navigate(absoluteURL)
				if err == nil {
					time.Sleep(time.Duration(waitAfter) * time.Millisecond)
					return true, nil
				}
			}
		}
	}

	// Fall back to click
	err = targetLocator.(interface{ Click() error }).Click()
	if err != nil {
		return false, fmt.Errorf("click failed: %w", err)
	}

	time.Sleep(time.Duration(waitAfter) * time.Millisecond)
	return true, nil
}
