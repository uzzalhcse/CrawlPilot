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

// RecoveryRule represents a recovery rule from recovery_rules table
type RecoveryRule struct {
	ID           string                 `json:"id" db:"id"`
	Name         string                 `json:"name" db:"name"`
	Description  string                 `json:"description" db:"description"`
	Pattern      string                 `json:"pattern" db:"pattern"`
	Domain       string                 `json:"domain,omitempty" db:"domain"`
	Conditions   map[string]interface{} `json:"conditions,omitempty" db:"conditions"`
	Action       string                 `json:"action" db:"action"`
	Params       map[string]interface{} `json:"params,omitempty" db:"params"`
	Priority     int                    `json:"priority" db:"priority"`
	Enabled      bool                   `json:"enabled" db:"enabled"`
	Source       string                 `json:"source" db:"source"`
	SuccessCount int                    `json:"success_count" db:"success_count"`
	FailureCount int                    `json:"failure_count" db:"failure_count"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

// Proxy represents a proxy from proxies table
type Proxy struct {
	ID           string     `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	Host         string     `json:"host" db:"host"`
	Port         int        `json:"port" db:"port"`
	Username     string     `json:"username,omitempty" db:"username"`
	Password     string     `json:"password,omitempty" db:"password"`
	Protocol     string     `json:"protocol" db:"protocol"`
	Location     string     `json:"location,omitempty" db:"location"`
	Provider     string     `json:"provider,omitempty" db:"provider"`
	Enabled      bool       `json:"enabled" db:"enabled"`
	SuccessCount int        `json:"success_count" db:"success_count"`
	FailureCount int        `json:"failure_count" db:"failure_count"`
	LastUsed     *time.Time `json:"last_used,omitempty" db:"last_used"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
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
	query := `SELECT id, name, description, pattern, domain, conditions, action, params, 
	          priority, enabled, source, success_count, failure_count, created_at, updated_at 
	          FROM recovery_rules ORDER BY priority DESC, created_at`
	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]RecoveryRule, 0)
	for rows.Next() {
		var rule RecoveryRule
		var conditionsJSON, paramsJSON []byte
		var domain *string
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.Description, &rule.Pattern, &domain,
			&conditionsJSON, &rule.Action, &paramsJSON, &rule.Priority, &rule.Enabled,
			&rule.Source, &rule.SuccessCount, &rule.FailureCount, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			continue
		}
		if domain != nil {
			rule.Domain = *domain
		}
		json.Unmarshal(conditionsJSON, &rule.Conditions)
		json.Unmarshal(paramsJSON, &rule.Params)
		rules = append(rules, rule)
	}
	return rules, nil
}

// GetRuleByID returns a rule by ID
func (r *SystemConfigRepository) GetRuleByID(ctx context.Context, id string) (*RecoveryRule, error) {
	query := `SELECT id, name, description, pattern, domain, conditions, action, params, 
	          priority, enabled, source, success_count, failure_count, created_at, updated_at 
	          FROM recovery_rules WHERE id = $1`
	row := r.db.Pool.QueryRow(ctx, query, id)

	var rule RecoveryRule
	var conditionsJSON, paramsJSON []byte
	var domain *string
	if err := row.Scan(&rule.ID, &rule.Name, &rule.Description, &rule.Pattern, &domain,
		&conditionsJSON, &rule.Action, &paramsJSON, &rule.Priority, &rule.Enabled,
		&rule.Source, &rule.SuccessCount, &rule.FailureCount, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
		return nil, err
	}
	if domain != nil {
		rule.Domain = *domain
	}
	json.Unmarshal(conditionsJSON, &rule.Conditions)
	json.Unmarshal(paramsJSON, &rule.Params)
	return &rule, nil
}

// CreateRule creates a new recovery rule
func (r *SystemConfigRepository) CreateRule(ctx context.Context, rule *RecoveryRule) error {
	conditionsJSON, _ := json.Marshal(rule.Conditions)
	paramsJSON, _ := json.Marshal(rule.Params)

	query := `INSERT INTO recovery_rules (id, name, description, pattern, domain, conditions, action, params, priority, enabled, source)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.db.Pool.Exec(ctx, query, rule.ID, rule.Name, rule.Description, rule.Pattern,
		nilIfEmpty(rule.Domain), conditionsJSON, rule.Action, paramsJSON, rule.Priority, rule.Enabled, rule.Source)
	return err
}

