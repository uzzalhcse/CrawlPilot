package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"crawer-agent/exp/v2/pkg/apikey"

	"github.com/charmbracelet/log"
	"github.com/nerdface-ai/browser-use-go/pkg/dotenv"
)

func main() {
	// Parse command line flags
	var (
		filePath     = flag.String("file", "", "Path to the API keys file")
		provider     = flag.String("provider", "", "Provider name (gemini, openai, etc.)")
		rateLimit    = flag.Int("rate-limit", 60, "Rate limit per minute")
		dailyLimit   = flag.Int("daily-limit", 1000, "Daily request limit")
		skipExisting = flag.Bool("skip-existing", true, "Skip keys that already exist")
		deleteAll    = flag.Bool("delete-all", false, "Delete all existing keys for provider")
		listKeys     = flag.Bool("list", false, "List all keys for provider")
		stats        = flag.Bool("stats", false, "Show statistics for provider")
		mongoURI     = flag.String("mongo-uri", "", "MongoDB connection URI")
		mongoDB      = flag.String("mongo-db", "", "MongoDB database name")
		logLevel     = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	)

	flag.Parse()

	// Set log level
	switch *logLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	}

	// Load environment variables
	dotenv.LoadEnv(".env")

	// Get MongoDB configuration
	if *mongoURI == "" {
		*mongoURI = os.Getenv("MONGODB_URI")
		if *mongoURI == "" {
			*mongoURI = "mongodb://localhost:27017"
		}
	}

	if *mongoDB == "" {
		*mongoDB = os.Getenv("MONGODB_DATABASE")
		if *mongoDB == "" {
			*mongoDB = "crawler_agent"
		}
	}

	ctx := context.Background()

	// Connect to MongoDB
	log.Infof("📡 Connecting to MongoDB: %s/%s", *mongoURI, *mongoDB)
	mongoClient, err := apikey.NewMongoClient(ctx, &apikey.MongoConfig{
		URI:      *mongoURI,
		Database: *mongoDB,
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

	// Create seeder
	seeder := apikey.NewSeeder(mongoClient)

	// Handle different operations
	switch {
	case *deleteAll:
		if *provider == "" {
			log.Fatal("Provider is required for delete operation")
		}
		if err := confirmDelete(*provider); err != nil {
			log.Fatal(err)
		}
		if err := seeder.DeleteAllKeys(ctx, *provider); err != nil {
			log.Fatalf("Failed to delete keys: %v", err)
		}
		log.Infof("✅ Successfully deleted all keys for provider: %s", *provider)

	case *listKeys:
		if *provider == "" {
			log.Fatal("Provider is required for list operation")
		}
		keys, err := seeder.ListAllKeys(ctx, *provider)
		if err != nil {
			log.Fatalf("Failed to list keys: %v", err)
		}
		printKeys(keys)

	case *stats:
		if *provider == "" {
			log.Fatal("Provider is required for stats operation")
		}
		stats, err := seeder.GetStats(ctx, *provider)
		if err != nil {
			log.Fatalf("Failed to get stats: %v", err)
		}
		printStats(stats)

	case *filePath != "":
		if *provider == "" {
			log.Fatal("Provider is required for seeding operation")
		}

		config := &apikey.SeedConfig{
			FilePath:     *filePath,
			Provider:     *provider,
			RateLimit:    *rateLimit,
			DailyLimit:   *dailyLimit,
			SkipExisting: *skipExisting,
		}

		log.Infof("🌱 Starting seeding process...")
		log.Infof("   File: %s", *filePath)
		log.Infof("   Provider: %s", *provider)
		log.Infof("   Rate Limit: %d/min", *rateLimit)
		log.Infof("   Daily Limit: %d/day", *dailyLimit)
		log.Infof("   Skip Existing: %v", *skipExisting)

		if err := seeder.SeedFromFile(ctx, config); err != nil {
			log.Fatalf("Seeding failed: %v", err)
		}

		log.Info("✅ Seeding completed successfully!")

	default:
		printUsage()
	}
}

func confirmDelete(provider string) error {
	fmt.Printf("⚠️  WARNING: This will delete ALL API keys for provider '%s'\n", provider)
	fmt.Print("Type 'yes' to confirm: ")

	var response string
	fmt.Scanln(&response)

	if response != "yes" {
		return fmt.Errorf("deletion cancelled")
	}

	return nil
}

func printKeys(keys []*apikey.APIKey) {
	if len(keys) == 0 {
		log.Info("No keys found")
		return
	}

	log.Info("═══════════════════════════════════════")
	log.Infof("Found %d key(s):", len(keys))
	log.Info("═══════════════════════════════════════")

	for i, key := range keys {
		status := "🟢 Active"
		if !key.IsActive {
			status = "🔴 Inactive"
		}

		maskedKey := key.Key
		if len(maskedKey) > 20 {
			maskedKey = maskedKey[:10] + "..." + maskedKey[len(maskedKey)-8:]
		}

		log.Infof("%d. %s %s", i+1, status, maskedKey)
		log.Infof("   Provider: %s", key.Provider)
		log.Infof("   Usage: %d times", key.UsageCount)
		log.Infof("   Failures: %d", key.FailureCount)

		// Check metadata for rate limits
		if key.Metadata != nil {
			if rateLimit, ok := key.Metadata["rate_limit"]; ok && rateLimit != "0" {
				log.Infof("   Rate Limit: %s/min", rateLimit)
			}
			if dailyLimit, ok := key.Metadata["daily_limit"]; ok && dailyLimit != "0" {
				log.Infof("   Daily Limit: %s/day", dailyLimit)
			}
			if name, ok := key.Metadata["name"]; ok {
				log.Infof("   Name: %s", name)
			}
		}

		if key.LastUsedAt != nil && !key.LastUsedAt.IsZero() {
			log.Infof("   Last Used: %s", key.LastUsedAt.Format("2006-01-02 15:04:05"))
		}
		log.Info("   ---")
	}

	log.Info("═══════════════════════════════════════")
}

func printStats(stats map[string]interface{}) {
	log.Info("═══════════════════════════════════════")
	log.Infof("📊 Statistics for: %s", stats["provider"])
	log.Info("═══════════════════════════════════════")
	log.Infof("   Total Keys: %d", stats["total_keys"])
	log.Infof("   Active Keys: %d", stats["active_keys"])
	log.Infof("   Inactive Keys: %d", stats["inactive_keys"])
	log.Infof("   Total Usage: %d requests", stats["total_usage"])
	log.Info("═══════════════════════════════════════")
}

func printUsage() {
	fmt.Println(`
API Key Seeder - Bulk import API keys into MongoDB

USAGE:
    seeder [OPTIONS]

OPERATIONS:
    --file <path>           Seed API keys from file
    --list                  List all keys for a provider
    --stats                 Show statistics for a provider
    --delete-all            Delete all keys for a provider (requires confirmation)

REQUIRED FLAGS:
    --provider <name>       Provider name (gemini, openai, etc.)

OPTIONAL FLAGS:
    --rate-limit <n>        Rate limit per minute (default: 60)
    --daily-limit <n>       Daily request limit (default: 1000)
    --skip-existing         Skip keys that already exist (default: true)
    --mongo-uri <uri>       MongoDB connection URI (default: from MONGODB_URI env or localhost)
    --mongo-db <name>       MongoDB database name (default: from MONGODB_DATABASE env or crawler_agent)
    --log-level <level>     Log level: debug, info, warn, error (default: info)

EXAMPLES:
    # Seed Gemini keys from file
    seeder --file gemini_api_keys.txt --provider gemini

    # Seed with custom limits
    seeder --file gemini_api_keys.txt --provider gemini --rate-limit 100 --daily-limit 2000

    # List all keys for a provider
    seeder --list --provider gemini

    # Show statistics
    seeder --stats --provider gemini

    # Delete all keys for a provider
    seeder --delete-all --provider gemini

    # Use custom MongoDB connection
    seeder --file keys.txt --provider gemini --mongo-uri mongodb://user:pass@host:27017 --mongo-db mydb
`)
}
