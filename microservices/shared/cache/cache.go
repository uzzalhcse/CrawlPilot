package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// Cache wraps Redis client
type Cache struct {
	client *redis.Client
}

// NewCache creates a new Redis cache client
func NewCache(cfg *config.RedisConfig) (*Cache, error) {
	if !cfg.Enabled {
		logger.Warn("Redis cache is disabled")
		return nil, fmt.Errorf("redis cache is disabled in config")
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Address(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     100,
		MinIdleConns: 10,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Redis cache connected",
		zap.String("address", cfg.Address()),
		zap.Int("db", cfg.DB),
	)

	return &Cache{client: client}, nil
}

// Close closes the Redis connection
func (c *Cache) Close() error {
	return c.client.Close()
}

// Get retrieves a value from cache
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Set stores a value in cache
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// SetNX sets a value only if it doesn't exist (for deduplication)
func (c *Cache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	return c.client.SetNX(ctx, key, value, ttl).Result()
}

// SetNXMulti executes multiple SetNX commands in a single pipeline
// Returns a slice of booleans indicating which keys were successfully set (new)
// Use this for batch deduplication to reduce N round-trips to 1
func (c *Cache) SetNXMulti(ctx context.Context, keys []string, value interface{}, ttl time.Duration) ([]bool, error) {
	if len(keys) == 0 {
		return []bool{}, nil
	}

	pipe := c.client.Pipeline()
	cmds := make([]*redis.BoolCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.SetNX(ctx, key, value, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	results := make([]bool, len(keys))
	for i, cmd := range cmds {
		results[i] = cmd.Val()
	}

	return results, nil
}

// Delete removes a key from cache
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, key).Result()
	return count > 0, err
}

// GetJSON retrieves and unmarshals JSON from cache
func (c *Cache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := c.Get(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// SetJSON marshals and stores JSON in cache
func (c *Cache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.Set(ctx, key, data, ttl)
}

// Increment increments a counter
func (c *Cache) Increment(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// Decrement decrements a counter
func (c *Cache) Decrement(ctx context.Context, key string) (int64, error) {
	return c.client.Decr(ctx, key).Result()
}

// IncrBy increments a counter by a specific amount
func (c *Cache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.IncrBy(ctx, key, value).Result()
}

// IncrByAndExpire atomically increments a counter and sets TTL in one round-trip
// Returns the new value after increment
func (c *Cache) IncrByAndExpire(ctx context.Context, key string, value int64, ttl time.Duration) (int64, error) {
	pipe := c.client.Pipeline()
	incrCmd := pipe.IncrBy(ctx, key, value)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incrCmd.Val(), nil
}

// DecrBy decrements a counter by a specific amount
func (c *Cache) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.DecrBy(ctx, key, value).Result()
}

// Expire sets a TTL on an existing key
func (c *Cache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

// Keys returns all keys matching a pattern
// Note: Use sparingly in production as SCAN can be slow
func (c *Cache) Keys(ctx context.Context, pattern string) ([]string, error) {
	var keys []string
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

// =====================================================
// SORTED SET OPERATIONS (for distributed proxy rotation)
// =====================================================

// ZAdd adds a member to a sorted set with a score
func (c *Cache) ZAdd(ctx context.Context, key string, score float64, member string) error {
	return c.client.ZAdd(ctx, key, redis.Z{Score: score, Member: member}).Err()
}

// ZIncrBy increments the score of a member in a sorted set
func (c *Cache) ZIncrBy(ctx context.Context, key string, increment float64, member string) error {
	return c.client.ZIncrBy(ctx, key, increment, member).Err()
}

// ZRangeByScore returns members with scores in the given range (ascending)
func (c *Cache) ZRangeByScore(ctx context.Context, key string, min, max int, limit int) ([]string, error) {
	return c.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min:   fmt.Sprintf("%d", min),
		Max:   fmt.Sprintf("%d", max),
		Count: int64(limit),
	}).Result()
}

// ZRevRangeByScore returns members with scores in the given range (descending)
func (c *Cache) ZRevRangeByScore(ctx context.Context, key string, max, min int, limit int) ([]string, error) {
	return c.client.ZRevRangeByScore(ctx, key, &redis.ZRangeBy{
		Min:   fmt.Sprintf("%d", min),
		Max:   fmt.Sprintf("%d", max),
		Count: int64(limit),
	}).Result()
}

// ZScore returns the score of a member in a sorted set
func (c *Cache) ZScore(ctx context.Context, key, member string) (float64, error) {
	return c.client.ZScore(ctx, key, member).Result()
}

// ZRem removes a member from a sorted set
func (c *Cache) ZRem(ctx context.Context, key string, members ...string) error {
	args := make([]interface{}, len(members))
	for i, m := range members {
		args[i] = m
	}
	return c.client.ZRem(ctx, key, args...).Err()
}

// ZCard returns the number of members in a sorted set
func (c *Cache) ZCard(ctx context.Context, key string) (int64, error) {
	return c.client.ZCard(ctx, key).Result()
}

// =====================================================
// HASH OPERATIONS (for proxy health metrics)
// =====================================================

// HSet sets a field in a hash
func (c *Cache) HSet(ctx context.Context, key, field string, value interface{}) error {
	return c.client.HSet(ctx, key, field, value).Err()
}

// HGet gets a field from a hash
func (c *Cache) HGet(ctx context.Context, key, field string) (string, error) {
	return c.client.HGet(ctx, key, field).Result()
}

// HGetAll gets all fields from a hash
func (c *Cache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.client.HGetAll(ctx, key).Result()
}

// HIncrBy increments a hash field by the given amount
func (c *Cache) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	return c.client.HIncrBy(ctx, key, field, incr).Result()
}

// HDel deletes a field from a hash
func (c *Cache) HDel(ctx context.Context, key string, fields ...string) error {
	return c.client.HDel(ctx, key, fields...).Err()
}

// =====================================================
// SET OPERATIONS (for domain tracking)
// =====================================================

// SAdd adds members to a set
func (c *Cache) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return c.client.SAdd(ctx, key, members...).Err()
}

// SMembers returns all members of a set
func (c *Cache) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.client.SMembers(ctx, key).Result()
}

