package apikey

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrNoKeysAvailable  = errors.New("no API keys available")
	ErrProviderNotFound = errors.New("provider not found")
)

// Manager handles API key rotation and usage tracking
type Manager struct {
	mongo    *MongoClient
	mu       sync.RWMutex
	keyCache map[string][]*APIKey // provider -> keys
	lastSync time.Time
	syncTTL  time.Duration
}

// ManagerConfig holds configuration for the API key manager
type ManagerConfig struct {
	MongoClient *MongoClient
	SyncTTL     time.Duration // how often to refresh keys from DB
}

// NewManager creates a new API key manager
func NewManager(config *ManagerConfig) *Manager {
	if config.SyncTTL == 0 {
		config.SyncTTL = 5 * time.Minute
	}

	return &Manager{
		mongo:    config.MongoClient,
		keyCache: make(map[string][]*APIKey),
		syncTTL:  config.SyncTTL,
	}
}

// SyncKeys loads API keys from MongoDB into cache
func (m *Manager) SyncKeys(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	collection := m.mongo.GetKeysCollection()

	// Find all active keys
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to fetch API keys: %w", err)
	}
	defer cursor.Close(ctx)

	// Clear cache
	m.keyCache = make(map[string][]*APIKey)

	// Load keys into cache
	for cursor.Next(ctx) {
		var key APIKey
		if err := cursor.Decode(&key); err != nil {
			continue
		}

		if m.keyCache[key.Provider] == nil {
			m.keyCache[key.Provider] = make([]*APIKey, 0)
		}
		m.keyCache[key.Provider] = append(m.keyCache[key.Provider], &key)
	}

	m.lastSync = time.Now()
	return nil
}

// shouldSync checks if we need to sync with MongoDB
func (m *Manager) shouldSync() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return time.Since(m.lastSync) > m.syncTTL
}

// GetKey gets the next available API key using round-robin
func (m *Manager) GetKey(ctx context.Context, provider string) (*APIKey, error) {
	// Sync if needed
	if m.shouldSync() {
		if err := m.SyncKeys(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.RLock()
	keys := m.keyCache[provider]
	m.mu.RUnlock()

	if len(keys) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, provider)
	}

	// Find the key with lowest usage count that is available
	var selectedKey *APIKey
	minUsage := int64(-1)

	for _, key := range keys {
		if !key.IsAvailable() {
			continue
		}

		if minUsage == -1 || key.UsageCount < minUsage {
			minUsage = key.UsageCount
			selectedKey = key
		}
	}

	if selectedKey == nil {
		return nil, ErrNoKeysAvailable
	}

	return selectedKey, nil
}

// UseKey marks a key as used and updates usage statistics
func (m *Manager) UseKey(ctx context.Context, keyID primitive.ObjectID, success bool, errorMsg string, responseTimeMs int64) error {
	collection := m.mongo.GetKeysCollection()
	now := time.Now()

	// Update key usage
	update := bson.M{
		"$inc": bson.M{
			"usage_count": 1,
		},
		"$set": bson.M{
			"last_used_at": now,
			"updated_at":   now,
		},
	}

	if success {
		// Reset failure count on success
		update["$set"].(bson.M)["failure_count"] = 0
	} else {
		// Increment failure count on error
		update["$inc"].(bson.M)["failure_count"] = 1
	}

	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": keyID},
		update,
	)
	if err != nil {
		return fmt.Errorf("failed to update key usage: %w", err)
	}

	// Record usage in history
	usageCollection := m.mongo.GetUsageCollection()

	// Get key info for provider
	var key APIKey
	if err := collection.FindOne(ctx, bson.M{"_id": keyID}).Decode(&key); err != nil {
		// Non-critical error, log but continue
		fmt.Printf("Warning: failed to get key info for usage logging: %v\n", err)
	}

	usage := &APIKeyUsage{
		KeyID:        keyID,
		Provider:     key.Provider,
		Timestamp:    now,
		RequestType:  "generation",
		Success:      success,
		ErrorMessage: errorMsg,
		ResponseTime: responseTimeMs,
	}

	_, err = usageCollection.InsertOne(ctx, usage)
	if err != nil {
		// Non-critical error, just log it
		fmt.Printf("Warning: failed to record usage: %v\n", err)
	}

	// Update local cache
	m.mu.Lock()
	if keys, ok := m.keyCache[key.Provider]; ok {
		for _, k := range keys {
			if k.ID == keyID {
				k.UsageCount++
				k.LastUsedAt = &now
				if success {
					k.FailureCount = 0
				} else {
					k.FailureCount++
				}
				break
			}
		}
	}
	m.mu.Unlock()

	return nil
}

