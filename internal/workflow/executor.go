package workflow

import (
	"context"
	"encoding/json"
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

	// Execute data extraction nodes only if URL matches extraction patterns
	if len(workflow.Config.DataExtraction) > 0 {
		shouldExtract := e.shouldExtractData(item.URL, workflow.Config.DataExtractionPatterns)

		if shouldExtract {
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
		} else {
			logger.Debug("Skipping data extraction - URL doesn't match extraction patterns", zap.String("url", item.URL))
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
		links, linkErr := extractionEngine.ExtractLinks(selector)
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

// shouldExtractData determines if data extraction should run for this URL
func (e *Executor) shouldExtractData(url string, patterns []string) bool {
	// If no patterns specified, extract from all URLs (backward compatible)
	if len(patterns) == 0 {
		return true
	}

	// Check if URL matches any of the extraction patterns
	for _, pattern := range patterns {
		if matchesPattern(url, pattern) {
			return true
		}
	}

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
