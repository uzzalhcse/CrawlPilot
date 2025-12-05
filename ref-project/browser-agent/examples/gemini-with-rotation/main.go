/*
 * Example: Gemini with API Key Rotation
 *
 * This example demonstrates how to use the API key rotation system
 * to handle rate limits and ensure even distribution of API usage
 * across multiple keys.
 */

package main

import (
	"context"
	"crawer-agent/exp/v2/pkg/agent"
	"crawer-agent/exp/v2/pkg/apikey"
	"os"
	"time"

	"github.com/nerdface-ai/browser-use-go/pkg/dotenv"
	"google.golang.org/genai"

	"github.com/charmbracelet/log"
)

func main() {
	// Set log level to debug for more detailed output
	log.SetLevel(log.DebugLevel)

	dotenv.LoadEnv(".env")

	ctx := context.Background()

	log.Info("🔄 Initializing API Key Rotation")

	// Get MongoDB configuration
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	mongoDatabase := os.Getenv("MONGODB_DATABASE")
	if mongoDatabase == "" {
		mongoDatabase = "crawler_agent"
	}

	log.Infof("Connecting to MongoDB: %s/%s", mongoURI, mongoDatabase)

	// Initialize MongoDB client
	mongoClient, err := apikey.NewMongoClient(ctx, &apikey.MongoConfig{
		URI:      mongoURI,
		Database: mongoDatabase,
		Timeout:  10 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Close(ctx)

	// Create indexes (if not already created)
	if err := mongoClient.CreateIndexes(ctx); err != nil {
		log.Warnf("Failed to create indexes: %v", err)
	}

	// Initialize API key manager
	manager := apikey.NewManager(&apikey.ManagerConfig{
		MongoClient: mongoClient,
		SyncTTL:     5 * time.Minute,
	})

	// Sync keys from database
	if err := manager.SyncKeys(ctx); err != nil {
		log.Fatalf("Failed to sync API keys: %v", err)
	}

	log.Info("✅ API Key Manager initialized")

	// Create Gemini model with rotation support
	modelName := "gemini-2.0-flash-exp"
	chatModel, err := apikey.NewGeminiModel(
		ctx,
		manager,
		modelName,
		&genai.ThinkingConfig{
			IncludeThoughts: true,
			ThinkingBudget:  nil,
		},
	)
	if err != nil {
		log.Fatalf("Failed to create Gemini model: %v", err)
	}

	currentKey := chatModel.GetCurrentKey()
	log.Infof("✅ Using initial API key: %s... (Usage: %d)",
		currentKey.Key[:8], currentKey.UsageCount)

	// Define the task
	task := "Go to https://merrell.jp/collections/men/products/mens_speed-arc-matis-gore-tex?variant=41939120193579 and extract product name, category and price"

	log.Infof("Starting browser automation task: %s", task)

	// Create agent with the model
	ag := agent.NewAgent(
		task,
		chatModel,
		agent.WithAgentSettings(agent.AgentSettingsConfig{
			"use_vision":   false, // Disable vision for Gemini
			"max_failures": 5,     // Allow more retries
		}),
	)

	// Run the agent
	result, err := ag.Run()
	if err != nil {
		log.Fatalf("Agent execution failed: %v", err)
	}

	log.Infof("Task completed successfully!")
	log.Infof("Result: %v", result)
}
