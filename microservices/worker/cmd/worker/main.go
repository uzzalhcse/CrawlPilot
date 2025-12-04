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

	// Initialize Pub/Sub client
	ctx := context.Background()
	pubsubClient, err := queue.NewPubSubClient(ctx, &cfg.GCP)
	if err != nil {
		logger.Fatal("Failed to initialize Pub/Sub client", zap.Error(err))
	}
	defer pubsubClient.Close()

	// Initialize Fiber app for health checks
	app := fiber.New(fiber.Config{
		AppName: "Crawlify Worker",
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"service":   "worker",
			"worker_id": os.Getenv("K_REVISION"),
		})
	})

	// Start HTTP server in background
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8081"
		}

		logger.Info("Worker HTTP server starting", zap.String("port", port))

		if err := app.Listen(":" + port); err != nil {
			logger.Error("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Start task processing
	logger.Info("Starting to pull tasks from Pub/Sub")

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
		time.Sleep(30 * time.Second)
	}()

	// Initialize task executor
	orchestratorURL := os.Getenv("ORCHESTRATOR_URL")
	if orchestratorURL == "" {
		orchestratorURL = "http://localhost:8080" // Default for local development
	}

	taskExecutor, err := executor.NewTaskExecutor(&cfg.Browser, &cfg.GCP, pubsubClient, redisCache, orchestratorURL)
	if err != nil {
		logger.Fatal("Failed to create task executor", zap.Error(err))
	}
	defer taskExecutor.Close()

	// Process tasks from Pub/Sub
	err = pubsubClient.Subscribe(shutdownCtx, func(ctx context.Context, task *models.Task) error {
		logger.Info("Processing task",
			zap.String("task_id", task.TaskID),
			zap.String("execution_id", task.ExecutionID),
			zap.String("url", task.URL),
		)

		// Execute task
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

	logger.Info("Worker shutdown complete")
}
