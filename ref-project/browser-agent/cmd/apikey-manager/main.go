package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"crawer-agent/exp/v2/pkg/apikey"
	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(".env"); err != nil {
		log.Debug("No .env file found")
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Get MongoDB configuration
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	mongoDatabase := os.Getenv("MONGODB_DATABASE")
	if mongoDatabase == "" {
		mongoDatabase = "crawler_agent"
	}

	// Initialize MongoDB client
	ctx := context.Background()
	mongoClient, err := apikey.NewMongoClient(ctx, &apikey.MongoConfig{
		URI:      mongoURI,
		Database: mongoDatabase,
		Timeout:  10 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Close(ctx)

	// Create indexes
	if err := mongoClient.CreateIndexes(ctx); err != nil {
		log.Warnf("Failed to create indexes: %v", err)
	}

	// Initialize API key manager
	manager := apikey.NewManager(&apikey.ManagerConfig{
		MongoClient: mongoClient,
		SyncTTL:     5 * time.Minute,
	})

	// Parse command
	command := os.Args[1]

	switch command {
	case "add":
		handleAdd(ctx, manager)
	case "list":
		handleList(ctx, manager)
	case "remove":
		handleRemove(ctx, manager)
	case "stats":
		handleStats(ctx, manager)
	case "test":
		handleTest(ctx, manager)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleAdd(ctx context.Context, manager *apikey.Manager) {
	if len(os.Args) < 4 {
		fmt.Println("Usage: apikey-manager add <provider> <api-key> [name=value...]")
		fmt.Println("Example: apikey-manager add gemini AIza... name=primary-key tier=pro")
		os.Exit(1)
	}

	provider := os.Args[2]
	key := os.Args[3]

	// Parse metadata
	metadata := make(map[string]string)
	for i := 4; i < len(os.Args); i++ {
		parts := strings.SplitN(os.Args[i], "=", 2)
		if len(parts) == 2 {
			metadata[parts[0]] = parts[1]
		}
	}

	err := manager.AddKey(ctx, provider, key, metadata)
	if err != nil {
		log.Fatalf("Failed to add key: %v", err)
	}

	fmt.Printf("✅ Successfully added %s API key\n", provider)
	if len(metadata) > 0 {
		fmt.Println("Metadata:")
		for k, v := range metadata {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}
}

func handleList(ctx context.Context, manager *apikey.Manager) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: apikey-manager list <provider>")
		fmt.Println("Example: apikey-manager list gemini")
		os.Exit(1)
	}

	provider := os.Args[2]

	keys, err := manager.GetAllKeys(ctx, provider)
	if err != nil {
		log.Fatalf("Failed to list keys: %v", err)
	}

	if len(keys) == 0 {
		fmt.Printf("No API keys found for provider: %s\n", provider)
		return
	}

	fmt.Printf("\n%s API Keys (%d total):\n", strings.ToUpper(provider), len(keys))
	fmt.Println(strings.Repeat("=", 80))

	for i, key := range keys {
		status := "🟢 Active"
		if !key.IsActive {
			status = "🔴 Inactive"
		} else if key.RateLimitedUntil != nil && time.Now().Before(*key.RateLimitedUntil) {
			status = fmt.Sprintf("⏸️  Rate Limited (until %s)", key.RateLimitedUntil.Format("15:04:05"))
		}

		fmt.Printf("\n%d. %s\n", i+1, status)
		fmt.Printf("   ID: %s\n", key.ID.Hex())
		fmt.Printf("   Key: %s\n", maskKey(key.Key))
		fmt.Printf("   Usage Count: %d\n", key.UsageCount)
		fmt.Printf("   Failure Count: %d\n", key.FailureCount)

		if key.LastUsedAt != nil {
			fmt.Printf("   Last Used: %s\n", key.LastUsedAt.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("   Last Used: Never\n")
		}

		if len(key.Metadata) > 0 {
			fmt.Printf("   Metadata:\n")
			for k, v := range key.Metadata {
				fmt.Printf("     %s: %s\n", k, v)
			}
		}
	}
	fmt.Println()
}

func handleRemove(ctx context.Context, manager *apikey.Manager) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: apikey-manager remove <key-id>")
		fmt.Println("Example: apikey-manager remove 507f1f77bcf86cd799439011")
		os.Exit(1)
	}

	keyIDStr := os.Args[2]

	// Convert string to ObjectID
	keyID, err := parseObjectID(keyIDStr)
	if err != nil {
		log.Fatalf("Invalid key ID: %v", err)
	}

	err = manager.DeactivateKey(ctx, keyID)
	if err != nil {
		log.Fatalf("Failed to deactivate key: %v", err)
	}

	fmt.Printf("✅ Successfully deactivated key: %s\n", keyIDStr)
}

func handleStats(ctx context.Context, manager *apikey.Manager) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: apikey-manager stats <provider> [hours]")
		fmt.Println("Example: apikey-manager stats gemini 24")
		os.Exit(1)
	}

	provider := os.Args[2]
	hours := 24

	if len(os.Args) >= 4 {
		fmt.Sscanf(os.Args[3], "%d", &hours)
	}

	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	stats, err := manager.GetUsageStats(ctx, provider, since)
	if err != nil {
		log.Fatalf("Failed to get usage stats: %v", err)
	}

	fmt.Printf("\nUsage Statistics for %s (last %d hours):\n", strings.ToUpper(provider), hours)
	fmt.Println(strings.Repeat("=", 80))

	keys := stats["keys"].([]interface{})
	if len(keys) == 0 {
		fmt.Println("No usage data available")
		return
	}

	totalRequests := 0
	totalSuccess := 0
	totalFailed := 0

	for i, keyData := range keys {
		data := keyData.(map[string]interface{})
		requests := int(data["total_requests"].(int32))
		success := int(data["successful_requests"].(int32))
		failed := int(data["failed_requests"].(int32))
		avgTime := data["avg_response_time"].(float64)

		totalRequests += requests
		totalSuccess += success
		totalFailed += failed

		successRate := float64(success) / float64(requests) * 100

		fmt.Printf("\n%d. Key ID: %v\n", i+1, data["_id"])
		fmt.Printf("   Total Requests: %d\n", requests)
		fmt.Printf("   Successful: %d (%.1f%%)\n", success, successRate)
		fmt.Printf("   Failed: %d (%.1f%%)\n", failed, 100-successRate)
		fmt.Printf("   Avg Response Time: %.0f ms\n", avgTime)
	}

	if totalRequests > 0 {
		fmt.Printf("\nOverall:\n")
		fmt.Printf("   Total Requests: %d\n", totalRequests)
		fmt.Printf("   Success Rate: %.1f%%\n", float64(totalSuccess)/float64(totalRequests)*100)
		fmt.Printf("   Failure Rate: %.1f%%\n", float64(totalFailed)/float64(totalRequests)*100)
	}
	fmt.Println()
}

