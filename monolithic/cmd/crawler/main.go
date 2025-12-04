package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/uzzalhcse/crawlify/api/handlers"
	"github.com/uzzalhcse/crawlify/internal/ai"
	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/config"
	"github.com/uzzalhcse/crawlify/internal/error_recovery"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/monitoring"
	"github.com/uzzalhcse/crawlify/internal/plugin"
	"github.com/uzzalhcse/crawlify/internal/queue"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/internal/workflow"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(true); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting Crawlify API Server")

	// Initialize database
	db, err := storage.NewPostgresDB(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize repositories
	workflowRepo := storage.NewWorkflowRepository(db)
	workflowVersionRepo := storage.NewWorkflowVersionRepository(db) // NEW
	executionRepo := storage.NewExecutionRepository(db)
	extractedItemsRepo := storage.NewExtractedItemsRepository(db)
	nodeExecRepo := storage.NewNodeExecutionRepository(db)
	urlQueue := queue.NewURLQueue(db)
	monitoringRepo := storage.NewMonitoringRepository(db)
	pluginRepo := storage.NewPluginRepository(db)                 // Plugin marketplace
	browserProfileRepo := storage.NewBrowserProfileRepository(db) // Browser profiles

	// Initialize browser pool
	browserPool, err := browser.NewBrowserPool(&cfg.Browser, browserProfileRepo)
	if err != nil {
		logger.Fatal("Failed to initialize browser pool", zap.Error(err))
	}
	defer browserPool.Close()

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "Crawlify API",
		DisableStartupMessage: false,
		ErrorHandler:          errorHandler,
		ReadTimeout:           time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout:          time.Duration(cfg.Server.WriteTimeout) * time.Second,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,PATCH",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Custom logging middleware
	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		logger.Info("Request",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("duration", duration),
			zap.String("ip", c.IP()),
		)

		return err
	})

	// Initialize handlers (note: executionHandler will be created after aiClient)
	workflowHandler := handlers.NewWorkflowHandler(workflowRepo)
	workflowVersionHandler := handlers.NewWorkflowVersionHandler(workflowVersionRepo, workflowRepo) // NEW
	analyticsHandler := handlers.NewAnalyticsHandler(nodeExecRepo, extractedItemsRepo, urlQueue)
	selectorHandler := handlers.NewSelectorHandler(browserPool)

	// Create a node registry for monitorings
	nodeRegistry := workflow.NewNodeRegistry()
	if err := nodeRegistry.RegisterDefaultNodes(); err != nil {
		logger.Warn("Failed to register default nodes for monitorings", zap.Error(err))
	}

	// Initialize snapshot repository and service
	snapshotRepo := storage.NewSnapshotRepository(db)
	snapshotStoragePath := "./data/snapshots"
	var snapshotLogger *zap.Logger
	snapshotLogger, _ = zap.NewProduction()
	snapshotService := monitoring.NewSnapshotService(snapshotRepo, snapshotStoragePath, snapshotLogger)

	monitoringHandler := handlers.NewMonitoringHandler(workflowRepo, monitoringRepo, browserPool, nodeRegistry, snapshotService)

	// Initialize schedule repository and handler
	scheduleRepo := storage.NewMonitoringScheduleRepository(db)

	// Create monitoring orchestrator for scheduler with snapshot service
	monitoringOrchestrator := monitoring.NewOrchestrator(browserPool, nodeRegistry, nil, snapshotService)

	// Create and start scheduler service
	schedulerService := monitoring.NewSchedulerService(scheduleRepo, monitoringRepo, workflowRepo, monitoringOrchestrator)
	go schedulerService.Start()
	logger.Info("Monitoring scheduler started")

	// Initialize snapshot handler
	snapshotHandler := handlers.NewSnapshotHandler(snapshotService)

	// Initialize AI services for auto-fix with key rotation
	zapLogger, _ := zap.NewProduction()

	// Initialize key repository and manager
	aiKeyRepo := storage.NewAIKeyRepository(db, zapLogger)
	keyManager := ai.NewAIKeyManager(aiKeyRepo, zapLogger)

	// Initialize AI client based on configured provider
	var aiClient interface {
		GenerateText(context.Context, string) (string, error)
		GenerateWithImage(context.Context, string, []byte, string) (string, error)
		Close() error
	}

	if cfg.AI.Provider == "openrouter" {
		aiClient, err = ai.NewOpenRouterClient(keyManager, cfg.AI.OpenRouterModel, zapLogger)
		if err != nil {
			logger.Fatal("Failed to initialize OpenRouter client", zap.Error(err))
		}
		logger.Info("Using OpenRouter AI provider", zap.String("model", cfg.AI.OpenRouterModel))
	} else {
		aiClient, err = ai.NewGeminiClient(keyManager, cfg.AI.GeminiModel, zapLogger)
		if err != nil {
			logger.Fatal("Failed to initialize Gemini client", zap.Error(err))
		}
		logger.Info("Using Gemini AI provider", zap.String("model", cfg.AI.GeminiModel))
	}
	defer aiClient.Close()

	// Initialize Error Recovery System (after aiClient)
	errorRecoveryRepo := storage.NewErrorRecoveryRepository(db)
	ctx := context.Background()
	rules, err := errorRecoveryRepo.ListRules(ctx)
	if err != nil {
		rules = []error_recovery.ContextAwareRule{}
	}
	defaultRules := error_recovery.GetDefaultRules()
	existingNames := make(map[string]bool)
	for _, r := range rules {
		existingNames[r.Name] = true
	}
	for _, dr := range defaultRules {
		if !existingNames[dr.Name] {
			errorRecoveryRepo.CreateRule(ctx, &dr)
			rules = append(rules, dr)
		}
	}
	errorRecoverySystem := error_recovery.NewErrorRecoverySystem(
		error_recovery.SystemConfig{
			Enabled: true,
			AnalyzerConfig: error_recovery.AnalyzerConfig{
				WindowSize:            100,
				ErrorRateThreshold:    0.10,
				ConsecutiveErrorLimit: 5,
				SameErrorThreshold:    10,
				DomainErrorThreshold:  0.20,
			},
			MinSuccessRate: 0.90,
			MinUsageCount:  5,
			AIEnabled:      true,
		},
		rules,
		aiClient,
	)
	logger.Info("Error recovery system initialized", zap.Int("rules", len(rules)))

	// Initialize Error Recovery History Repository and Handler
	recoveryHistoryRepo := storage.NewErrorRecoveryHistoryRepository(db)
	recoveryHistoryHandler := handlers.NewErrorRecoveryHistoryHandler(recoveryHistoryRepo)

	errorRecoveryHandler := handlers.NewErrorRecoveryHandler(errorRecoveryRepo)

	// Create ExecutionHandler with errorRecoverySystem
	executionHandler := handlers.NewExecutionHandler(workflowRepo, executionRepo, extractedItemsRepo, nodeExecRepo, browserPool, urlQueue, errorRecoverySystem, recoveryHistoryRepo)

	autoFixService := ai.NewAutoFixService(aiClient, zapLogger)
	fixSuggestionRepo := storage.NewFixSuggestionRepository(db)
	autoFixHandler := handlers.NewAutoFixHandler(
		snapshotRepo,
		fixSuggestionRepo,
		workflowRepo,
		workflowVersionRepo,
		monitoringRepo,
		autoFixService,
		snapshotStoragePath,
	)

	logger.Info("AI auto-fix service initialized with key rotation")

	// Initialize schedule handler
	scheduleHandler := handlers.NewScheduleHandler(scheduleRepo, schedulerService)

	// Initialize plugin handler
	pluginHandler := handlers.NewPluginHandler(pluginRepo, zapLogger)

	// Initialize plugin code handler (for editing/building plugins)
	sourceManager := plugin.NewSourceManager("./examples/plugins")
	builder := plugin.NewBuilder("./examples/plugins", "./plugins")
	pluginCodeHandler := handlers.NewPluginCodeHandler(sourceManager, builder, pluginRepo)

	// Initialize browser profile handler
	browserProfileHandler := handlers.NewBrowserProfileHandler(browserProfileRepo, browserPool.GetLauncher())

	// Routes
	setupRoutes(app, workflowHandler, workflowVersionHandler, executionHandler, analyticsHandler, selectorHandler, monitoringHandler, scheduleHandler, snapshotHandler, autoFixHandler, pluginHandler, pluginCodeHandler, browserProfileHandler, errorRecoveryHandler, recoveryHistoryHandler)

	// Monitoring
	app.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := db.Health(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "unhealthy",
				"error":  "database connection failed",
			})
		}

		return c.JSON(fiber.Map{
			"status":  "healthy",
			"version": "1.0.0",
			"time":    time.Now().UTC(),
		})
	})

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info("Server starting", zap.String("address", addr))

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		logger.Info("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
		defer cancel()

		if err := app.ShutdownWithContext(ctx); err != nil {
			logger.Error("Server shutdown error", zap.Error(err))
		}

		// Stop scheduler
		schedulerService.Stop()
		logger.Info("Monitoring scheduler stopped")
	}()

	if err := app.Listen(addr); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}

