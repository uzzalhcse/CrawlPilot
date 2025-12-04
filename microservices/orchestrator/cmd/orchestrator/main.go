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
	"go.uber.org/zap"

	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/api/handlers"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/repository"
	"github.com/uzzalhcse/crawlify/microservices/orchestrator/internal/service"
	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/database"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/queue"
)

func main() {
	// Initialize logger
	if err := logger.Init(true); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting Crawlify Orchestrator Service")

	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Initialize database
	db, err := database.NewDB(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize Redis cache
	redisCache, err := cache.NewCache(&cfg.Redis)
	if err != nil {
		logger.Warn("Redis cache not available, running without cache", zap.Error(err))
	} else {
		defer redisCache.Close()
	}

	// Initialize Pub/Sub client
	ctx := context.Background()
	pubsubClient, err := queue.NewPubSubClient(ctx, &cfg.GCP)
	if err != nil {
		logger.Fatal("Failed to initialize Pub/Sub client", zap.Error(err))
	}
	defer pubsubClient.Close()

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Crawlify Orchestrator",
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,PATCH",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Logging middleware
	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		logger.Info("Request",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("duration", duration),
		)

		return err
	})

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Check database
		if err := db.Health(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "unhealthy",
				"error":  "database connection failed",
			})
		}

		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "orchestrator",
			"version": "1.0.0",
			"time":    time.Now().UTC(),
		})
	})

	// Initialize repositories
	workflowRepo := repository.NewWorkflowRepository(db)
	executionRepo := repository.NewExecutionRepository(db)

	// Initialize services
	workflowSvc := service.NewWorkflowService(workflowRepo, redisCache)
	executionSvc := service.NewExecutionService(workflowRepo, executionRepo, workflowSvc, pubsubClient)

	// Initialize handlers
	workflowHandler := handlers.NewWorkflowHandler(workflowSvc)
	executionHandler := handlers.NewExecutionHandler(executionSvc)
	statsHandler := handlers.NewStatsHandler(executionRepo)

	// API routes
	api := app.Group("/api/v1")

	// Workflow routes
	workflows := api.Group("/workflows")
	workflows.Post("/", workflowHandler.CreateWorkflow)
	workflows.Get("/", workflowHandler.ListWorkflows)
	workflows.Get("/:id", workflowHandler.GetWorkflow)
	workflows.Put("/:id", workflowHandler.UpdateWorkflow)
	workflows.Delete("/:id", workflowHandler.DeleteWorkflow)

	// Execution routes
	workflows.Post("/:id/execute", executionHandler.StartExecution)
	workflows.Get("/:id/executions", executionHandler.ListExecutions)

	executions := api.Group("/executions")
	executions.Get("/:id", executionHandler.GetExecution)
	executions.Delete("/:id", executionHandler.StopExecution)

	// Internal API (for worker → orchestrator communication)
	internal := api.Group("/internal")
	internal.Post("/executions/:id/stats", statsHandler.UpdateExecutionStats)

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info("Orchestrator starting", zap.String("address", addr))

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		logger.Info("Shutting down orchestrator...")
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
		defer cancel()

		if err := app.ShutdownWithContext(ctx); err != nil {
			logger.Error("Server shutdown error", zap.Error(err))
		}
	}()

	if err := app.Listen(addr); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
