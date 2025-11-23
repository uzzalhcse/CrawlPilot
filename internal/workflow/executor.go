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

	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/extraction"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/queue"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
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
	registry           *NodeRegistry // NEW: Plugin registry for extensible node execution
}

func NewExecutor(browserPool *browser.BrowserPool, urlQueue *queue.URLQueue, extractedItemsRepo *storage.ExtractedItemsRepository, nodeExecRepo *storage.NodeExecutionRepository, executionRepo *storage.ExecutionRepository) *Executor {
	// Create registry and register default nodes
	registry := NewNodeRegistry()
	if err := registry.RegisterDefaultNodes(); err != nil {
		logger.Error("Failed to register default nodes", zap.Error(err))
	}

	return &Executor{
		browserPool:        browserPool,
		urlQueue:           urlQueue,
		parser:             NewParser(),
		extractedItemsRepo: extractedItemsRepo,
		nodeExecRepo:       nodeExecRepo,
		executionRepo:      executionRepo,
		registry:           registry,
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

// processURL processes a single URL using phase-based workflow
func (e *Executor) processURL(ctx context.Context, workflow *models.Workflow, executionID string, item *models.URLQueueItem) error {
	logger.Info("Processing URL",
		zap.String("url", item.URL),
		zap.Int("depth", item.Depth),
		zap.String("phase_id", item.PhaseID))

	// Find the phase to execute for this URL
	var phaseToExecute *models.WorkflowPhase
	for i := range workflow.Config.Phases {
		phase := &workflow.Config.Phases[i]

		// Check if this URL should be processed by this phase
		if e.urlMatchesPhase(item, phase) {
			phaseToExecute = phase
			break
		}
	}

	if phaseToExecute == nil {
		// No matching phase - this might be a start URL, use first phase
		if len(workflow.Config.Phases) > 0 {
			phaseToExecute = &workflow.Config.Phases[0]
			logger.Debug("Using first phase for start URL", zap.String("phase_id", phaseToExecute.ID))
		} else {
			return fmt.Errorf("no phases configured in workflow")
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
	execCtx.Set("phase_id", phaseToExecute.ID)

	// Execute phase nodes
	logger.Info("Executing phase nodes",
		zap.String("phase_id", phaseToExecute.ID),
		zap.String("phase_type", string(phaseToExecute.Type)),
		zap.Int("node_count", len(phaseToExecute.Nodes)))

	if err := e.executeNodeGroup(ctx, phaseToExecute.Nodes, browserCtx, &execCtx, executionID, item); err != nil {
		logger.Error("Phase execution failed",
			zap.String("phase_id", phaseToExecute.ID),
			zap.Error(err))
	}

	// Save extracted data if this is an extraction phase
	if phaseToExecute.Type == models.PhaseTypeExtraction {
		if e.extractedItemsRepo != nil {
			extractedData := e.collectExtractedData(&execCtx)
			if len(extractedData) > 0 {
				schemaName := phaseToExecute.ID
				// Get node execution ID from context (set by the last extraction node)
				var nodeExecIDPtr *string
				if nodeExecID, ok := execCtx.Get("_node_exec_id"); ok {
					if nodeExecIDStr, ok := nodeExecID.(string); ok {
						nodeExecIDPtr = &nodeExecIDStr
					}
				}
				if err := e.saveExtractedData(ctx, executionID, item.ID, schemaName, nodeExecIDPtr, extractedData); err != nil {
					logger.Error("Failed to save extracted data", zap.Error(err))
				} else {
					logger.Info("Saved extracted data",
						zap.String("url", item.URL),
						zap.Int("fields", len(extractedData)))
				}
			}
		}
	}

	// Check for phase transition
	if phaseToExecute.Transition != nil {
		if err := e.handlePhaseTransition(ctx, workflow, phaseToExecute, item, executionID, &execCtx); err != nil {
			logger.Error("Phase transition failed", zap.Error(err))
		}
	}

	return nil
}

// urlMatchesPhase checks if a URL should be processed in a given phase
func (e *Executor) urlMatchesPhase(item *models.URLQueueItem, phase *models.WorkflowPhase) bool {
	logger.Debug("Checking phase match",
		zap.String("url", item.URL),
		zap.String("phase_id", phase.ID),
		zap.String("item_phase_id", item.PhaseID),
		zap.String("item_marker", item.Marker),
		zap.Int("item_depth", item.Depth))

	// If URL has a phase ID assigned, match exactly
	if item.PhaseID != "" {
		matches := item.PhaseID == phase.ID
		logger.Debug("Phase ID match", zap.Bool("matches", matches))
		return matches
	}

	// Otherwise, check URLFilter if present
	if phase.URLFilter == nil {
		logger.Debug("No URL filter for phase", zap.String("phase_id", phase.ID))
		return false
	}

	filter := phase.URLFilter

	// Check markers
	if len(filter.Markers) > 0 {
		logger.Debug("Checking markers",
			zap.Strings("filter_markers", filter.Markers),
			zap.String("item_marker", item.Marker))
		for _, marker := range filter.Markers {
			if item.Marker == marker {
				logger.Debug("Marker matched!", zap.String("marker", marker))
				return true
			}
		}
	}

	// Check depth
	if filter.Depth != nil {
		logger.Debug("Checking depth",
			zap.Int("filter_depth", *filter.Depth),
			zap.Int("item_depth", item.Depth))
		if item.Depth == *filter.Depth {
			logger.Debug("Depth matched!")
			return true
		}
	}

	// Check patterns (regex)
	if len(filter.Patterns) > 0 {
		for _, pattern := range filter.Patterns {
			matched, err := regexp.MatchString(pattern, item.URL)
			if err == nil && matched {
				logger.Debug("Pattern matched!", zap.String("pattern", pattern))
				return true
			}
		}
	}

	logger.Debug("No match for phase", zap.String("phase_id", phase.ID))
	return false
}

// handlePhaseTransition handles transitioning discovered URLs to the next phase
func (e *Executor) handlePhaseTransition(ctx context.Context, workflow *models.Workflow, currentPhase *models.WorkflowPhase, item *models.URLQueueItem, executionID string, execCtx *models.ExecutionContext) error {
	transition := currentPhase.Transition

	// Check transition condition
	shouldTransition := false
	switch transition.Condition {
	case "all_nodes_complete":
		// Always transition after completing all nodes in current URL
		shouldTransition = true
	case "url_count":
		// Check if we've processed enough URLs
		if _, ok := transition.Params["threshold"].(int); ok {
			// This would need tracking - simplified for now
			shouldTransition = true
		}
	default:
		logger.Warn("Unknown transition condition", zap.String("condition", transition.Condition))
	}

	if !shouldTransition {
		return nil
	}

	// If there's a next phase, mark discovered URLs for that phase
	if transition.NextPhase != "" {
		// Get discovered URLs from exec context
		discoveredData, ok := execCtx.Get("discovered_urls")
		if ok {
			if discoveredURLs, ok := discoveredData.([]string); ok && len(discoveredURLs) > 0 {
				// These URLs were discovered in this phase, assign to next phase
				logger.Info("Transitioning URLs to next phase",
					zap.Int("url_count", len(discoveredURLs)),
					zap.String("next_phase", transition.NextPhase))
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
			// Store in context for later retrieval (e.g., when saving extracted data)
			execCtx.Set("_node_exec_id", nodeExecID)
		}
	}

	interactionEngine := browser.NewInteractionEngine(browserCtx)
	extractionEngine := extraction.NewExtractionEngine(browserCtx.Page)

	var result interface{}
	var err error

	// TRY REGISTRY FIRST - New plugin-based execution
	if e.registry != nil && e.registry.IsRegistered(node.Type) {
		executor, regErr := e.registry.Get(node.Type)
		if regErr == nil {
			// Validate node parameters
			if validErr := executor.Validate(node.Params); validErr != nil {
				err = fmt.Errorf("node validation failed: %w", validErr)
			} else {
				// Prepare input for plugin execution
				input := &nodes.ExecutionInput{
					BrowserContext:   browserCtx,
					ExecutionContext: execCtx,
					Params:           node.Params,
					URLItem:          item,
					ExecutionID:      executionID,
				}

				// Execute using plugin
				output, execErr := executor.Execute(ctx, input)
				if execErr != nil {
					err = execErr
				} else {
					result = output.Result

					// Handle discovered URLs
					if len(output.DiscoveredURLs) > 0 {
						if enqErr := e.enqueueLinks(ctx, executionID, item, output.DiscoveredURLs, node.Params, node.ID, nodeExecID); enqErr != nil {
							logger.Error("Failed to enqueue links", zap.Error(enqErr))
						} else if e.nodeExecRepo != nil && nodeExecID != "" {
							// Update node execution with URLs discovered count
							if nodeExec, getErr := e.nodeExecRepo.GetByID(ctx, nodeExecID); getErr == nil {
								nodeExec.URLsDiscovered = len(output.DiscoveredURLs)
								e.nodeExecRepo.Update(ctx, nodeExec)
							}
						}
					}
				}
			}
		} else {
			logger.Warn("Failed to get executor from registry", zap.String("type", string(node.Type)), zap.Error(regErr))
		}
	} else {
		// FALLBACK TO LEGACY SWITCH - For backward compatibility and special nodes
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
	}

	// Store extracted data directly in context (only for non-extraction nodes)
	// Extraction nodes handle their own context storage
	if node.Type != models.NodeTypeExtract {
		if resultMap, ok := result.(map[string]interface{}); ok {
			for k, v := range resultMap {
				execCtx.Set(k, v)
			}
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

	// Get marker from params (for phase-based routing)
	marker := getStringParam(params, "marker")

	// For backward compatibility, also check url_type
	if marker == "" {
		marker = getStringParam(params, "url_type")
	}

	logger.Debug("Enqueuing links with marker",
		zap.String("marker", marker),
		zap.Int("link_count", len(links)),
		zap.String("node_id", nodeID))

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
			ParentURLID:      &parentItem.ID,
			DiscoveredByNode: &nodeID,
			URLType:          marker, // For backward compatibility
			Marker:           marker, // NEW: Set marker for phase matching
			PhaseID:          "",     // Will be set by phase transition logic
		}

		logger.Debug("Enqueued URL",
			zap.String("url", absoluteURL),
			zap.String("marker", marker),
			zap.Int("depth", parentItem.Depth+1))

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

// saveExtractedData saves extracted data to storage
func (e *Executor) saveExtractedData(ctx context.Context, executionID, urlID, schemaName string, nodeExecID *string, result interface{}) error {
	// Marshal entire result to JSON
	dataJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal extracted data: %w", err)
	}

	item := &models.ExtractedItem{
		ID:              uuid.New().String(),
		ExecutionID:     executionID,
		URLID:           urlID,
		NodeExecutionID: nodeExecID,
		SchemaName:      &schemaName,
		Data:            string(dataJSON),
		ExtractedAt:     time.Now(),
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
