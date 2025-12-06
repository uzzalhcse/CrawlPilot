package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/database"
)

// SystemConfigRepository handles system configuration CRUD operations
type SystemConfigRepository struct {
	db *database.DB
}

// ConfigItem represents a configuration item from system_config table
type ConfigItem struct {
	Key         string      `json:"key" db:"key"`
	Value       interface{} `json:"value" db:"value"`
	Description string      `json:"description" db:"description"`
	Category    string      `json:"category" db:"category"`
	Editable    bool        `json:"editable" db:"editable"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
}

// RecoveryRule represents a recovery rule from recovery_rules table (matches actual DB schema)
type RecoveryRule struct {
	ID           string                 `json:"id" db:"id"`
	Name         string                 `json:"name" db:"name"`
	Description  string                 `json:"description" db:"description"`
	Priority     int                    `json:"priority" db:"priority"`
	Enabled      bool                   `json:"enabled" db:"enabled"`
	Pattern      string                 `json:"pattern" db:"pattern"`
	Conditions   []interface{}          `json:"conditions,omitempty" db:"conditions"`
	Action       string                 `json:"action" db:"action"`
	ActionParams map[string]interface{} `json:"action_params,omitempty" db:"action_params"`
	MaxRetries   int                    `json:"max_retries" db:"max_retries"`
	RetryDelay   int                    `json:"retry_delay" db:"retry_delay"`
	IsLearned    bool                   `json:"is_learned" db:"is_learned"`
	LearnedFrom  string                 `json:"learned_from,omitempty" db:"learned_from"`
	SuccessCount int                    `json:"success_count" db:"success_count"`
	FailureCount int                    `json:"failure_count" db:"failure_count"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

// Proxy represents a proxy from proxies table (matches actual DB schema)
type Proxy struct {
	ID             string     `json:"id" db:"id"`
	ProxyID        string     `json:"proxy_id" db:"proxy_id"`
	Server         string     `json:"server" db:"server"`
	Username       string     `json:"username,omitempty" db:"username"`
	Password       string     `json:"password,omitempty" db:"password"`
	ProxyAddress   string     `json:"proxy_address" db:"proxy_address"`
	Port           int        `json:"port" db:"port"`
	Valid          bool       `json:"valid" db:"valid"`
	LastVerified   *time.Time `json:"last_verified,omitempty" db:"last_verified"`
	CountryCode    string     `json:"country_code,omitempty" db:"country_code"`
	CityName       string     `json:"city_name,omitempty" db:"city_name"`
	ASNName        string     `json:"asn_name,omitempty" db:"asn_name"`
	ASNNumber      int        `json:"asn_number,omitempty" db:"asn_number"`
	ConfidenceHigh bool       `json:"confidence_high" db:"confidence_high"`
	ProxyType      string     `json:"proxy_type" db:"proxy_type"`
	FailureCount   int        `json:"failure_count" db:"failure_count"`
	SuccessCount   int        `json:"success_count" db:"success_count"`
	LastUsed       *time.Time `json:"last_used,omitempty" db:"last_used"`
	IsHealthy      bool       `json:"is_healthy" db:"is_healthy"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// NewSystemConfigRepository creates a new system config repository
func NewSystemConfigRepository(db *database.DB) *SystemConfigRepository {
	return &SystemConfigRepository{db: db}
}

// GetAllConfigs returns all configuration items
func (r *SystemConfigRepository) GetAllConfigs(ctx context.Context) ([]ConfigItem, error) {
	query := `SELECT key, value, description, category, editable, updated_at 
	          FROM system_config ORDER BY category, key`
	rows, err := r.db.Pool.Query(ctx, query)
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

// GetConfigsByCategory returns configs for a specific category
func (r *SystemConfigRepository) GetConfigsByCategory(ctx context.Context, category string) ([]ConfigItem, error) {
	query := `SELECT key, value, description, category, editable, updated_at 
	          FROM system_config WHERE category = $1 ORDER BY key`
	rows, err := r.db.Pool.Query(ctx, query, category)
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

// GetConfig returns a single config item
func (r *SystemConfigRepository) GetConfig(ctx context.Context, key string) (*ConfigItem, error) {
	query := `SELECT key, value, description, category, editable, updated_at 
	          FROM system_config WHERE key = $1`
	row := r.db.Pool.QueryRow(ctx, query, key)

	var item ConfigItem
	var valueJSON []byte
	if err := row.Scan(&item.Key, &valueJSON, &item.Description, &item.Category, &item.Editable, &item.UpdatedAt); err != nil {
		return nil, err
	}
	json.Unmarshal(valueJSON, &item.Value)
	return &item, nil
}

// UpdateConfig updates a config value
func (r *SystemConfigRepository) UpdateConfig(ctx context.Context, key string, value interface{}) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}
	query := `UPDATE system_config SET value = $1, updated_at = NOW() WHERE key = $2`
	_, err = r.db.Pool.Exec(ctx, query, valueJSON, key)
	return err
}

// UpdateConfigs updates multiple configs
func (r *SystemConfigRepository) UpdateConfigs(ctx context.Context, updates map[string]interface{}) error {
	for key, value := range updates {
		if err := r.UpdateConfig(ctx, key, value); err != nil {
			return fmt.Errorf("failed to update %s: %w", key, err)
		}
	}
	return nil
}

// =====================================================
// RECOVERY RULES
// =====================================================

// GetAllRules returns all recovery rules
func (r *SystemConfigRepository) GetAllRules(ctx context.Context) ([]RecoveryRule, error) {
	query := `SELECT id, name, description, priority, enabled, pattern, conditions, action, action_params,
	          max_retries, retry_delay, is_learned, learned_from, success_count, failure_count, created_at, updated_at 
	          FROM recovery_rules ORDER BY priority DESC, created_at`
	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]RecoveryRule, 0)
	for rows.Next() {
		var rule RecoveryRule
		var conditionsJSON, actionParamsJSON []byte
		var description, learnedFrom *string
		if err := rows.Scan(&rule.ID, &rule.Name, &description, &rule.Priority, &rule.Enabled,
			&rule.Pattern, &conditionsJSON, &rule.Action, &actionParamsJSON,
			&rule.MaxRetries, &rule.RetryDelay, &rule.IsLearned, &learnedFrom,
			&rule.SuccessCount, &rule.FailureCount, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			continue
		}
		if description != nil {
			rule.Description = *description
		}
		if learnedFrom != nil {
			rule.LearnedFrom = *learnedFrom
		}
		json.Unmarshal(conditionsJSON, &rule.Conditions)
		json.Unmarshal(actionParamsJSON, &rule.ActionParams)
		rules = append(rules, rule)
	}
	return rules, nil
}

// GetRuleByID returns a rule by ID
func (r *SystemConfigRepository) GetRuleByID(ctx context.Context, id string) (*RecoveryRule, error) {
	query := `SELECT id, name, description, priority, enabled, pattern, conditions, action, action_params,
	          max_retries, retry_delay, is_learned, learned_from, success_count, failure_count, created_at, updated_at 
	          FROM recovery_rules WHERE id = $1`
	row := r.db.Pool.QueryRow(ctx, query, id)

	var rule RecoveryRule
	var conditionsJSON, actionParamsJSON []byte
	var description, learnedFrom *string
	if err := row.Scan(&rule.ID, &rule.Name, &description, &rule.Priority, &rule.Enabled,
		&rule.Pattern, &conditionsJSON, &rule.Action, &actionParamsJSON,
		&rule.MaxRetries, &rule.RetryDelay, &rule.IsLearned, &learnedFrom,
		&rule.SuccessCount, &rule.FailureCount, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
		return nil, err
	}
	if description != nil {
		rule.Description = *description
	}
	if learnedFrom != nil {
		rule.LearnedFrom = *learnedFrom
	}
	json.Unmarshal(conditionsJSON, &rule.Conditions)
	json.Unmarshal(actionParamsJSON, &rule.ActionParams)
	return &rule, nil
}

// CreateRule creates a new recovery rule
func (r *SystemConfigRepository) CreateRule(ctx context.Context, rule *RecoveryRule) error {
	conditionsJSON, _ := json.Marshal(rule.Conditions)
	actionParamsJSON, _ := json.Marshal(rule.ActionParams)

	query := `INSERT INTO recovery_rules (id, name, description, priority, enabled, pattern, conditions, action, action_params, max_retries, retry_delay, is_learned)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := r.db.Pool.Exec(ctx, query, rule.ID, rule.Name, nilIfEmpty(rule.Description), rule.Priority, rule.Enabled,
		rule.Pattern, string(conditionsJSON), rule.Action, string(actionParamsJSON), rule.MaxRetries, rule.RetryDelay, rule.IsLearned)
	return err
}

// UpdateRule updates a recovery rule
func (r *SystemConfigRepository) UpdateRule(ctx context.Context, rule *RecoveryRule) error {
	conditionsJSON, _ := json.Marshal(rule.Conditions)
	actionParamsJSON, _ := json.Marshal(rule.ActionParams)

	query := `UPDATE recovery_rules SET name = $1, description = $2, priority = $3, enabled = $4, pattern = $5,
	          conditions = $6, action = $7, action_params = $8, max_retries = $9, retry_delay = $10, updated_at = NOW()
	          WHERE id = $11`
	_, err := r.db.Pool.Exec(ctx, query, rule.Name, nilIfEmpty(rule.Description), rule.Priority, rule.Enabled, rule.Pattern,
		string(conditionsJSON), rule.Action, string(actionParamsJSON), rule.MaxRetries, rule.RetryDelay, rule.ID)
	return err
}

// DeleteRule deletes a recovery rule
func (r *SystemConfigRepository) DeleteRule(ctx context.Context, id string) error {
	query := `DELETE FROM recovery_rules WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}

// ToggleRule enables/disables a rule
func (r *SystemConfigRepository) ToggleRule(ctx context.Context, id string, enabled bool) error {
	query := `UPDATE recovery_rules SET enabled = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Pool.Exec(ctx, query, enabled, id)
	return err
}

// =====================================================
// PROXIES
// =====================================================

// GetAllProxies returns all proxies
func (r *SystemConfigRepository) GetAllProxies(ctx context.Context) ([]Proxy, error) {
	query := `SELECT id, proxy_id, server, username, password, proxy_address, port, valid,
	          last_verified, country_code, city_name, asn_name, asn_number, confidence_high,
	          proxy_type, failure_count, success_count, last_used, is_healthy, created_at, updated_at 
	          FROM proxies ORDER BY is_healthy DESC, success_count DESC`
	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	proxies := make([]Proxy, 0)
	for rows.Next() {
		var proxy Proxy
		var username, password, countryCode, cityName, asnName *string
		var asnNumber *int
		if err := rows.Scan(&proxy.ID, &proxy.ProxyID, &proxy.Server, &username, &password,
			&proxy.ProxyAddress, &proxy.Port, &proxy.Valid, &proxy.LastVerified,
			&countryCode, &cityName, &asnName, &asnNumber, &proxy.ConfidenceHigh,
			&proxy.ProxyType, &proxy.FailureCount, &proxy.SuccessCount,
			&proxy.LastUsed, &proxy.IsHealthy, &proxy.CreatedAt, &proxy.UpdatedAt); err != nil {
			continue
		}
		if username != nil {
			proxy.Username = *username
		}
		if password != nil {
			proxy.Password = *password
		}
		if countryCode != nil {
			proxy.CountryCode = *countryCode
		}
		if cityName != nil {
			proxy.CityName = *cityName
		}
		if asnName != nil {
			proxy.ASNName = *asnName
		}
		if asnNumber != nil {
			proxy.ASNNumber = *asnNumber
		}
		proxies = append(proxies, proxy)
	}
	return proxies, nil
}

// CreateProxy creates a new proxy
func (r *SystemConfigRepository) CreateProxy(ctx context.Context, proxy *Proxy) error {
	query := `INSERT INTO proxies (id, proxy_id, server, username, password, proxy_address, port, proxy_type, is_healthy)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.Pool.Exec(ctx, query, proxy.ID, proxy.ProxyID, proxy.Server,
		nilIfEmpty(proxy.Username), nilIfEmpty(proxy.Password), proxy.ProxyAddress,
		proxy.Port, proxy.ProxyType, true)
	return err
}

// UpdateProxy updates a proxy
func (r *SystemConfigRepository) UpdateProxy(ctx context.Context, proxy *Proxy) error {
	query := `UPDATE proxies SET server = $1, username = $2, password = $3, proxy_address = $4,
	          port = $5, proxy_type = $6, is_healthy = $7, updated_at = NOW()
	          WHERE id = $8`
	_, err := r.db.Pool.Exec(ctx, query, proxy.Server,
		nilIfEmpty(proxy.Username), nilIfEmpty(proxy.Password), proxy.ProxyAddress,
		proxy.Port, proxy.ProxyType, proxy.IsHealthy, proxy.ID)
	return err
}

// DeleteProxy deletes a proxy
func (r *SystemConfigRepository) DeleteProxy(ctx context.Context, id string) error {
	query := `DELETE FROM proxies WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}

// ToggleProxy enables/disables a proxy
func (r *SystemConfigRepository) ToggleProxy(ctx context.Context, id string, isHealthy bool) error {
	query := `UPDATE proxies SET is_healthy = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Pool.Exec(ctx, query, isHealthy, id)
	return err
}

// GetProxyStats returns proxy statistics
func (r *SystemConfigRepository) GetProxyStats(ctx context.Context) (map[string]interface{}, error) {
	query := `SELECT 
	          COUNT(*) as total,
	          COUNT(*) FILTER (WHERE is_healthy = true) as healthy,
	          COALESCE(SUM(success_count), 0) as total_success,
	          COALESCE(SUM(failure_count), 0) as total_failure
	          FROM proxies`
	row := r.db.Pool.QueryRow(ctx, query)

	var total, healthy int
	var totalSuccess, totalFailure int64
	if err := row.Scan(&total, &healthy, &totalSuccess, &totalFailure); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total":         total,
		"healthy":       healthy,
		"unhealthy":     total - healthy,
		"total_success": totalSuccess,
		"total_failure": totalFailure,
	}, nil
}

// nilIfEmpty returns nil if string is empty, otherwise returns pointer to string
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
