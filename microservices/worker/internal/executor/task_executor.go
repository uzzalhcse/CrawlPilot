package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/shared/queue"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/browser"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/dedup"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/nodes"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/reporter"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/storage"
	"go.uber.org/zap"
)

// TaskExecutor handles execution of individual tasks
type TaskExecutor struct {
	browserPool   *browser.Pool
	nodeRegistry  *nodes.Registry
	pubsubClient  *queue.PubSubClient
	gcsClient     *storage.GCSClient
	deduplicator  *dedup.URLDeduplicator
	statsReporter *reporter.StatsReporter
}

// NewTaskExecutor creates a new task executor
func NewTaskExecutor(
	cfg *config.BrowserConfig,
	gcpCfg *config.GCPConfig,
	pubsubClient *queue.PubSubClient,
	redisCache *cache.Cache,
	orchestratorURL string,
) (*TaskExecutor, error) {
	// Initialize browser pool
	browserPool, err := browser.NewPool(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser pool: %w", err)
	}

	// Initialize node registry
	nodeRegistry := nodes.NewRegistry()

	// Initialize GCS client
	ctx := context.Background()
	gcsClient, err := storage.NewGCSClient(ctx, gcpCfg)
	if err != nil {
		logger.Warn("Cloud Storage client not available", zap.Error(err))
		gcsClient = nil // Continue without GCS
	}

	// Initialize deduplicator
	deduplicator := dedup.NewURLDeduplicator(redisCache)

	// Initialize stats reporter
	statsReporter := reporter.NewStatsReporter(orchestratorURL)

	logger.Info("Task executor initialized",
		zap.Int("registered_nodes", len(nodeRegistry.List())),
		zap.Bool("gcs_enabled", gcsClient != nil),
	)

	return &TaskExecutor{
		browserPool:   browserPool,
		nodeRegistry:  nodeRegistry,
		pubsubClient:  pubsubClient,
		gcsClient:     gcsClient,
		deduplicator:  deduplicator,
		statsReporter: statsReporter,
	}, nil
}