// SCard returns the number of members in a set
func (c *Cache) SCard(ctx context.Context, key string) (int64, error) {
	return c.client.SCard(ctx, key).Result()
}

// =====================================================
// LIST OPERATIONS (for atomic append to recovery history)
// =====================================================

// RPush appends values to the end of a list (atomic operation)
func (c *Cache) RPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return c.client.RPush(ctx, key, values...).Result()
}

// LRange returns a range of elements from a list
// Use 0, -1 to get all elements
func (c *Cache) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.LRange(ctx, key, start, stop).Result()
}

// LLen returns the length of a list
func (c *Cache) LLen(ctx context.Context, key string) (int64, error) {
	return c.client.LLen(ctx, key).Result()
}

// =====================================================
// PIPELINE OPERATIONS (for high-throughput batching)
// =====================================================

// Pipeliner wraps Redis pipeline for batched commands
type Pipeliner struct {
	pipe redis.Pipeliner
}

// Pipeline creates a new pipeline for batching commands
// Use Exec() to execute all commands in a single round-trip
func (c *Cache) Pipeline() *Pipeliner {
	return &Pipeliner{pipe: c.client.Pipeline()}
}

// HIncrBy queues a hash increment in the pipeline
func (p *Pipeliner) HIncrBy(ctx context.Context, key, field string, incr int64) {
	p.pipe.HIncrBy(ctx, key, field, incr)
}

// HSet queues a hash set in the pipeline
func (p *Pipeliner) HSet(ctx context.Context, key, field string, value interface{}) {
	p.pipe.HSet(ctx, key, field, value)
}

// Incr queues an increment in the pipeline
func (p *Pipeliner) Incr(ctx context.Context, key string) {
	p.pipe.Incr(ctx, key)
}

// Expire queues an expire in the pipeline
func (p *Pipeliner) Expire(ctx context.Context, key string, ttl time.Duration) {
	p.pipe.Expire(ctx, key, ttl)
}

// Del queues a delete in the pipeline
func (p *Pipeliner) Del(ctx context.Context, keys ...string) {
	p.pipe.Del(ctx, keys...)
}

// SAdd queues a set add in the pipeline
func (p *Pipeliner) SAdd(ctx context.Context, key string, members ...interface{}) {
	p.pipe.SAdd(ctx, key, members...)
}

// ZIncrBy queues a sorted set increment in the pipeline
func (p *Pipeliner) ZIncrBy(ctx context.Context, key string, incr float64, member string) {
	p.pipe.ZIncrBy(ctx, key, incr, member)
}

// Set queues a string set in the pipeline
func (p *Pipeliner) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	p.pipe.Set(ctx, key, value, ttl)
}

// SetNX queues a set-if-not-exists in the pipeline
func (p *Pipeliner) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	p.pipe.SetNX(ctx, key, value, ttl)
}

// Exec executes all queued commands in a single round-trip
// Returns the number of completed commands and any error
func (p *Pipeliner) Exec(ctx context.Context) (int, error) {
	cmds, err := p.pipe.Exec(ctx)
	return len(cmds), err
}
