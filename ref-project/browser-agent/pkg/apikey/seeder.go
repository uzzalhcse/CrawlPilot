package apikey

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

// SeedConfig holds configuration for seeding API keys
type SeedConfig struct {
	FilePath     string
	Provider     string
	RateLimit    int
	DailyLimit   int
	SkipExisting bool
}

// Seeder handles bulk insertion of API keys from files
type Seeder struct {
	mongoClient *MongoClient
}

// NewSeeder creates a new seeder instance
func NewSeeder(client *MongoClient) *Seeder {
	return &Seeder{
		mongoClient: client,
	}
}

// SeedFromFile reads API keys from a file and inserts them into MongoDB
// File format: one API key per line, lines starting with # are comments
// Optionally support key=value format for additional metadata
func (s *Seeder) SeedFromFile(ctx context.Context, config *SeedConfig) error {
	log.Infof("📁 Reading API keys from: %s", config.FilePath)

	// Open the file
	file, err := os.Open(config.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read file line by line
	scanner := bufio.NewScanner(file)
	lineNum := 0
	insertedCount := 0
	skippedCount := 0
	errorCount := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse the line
		apiKey, metadata := s.parseLine(line)
		if apiKey == "" {
			log.Warnf("Line %d: Invalid format, skipping", lineNum)
			skippedCount++
			continue
		}

		// Check if key already exists
		if config.SkipExisting {
			existing, err := s.mongoClient.GetAPIKey(ctx, config.Provider, apiKey)
			if err == nil && existing != nil {
				log.Debugf("Line %d: Key already exists, skipping", lineNum)
				skippedCount++
				continue
			}
		}

		// Create API key entry
		now := time.Now()
		key := &APIKey{
			Provider:     config.Provider,
			Key:          apiKey,
			UsageCount:   0,
			LastUsedAt:   nil,
			FailureCount: 0,
			IsActive:     true,
			CreatedAt:    now,
			UpdatedAt:    now,
			Metadata:     make(map[string]string),
		}

		// Store rate limits in metadata
		key.Metadata["rate_limit"] = fmt.Sprintf("%d", config.RateLimit)
		key.Metadata["daily_limit"] = fmt.Sprintf("%d", config.DailyLimit)

		// Apply any metadata from the line
		if metadata != nil {
			for k, v := range metadata {
				key.Metadata[k] = v
			}
		}

		// Insert the key
		err := s.mongoClient.InsertAPIKey(ctx, key)
		if err != nil {
			log.Errorf("Line %d: Failed to insert key: %v", lineNum, err)
			errorCount++
			continue
		}

		insertedCount++
		log.Debugf("✓ Inserted key from line %d: %s...", lineNum, maskKey(apiKey))
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Print summary
	log.Info("═══════════════════════════════════════")
	log.Infof("📊 Seeding Summary:")
	log.Infof("   Provider: %s", config.Provider)
	log.Infof("   Total lines processed: %d", lineNum)
	log.Infof("   ✅ Successfully inserted: %d", insertedCount)
	log.Infof("   ⏭️  Skipped: %d", skippedCount)
	log.Infof("   ❌ Errors: %d", errorCount)
	log.Info("═══════════════════════════════════════")

	return nil
}

// parseLine parses a line from the file
// Supports formats:
//   - Simple: "sk-proj-abc123..."
//   - With metadata: "sk-proj-abc123... name=my-key rate_limit=60"
func (s *Seeder) parseLine(line string) (string, map[string]string) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return "", nil
	}

	apiKey := parts[0]
	metadata := make(map[string]string)

	// Parse metadata if present
	for i := 1; i < len(parts); i++ {
		kv := strings.SplitN(parts[i], "=", 2)
		if len(kv) == 2 {
			metadata[kv[0]] = kv[1]
		}
	}

	return apiKey, metadata
}

// SeedMultipleProviders seeds keys from multiple files
func (s *Seeder) SeedMultipleProviders(ctx context.Context, configs []*SeedConfig) error {
	totalInserted := 0

	for _, config := range configs {
		if err := s.SeedFromFile(ctx, config); err != nil {
			log.Errorf("Failed to seed from %s: %v", config.FilePath, err)
			continue
		}
		totalInserted++
	}

	log.Infof("🎉 Completed seeding from %d file(s)", totalInserted)
	return nil
}

// parseIntSafe safely parses an integer from string
func parseIntSafe(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

// DeleteAllKeys removes all API keys for a provider (use with caution!)
func (s *Seeder) DeleteAllKeys(ctx context.Context, provider string) error {
	log.Warnf("⚠️  Deleting all API keys for provider: %s", provider)
	return s.mongoClient.DeleteAllAPIKeys(ctx, provider)
}

// ListAllKeys lists all API keys for a provider
func (s *Seeder) ListAllKeys(ctx context.Context, provider string) ([]*APIKey, error) {
	return s.mongoClient.GetAllAPIKeys(ctx, provider)
}

// GetStats returns statistics about API keys
func (s *Seeder) GetStats(ctx context.Context, provider string) (map[string]interface{}, error) {
	keys, err := s.mongoClient.GetAllAPIKeys(ctx, provider)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"provider":      provider,
		"total_keys":    len(keys),
		"active_keys":   0,
		"inactive_keys": 0,
		"total_usage":   int64(0),
	}

	for _, key := range keys {
		if key.IsActive {
			stats["active_keys"] = stats["active_keys"].(int) + 1
		} else {
			stats["inactive_keys"] = stats["inactive_keys"].(int) + 1
		}
		stats["total_usage"] = stats["total_usage"].(int64) + key.UsageCount
	}

	return stats, nil
}
