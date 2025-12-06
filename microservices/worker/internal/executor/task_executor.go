package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/database"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/shared/queue"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/browser"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/dedup"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/driver"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/nodes"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/recovery"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/reporter"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/storage"
	"go.uber.org/zap"
)

// TaskExecutor handles execution of individual tasks
type TaskExecutor struct {
	drivers              map[string]driver.Driver
	defaultDriver        driver.Driver
	driverFactory        *driver.Factory                   // Factory for creating profile-based drivers
	browserConfig        *config.BrowserConfig             // Browser config for factory
	orchestratorURL      string                            // URL for fetching profiles
	profileCache         map[string]*models.BrowserProfile // Cache of loaded profiles
	nodeRegistry         *nodes.Registry
	pubsubClient         *queue.PubSubClient
	gcsClient            *storage.GCSClient
	itemWriter           storage.Writer                 // Primary storage: COPY protocol for max throughput
	deduplicator         dedup.Deduplicator             // URL deduplication (interface)
	batchedStatsReporter *reporter.BatchedStatsReporter // High-throughput batched stats
	retryConfig          RetryConfig                    // Retry configuration for transient failures
	recoveryManager      *recovery.RecoveryManager      // AI-powered error recovery
}

// NewTaskExecutor creates a new task executor
func NewTaskExecutor(
	cfg *config.BrowserConfig,
	gcpCfg *config.GCPConfig,
	pubsubClient *queue.PubSubClient,
	redisCache *cache.Cache,
	orchestratorURL string,
	db *database.DB,
) (*TaskExecutor, error) {
	// Initialize drivers registry
	drivers := make(map[string]driver.Driver)

	// 1. Playwright Driver
	pwDriver, err := driver.NewPlaywrightDriver(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create playwright driver: %w", err)
	}
	drivers[pwDriver.Name()] = pwDriver

	// 2. HTTP Driver
	httpDriver := driver.NewHttpDriver()
	drivers[httpDriver.Name()] = httpDriver

	// 3. Chromedp Driver
	chromedpDriver := driver.NewChromedpDriver(cfg)
	drivers[chromedpDriver.Name()] = chromedpDriver

	// Determine default driver based on config
	// Default to Playwright if not specified or not found
	defaultDriverName := cfg.Driver
	if defaultDriverName == "" {
		defaultDriverName = "playwright"
	}

	defaultDriver, ok := drivers[defaultDriverName]
	if !ok {
		logger.Warn("Configured default driver not found, falling back to playwright",
			zap.String("configured", defaultDriverName),
		)
		defaultDriver = pwDriver
	}

	// Initialize node registry
	nodeRegistry := nodes.NewRegistry()

	// Initialize GCS client only if enabled
	var gcsClient *storage.GCSClient
	if gcpCfg.StorageEnabled {
		ctx := context.Background()
		client, err := storage.NewGCSClient(ctx, gcpCfg)
		if err != nil {
			logger.Warn("Cloud Storage client not available", zap.Error(err))
			gcsClient = nil // Continue without GCS
		} else {
			gcsClient = client
		}
	} else {
		logger.Info("Cloud Storage disabled for local development")
		gcsClient = nil
	}

	// Initialize deduplicator (Redis-based for distributed safety)
	// Uses SETNX for atomic check-and-set
	deduplicator := dedup.NewURLDeduplicator(redisCache)

	// Initialize COPY writer for DB (primary storage)
	// Uses PostgreSQL COPY protocol for 10x faster inserts than batch INSERT
	var itemWriter storage.Writer
	if db != nil {
		copyConfig := storage.DefaultCopyWriterConfig()
		itemWriter = storage.NewCopyWriter(db, copyConfig)
	}

	// Initialize batched stats reporter (high-throughput: aggregates locally, flushes periodically)
	batchedStatsReporter := reporter.NewBatchedStatsReporter(orchestratorURL, redisCache)

	// Initialize recovery manager for smart error recovery
	var recoveryManager *recovery.RecoveryManager
	if db != nil && redisCache != nil {
		recoveryConfig := recovery.DefaultManagerConfig()
		rm, err := recovery.NewRecoveryManager(db.Pool, redisCache, pubsubClient, recoveryConfig)
		if err != nil {
			logger.Warn("Failed to initialize recovery manager", zap.Error(err))
		} else {
			recoveryManager = rm
		}
	}

	logger.Info("Task executor initialized",
		zap.Int("registered_nodes", len(nodeRegistry.List())),
		zap.Bool("gcs_archive_enabled", gcsClient != nil),
		zap.Bool("item_writer_enabled", itemWriter != nil),
		zap.Bool("recovery_enabled", recoveryManager != nil),
		zap.String("dedup_type", "redis_distributed"),
		zap.String("writer_type", "copy_protocol"),
		zap.String("default_driver", defaultDriver.Name()),
	)

	// Create driver factory for profile-based driver creation
	driverFactory := driver.NewFactory(cfg)

	return &TaskExecutor{
		drivers:              drivers,
		defaultDriver:        defaultDriver,
		driverFactory:        driverFactory,
		browserConfig:        cfg,
		orchestratorURL:      orchestratorURL,
		profileCache:         make(map[string]*models.BrowserProfile),
		nodeRegistry:         nodeRegistry,
		pubsubClient:         pubsubClient,
		gcsClient:            gcsClient,
		itemWriter:           itemWriter,
		deduplicator:         deduplicator,
		batchedStatsReporter: batchedStatsReporter,
		retryConfig:          DefaultRetryConfig(),
		recoveryManager:      recoveryManager,
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

	// Prepare context with proxy if needed
	execCtx := ctx
	if task.ProxyURL != "" {
		proxyConfig := &browser.ProxyConfig{
			Server: task.ProxyURL,
		}
		// Pass proxy config via context to driver
		execCtx = context.WithValue(ctx, driver.ProxyKey, proxyConfig)
		logger.Info("Using proxy for retry",
			zap.String("task_id", task.TaskID),
			zap.String("proxy_id", task.ProxyID),
		)
	}

	// If first node uses HTTP driver, set JA3 config from node or workflow defaults
	if firstNodeDriver := e.getFirstNodeDriver(task); firstNodeDriver == "http" {
		// Priority: node-level browser_name > workflow-level default_browser_name > "chrome"
		browserName := e.getFirstNodeBrowserName(task)
		if browserName == "" && task.WorkflowConfig != nil {
			browserName = task.WorkflowConfig.DefaultBrowserName
		}
		if browserName == "" {
			browserName = "chrome" // Ultimate default
		}
		execCtx = context.WithValue(execCtx, driver.JA3Key, &driver.JA3Config{
			BrowserName: browserName,
		})
		logger.Debug("Setting browser_name for HTTP driver",
			zap.String("browser_name", browserName),
			zap.String("source", func() string {
				if e.getFirstNodeBrowserName(task) != "" {
					return "node"
				} else if task.WorkflowConfig != nil && task.WorkflowConfig.DefaultBrowserName != "" {
					return "workflow"
				}
				return "default"
			}()),
		)
	}

	// Get driver for this task (profile-based or default)
	taskDriver, shouldClose, err := e.getDriverForTask(task)
	if err != nil {
		taskStats.Record(0, 0, 1)
		e.reportStats(ctx, task.ExecutionID, taskStats)
		return fmt.Errorf("failed to get driver for task: %w", err)
	}

	// Create new page via driver
	page, err := taskDriver.NewPage(execCtx)
	if err != nil {
		taskStats.Record(0, 0, 1)
		e.reportStats(ctx, task.ExecutionID, taskStats)
		// Close profile-specific driver if it was created
		if shouldClose {
			taskDriver.Close()
		}
		return fmt.Errorf("failed to create page: %w", err)
	}
	// Track the current page to ensure the final page is closed
	// (SwitchDriver might change the page instance)
	currentPage := page
	defer func() {
		if currentPage != nil {
			currentPage.Close()
		}
		// Close profile-specific driver if it was created for this task
		if shouldClose {
			taskDriver.Close()
		}
	}()

	// Execute workflow nodes for this phase
	finalPage, result, err := e.executePhase(ctx, task, page)
	if finalPage != nil {
		currentPage = finalPage
	}
	if err != nil {
		logger.Error("Phase execution failed",
			zap.String("task_id", task.TaskID),
			zap.Error(err),
		)
		taskStats.Record(0, 0, 1)
		e.reportStats(ctx, task.ExecutionID, taskStats)

		// Try smart recovery (if enabled and thresholds met)
		if e.recoveryManager != nil {
			plan, recoverErr := e.recoveryManager.TryRecover(ctx, task.TaskID, task.ExecutionID, task.URL, err, "")
			if recoverErr != nil {
				logger.Warn("Recovery attempt failed", zap.Error(recoverErr))
			} else if plan != nil {
				logger.Info("Recovery plan generated",
					zap.String("action", string(plan.Action)),
					zap.String("source", plan.Source),
					zap.String("reason", plan.Reason),
				)

				// Execute the recovery plan
				if execErr := e.recoveryManager.ExecutePlan(ctx, plan, task.URL); execErr != nil {
					logger.Warn("Failed to execute recovery plan", zap.Error(execErr))
				}

				// Handle based on recovery action
				if plan.Action == recovery.ActionSendToDLQ {
					// Recovery determined this is a permanent failure
					if e.pubsubClient != nil {
						if dlqErr := e.pubsubClient.PublishToDLQ(ctx, task, plan.Reason); dlqErr != nil {
							logger.Error("Failed to publish to DLQ",
								zap.String("task_id", task.TaskID),
								zap.Error(dlqErr),
							)
						}
					}
					e.recoveryManager.ClearHistory(task.TaskID)
					return fmt.Errorf("recovery sent to DLQ: %s", plan.Reason)
				}

				if plan.ShouldRetry {
					// Apply retry delay if specified
					if plan.RetryDelay > 0 {
						time.Sleep(plan.RetryDelay)
					}

					// If proxy was switched, update task with new proxy info
					if plan.Action == recovery.ActionSwitchProxy {
						if proxyURL, ok := plan.Params["proxy_url"].(string); ok {
							task.ProxyURL = proxyURL
						}
						if proxyID, ok := plan.Params["proxy_id"].(string); ok {
							task.ProxyID = proxyID
						}
						logger.Info("Retrying with new proxy",
							zap.String("task_id", task.TaskID),
							zap.String("proxy_id", task.ProxyID),
						)
					}

					// Republish task with updated info (proxy, retry count)
					task.RetryCount++
					if e.pubsubClient != nil {
						if pubErr := e.pubsubClient.PublishTask(ctx, task); pubErr != nil {
							logger.Error("Failed to republish task for retry", zap.Error(pubErr))
							return fmt.Errorf("recovery action %s: %s", plan.Action, plan.Reason)
						}
						// Successfully republished - return nil to ack original message
						return nil
					}

					// No pubsub client - return error for manual retry
					return fmt.Errorf("recovery action %s: %s", plan.Action, plan.Reason)
				}
			}
		}

		// Record proxy failure if proxy was used
		if task.ProxyID != "" && e.recoveryManager != nil {
			domain := extractDomain(task.URL)
			if err := e.recoveryManager.RecordProxyFailure(ctx, task.ProxyID, domain, recovery.PatternUnknown); err != nil {
				logger.Warn("Failed to record proxy failure", zap.Error(err))
			}
		}

		// Fallback: Send to Dead Letter Queue if max retries exceeded
		// This captures permanently failed tasks for analysis/debugging
		if task.RetryCount >= e.retryConfig.MaxRetries && e.pubsubClient != nil {
			// Create incident report for human investigation
			if e.recoveryManager != nil {
				incident, incErr := e.recoveryManager.CreateIncident(
					ctx,
					task.TaskID, task.ExecutionID, task.WorkflowID, task.URL,
					nil, // detected error (not available here)
					"",  // AI reasoning
					"All automated recovery attempts exhausted", // AI failure reason
					nil, // page snapshot (not captured here)
				)
				if incErr != nil {
					logger.Warn("Failed to create incident", zap.Error(incErr))
				} else if incident != nil {
					logger.Info("Incident created for human investigation",
						zap.String("incident_id", incident.ID),
						zap.String("task_id", task.TaskID),
						zap.String("priority", string(incident.Priority)),
					)
				}
			}

			if dlqErr := e.pubsubClient.PublishToDLQ(ctx, task, err.Error()); dlqErr != nil {
				logger.Error("Failed to publish to DLQ",
					zap.String("task_id", task.TaskID),
					zap.Error(dlqErr),
				)
			}
		}

		return fmt.Errorf("phase execution failed: %w", err)
	}

	// Record success for error rate tracking
	if e.recoveryManager != nil {
		e.recoveryManager.RecordSuccess(ctx, task.URL)
		e.recoveryManager.ClearHistory(task.TaskID)

		// Record proxy success if proxy was used
		if task.ProxyID != "" {
			domain := extractDomain(task.URL)
			if err := e.recoveryManager.RecordProxySuccess(ctx, task.ProxyID, domain); err != nil {
				logger.Warn("Failed to record proxy success", zap.Error(err))
			}
		}
	}

	// Save extracted items: DB (primary) + optional GCS (archive)
	if len(result.ExtractedItems) > 0 {
		// Primary: Write to database using COPY protocol (high throughput)
		if e.itemWriter != nil {
			items := make([]storage.ExtractedItem, 0, len(result.ExtractedItems))
			for _, data := range result.ExtractedItems {
				items = append(items, storage.ExtractedItem{
					ExecutionID: task.ExecutionID,
					WorkflowID:  task.WorkflowID,
					TaskID:      task.TaskID,
					URL:         task.URL,
					Data:        data,
				})
			}
			if err := e.itemWriter.AddBatch(ctx, items); err != nil {
				logger.Error("Failed to add items to writer",
					zap.String("task_id", task.TaskID),
					zap.Error(err),
				)
			}
		}

		// Archive: Async upload to GCS (if enabled)
		if e.gcsClient != nil {
			go func() {
				gcsCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				gcsPath, err := e.gcsClient.UploadExtractedItems(gcsCtx, task.ExecutionID, result.ExtractedItems)
				if err != nil {
					logger.Warn("Failed to archive to GCS (non-critical)",
						zap.String("task_id", task.TaskID),
						zap.Error(err),
					)
				} else {
					logger.Debug("Archived to GCS", zap.String("path", gcsPath))
				}
			}()
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
func (e *TaskExecutor) executePhase(ctx context.Context, task *models.Task, page driver.Page) (driver.Page, *TaskResult, error) {
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

	// Track profile-based drivers created during execution for cleanup
	var profileDriversToClose []driver.Driver
	defer func() {
		for _, d := range profileDriversToClose {
			d.Close()
		}
	}()

	// Implement SwitchDriver callback
	execCtx.SwitchDriver = func(targetType string) error {
		var currentType string
		switch execCtx.Page.(type) {
		case *driver.PlaywrightPage:
			currentType = "playwright"
		case *driver.HttpPage:
			currentType = "http"
		default:
			return fmt.Errorf("unknown page type")
		}

		if currentType == targetType {
			return nil // Already on target driver
		}

		logger.Info("Switching driver",
			zap.String("from", currentType),
			zap.String("to", targetType),
			zap.String("task_id", task.TaskID),
		)

		// Get cookies from current page
		cookies, err := execCtx.Page.GetCookies()
		if err != nil {
			return fmt.Errorf("failed to get cookies for driver switch: %w", err)
		}

		// Close current page
		if err := execCtx.Page.Close(); err != nil {
			logger.Warn("Failed to close page during switch", zap.Error(err))
		}

		// Create new page
		var newPage driver.Page
		var createErr error

		// Reuse the same context (with proxy if any)
		// Note: We need to ensure the context passed to NewPage has the proxy info if needed.
		// executePhase receives 'ctx' which might not have the proxy info if it was added in Execute.
		// Wait, Execute creates 'execCtx' (context.Context) with proxy and passes it to NewPage.
		// But executePhase receives 'ctx' (context.Context) and 'page'.
		// We need the context that has the proxy info.
		// In Execute: execCtx = context.WithValue(ctx, driver.ProxyKey, proxyConfig)
		// But executePhase receives 'ctx' which is the *original* context (or the one passed to Execute).
		// Actually Execute passes 'ctx' to executePhase.
		// Let's check Execute again.
		// In Execute:
		// execCtx := ctx
		// if task.ProxyURL != "" { ... execCtx = context.WithValue(...) }
		// page, err := e.driver.NewPage(execCtx)
		// result, err := e.executePhase(ctx, task, page)
		// Ah, it passes 'ctx' (original) to executePhase, not 'execCtx' (with proxy).
		// This is a bug if we want to reuse proxy in new driver.
		// However, the 'page' already has the proxy context if it was created with it.
		// But when creating a NEW page, we need that context.
		// I should update Execute to pass the correct context or reconstruct it.
		// For now, I'll assume we can use 'ctx' but we might lose proxy if it was added in Execute.
		// I'll fix Execute to pass the correct context in a separate step if needed.
		// Or I can reconstruct it from task.ProxyURL.

		pageCtx := ctx
		if task.ProxyURL != "" {
			proxyConfig := &browser.ProxyConfig{
				Server: task.ProxyURL,
			}
			pageCtx = context.WithValue(ctx, driver.ProxyKey, proxyConfig)
		}

		// Look up target driver in registry
		targetDriverInstance, ok := e.drivers[targetType]
		if !ok {
			return fmt.Errorf("unsupported driver type: %s", targetType)
		}

		newPage, createErr = targetDriverInstance.NewPage(pageCtx)

		if createErr != nil {
			return fmt.Errorf("failed to create new page for driver switch: %w", createErr)
		}

		// Set cookies on new page
		if err := newPage.SetCookies(cookies); err != nil {
			newPage.Close()
			return fmt.Errorf("failed to set cookies on new page: %w", err)
		}

		// Update execution context
		execCtx.Page = newPage
		return nil
	}

	// Implement SwitchDriverWithProfile callback for per-node profile support
	// Uses embedded profiles from task metadata (ZERO API calls)
	execCtx.SwitchDriverWithProfile = func(targetType, profileID string) error {
		logger.Info("Switching driver with profile",
			zap.String("target_driver", targetType),
			zap.String("profile_id", profileID),
			zap.String("task_id", task.TaskID),
		)

		// Get embedded profiles from metadata
		var profile *models.BrowserProfile
		if nodeProfiles, ok := task.Metadata["node_profiles"].(map[string]interface{}); ok {
			if profileData, exists := nodeProfiles[profileID]; exists {
				// Convert from interface{} to BrowserProfile via JSON
				profileBytes, err := json.Marshal(profileData)
				if err == nil {
					var p models.BrowserProfile
					if err := json.Unmarshal(profileBytes, &p); err == nil {
						profile = &p
					}
				}
			}
		}

		if profile == nil {
			logger.Warn("Profile not found in embedded metadata, using default driver",
				zap.String("profile_id", profileID),
			)
			// Fall back to legacy driver switch
			return execCtx.SwitchDriver(targetType)
		}

		// Validate driver type matches profile
		if profile.DriverType != targetType {
			logger.Warn("Profile driver type doesn't match requested driver, using default",
				zap.String("profile_driver", profile.DriverType),
				zap.String("requested_driver", targetType),
			)
			return execCtx.SwitchDriver(targetType)
		}

		// Close current page
		if err := execCtx.Page.Close(); err != nil {
			logger.Warn("Failed to close page during profile switch", zap.Error(err))
		}

		// Create driver from profile (no API call - profile is already in memory)
		profileDriver, err := e.driverFactory.CreateDriverFromProfile(profile)
		if err != nil {
			logger.Warn("Failed to create profile driver, using default",
				zap.String("profile_id", profileID),
				zap.Error(err),
			)
			return execCtx.SwitchDriver(targetType)
		}

		// Create new page from profile driver
		pageCtx := ctx
		if task.ProxyURL != "" {
			proxyConfig := &browser.ProxyConfig{Server: task.ProxyURL}
			pageCtx = context.WithValue(ctx, driver.ProxyKey, proxyConfig)
		}

		newPage, err := profileDriver.NewPage(pageCtx)
		if err != nil {
			profileDriver.Close()
			return fmt.Errorf("failed to create page from profile driver: %w", err)
		}

		execCtx.Page = newPage
		profileDriversToClose = append(profileDriversToClose, profileDriver) // Track for cleanup
		logger.Info("Switched to profile-based driver",
			zap.String("profile_id", profileID),
			zap.String("driver_type", targetType),
			zap.String("browser_type", profile.BrowserType),
		)
		return nil
	}

	// Implement SwitchDriverWithBrowser callback for HTTP driver with browser_name
	// Used for JA3 fingerprint + matching user agent generation
	execCtx.SwitchDriverWithBrowser = func(targetType, browserName string) error {
		if targetType != "http" {
			// Only HTTP driver uses this - others should use profile
			return execCtx.SwitchDriver(targetType)
		}

		logger.Info("Switching to HTTP driver with browser",
			zap.String("browser_name", browserName),
			zap.String("task_id", task.TaskID),
		)

		// Close current page
		if err := execCtx.Page.Close(); err != nil {
			logger.Warn("Failed to close page during HTTP driver switch", zap.Error(err))
		}

		// Create context with JA3 config for the specified browser
		pageCtx := ctx
		ja3Config := &driver.JA3Config{BrowserName: browserName}
		pageCtx = context.WithValue(pageCtx, driver.JA3Key, ja3Config)

		if task.ProxyURL != "" {
			proxyConfig := &browser.ProxyConfig{Server: task.ProxyURL}
			pageCtx = context.WithValue(pageCtx, driver.ProxyKey, proxyConfig)
		}

		// Get HTTP driver from registry
		httpDriver, ok := e.drivers["http"]
		if !ok {
			return fmt.Errorf("HTTP driver not available")
		}

		// Create new page with JA3 config
		newPage, err := httpDriver.NewPage(pageCtx)
		if err != nil {
			return fmt.Errorf("failed to create HTTP page: %w", err)
		}

		execCtx.Page = newPage
		logger.Info("Switched to HTTP driver with browser-specific JA3 and user agent",
			zap.String("browser_name", browserName),
		)
		return nil
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

		// Execute node with retry for transient failures
		err = WithRetry(func() error {
			return executor.Execute(ctx, execCtx, node)
		}, e.retryConfig)

		if err != nil {
			logger.Error("Node execution failed after retries",
				zap.String("node_id", node.ID),
				zap.String("node_type", node.Type),
				zap.Error(err),
			)
			result.Errors = append(result.Errors, err)

			// Continue with other nodes (non-fatal)
			continue
		}

		// Execute any branch nodes from conditional execution
		for _, branchNode := range execCtx.BranchNodes {
			branchExecutor, err := e.nodeRegistry.Get(branchNode.Type)
			if err != nil {
				logger.Warn("No executor for branch node", zap.String("type", branchNode.Type))
				continue
			}
			if err := branchExecutor.Execute(ctx, execCtx, branchNode); err != nil {
				logger.Warn("Branch node execution failed", zap.Error(err))
			}
		}
		execCtx.BranchNodes = nil // Clear after execution
	}

	// Extract results from execution context
	// From Variables (extract nodes)
	if items, ok := execCtx.Variables["extracted_items"].([]map[string]interface{}); ok {
		result.ExtractedItems = items
	}
	// From ExecutionContext fields (screenshot, paginate nodes)
	if len(execCtx.ExtractedItems) > 0 {
		result.ExtractedItems = append(result.ExtractedItems, execCtx.ExtractedItems...)
	}

	// Handle discovered_urls (can be []string or []map[string]interface{})
	// Prefer Variables["discovered_urls"] which may contain markers
	if discoveredURLs, ok := execCtx.Variables["discovered_urls"]; ok {
		result.DiscoveredURLs = discoveredURLs
	} else if len(execCtx.DiscoveredURLs) > 0 {
		// Fallback to ExecutionContext.DiscoveredURLs (plain []string) only if Variables not set
		result.DiscoveredURLs = execCtx.DiscoveredURLs
	}

	return execCtx.Page, result, nil
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
			TaskID:           fmt.Sprintf("%s-%d", task.TaskID, len(tasks)),
			ExecutionID:      task.ExecutionID,
			WorkflowID:       task.WorkflowID,
			URL:              urlData.URL,
			Depth:            task.Depth + 1,
			ParentURLID:      &task.TaskID,
			Marker:           urlData.Marker, // Propagate marker
			PhaseID:          nextPhase.ID,
			PhaseConfig:      nextPhase,
			WorkflowConfig:   task.WorkflowConfig, // Propagate workflow config to child tasks
			Metadata:         task.Metadata,
			RetryCount:       0,
			BrowserProfileID: task.BrowserProfileID, // Propagate browser profile to child tasks
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

// reportStats reports task statistics to orchestrator using batched reporter
// Stats are aggregated locally and flushed periodically (no HTTP call per task)
func (e *TaskExecutor) reportStats(ctx context.Context, executionID string, taskStats *reporter.TaskStats) {
	if e.batchedStatsReporter == nil {
		return
	}

	// Record stats locally (atomic, no network call)
	// Batched reporter will flush to orchestrator every 5 seconds
	e.batchedStatsReporter.Record(executionID, *taskStats)
}

// Close cleans up resources
func (e *TaskExecutor) Close() error {
	// Flush batched stats reporter first (ensures all stats are sent)
	if e.batchedStatsReporter != nil {
		if err := e.batchedStatsReporter.Close(); err != nil {
			logger.Error("Failed to close batched stats reporter", zap.Error(err))
		}
	}

	// Flush item writer (ensures all data is written)
	if e.itemWriter != nil {
		if err := e.itemWriter.Close(); err != nil {
			logger.Error("Failed to close item writer", zap.Error(err))
		}
	}

	if e.gcsClient != nil {
		e.gcsClient.Close()
	}

	if e.recoveryManager != nil {
		e.recoveryManager.Close()
	}

	for name, d := range e.drivers {
		if err := d.Close(); err != nil {
			logger.Error("Failed to close driver",
				zap.String("driver", name),
				zap.Error(err),
			)
		}
	}
	return nil
}

// extractDomain extracts the domain from a URL
func extractDomain(url string) string {
	// Simple extraction - could use net/url for more robust parsing
	if len(url) > 8 {
		start := 0
		if url[:8] == "https://" {
			start = 8
		} else if url[:7] == "http://" {
			start = 7
		}

		url = url[start:]
		end := len(url)
		for i, c := range url {
			if c == '/' || c == '?' || c == ':' {
				end = i
				break
			}
		}
		return url[:end]
	}
	return url
}

// getDriverForTask returns the appropriate driver for the task
// Returns (driver, shouldClose, error) where shouldClose indicates if the driver should be closed after use
func (e *TaskExecutor) getDriverForTask(task *models.Task) (driver.Driver, bool, error) {
	// Check if first node explicitly specifies a driver override
	// This takes priority over workflow-level browser_profile_id
	if firstNodeDriver := e.getFirstNodeDriver(task); firstNodeDriver != "" {
		logger.Info("First node specifies driver override, using direct driver",
			zap.String("task_id", task.TaskID),
			zap.String("phase_id", task.PhaseID),
			zap.String("driver", firstNodeDriver),
		)

		switch firstNodeDriver {
		case "http":
			// Use registered HTTP driver or create on demand
			if httpDriver, ok := e.drivers["http"]; ok && httpDriver != nil {
				return httpDriver, false, nil
			}
			return driver.NewHttpDriver(), false, nil

		case "chromedp":
			// Create chromedp driver on demand (doesn't use profile, faster startup)
			chromedpDriver := driver.NewChromedpDriver(e.browserConfig)
			return chromedpDriver, true, nil // shouldClose=true to clean up after task

		case "playwright":
			// Create playwright driver on demand
			playwrightDriver, err := driver.NewPlaywrightDriver(e.browserConfig)
			if err != nil {
				logger.Warn("Failed to create playwright driver, falling back to default",
					zap.Error(err),
				)
				return e.defaultDriver, false, nil
			}
			return playwrightDriver, true, nil // shouldClose=true to clean up after task
		}
	}

	// If no browser profile ID, use default driver
	if task.BrowserProfileID == nil || *task.BrowserProfileID == "" {
		return e.defaultDriver, false, nil
	}

	profileID := *task.BrowserProfileID

	// Fetch or use cached profile
	profile, err := e.fetchBrowserProfile(profileID)
	if err != nil {
		logger.Warn("Failed to fetch browser profile, using default driver",
			zap.String("profile_id", profileID),
			zap.Error(err),
		)
		return e.defaultDriver, false, nil
	}

	// Log profile usage
	logger.Info("Using browser profile for task",
		zap.String("task_id", task.TaskID),
		zap.String("profile_id", profileID),
		zap.String("driver_type", profile.DriverType),
		zap.String("browser_type", profile.BrowserType),
	)

	// Create driver from profile
	profileDriver, err := e.driverFactory.CreateDriverFromProfile(profile)
	if err != nil {
		logger.Warn("Failed to create profile driver, using default",
			zap.String("profile_id", profileID),
			zap.Error(err),
		)
		return e.defaultDriver, false, nil
	}

	// Profile-based drivers should be closed after single task (new browser per task)
	return profileDriver, true, nil
}

// getFirstNodeDriver returns the driver type specified in the first node of the phase, or empty if not specified
func (e *TaskExecutor) getFirstNodeDriver(task *models.Task) string {
	// Check first node in the phase config
	if len(task.PhaseConfig.Nodes) > 0 {
		firstNode := task.PhaseConfig.Nodes[0]
		if driverType, ok := firstNode.Params["driver"].(string); ok && driverType != "" && driverType != "default" {
			return driverType
		}
	}
	return ""
}

// getFirstNodeBrowserName returns the browser_name specified in the first node of the phase
func (e *TaskExecutor) getFirstNodeBrowserName(task *models.Task) string {
	if len(task.PhaseConfig.Nodes) > 0 {
		firstNode := task.PhaseConfig.Nodes[0]
		if browserName, ok := firstNode.Params["browser_name"].(string); ok && browserName != "" {
			return browserName
		}
	}
	return ""
}

// fetchBrowserProfile fetches a browser profile from orchestrator or cache
func (e *TaskExecutor) fetchBrowserProfile(profileID string) (*models.BrowserProfile, error) {
	// Check cache first
	if profile, ok := e.profileCache[profileID]; ok {
		return profile, nil
	}

	// Fetch from orchestrator
	url := fmt.Sprintf("%s/api/v1/profiles/%s", e.orchestratorURL, profileID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("profile fetch failed with status: %d", resp.StatusCode)
	}

	var profile models.BrowserProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode profile: %w", err)
	}

	// Cache the profile (profiles don't change frequently)
	e.profileCache[profileID] = &profile

	return &profile, nil
}