func handleTest(ctx context.Context, manager *apikey.Manager) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: apikey-manager test <provider>")
		fmt.Println("Example: apikey-manager test gemini")
		os.Exit(1)
	}

	provider := os.Args[2]

	fmt.Printf("Testing %s API key rotation...\n\n", provider)

	// Sync keys
	if err := manager.SyncKeys(ctx); err != nil {
		log.Fatalf("Failed to sync keys: %v", err)
	}

	// Get all keys
	keys, err := manager.GetAllKeys(ctx, provider)
	if err != nil {
		log.Fatalf("Failed to get keys: %v", err)
	}

	if len(keys) == 0 {
		fmt.Printf("❌ No keys found for provider: %s\n", provider)
		os.Exit(1)
	}

	fmt.Printf("Found %d keys for %s\n", len(keys), provider)

	// Test round-robin rotation
	fmt.Println("\nTesting round-robin rotation (10 iterations):")
	usageCounts := make(map[string]int)

	for i := 0; i < 10; i++ {
		key, err := manager.GetKey(ctx, provider)
		if err != nil {
			log.Fatalf("Failed to get key: %v", err)
		}

		maskedKey := maskKey(key.Key)
		usageCounts[maskedKey]++

		fmt.Printf("  Iteration %2d: %s (Usage: %d)\n", i+1, maskedKey, key.UsageCount)

		// Simulate usage
		if err := manager.UseKey(ctx, key.ID, true, "", 100); err != nil {
			log.Warnf("Failed to record usage: %v", err)
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Print distribution
	fmt.Println("\nUsage Distribution:")
	for key, count := range usageCounts {
		fmt.Printf("  %s: %d requests\n", key, count)
	}

	fmt.Println("\n✅ Test completed successfully!")
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func parseObjectID(id string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(id)
}

func printUsage() {
	fmt.Println("API Key Manager - Manage API keys for rotation")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  apikey-manager <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  add <provider> <key> [metadata...]  Add a new API key")
	fmt.Println("  list <provider>                      List all keys for a provider")
	fmt.Println("  remove <key-id>                      Deactivate an API key")
	fmt.Println("  stats <provider> [hours]             Show usage statistics")
	fmt.Println("  test <provider>                      Test key rotation")
	fmt.Println()
	fmt.Println("Providers:")
	fmt.Println("  openai, gemini, claude")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  apikey-manager add gemini AIza... name=primary-key")
	fmt.Println("  apikey-manager list gemini")
	fmt.Println("  apikey-manager stats gemini 24")
	fmt.Println("  apikey-manager test gemini")
	fmt.Println("  apikey-manager remove 507f1f77bcf86cd799439011")
}
