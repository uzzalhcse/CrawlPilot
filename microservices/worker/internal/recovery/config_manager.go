package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/recovery/llm"
	"go.uber.org/zap"
)

// ConfigManager manages system configuration from database
// All settings are manageable from frontend via API
type ConfigManager struct {
	pool     *pgxpool.Pool
	cache    map[string]interface{}
	mu       sync.RWMutex
	lastLoad time.Time
	cacheTTL time.Duration
}

// ConfigItem represents a configuration item
type ConfigItem struct {
	Key         string      `json:"key" db:"key"`
	Value       interface{} `json:"value" db:"value"`
	Description string      `json:"description" db:"description"`
	Category    string      `json:"category" db:"category"`
	Editable    bool        `json:"editable" db:"editable"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
}

// NewConfigManager creates a new config manager
func NewConfigManager(pool *pgxpool.Pool) *ConfigManager {
	cm := &ConfigManager{
		pool:     pool,
		cache:    make(map[string]interface{}),
		cacheTTL: 1 * time.Minute, // Refresh every minute
	}

	// Load initial config
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cm.Refresh(ctx)

	return cm
}

// Refresh reloads all config from database
func (cm *ConfigManager) Refresh(ctx context.Context) error {
	query := `SELECT key, value FROM system_config`
	rows, err := cm.pool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	defer rows.Close()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	for rows.Next() {
		var key string
		var valueJSON []byte
		if err := rows.Scan(&key, &valueJSON); err != nil {
			continue
		}

		var value interface{}
		if err := json.Unmarshal(valueJSON, &value); err != nil {
			// Try as raw string
			value = string(valueJSON)
		}
		cm.cache[key] = value
	}

	cm.lastLoad = time.Now()
	logger.Debug("Config refreshed from database", zap.Int("items", len(cm.cache)))
	return nil
}

// ensureFresh refreshes cache if stale
func (cm *ConfigManager) ensureFresh(ctx context.Context) {
	if time.Since(cm.lastLoad) > cm.cacheTTL {
		go cm.Refresh(ctx)
	}
}

// Get returns a config value
func (cm *ConfigManager) Get(ctx context.Context, key string) interface{} {
	cm.ensureFresh(ctx)
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.cache[key]
}

// GetString returns a string config value
func (cm *ConfigManager) GetString(ctx context.Context, key, defaultVal string) string {
	val := cm.Get(ctx, key)
	if val == nil {
		return defaultVal
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", val)
}

// GetInt returns an int config value
func (cm *ConfigManager) GetInt(ctx context.Context, key string, defaultVal int) int {
	val := cm.Get(ctx, key)
	if val == nil {
		return defaultVal
	}
	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

// GetFloat returns a float config value
func (cm *ConfigManager) GetFloat(ctx context.Context, key string, defaultVal float64) float64 {
	val := cm.Get(ctx, key)
	if val == nil {
		return defaultVal
	}
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

// GetBool returns a bool config value
func (cm *ConfigManager) GetBool(ctx context.Context, key string, defaultVal bool) bool {
	val := cm.Get(ctx, key)
	if val == nil {
		return defaultVal
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1" || v == "yes"
	}
	return defaultVal
}

// Set updates a config value
func (cm *ConfigManager) Set(ctx context.Context, key string, value interface{}) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}

	query := `UPDATE system_config SET value = $1, updated_at = NOW() WHERE key = $2`
	_, err = cm.pool.Exec(ctx, query, valueJSON, key)
	if err != nil {
		return err
	}

	// Update cache
	cm.mu.Lock()
	cm.cache[key] = value
	cm.mu.Unlock()

	return nil
}

// GetAll returns all config items
func (cm *ConfigManager) GetAll(ctx context.Context) ([]ConfigItem, error) {
	query := `SELECT key, value, description, category, editable, updated_at FROM system_config ORDER BY category, key`
	rows, err := cm.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ConfigItem, 0)
	for rows.Next() {
		var item ConfigItem
		var valueJSON []byte
		if err := rows.Scan(&item.Key, &valueJSON, &item.Description, &item.Category, &item.Editable, &item.UpdatedAt); err != nil {
			continue
		}
		json.Unmarshal(valueJSON, &item.Value)
		items = append(items, item)
	}

	return items, nil
}

// GetByCategory returns config items for a category
func (cm *ConfigManager) GetByCategory(ctx context.Context, category string) ([]ConfigItem, error) {
	query := `SELECT key, value, description, category, editable, updated_at FROM system_config WHERE category = $1 ORDER BY key`
	rows, err := cm.pool.Query(ctx, query, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ConfigItem, 0)
	for rows.Next() {
		var item ConfigItem
		var valueJSON []byte
		if err := rows.Scan(&item.Key, &valueJSON, &item.Description, &item.Category, &item.Editable, &item.UpdatedAt); err != nil {
			continue
		}
		json.Unmarshal(valueJSON, &item.Value)
		items = append(items, item)
	}

	return items, nil
}

// GetManagerConfig builds ManagerConfig from database settings
func (cm *ConfigManager) GetManagerConfig(ctx context.Context) *ManagerConfig {
	return &ManagerConfig{
		Enabled:              cm.GetBool(ctx, "recovery.enabled", true),
		MaxRecoveryAttempts:  cm.GetInt(ctx, "recovery.max_attempts", 3),
		AIFallbackEnabled:    cm.GetBool(ctx, "ai.enabled", true),
		WindowSize:           cm.GetInt(ctx, "recovery.window_size", 100),
		ErrorRateThreshold:   cm.GetFloat(ctx, "recovery.error_rate_threshold", 0.10),
		ConsecutiveThreshold: cm.GetInt(ctx, "recovery.consecutive_threshold", 3),
		LLMConfig: llm.Config{
			Provider: cm.GetString(ctx, "ai.provider", "ollama"),
			Model:    cm.GetString(ctx, "ai.model", "qwen2.5"),
			Endpoint: cm.GetString(ctx, "ai.endpoint", "http://localhost:11434"),
			Timeout:  cm.GetInt(ctx, "ai.timeout", 30),
		},
	}
}

// GetTrackerConfig builds TrackerConfig from database settings
func (cm *ConfigManager) GetTrackerConfig(ctx context.Context) *TrackerConfig {
	return &TrackerConfig{
		WindowSize:           cm.GetInt(ctx, "recovery.window_size", 100),
		ErrorRateThreshold:   cm.GetFloat(ctx, "recovery.error_rate_threshold", 0.10),
		ConsecutiveThreshold: cm.GetInt(ctx, "recovery.consecutive_threshold", 3),
		WindowTTL:            1 * time.Hour,
	}
}

// GetLearningPromotionThreshold returns the learning promotion threshold
func (cm *ConfigManager) GetLearningPromotionThreshold(ctx context.Context) int {
	return cm.GetInt(ctx, "learning.promotion_threshold", 3)
}

// IsProxyEnabled returns whether proxy rotation is enabled
func (cm *ConfigManager) IsProxyEnabled(ctx context.Context) bool {
	return cm.GetBool(ctx, "proxy.enabled", true)
}

// GetProxyMaxFailures returns proxy max failures before disable
func (cm *ConfigManager) GetProxyMaxFailures(ctx context.Context) int {
	return cm.GetInt(ctx, "proxy.max_failures_before_disable", 5)
}
