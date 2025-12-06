package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/database"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/shared/queue"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/executor"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/handler"
)

func main() {
	// Initialize logger
	if err := logger.Init(true); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting Crawlify Worker Service")

	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Initialize database with limited connections (for worker)
	cfg.Database.MaxConnections = 5 // Workers use fewer connections
	db, err := database.NewDB(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize Redis cache
	redisCache, err := cache.NewCache(&cfg.Redis)
	if err != nil {
		logger.Fatal("Redis cache required for workers", zap.Error(err))
	}
	defer redisCache.Close()

	// Determine Pub/Sub mode (push for Cloud Run, pull for local/VMs)
	pubsubMode := cfg.GCP.PubSubMode
	if pubsubMode == "" {
		pubsubMode = "pull" // Default to pull mode
	}

	logger.Info("Pub/Sub mode configured",
		zap.String("mode", pubsubMode),
	)

	// Initialize Pub/Sub client (only needed for pull mode and publishing)
	ctx := context.Background()
	var pubsubClient *queue.PubSubClient
	if pubsubMode == "pull" {
		pubsubClient, err = queue.NewPubSubClient(ctx, &cfg.GCP)
		if err != nil {
			logger.Fatal("Failed to initialize Pub/Sub client", zap.Error(err))
		}
		defer pubsubClient.Close()
	}

	// Initialize task executor
	orchestratorURL := cfg.GCP.OrchestratorURL
	if orchestratorURL == "" {
		orchestratorURL = "http://localhost:8080"
		logger.Warn("ORCHESTRATOR_URL not set in config, using default", zap.String("url", orchestratorURL))
	}

	taskExecutor, err := executor.NewTaskExecutor(&cfg.Browser, &cfg.GCP, pubsubClient, redisCache, orchestratorURL, db)
	if err != nil {
		logger.Fatal("Failed to create task executor", zap.Error(err))
	}
	defer taskExecutor.Close()

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Crawlify Worker",
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	})

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "worker",
			"mode":    pubsubMode,
		})
	})

	// Push mode: Register push endpoint for Cloud Run
	if pubsubMode == "push" {
		pushHandler := handler.NewPushHandler(taskExecutor)
		app.Post("/tasks/push", pushHandler.Handler())
		logger.Info("Push handler registered at /tasks/push")
	}

	// Setup graceful shutdown
	shutdownCtx, shutdownCancel := context.WithCancel(ctx)
	defer shutdownCancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		logger.Info("Shutting down worker...")
		shutdownCancel()

		// Give time for current tasks to complete
		time.Sleep(time.Duration(cfg.Server.ShutdownTimeout) * time.Second)

		logger.Warn("Shutdown timeout reached, forcing exit")
		os.Exit(1)
	}()

	// Start HTTP server
	go func() {
		port := fmt.Sprintf("%d", cfg.Server.Port)
		logger.Info("Worker HTTP server starting", zap.String("port", port))

		if err := app.Listen(":" + port); err != nil {
			logger.Error("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Pull mode: Start Pub/Sub subscription loop
	if pubsubMode == "pull" {
		logger.Info("Starting to pull tasks from Pub/Sub")

		err = pubsubClient.Subscribe(shutdownCtx, func(ctx context.Context, task *models.Task) error {
			logger.Info("Processing task",
				zap.String("task_id", task.TaskID),
				zap.String("execution_id", task.ExecutionID),
				zap.String("url", task.URL),
			)

			if err := taskExecutor.Execute(ctx, task); err != nil {
				logger.Error("Task execution failed",
					zap.String("task_id", task.TaskID),
					zap.Error(err),
				)
				return err
			}

			logger.Info("Task completed successfully",
				zap.String("task_id", task.TaskID),
			)

			return nil
		})

		if err != nil && err != context.Canceled {
			logger.Fatal("Error in task subscription", zap.Error(err))
		}
	} else {
		// Push mode: Just wait for shutdown signal
		logger.Info("Push mode: waiting for HTTP requests on /tasks/push")
		<-shutdownCtx.Done()
	}

	logger.Info("Worker shutdown complete")
}