// MarkRateLimited marks a key as rate limited
func (m *Manager) MarkRateLimited(ctx context.Context, keyID primitive.ObjectID, duration time.Duration) error {
	collection := m.mongo.GetKeysCollection()
	rateLimitedUntil := time.Now().Add(duration)

	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": keyID},
		bson.M{
			"$set": bson.M{
				"rate_limited_until": rateLimitedUntil,
				"updated_at":         time.Now(),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to mark key as rate limited: %w", err)
	}

	// Update local cache
	m.mu.Lock()
	for _, keys := range m.keyCache {
		for _, k := range keys {
			if k.ID == keyID {
				k.RateLimitedUntil = &rateLimitedUntil
				break
			}
		}
	}
	m.mu.Unlock()

	return nil
}

// GetAllKeys returns all keys for a provider
func (m *Manager) GetAllKeys(ctx context.Context, provider string) ([]*APIKey, error) {
	collection := m.mongo.GetKeysCollection()

	cursor, err := collection.Find(ctx, bson.M{"provider": provider})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch keys: %w", err)
	}
	defer cursor.Close(ctx)

	var keys []*APIKey
	for cursor.Next(ctx) {
		var key APIKey
		if err := cursor.Decode(&key); err != nil {
			continue
		}
		keys = append(keys, &key)
	}

	return keys, nil
}

// AddKey adds a new API key to the database
func (m *Manager) AddKey(ctx context.Context, provider, key string, metadata map[string]string) error {
	collection := m.mongo.GetKeysCollection()

	now := time.Now()
	apiKey := &APIKey{
		Provider:     provider,
		Key:          key,
		IsActive:     true,
		UsageCount:   0,
		FailureCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
		Metadata:     metadata,
	}

	_, err := collection.InsertOne(ctx, apiKey)
	if err != nil {
		return fmt.Errorf("failed to add key: %w", err)
	}

	// Refresh cache
	return m.SyncKeys(ctx)
}

// DeactivateKey deactivates an API key
func (m *Manager) DeactivateKey(ctx context.Context, keyID primitive.ObjectID) error {
	collection := m.mongo.GetKeysCollection()

	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": keyID},
		bson.M{
			"$set": bson.M{
				"is_active":  false,
				"updated_at": time.Now(),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to deactivate key: %w", err)
	}

	// Refresh cache
	return m.SyncKeys(ctx)
}

// GetUsageStats returns usage statistics for a provider
func (m *Manager) GetUsageStats(ctx context.Context, provider string, since time.Time) (map[string]interface{}, error) {
	usageCollection := m.mongo.GetUsageCollection()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "provider", Value: provider},
			{Key: "timestamp", Value: bson.D{{Key: "$gte", Value: since}}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$key_id"},
			{Key: "total_requests", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "successful_requests", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: []interface{}{"$success", 1, 0}}}}}},
			{Key: "failed_requests", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: []interface{}{"$success", 0, 1}}}}}},
			{Key: "avg_response_time", Value: bson.D{{Key: "$avg", Value: "$response_time_ms"}}},
		}}},
	}

	cursor, err := usageCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode usage stats: %w", err)
	}

	return map[string]interface{}{
		"provider": provider,
		"since":    since,
		"keys":     results,
	}, nil
}

// GetNextAvailableKey tries all keys in round-robin fashion until one works
func (m *Manager) GetNextAvailableKey(ctx context.Context, provider string, excludeKeys []primitive.ObjectID) (*APIKey, error) {
	if m.shouldSync() {
		if err := m.SyncKeys(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.RLock()
	keys := m.keyCache[provider]
	m.mu.RUnlock()

	if len(keys) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, provider)
	}

	// Create exclude map for quick lookup
	excludeMap := make(map[primitive.ObjectID]bool)
	for _, id := range excludeKeys {
		excludeMap[id] = true
	}

	// Find the key with lowest usage count that is available and not excluded
	var selectedKey *APIKey
	minUsage := int64(-1)

	for _, key := range keys {
		if !key.IsAvailable() || excludeMap[key.ID] {
			continue
		}

		if minUsage == -1 || key.UsageCount < minUsage {
			minUsage = key.UsageCount
			selectedKey = key
		}
	}

	if selectedKey == nil {
		return nil, ErrNoKeysAvailable
	}

	return selectedKey, nil
}