func setupRoutes(app *fiber.App, workflowHandler *handlers.WorkflowHandler, workflowVersionHandler *handlers.WorkflowVersionHandler, executionHandler *handlers.ExecutionHandler, analyticsHandler *handlers.AnalyticsHandler, selectorHandler *handlers.SelectorHandler, monitoringHandler *handlers.MonitoringHandler, scheduleHandler *handlers.ScheduleHandler, snapshotHandler *handlers.SnapshotHandler, autoFixHandler *handlers.AutoFixHandler, pluginHandler *handlers.PluginHandler, pluginCodeHandler *handlers.PluginCodeHandler, browserProfileHandler *handlers.BrowserProfileHandler, errorRecoveryHandler *handlers.ErrorRecoveryHandler, recoveryHistoryHandler *handlers.ErrorRecoveryHistoryHandler) {
	api := app.Group("/api/v1")

	// Workflow routes
	workflows := api.Group("/workflows")
	workflows.Post("/", workflowHandler.CreateWorkflow)
	workflows.Get("/", workflowHandler.ListWorkflows)
	workflows.Get("/:id", workflowHandler.GetWorkflow)
	workflows.Put("/:id", workflowHandler.UpdateWorkflow)
	workflows.Delete("/:id", workflowHandler.DeleteWorkflow)
	workflows.Patch("/:id/status", workflowHandler.UpdateWorkflowStatus)

	// Workflow Version routes
	workflows.Get("/:id/versions", workflowVersionHandler.ListVersions)
	workflows.Post("/:id/rollback/:version", workflowVersionHandler.RollbackVersion)

	// Execution routes
	workflows.Post("/:id/execute", executionHandler.StartExecution)

	// Health Check routes (under monitoring)
	workflows.Post("/:id/monitoring/run", monitoringHandler.RunMonitoring)
	workflows.Get("/:id/monitoring", monitoringHandler.ListMonitoring)

	// Monitoring reports (by ID)
	monitoring := api.Group("/monitoring")
	monitoring.Get("/:report_id", monitoringHandler.GetMonitoringReport)
	monitoring.Post("/:report_id/set-baseline", monitoringHandler.SetBaseline)
	monitoring.Get("/:report_id/compare", monitoringHandler.CompareWithBaseline)

	// Baseline routes
	workflows.Get("/:id/baseline", monitoringHandler.GetBaseline)

	// Schedule management routes
	workflows.Get("/:id/schedule", scheduleHandler.GetSchedule)
	workflows.Post("/:id/schedule", scheduleHandler.CreateSchedule)
	workflows.Delete("/:id/schedule", scheduleHandler.DeleteSchedule)
	workflows.Post("/:id/test-notification", scheduleHandler.TestNotification)

	// Snapshot routes
	snapshots := api.Group("/snapshots")
	snapshots.Get("/:snapshot_id", snapshotHandler.GetSnapshot)
	snapshots.Get("/:snapshot_id/screenshot", snapshotHandler.GetScreenshot)
	snapshots.Get("/:snapshot_id/dom", snapshotHandler.GetDOM)
	snapshots.Delete("/:snapshot_id", snapshotHandler.DeleteSnapshot)

	// Auto-fix AI routes
	snapshots.Post("/:id/analyze", autoFixHandler.AnalyzeSnapshot)
	snapshots.Get("/:id/suggestions", autoFixHandler.GetSuggestions)

	suggestions := api.Group("/suggestions")
	suggestions.Post("/:id/approve", autoFixHandler.ApproveSuggestion)
	suggestions.Post("/:id/reject", autoFixHandler.RejectSuggestion)
	suggestions.Post("/:id/apply", autoFixHandler.ApplySuggestion)
	suggestions.Post("/:id/revert", autoFixHandler.RevertSuggestion)

	// Snapshots by report
	monitoring.Get("/:report_id/snapshots", snapshotHandler.ListSnapshotsByReport)

	executions := api.Group("/executions")
	executions.Get("/", executionHandler.ListExecutions)
	executions.Get("/:execution_id", executionHandler.GetExecutionStatus)
	executions.Delete("/:execution_id", executionHandler.StopExecution)
	executions.Post("/:execution_id/pause", executionHandler.PauseExecution)   // NEW
	executions.Post("/:execution_id/resume", executionHandler.ResumeExecution) // NEW
	executions.Get("/:execution_id/stats", executionHandler.GetQueueStats)
	executions.Get("/:execution_id/data", executionHandler.GetExtractedData)

	// Stream route
	streamHandler := handlers.NewStreamHandler(executionHandler)
	api.Get("/executions/:id/stream", streamHandler.StreamExecutionEvents)

	// Analytics/Visualization routes
	analyticsHandler.RegisterRoutes(api)

	// Selector routes
	selector := api.Group("/selector")
	selector.Post("/sessions", selectorHandler.CreateSelectorSession)
	selector.Get("/sessions/:session_id", selectorHandler.GetSessionStatus)
	selector.Get("/sessions/:session_id/fields", selectorHandler.GetSelectedFields)
	selector.Delete("/sessions/:session_id", selectorHandler.CloseSelectorSession)

	// Plugin Marketplace routes
	plugins := api.Group("/plugins")
	plugins.Get("/categories", pluginHandler.GetCategories)
	plugins.Get("/search", pluginHandler.SearchPlugins)
	plugins.Get("/popular", pluginHandler.GetPopularPlugins)
	plugins.Get("/installed", pluginHandler.ListInstalledPlugins)
	plugins.Post("/", pluginHandler.CreatePlugin)
	plugins.Get("/", pluginHandler.ListPlugins)
	plugins.Get("/:slug", pluginHandler.GetPlugin)
	plugins.Put("/:id", pluginHandler.UpdatePlugin)
	plugins.Delete("/:id", pluginHandler.DeletePlugin)
	plugins.Post("/:id/versions", pluginHandler.PublishVersion)
	plugins.Get("/:id/versions", pluginHandler.ListVersions)
	plugins.Get("/:id/versions/:version", pluginHandler.GetVersion)
	plugins.Post("/:id/install", pluginHandler.InstallPlugin)
	plugins.Post("/:id/uninstall", pluginHandler.UninstallPlugin)
	plugins.Post("/:id/reviews", pluginHandler.CreateReview)
	plugins.Get("/:id/reviews", pluginHandler.ListReviews)

	// Plugin code management routes
	plugins.Get("/:id/code", pluginCodeHandler.GetPluginSource)
	plugins.Put("/:id/code", pluginCodeHandler.UpdatePluginSource)
	plugins.Post("/:id/build", pluginCodeHandler.BuildPlugin)
	plugins.Post("/scaffold", pluginCodeHandler.ScaffoldPlugin)
	plugins.Get("/:id/readme", pluginCodeHandler.GetPluginReadme)

	// Build status routes
	builds := api.Group("/builds")
	builds.Get("/:build_id/status", pluginCodeHandler.GetBuildStatus)

	// Browser Profile routes
	profiles := api.Group("/profiles")
	profiles.Post("/", browserProfileHandler.CreateProfile)
	profiles.Get("/", browserProfileHandler.ListProfiles)
	profiles.Get("/folders", browserProfileHandler.GetFolders)
	profiles.Post("/generate-fingerprint", browserProfileHandler.GenerateFingerprint)
	profiles.Post("/test-browser-config", browserProfileHandler.TestBrowserConfig)
	profiles.Get("/browser-types", browserProfileHandler.GetBrowserTypes)
	profiles.Get("/:id", browserProfileHandler.GetProfile)
	profiles.Put("/:id", browserProfileHandler.UpdateProfile)
	profiles.Delete("/:id", browserProfileHandler.DeleteProfile)
	profiles.Post("/:id/duplicate", browserProfileHandler.DuplicateProfile)
	profiles.Post("/:id/test", browserProfileHandler.TestProfile)
	profiles.Post("/:id/launch", browserProfileHandler.LaunchProfile)
	profiles.Post("/:id/stop", browserProfileHandler.StopProfile)

	// Error Recovery routes
	errorRecovery := api.Group("/error-recovery")
	errorRecovery.Get("/rules", errorRecoveryHandler.ListRules)
	errorRecovery.Get("/rules/:id", errorRecoveryHandler.GetRule)
	errorRecovery.Post("/rules", errorRecoveryHandler.CreateRule)
	errorRecovery.Put("/rules/:id", errorRecoveryHandler.UpdateRule)
	errorRecovery.Delete("/rules/:id", errorRecoveryHandler.DeleteRule)
	errorRecovery.Get("/config/:key", errorRecoveryHandler.GetConfig)
	errorRecovery.Put("/config", errorRecoveryHandler.UpdateConfig)

	// Error Recovery History routes
	errorRecovery.Get("/history/recent", recoveryHistoryHandler.GetRecentHistory)
	errorRecovery.Get("/history/stats", recoveryHistoryHandler.GetStats)

	// Execution-specific recovery history
	api.Get("/executions/:id/recovery-history", recoveryHistoryHandler.GetExecutionHistory)

	// Workflow-specific recovery history
	api.Get("/workflows/:id/recovery-history", recoveryHistoryHandler.GetWorkflowHistory)
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	logger.Error("Request error",
		zap.Error(err),
		zap.String("path", c.Path()),
		zap.Int("status", code),
	)

	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
	})
}