// UpdateRule updates a recovery rule
func (r *SystemConfigRepository) UpdateRule(ctx context.Context, rule *RecoveryRule) error {
	conditionsJSON, _ := json.Marshal(rule.Conditions)
	paramsJSON, _ := json.Marshal(rule.Params)

	query := `UPDATE recovery_rules SET name = $1, description = $2, pattern = $3, domain = $4, 
	          conditions = $5, action = $6, params = $7, priority = $8, enabled = $9, updated_at = NOW()
	          WHERE id = $10`
	_, err := r.db.Pool.Exec(ctx, query, rule.Name, rule.Description, rule.Pattern, nilIfEmpty(rule.Domain),
		conditionsJSON, rule.Action, paramsJSON, rule.Priority, rule.Enabled, rule.ID)
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
	query := `SELECT id, name, host, port, username, password, protocol, location, provider,
	          enabled, success_count, failure_count, last_used, created_at, updated_at 
	          FROM proxies ORDER BY enabled DESC, success_count DESC`
	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	proxies := make([]Proxy, 0)
	for rows.Next() {
		var proxy Proxy
		var username, password, location, provider *string
		if err := rows.Scan(&proxy.ID, &proxy.Name, &proxy.Host, &proxy.Port,
			&username, &password, &proxy.Protocol, &location, &provider,
			&proxy.Enabled, &proxy.SuccessCount, &proxy.FailureCount,
			&proxy.LastUsed, &proxy.CreatedAt, &proxy.UpdatedAt); err != nil {
			continue
		}
		if username != nil {
			proxy.Username = *username
		}
		if password != nil {
			proxy.Password = *password
		}
		if location != nil {
			proxy.Location = *location
		}
		if provider != nil {
			proxy.Provider = *provider
		}
		proxies = append(proxies, proxy)
	}
	return proxies, nil
}

// CreateProxy creates a new proxy
func (r *SystemConfigRepository) CreateProxy(ctx context.Context, proxy *Proxy) error {
	query := `INSERT INTO proxies (id, name, host, port, username, password, protocol, location, provider, enabled)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.db.Pool.Exec(ctx, query, proxy.ID, proxy.Name, proxy.Host, proxy.Port,
		nilIfEmpty(proxy.Username), nilIfEmpty(proxy.Password), proxy.Protocol,
		nilIfEmpty(proxy.Location), nilIfEmpty(proxy.Provider), proxy.Enabled)
	return err
}

// UpdateProxy updates a proxy
func (r *SystemConfigRepository) UpdateProxy(ctx context.Context, proxy *Proxy) error {
	query := `UPDATE proxies SET name = $1, host = $2, port = $3, username = $4, password = $5,
	          protocol = $6, location = $7, provider = $8, enabled = $9, updated_at = NOW()
	          WHERE id = $10`
	_, err := r.db.Pool.Exec(ctx, query, proxy.Name, proxy.Host, proxy.Port,
		nilIfEmpty(proxy.Username), nilIfEmpty(proxy.Password), proxy.Protocol,
		nilIfEmpty(proxy.Location), nilIfEmpty(proxy.Provider), proxy.Enabled, proxy.ID)
	return err
}

// DeleteProxy deletes a proxy
func (r *SystemConfigRepository) DeleteProxy(ctx context.Context, id string) error {
	query := `DELETE FROM proxies WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}

// ToggleProxy enables/disables a proxy
func (r *SystemConfigRepository) ToggleProxy(ctx context.Context, id string, enabled bool) error {
	query := `UPDATE proxies SET enabled = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Pool.Exec(ctx, query, enabled, id)
	return err
}

// GetProxyStats returns proxy statistics
func (r *SystemConfigRepository) GetProxyStats(ctx context.Context) (map[string]interface{}, error) {
	query := `SELECT 
	          COUNT(*) as total,
	          COUNT(*) FILTER (WHERE enabled = true) as enabled,
	          SUM(success_count) as total_success,
	          SUM(failure_count) as total_failure
	          FROM proxies`
	row := r.db.Pool.QueryRow(ctx, query)

	var total, enabled int
	var totalSuccess, totalFailure int64
	if err := row.Scan(&total, &enabled, &totalSuccess, &totalFailure); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total":         total,
		"enabled":       enabled,
		"disabled":      total - enabled,
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