// Execute processes a single task
func (e *TaskExecutor) Execute(ctx context.Context, task *models.Task) error {
	logger.Info("Executing task",
		zap.String("task_id", task.TaskID),
		zap.String("execution_id", task.ExecutionID),
		zap.String("url", task.URL),
		zap.String("phase_id", task.PhaseID),
		zap.String("marker", task.Marker),
		zap.Int("depth", task.Depth),
	)

	// Check if task passes URL filter for this phase
	if !e.passesURLFilter(task) {
		logger.Info("Task filtered out by URL filter",
			zap.String("url", task.URL),
			zap.String("phase_id", task.PhaseID),
		)
		return nil // Not an error, just skip
	}

	startTime := time.Now()
	taskStats := reporter.NewTaskStats()

	// Check for duplicate URL
	isDuplicate, err := e.deduplicator.IsDuplicate(ctx, task.ExecutionID, task.PhaseID, task.URL)
	if err != nil {
		logger.Warn("Deduplication check failed", zap.Error(err))
	} else if isDuplicate {
		logger.Info("Skipping duplicate URL",
			zap.String("url", task.URL),
		)
		return nil
	}

	// Acquire browser context
	browserCtx, err := e.browserPool.Acquire(ctx)
	if err != nil {
		taskStats.Record(0, 0, 1)
		e.reportStats(ctx, task.ExecutionID, taskStats)
		return fmt.Errorf("failed to acquire browser context: %w", err)
	}
	defer e.browserPool.Release(browserCtx)

	// Create new page
	page, err := browserCtx.NewPage()
	if err != nil {
		taskStats.Record(0, 0, 1)
		e.reportStats(ctx, task.ExecutionID, taskStats)
		return fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Execute workflow nodes for this phase
	result, err := e.executePhase(ctx, task, page)
	if err != nil {
		logger.Error("Phase execution failed",
			zap.String("task_id", task.TaskID),
			zap.Error(err),
		)
		taskStats.Record(0, 0, 1)
		e.reportStats(ctx, task.ExecutionID, taskStats)
		return fmt.Errorf("phase execution failed: %w", err)
	}

	// Upload extracted items to Cloud Storage
	if len(result.ExtractedItems) > 0 && e.gcsClient != nil {
		gcsPath, err := e.gcsClient.UploadExtractedItems(ctx, task.ExecutionID, result.ExtractedItems)
		if err != nil {
			logger.Error("Failed to upload extracted items",
				zap.String("task_id", task.TaskID),
				zap.Error(err),
			)
		} else {
			logger.Info("Extracted items uploaded",
				zap.String("gcs_path", gcsPath),
			)
		}
	}

	// Process discovered URLs with marker propagation
	uniqueURLs := e.processDiscoveredURLs(ctx, task, result.DiscoveredURLs)

	// Handle discovered URLs - requeue or transition
	if len(uniqueURLs) > 0 {
		if err := e.requeueDiscoveredURLs(ctx, task, uniqueURLs); err != nil {
			logger.Error("Failed to requeue URLs",
				zap.String("task_id", task.TaskID),
				zap.Error(err),
			)
		}
	}

	// Record stats
	taskStats.Record(len(result.ExtractedItems), len(uniqueURLs), len(result.Errors))

	// Report stats to orchestrator
	e.reportStats(ctx, task.ExecutionID, taskStats)

	duration := time.Since(startTime)

	// Get count of discovered URLs (handle both types)
	discoveredCount := 0
	switch v := result.DiscoveredURLs.(type) {
	case []string:
		discoveredCount = len(v)
	case []map[string]interface{}:
		discoveredCount = len(v)
	case []interface{}:
		discoveredCount = len(v)
	}

	logger.Info("Task completed",
		zap.String("task_id", task.TaskID),
		zap.Duration("duration", duration),
		zap.Int("items_extracted", len(result.ExtractedItems)),
		zap.Int("urls_discovered", len(uniqueURLs)),
		zap.Int("total_discovered", discoveredCount),
		zap.Int("duplicates_filtered", discoveredCount-len(uniqueURLs)),
	)

	return nil
}

// TaskResult holds the result of task execution
type TaskResult struct {
	ExtractedItems []map[string]interface{}
	DiscoveredURLs interface{} // Can be []string or []map[string]interface{}
	Errors         []error
}

// executePhase executes all nodes in a phase
func (e *TaskExecutor) executePhase(ctx context.Context, task *models.Task, page playwright.Page) (*TaskResult, error) {
	result := &TaskResult{
		ExtractedItems: make([]map[string]interface{}, 0),
		DiscoveredURLs: make([]string, 0),
		Errors:         make([]error, 0),
	}

	// Get nodes for this phase
	phaseNodes := e.getPhaseNodes(task)

	logger.Info("Executing phase nodes",
		zap.String("phase_id", task.PhaseID),
		zap.Int("node_count", len(phaseNodes)),
	)

	// Create execution context
	execCtx := &nodes.ExecutionContext{
		Page:      page,
		Task:      task,
		Variables: make(map[string]interface{}),
	}

	// Execute each node in sequence
	for i, node := range phaseNodes {
		logger.Debug("Executing node",
			zap.String("node_id", node.ID),
			zap.String("node_type", node.Type),
			zap.Int("node_index", i),
		)

		// Get executor for this node type
		executor, err := e.nodeRegistry.Get(node.Type)
		if err != nil {
			logger.Error("No executor found for node type",
				zap.String("node_type", node.Type),
				zap.Error(err),
			)
			result.Errors = append(result.Errors, err)
			continue
		}

		// Execute node
		if err := executor.Execute(ctx, execCtx, node); err != nil {
			logger.Error("Node execution failed",
				zap.String("node_id", node.ID),
				zap.String("node_type", node.Type),
				zap.Error(err),
			)
			result.Errors = append(result.Errors, err)

			// Continue with other nodes (non-fatal)
			continue
		}
	}

	// Extract results from execution context
	if items, ok := execCtx.Variables["extracted_items"].([]map[string]interface{}); ok {
		result.ExtractedItems = items
	}

	// Handle discovered_urls (can be []string or []map[string]interface{})
	if discoveredURLs, ok := execCtx.Variables["discovered_urls"]; ok {
		result.DiscoveredURLs = discoveredURLs
	}

	return result, nil
}

// getPhaseNodes returns nodes from the current phase
func (e *TaskExecutor) getPhaseNodes(task *models.Task) []models.Node {
	// Nodes are directly in the phase config
	return task.PhaseConfig.Nodes
}

// requeueDiscoveredURLs re-enqueues discovered URLs for processing
func (e *TaskExecutor) requeueDiscoveredURLs(ctx context.Context, task *models.Task, urls []URLWithMarker) error {
	if e.pubsubClient == nil {
		logger.Warn("Pub/Sub client not available, cannot requeue URLs")
		return nil
	}

	// Determine next phase based on transition rules
	nextPhase := e.getNextPhase(task)

	// Check max depth (if specified in workflow config)
	maxDepth := task.Metadata["max_depth"]
	if maxDepth != nil {
		if maxDepthInt, ok := maxDepth.(int); ok {
			if task.Depth+1 > maxDepthInt {
				logger.Info("Max depth reached, not requeuing URLs",
					zap.Int("current_depth", task.Depth),
					zap.Int("max_depth", maxDepthInt),
				)
				return nil
			}
		}
	}

	// Create tasks for discovered URLs
	tasks := make([]*models.Task, 0, len(urls))

	for _, urlData := range urls {
		newTask := &models.Task{
			TaskID:      fmt.Sprintf("%s-%d", task.TaskID, len(tasks)),
			ExecutionID: task.ExecutionID,
			WorkflowID:  task.WorkflowID,
			URL:         urlData.URL,
			Depth:       task.Depth + 1,
			ParentURLID: &task.TaskID,
			Marker:      urlData.Marker, // Propagate marker
			PhaseID:     nextPhase.ID,
			PhaseConfig: nextPhase,
			Metadata:    task.Metadata,
			RetryCount:  0,
		}

		tasks = append(tasks, newTask)
	}

	// Apply rate limiting if specified
	if rateLimitDelay, ok := task.Metadata["rate_limit_delay"].(int); ok && rateLimitDelay > 0 {
		logger.Debug("Applying rate limit", zap.Int("delay_ms", rateLimitDelay))
		time.Sleep(time.Duration(rateLimitDelay) * time.Millisecond)
	}

	// Publish batch
	if err := e.pubsubClient.PublishBatch(ctx, tasks); err != nil {
		return fmt.Errorf("failed to publish discovered URLs: %w", err)
	}

	logger.Info("Discovered URLs requeued",
		zap.Int("count", len(tasks)),
		zap.String("next_phase", nextPhase.ID),
	)

	return nil
}

// getNextPhase determines the next phase based on transition rules
func (e *TaskExecutor) getNextPhase(task *models.Task) models.WorkflowPhase {
	// Check if current phase has a transition
	if task.PhaseConfig.Transition != nil && task.PhaseConfig.Transition.NextPhase != "" {
		// Find the next phase in metadata
		if phasesData, ok := task.Metadata["phases"].([]interface{}); ok {
			for _, phaseData := range phasesData {
				// Try to convert via JSON marshaling/unmarshaling
				phaseBytes, err := json.Marshal(phaseData)
				if err != nil {
					continue
				}

				var phase models.WorkflowPhase
				if err := json.Unmarshal(phaseBytes, &phase); err != nil {
					continue
				}

				if phase.ID == task.PhaseConfig.Transition.NextPhase {
					logger.Info("Phase transition",
						zap.String("from", task.PhaseID),
						zap.String("to", phase.ID),
					)
					return phase
				}
			}
		}

		logger.Warn("Next phase not found, staying in current phase",
			zap.String("current_phase", task.PhaseID),
			zap.String("expected_next", task.PhaseConfig.Transition.NextPhase),
		)
	}

	// No transition or next phase not found, stay in current phase
	return task.PhaseConfig
}

// reportStats reports task statistics to orchestrator
func (e *TaskExecutor) reportStats(ctx context.Context, executionID string, taskStats *reporter.TaskStats) {
	if e.statsReporter == nil {
		return
	}

	stats := taskStats.ToExecutionStats(executionID)
	if err := e.statsReporter.ReportStats(ctx, stats); err != nil {
		logger.Warn("Failed to report stats",
			zap.String("execution_id", executionID),
			zap.Error(err),
		)
	}
}

// Close cleans up resources
func (e *TaskExecutor) Close() error {
	if e.gcsClient != nil {
		e.gcsClient.Close()
	}

	if e.browserPool != nil {
		return e.browserPool.Close()
	}
	return nil
}
