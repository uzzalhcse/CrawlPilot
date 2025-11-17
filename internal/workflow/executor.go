package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/extraction"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/queue"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

type Executor struct {
	browserPool       *browser.BrowserPool
	urlQueue          *queue.URLQueue
	parser            *Parser
	extractedDataRepo *storage.ExtractedDataRepository
	nodeExecRepo      *storage.NodeExecutionRepository
	executionRepo     *storage.ExecutionRepository
}

func NewExecutor(browserPool *browser.BrowserPool, urlQueue *queue.URLQueue, extractedDataRepo *storage.ExtractedDataRepository, nodeExecRepo *storage.NodeExecutionRepository, executionRepo *storage.ExecutionRepository) *Executor {
	return &Executor{
		browserPool:       browserPool,
		urlQueue:          urlQueue,
		parser:            NewParser(),
		extractedDataRepo: extractedDataRepo,
		nodeExecRepo:      nodeExecRepo,
		executionRepo:     executionRepo,
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
			if e.extractedDataRepo != nil {
				count, err := e.extractedDataRepo.Count(ctx, executionID)
				if err == nil {
					stats.ItemsExtracted = int(count)
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
				if e.extractedDataRepo != nil {
					count, err := e.extractedDataRepo.Count(ctx, executionID)
					if err == nil {
						stats.ItemsExtracted = int(count)
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
	logger.Info("Processing URL", zap.String("url", item.URL), zap.Int("depth", item.Depth))

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

	// Execute URL discovery nodes
	if len(workflow.Config.URLDiscovery) > 0 && item.Depth < workflow.Config.MaxDepth {
		if err := e.executeNodeGroup(ctx, workflow.Config.URLDiscovery, browserCtx, &execCtx, executionID, item); err != nil {
			logger.Error("URL discovery failed", zap.Error(err))
		}
	}

	// Execute data extraction nodes
	if len(workflow.Config.DataExtraction) > 0 {
		if err := e.executeNodeGroup(ctx, workflow.Config.DataExtraction, browserCtx, &execCtx, executionID, item); err != nil {
			logger.Error("Data extraction failed", zap.Error(err))
		}

		// Save extracted data to database
		if e.extractedDataRepo != nil {
			extractedData := e.collectExtractedData(&execCtx)
			if len(extractedData) > 0 {
				if err := e.saveExtractedData(ctx, executionID, item.URL, extractedData); err != nil {
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
		nodeExec := &models.NodeExecution{
			ExecutionID: executionID,
			NodeID:      node.ID,
			Status:      models.ExecutionStatusRunning,
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
		result, err = extractionEngine.Extract(config)

	case models.NodeTypeExtractLinks:
		selector := getStringParam(node.Params, "selector")
		if selector == "" {
			selector = "a"
		}
		links, linkErr := extractionEngine.ExtractLinks(selector)
		if linkErr == nil {
			result = links
			// Enqueue discovered URLs
			if err := e.enqueueLinks(ctx, executionID, item, links, node.Params); err != nil {
				logger.Error("Failed to enqueue links", zap.Error(err))
			}
		}
		err = linkErr

	case models.NodeTypeNavigate:
		targetURL := getStringParam(node.Params, "url")
		_, err = browserCtx.Navigate(targetURL)

	case models.NodeTypeHover:
		selector := getStringParam(node.Params, "selector")
		err = interactionEngine.Hover(selector)

	default:
		logger.Warn("Unknown node type", zap.String("type", string(node.Type)))
	}

	// Store result in context if output key is specified
	if node.OutputKey != "" && result != nil {
		execCtx.Set(node.OutputKey, result)
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

// enqueueLinks enqueues discovered links
func (e *Executor) enqueueLinks(ctx context.Context, executionID string, parentItem *models.URLQueueItem, links []string, params map[string]interface{}) error {
	baseURL, err := url.Parse(parentItem.URL)
	if err != nil {
		return err
	}

	var items []*models.URLQueueItem

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
			ExecutionID: executionID,
			URL:         absoluteURL,
			Depth:       parentItem.Depth + 1,
			Priority:    parentItem.Priority - 10,
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

// collectExtractedData collects all extracted data from execution context
func (e *Executor) collectExtractedData(execCtx *models.ExecutionContext) map[string]interface{} {
	data := make(map[string]interface{})

	// Get all values from execution context, excluding internal fields
	contextData := execCtx.GetAll()
	for key, value := range contextData {
		// Skip internal fields like url and depth
		if key != "url" && key != "depth" {
			data[key] = value
		}
	}

	return data
}

// saveExtractedData saves extracted data to database
func (e *Executor) saveExtractedData(ctx context.Context, executionID, url string, data map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	extracted := &models.ExtractedData{
		ExecutionID: executionID,
		URL:         url,
		Data:        models.JSONMap(data),
	}

	return e.extractedDataRepo.Create(ctx, extracted)
}
