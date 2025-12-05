package apikey

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoConfig holds MongoDB connection configuration
type MongoConfig struct {
	URI      string
	Database string
	Timeout  time.Duration
}

// MongoClient wraps MongoDB client with helper methods
type MongoClient struct {
	client   *mongo.Client
	database *mongo.Database
	config   *MongoConfig
}

// NewMongoClient creates a new MongoDB client
func NewMongoClient(ctx context.Context, config *MongoConfig) (*MongoClient, error) {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	clientOptions := options.Client().ApplyURI(config.URI)

	ctx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return &MongoClient{
		client:   client,
		database: client.Database(config.Database),
		config:   config,
	}, nil
}

// GetKeysCollection returns the API keys collection
func (m *MongoClient) GetKeysCollection() *mongo.Collection {
	return m.database.Collection("api_keys")
}

// GetUsageCollection returns the usage statistics collection
func (m *MongoClient) GetUsageCollection() *mongo.Collection {
	return m.database.Collection("api_key_usage")
}

// Close closes the MongoDB connection
func (m *MongoClient) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

// CreateIndexes creates necessary indexes for optimal performance
func (m *MongoClient) CreateIndexes(ctx context.Context) error {
	keysCollection := m.GetKeysCollection()
	usageCollection := m.GetUsageCollection()

	// Index for API keys collection
	keyIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "provider", Value: 1},
				{Key: "is_active", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "provider", Value: 1},
				{Key: "usage_count", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "rate_limited_until", Value: 1},
			},
			Options: options.Index().SetSparse(true),
		},
		{
			Keys: bson.D{
				{Key: "provider", Value: 1},
				{Key: "key", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	}

	if _, err := keysCollection.Indexes().CreateMany(ctx, keyIndexes); err != nil {
		return fmt.Errorf("failed to create key indexes: %w", err)
	}

	// Index for usage collection
	usageIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "key_id", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "provider", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
	}

	if _, err := usageCollection.Indexes().CreateMany(ctx, usageIndexes); err != nil {
		return fmt.Errorf("failed to create usage indexes: %w", err)
	}

	return nil
}

// InsertAPIKey inserts a new API key into the database
func (m *MongoClient) InsertAPIKey(ctx context.Context, key *APIKey) error {
	collection := m.GetKeysCollection()

	_, err := collection.InsertOne(ctx, key)
	if err != nil {
		// Check if it's a duplicate key error
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("API key already exists for provider %s", key.Provider)
		}
		return fmt.Errorf("failed to insert API key: %w", err)
	}

	return nil
}

// GetAPIKey retrieves a specific API key by provider and key value
func (m *MongoClient) GetAPIKey(ctx context.Context, provider, key string) (*APIKey, error) {
	collection := m.GetKeysCollection()

	var apiKey APIKey
	err := collection.FindOne(ctx, bson.M{
		"provider": provider,
		"key":      key,
	}).Decode(&apiKey)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Key not found
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return &apiKey, nil
}

// GetAllAPIKeys retrieves all API keys for a provider
func (m *MongoClient) GetAllAPIKeys(ctx context.Context, provider string) ([]*APIKey, error) {
	collection := m.GetKeysCollection()

	filter := bson.M{}
	if provider != "" {
		filter["provider"] = provider
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
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

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return keys, nil
}

// DeleteAllAPIKeys deletes all API keys for a provider (use with caution!)
func (m *MongoClient) DeleteAllAPIKeys(ctx context.Context, provider string) error {
	collection := m.GetKeysCollection()

	filter := bson.M{}
	if provider != "" {
		filter["provider"] = provider
	}

	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete API keys: %w", err)
	}

	fmt.Printf("Deleted %d API key(s) for provider: %s\n", result.DeletedCount, provider)
	return nil
}

// UpdateAPIKey updates an existing API key
func (m *MongoClient) UpdateAPIKey(ctx context.Context, key *APIKey) error {
	collection := m.GetKeysCollection()

	key.UpdatedAt = time.Now()

	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": key.ID},
		bson.M{"$set": key},
	)

	if err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}

	return nil
}
