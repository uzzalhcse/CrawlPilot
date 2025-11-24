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
	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/config"
	"github.com/uzzalhcse/crawlify/internal/healthcheck"
	"github.com/uzzalhcse/crawlify/internal/logger"
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
	executionRepo := storage.NewExecutionRepository(db)
	extractedItemsRepo := storage.NewExtractedItemsRepository(db)
	nodeExecRepo := storage.NewNodeExecutionRepository(db)
	urlQueue := queue.NewURLQueue(db)
	healthCheckRepo := storage.NewHealthCheckRepository(db)

	// Initialize browser pool
	browserPool, err := browser.NewBrowserPool(&cfg.Browser)
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

	// Initialize handlers
	workflowHandler := handlers.NewWorkflowHandler(workflowRepo)
	executionHandler := handlers.NewExecutionHandler(workflowRepo, executionRepo, extractedItemsRepo, nodeExecRepo, browserPool, urlQueue)
	analyticsHandler := handlers.NewAnalyticsHandler(nodeExecRepo, extractedItemsRepo, urlQueue)
	selectorHandler := handlers.NewSelectorHandler(browserPool)

	// Create a node registry for health checks
	nodeRegistry := workflow.NewNodeRegistry()
	if err := nodeRegistry.RegisterDefaultNodes(); err != nil {
		logger.Warn("Failed to register default nodes for health checks", zap.Error(err))
	}
	healthCheckHandler := handlers.NewHealthCheckHandler(workflowRepo, healthCheckRepo, browserPool, nodeRegistry)

	// Initialize schedule repository and handler
	scheduleRepo := storage.NewHealthCheckScheduleRepository(db)

	// Create health check orchestrator for scheduler
	healthCheckOrchestrator := healthcheck.NewOrchestrator(browserPool, nodeRegistry, nil)

	// Create and start scheduler service
	schedulerService := healthcheck.NewSchedulerService(scheduleRepo, healthCheckRepo, workflowRepo, healthCheckOrchestrator)
	go schedulerService.Start()
	logger.Info("Health check scheduler started")

	scheduleHandler := handlers.NewScheduleHandler(scheduleRepo, schedulerService)

	// Routes
	setupRoutes(app, workflowHandler, executionHandler, analyticsHandler, selectorHandler, healthCheckHandler, scheduleHandler)

	// Health check
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
		logger.Info("Health check scheduler stopped")
	}()

	if err := app.Listen(addr); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}

func setupRoutes(app *fiber.App, workflowHandler *handlers.WorkflowHandler, executionHandler *handlers.ExecutionHandler, analyticsHandler *handlers.AnalyticsHandler, selectorHandler *handlers.SelectorHandler, healthCheckHandler *handlers.HealthCheckHandler, scheduleHandler *handlers.ScheduleHandler) {
	api := app.Group("/api/v1")

	// Workflow routes
	workflows := api.Group("/workflows")
	workflows.Post("/", workflowHandler.CreateWorkflow)
	workflows.Get("/", workflowHandler.ListWorkflows)
	workflows.Get("/:id", workflowHandler.GetWorkflow)
	workflows.Put("/:id", workflowHandler.UpdateWorkflow)
	workflows.Delete("/:id", workflowHandler.DeleteWorkflow)
	workflows.Patch("/:id/status", workflowHandler.UpdateWorkflowStatus)

	// Execution routes
	workflows.Post("/:id/execute", executionHandler.StartExecution)

	// Health Check routes
	workflows.Post("/:id/health-check", healthCheckHandler.RunHealthCheck)
	workflows.Get("/:id/health-checks", healthCheckHandler.ListHealthChecks)

	// Health check reports (by ID)
	healthChecks := api.Group("/health-checks")
	healthChecks.Get("/:report_id", healthCheckHandler.GetHealthCheckReport)
	healthChecks.Post("/:report_id/set-baseline", healthCheckHandler.SetBaseline)
	healthChecks.Get("/:report_id/compare", healthCheckHandler.CompareWithBaseline)

	// Baseline routes
	workflows.Get("/:id/baseline", healthCheckHandler.GetBaseline)

	// Schedule routes
	workflows.Get("/:id/schedule", scheduleHandler.GetSchedule)
	workflows.Post("/:id/schedule", scheduleHandler.CreateSchedule)
	workflows.Delete("/:id/schedule", scheduleHandler.DeleteSchedule)
	workflows.Post("/:id/test-notification", scheduleHandler.TestNotification)

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
