package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/queue"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/recovery/llm"
	"go.uber.org/zap"
)

// RecoveryManager orchestrates error recovery using rules and AI
// It implements the Manager interface
type RecoveryManager struct {
	detector         *ErrorDetector
	ruleEngine       *RuleEngine
	agent            *Agent
	learning         *LearningSystem
	domainHealth     *DomainHealth
	proxyManager     *ProxyManager            // Local proxy manager (deprecated, use distributed)
	distributedProxy *DistributedProxyManager // Redis-based distributed proxy rotation
	errorTracker     *ErrorTracker            // Smart triggering with sliding window
	configManager    *ConfigManager           // Dynamic config from database (frontend-manageable)
	incidentReporter *IncidentReporter        // Creates reports for human investigation
	pubsubClient     *queue.PubSubClient
	cache            *cache.Cache // Redis cache for distributed state

	config *ManagerConfig
}

// ManagerConfig holds configuration for the recovery manager
type ManagerConfig struct {
	Enabled             bool
	MaxRecoveryAttempts int
	AIFallbackEnabled   bool
	LLMConfig           llm.Config

	// Smart triggering settings
	WindowSize           int     // Number of results to track (default: 100)
	ErrorRateThreshold   float64 // Trigger if error rate exceeds this (default: 0.10)
	ConsecutiveThreshold int     // Trigger after N consecutive errors (default: 3)
}

// DefaultManagerConfig returns sensible defaults
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		Enabled:              true,
		MaxRecoveryAttempts:  3,
		AIFallbackEnabled:    true,
		WindowSize:           100,
		ErrorRateThreshold:   0.10, // 10%
		ConsecutiveThreshold: 3,
		LLMConfig: llm.Config{
			Provider: "ollama",
			Model:    "qwen2.5",
			Endpoint: "http://localhost:11434",
			Timeout:  30,
		},
	}
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(
	pool *pgxpool.Pool,
	cache *cache.Cache,
	pubsubClient *queue.PubSubClient,
	config *ManagerConfig,
) (*RecoveryManager, error) {
	// Initialize config manager for dynamic DB-based settings
	configManager := NewConfigManager(pool)

	// Use DB config if no explicit config provided
	if config == nil {
		ctx := context.Background()
		config = configManager.GetManagerConfig(ctx)
	}

	if !config.Enabled {
		return &RecoveryManager{config: config, configManager: configManager}, nil
	}

	// Initialize components
	detector := NewErrorDetector()

	ruleEngine, err := NewRuleEngine(pool)
	if err != nil {
		return nil, fmt.Errorf("failed to create rule engine: %w", err)
	}

	learning := NewLearningSystem(pool)
	domainHealth := NewDomainHealth(cache)

	proxyManager, err := NewProxyManager(pool)
	if err != nil {
		logger.Warn("Failed to initialize proxy manager", zap.Error(err))
		// Continue without proxy manager
	}

	// Initialize AI agent if enabled
	var agent *Agent
	if config.AIFallbackEnabled {
		factory := llm.NewProviderFactory()
		provider, err := factory.Create(config.LLMConfig)
		if err != nil {
			logger.Warn("Failed to create LLM provider, AI fallback disabled", zap.Error(err))
		} else {
			agent = NewAgent(provider, nil)
		}
	}

	// Initialize error tracker for smart triggering
	trackerConfig := &TrackerConfig{
		WindowSize:           config.WindowSize,
		ErrorRateThreshold:   config.ErrorRateThreshold,
		ConsecutiveThreshold: config.ConsecutiveThreshold,
		WindowTTL:            1 * time.Hour,
	}
	errorTracker := NewErrorTracker(cache, trackerConfig)

	// Initialize incident reporter for human escalation
	incidentReporter := NewIncidentReporter(pool)

	// Initialize distributed proxy manager for Redis-based coordination
	distributedProxy := NewDistributedProxyManager(cache, DefaultProxyRotationConfig())

	// Sync proxies from database to Redis for distributed coordination
	if distributedProxy != nil && pool != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Load proxies from database
			query := `SELECT id, proxy_id, server, username, password, proxy_address, port, 
			          valid, last_verified, country_code, city_name, asn_name, asn_number,
			          confidence_high, proxy_type, failure_count, success_count, 
			          last_used, is_healthy, created_at, updated_at
			          FROM proxies WHERE valid = true AND is_healthy = true`

			rows, err := pool.Query(ctx, query)
			if err != nil {
				logger.Warn("Failed to load proxies from database", zap.Error(err))
				return
			}
			defer rows.Close()

			proxies := make([]Proxy, 0)
			for rows.Next() {
				var p Proxy
				var lastVerified, lastUsed, createdAt, updatedAt *time.Time
				err := rows.Scan(&p.ID, &p.ProxyID, &p.Server, &p.Username, &p.Password, &p.ProxyAddress, &p.Port,
					&p.Valid, &lastVerified, &p.CountryCode, &p.CityName, &p.ASNName, &p.ASNNumber,
					&p.ConfidenceHigh, &p.ProxyType, &p.FailureCount, &p.SuccessCount,
					&lastUsed, &p.IsHealthy, &createdAt, &updatedAt)
				if err != nil {
					continue
				}
				if lastVerified != nil {
					p.LastVerified = *lastVerified
				}
				if lastUsed != nil {
					p.LastUsed = *lastUsed
				}
				if createdAt != nil {
					p.CreatedAt = *createdAt
				}
				if updatedAt != nil {
					p.UpdatedAt = *updatedAt
				}
				proxies = append(proxies, p)
			}

			if len(proxies) > 0 {
				if err := distributedProxy.SeedProxies(ctx, proxies); err != nil {
					logger.Error("Failed to seed proxies to Redis", zap.Error(err))
				} else {
					logger.Info("Proxies auto-synced from database to Redis",
						zap.Int("count", len(proxies)),
					)
				}
			}
		}()
	}

	logger.Info("Recovery manager initialized",
		zap.Bool("ai_fallback", agent != nil),
		zap.Bool("proxy_manager", proxyManager != nil),
		zap.Bool("distributed_proxy", distributedProxy != nil),
		zap.Bool("incident_reporter", incidentReporter != nil),
		zap.String("llm_provider", config.LLMConfig.Provider),
		zap.Int("window_size", config.WindowSize),
		zap.Float64("error_rate_threshold", config.ErrorRateThreshold),
		zap.Int("consecutive_threshold", config.ConsecutiveThreshold),
	)

	return &RecoveryManager{
		detector:         detector,
		ruleEngine:       ruleEngine,
		agent:            agent,
		learning:         learning,
		domainHealth:     domainHealth,
		proxyManager:     proxyManager,
		distributedProxy: distributedProxy,
		errorTracker:     errorTracker,
		configManager:    configManager,
		incidentReporter: incidentReporter,
		pubsubClient:     pubsubClient,
		cache:            cache,
		config:           config,
	}, nil
}

// TryRecover attempts to recover from an error
// Uses smart triggering: only activates when error rate exceeds threshold OR consecutive errors occur
func (m *RecoveryManager) TryRecover(ctx context.Context, taskID, executionID, url string, err error, pageContent string) (*RecoveryPlan, error) {
	if !m.config.Enabled {
		return nil, nil
	}

	// Detect error pattern first
	detected := m.detector.Detect(err, url, 0, pageContent)
	if detected == nil {
		// Not an error or undetectable - record as success
		if m.errorTracker != nil {
			m.errorTracker.RecordSuccess(ctx, extractDomain(url))
		}
		return nil, nil
	}

	// Record failure and check if recovery should trigger (smart triggering)
	if m.errorTracker != nil {
		shouldTrigger, reason := m.errorTracker.RecordFailure(ctx, detected.Domain, detected.Pattern)
		if !shouldTrigger {
			// Error rate/consecutive errors below threshold - don't trigger recovery yet
			logger.Debug("Error recorded but recovery not triggered",
				zap.String("domain", detected.Domain),
				zap.String("pattern", string(detected.Pattern)),
				zap.String("reason", "thresholds not met"),
			)
			return nil, nil
		}
		logger.Info("Recovery triggered",
			zap.String("domain", detected.Domain),
			zap.String("pattern", string(detected.Pattern)),
			zap.String("trigger_reason", reason),
		)
	}

	// Check recovery attempt limit for this specific task
	history := m.getHistory(taskID)
	if len(history) >= m.config.MaxRecoveryAttempts {
		logger.Warn("Max recovery attempts reached",
			zap.String("task_id", taskID),
			zap.Int("attempts", len(history)),
		)
		return &RecoveryPlan{
			Action:      ActionSendToDLQ,
			Reason:      "Max recovery attempts exceeded",
			ShouldRetry: false,
			Source:      "system",
		}, nil
	}

	// Check domain health
	if healthy, _ := m.domainHealth.IsHealthy(ctx, detected.Domain); !healthy {
		return &RecoveryPlan{
			Action:      ActionSkipDomain,
			Reason:      "Domain is currently blocked",
			ShouldRetry: false,
			Source:      "domain_health",
		}, nil
	}

	// Step 1: Try rule-based recovery
	if rule := m.ruleEngine.Match(ctx, detected); rule != nil {
		plan := m.ruleEngine.ToRecoveryPlan(rule)
		m.recordAttempt(taskID, executionID, detected, plan)
		return plan, nil
	}

	// Step 2: Try AI fallback if enabled
	if m.agent != nil && m.config.AIFallbackEnabled {
		plan, reasoning, err := m.agent.Analyze(ctx, detected, history)
		if err != nil {
			logger.Warn("AI analysis failed", zap.Error(err))
		} else if plan != nil {
			// Record for learning
			learnedAction := &LearnedAction{
				ID:           generateID(),
				ExecutionID:  executionID,
				TaskID:       taskID,
				ErrorPattern: detected.Pattern,
				Domain:       detected.Domain,
				Action:       plan.Action,
				ActionParams: plan.Params,
				AIReasoning:  reasoning,
				Success:      false, // Will be updated after execution
			}
			if err := m.learning.RecordAIAction(ctx, learnedAction); err != nil {
				logger.Warn("Failed to record AI action for learning", zap.Error(err))
			}
			plan.Params["learning_id"] = learnedAction.ID

			m.recordAttempt(taskID, executionID, detected, plan)
			return plan, nil
		}
	}

	// Step 3: Use default action based on pattern
	defaultPlan := &RecoveryPlan{
		Action:      GetRecommendedAction(detected.Pattern),
		Reason:      "Default action for " + string(detected.Pattern),
		ShouldRetry: detected.Pattern != PatternCaptcha && detected.Pattern != PatternAuthRequired,
		RetryDelay:  5 * time.Second,
		Source:      "default",
	}

	m.recordAttempt(taskID, executionID, detected, defaultPlan)
	return defaultPlan, nil
}

// RecordOutcome records whether a recovery attempt succeeded
func (m *RecoveryManager) RecordOutcome(ctx context.Context, attempt *RecoveryAttempt) error {
	if !m.config.Enabled {
		return nil
	}

	// Update rule stats if came from a rule
	if attempt.Plan.RuleID != "" {
		m.ruleEngine.RecordRuleOutcome(ctx, attempt.Plan.RuleID, attempt.Success)
	}

	// Update learning if came from AI
	if attempt.Plan.Source == "ai" {
		if learningID, ok := attempt.Plan.Params["learning_id"].(string); ok {
			m.learning.UpdateOutcome(ctx, learningID, attempt.Success)
		}
	}

	// Update domain health
	domain := extractDomain(attempt.DetectedError.URL)
	if attempt.Success {
		m.domainHealth.RecordSuccess(ctx, domain, "")
	} else {
		m.domainHealth.RecordFailure(ctx, domain, attempt.DetectedError.Pattern)
	}

	return nil
}

// ExecutePlan executes a recovery plan
func (m *RecoveryManager) ExecutePlan(ctx context.Context, plan *RecoveryPlan, taskURL string) error {
	domain := extractDomain(taskURL)

	switch plan.Action {
	case ActionSwitchProxy:
		// Prefer distributed proxy manager (Redis-based)
		if m.distributedProxy != nil {
			proxy, lease, err := m.distributedProxy.GetProxy(ctx, domain)
			if err != nil {
				return err
			}
			plan.Params["proxy_url"] = proxy.ProxyURL()
			plan.Params["proxy_id"] = proxy.ID
			if lease != nil {
				plan.Params["lease_expires"] = lease.ExpiresAt
			}
		} else if m.proxyManager != nil {
			// Fallback to local proxy manager
			proxy, err := m.proxyManager.GetNext(ctx)
			if err != nil {
				return err
			}
			plan.Params["proxy_url"] = proxy.ProxyURL()
			plan.Params["proxy_id"] = proxy.ID
		}

	case ActionAddDelay:
		if plan.RetryDelay > 0 {
			time.Sleep(plan.RetryDelay)
		}

	case ActionSkipDomain:
		duration := 30 * time.Minute
		if d, ok := plan.Params["duration"].(time.Duration); ok {
			duration = d
		}
		m.domainHealth.Block(ctx, domain, duration, plan.Reason)

	case ActionSendToDLQ:
		// Already handled by caller

	case ActionRetry:
		if clearCookies, ok := plan.Params["clear_cookies"].(bool); ok && clearCookies {
			plan.Params["clear_cookies"] = true
		}
	}

	return nil
}

// GetDomainStatus returns the health status of a domain
func (m *RecoveryManager) GetDomainStatus(ctx context.Context, domain string) (*DomainStatus, error) {
	if m.domainHealth == nil {
		return nil, fmt.Errorf("domain health not initialized")
	}
	return m.domainHealth.Get(ctx, domain)
}

// RefreshRules reloads rules from the database
func (m *RecoveryManager) RefreshRules(ctx context.Context) error {
	if m.ruleEngine == nil {
		return nil
	}
	return m.ruleEngine.Refresh(ctx)
}

// GetProxy returns a proxy for the given domain using distributed coordination
// This ensures even distribution across all workers in the cluster
func (m *RecoveryManager) GetProxy(ctx context.Context, domain string) (*Proxy, error) {
	// Prefer distributed proxy manager (Redis-based)
	if m.distributedProxy != nil {
		proxy, _, err := m.distributedProxy.GetProxy(ctx, domain)
		return proxy, err
	}

	// Fallback to local proxy manager (single-worker only)
	if m.proxyManager == nil {
		return nil, fmt.Errorf("proxy manager not initialized")
	}
	return m.proxyManager.GetForDomain(ctx, domain)
}

// GetProxyWithLease returns a proxy with its lease for tracking
func (m *RecoveryManager) GetProxyWithLease(ctx context.Context, domain string) (*Proxy, *ProxyLease, error) {
	if m.distributedProxy == nil {
		return nil, nil, fmt.Errorf("distributed proxy manager not initialized")
	}
	return m.distributedProxy.GetProxy(ctx, domain)
}

// RecordProxySuccess records a successful request with a proxy
func (m *RecoveryManager) RecordProxySuccess(ctx context.Context, proxyID, domain string) error {
	// Use distributed proxy manager for Redis-based tracking
	if m.distributedProxy != nil {
		return m.distributedProxy.RecordSuccess(ctx, proxyID, domain)
	}

	// Fallback to local proxy manager
	if m.proxyManager == nil {
		return nil
	}
	return m.proxyManager.RecordSuccess(ctx, proxyID, domain)
}

// RecordProxyFailure records a failed request with a proxy
func (m *RecoveryManager) RecordProxyFailure(ctx context.Context, proxyID, domain string, pattern ErrorPattern) error {
	// Use distributed proxy manager for Redis-based tracking
	if m.distributedProxy != nil {
		return m.distributedProxy.RecordFailure(ctx, proxyID, domain, pattern)
	}

	// Fallback to local proxy manager
	if m.proxyManager == nil {
		return nil
	}
	return m.proxyManager.RecordFailure(ctx, proxyID, domain, pattern)
}

// GetDistributedProxyManager returns the distributed proxy manager for direct access
func (m *RecoveryManager) GetDistributedProxyManager() *DistributedProxyManager {
	return m.distributedProxy
}

// GetDistributedProxyStats returns stats from the distributed proxy manager
func (m *RecoveryManager) GetDistributedProxyStats(ctx context.Context) (map[string]interface{}, error) {
	if m.distributedProxy == nil {
		return nil, nil
	}
	return m.distributedProxy.GetStats(ctx)
}

// Close cleans up resources
func (m *RecoveryManager) Close() error {
	if m.agent != nil {
		m.agent.Close()
	}
	return nil
}

// historyKeyFor returns the Redis key for task recovery history
func (m *RecoveryManager) historyKeyFor(taskID string) string {
	return fmt.Sprintf("recovery:history:%s", taskID)
}

// getHistory returns recovery history for a task from Redis
func (m *RecoveryManager) getHistory(taskID string) []*RecoveryAttempt {
	if m.cache == nil {
		return []*RecoveryAttempt{}
	}

	ctx := context.Background()
	key := m.historyKeyFor(taskID)
	data, err := m.cache.Get(ctx, key)
	if err != nil || data == "" {
		return []*RecoveryAttempt{}
	}

	var attempts []*RecoveryAttempt
	if err := json.Unmarshal([]byte(data), &attempts); err != nil {
		return []*RecoveryAttempt{}
	}

	return attempts
}

// recordAttempt records a recovery attempt to Redis
func (m *RecoveryManager) recordAttempt(taskID, executionID string, detected *DetectedError, plan *RecoveryPlan) {
	attempt := &RecoveryAttempt{
		ID:            generateID(),
		TaskID:        taskID,
		ExecutionID:   executionID,
		DetectedError: detected,
		Plan:          plan,
		Timestamp:     time.Now(),
	}

	if m.cache == nil {
		return
	}

	ctx := context.Background()
	key := m.historyKeyFor(taskID)

	// Get existing history
	history := m.getHistory(taskID)
	history = append(history, attempt)

	// Save to Redis with 1 hour TTL
	data, err := json.Marshal(history)
	if err != nil {
		return
	}

	m.cache.Set(ctx, key, string(data), 1*time.Hour)
}

// ClearHistory clears recovery history for a task from Redis
func (m *RecoveryManager) ClearHistory(taskID string) {
	if m.cache == nil {
		return
	}

	ctx := context.Background()
	key := m.historyKeyFor(taskID)
	m.cache.Delete(ctx, key)
}

// RecordSuccess records a successful task execution (for error rate tracking)
func (m *RecoveryManager) RecordSuccess(ctx context.Context, url string) {
	if m.errorTracker != nil {
		domain := extractDomain(url)
		m.errorTracker.RecordSuccess(ctx, domain)
	}
}

// GetErrorStats returns error statistics for a domain
func (m *RecoveryManager) GetErrorStats(ctx context.Context, domain string) map[string]interface{} {
	if m.errorTracker == nil {
		return nil
	}
	return m.errorTracker.GetStats(ctx, domain)
}

// GetAllErrorStats returns error statistics for all domains
func (m *RecoveryManager) GetAllErrorStats(ctx context.Context) []map[string]interface{} {
	if m.errorTracker == nil {
		return nil
	}
	return m.errorTracker.GetAllStats(ctx)
}

// SeedProxies seeds proxies from JSON data into both local and distributed managers
func (m *RecoveryManager) SeedProxies(ctx context.Context, jsonData []byte) error {
	// Parse JSON
	var proxies []Proxy
	if err := json.Unmarshal(jsonData, &proxies); err != nil {
		return fmt.Errorf("failed to parse proxy JSON: %w", err)
	}

	// Seed to distributed proxy manager (Redis)
	if m.distributedProxy != nil {
		if err := m.distributedProxy.SeedProxies(ctx, proxies); err != nil {
			logger.Error("Failed to seed proxies to Redis", zap.Error(err))
			// Don't return error, try local fallback
		} else {
			logger.Info("Proxies seeded to distributed proxy manager",
				zap.Int("count", len(proxies)),
			)
		}
	}

	// Also seed to local manager (fallback)
	if m.proxyManager != nil {
		return m.proxyManager.SeedFromJSON(ctx, jsonData)
	}

	return nil
}

// SyncProxiesFromDB loads proxies from database and seeds them to Redis
// Call this on worker startup to ensure Redis has all available proxies
func (m *RecoveryManager) SyncProxiesFromDB(ctx context.Context, proxies []Proxy) error {
	if m.distributedProxy == nil {
		return fmt.Errorf("distributed proxy manager not initialized")
	}

	if len(proxies) == 0 {
		logger.Warn("No proxies to sync from database")
		return nil
	}

	if err := m.distributedProxy.SeedProxies(ctx, proxies); err != nil {
		return fmt.Errorf("failed to sync proxies to Redis: %w", err)
	}

	logger.Info("Proxies synced from database to Redis",
		zap.Int("count", len(proxies)),
	)

	return nil
}

// GetProxyCount returns the number of available proxies
func (m *RecoveryManager) GetProxyCount() int {
	if m.proxyManager == nil {
		return 0
	}
	return m.proxyManager.Count()
}

// GetHealthyProxyCount returns the number of healthy proxies
func (m *RecoveryManager) GetHealthyProxyCount() int {
	if m.proxyManager == nil {
		return 0
	}
	return m.proxyManager.CountHealthy()
}

// =====================================================
// FRONTEND CONFIG MANAGEMENT METHODS
// These methods expose configuration for frontend APIs
// =====================================================

// GetConfigManager returns the config manager for direct access
func (m *RecoveryManager) GetConfigManager() *ConfigManager {
	return m.configManager
}

// GetAllConfigs returns all configuration items for frontend display
func (m *RecoveryManager) GetAllConfigs(ctx context.Context) ([]ConfigItem, error) {
	if m.configManager == nil {
		return nil, fmt.Errorf("config manager not initialized")
	}
	return m.configManager.GetAll(ctx)
}

// GetConfigsByCategory returns configs for a specific category
func (m *RecoveryManager) GetConfigsByCategory(ctx context.Context, category string) ([]ConfigItem, error) {
	if m.configManager == nil {
		return nil, fmt.Errorf("config manager not initialized")
	}
	return m.configManager.GetByCategory(ctx, category)
}

// UpdateConfig updates a single config value (for frontend API)
func (m *RecoveryManager) UpdateConfig(ctx context.Context, key string, value interface{}) error {
	if m.configManager == nil {
		return fmt.Errorf("config manager not initialized")
	}
	return m.configManager.Set(ctx, key, value)
}

// RefreshConfig reloads all configs from database
func (m *RecoveryManager) RefreshConfig(ctx context.Context) error {
	if m.configManager == nil {
		return nil
	}

	// Refresh config from DB
	if err := m.configManager.Refresh(ctx); err != nil {
		return err
	}

	// Update internal config (for sliding window, AI, etc.)
	m.config = m.configManager.GetManagerConfig(ctx)

	// Update error tracker config if settings changed
	if m.errorTracker != nil {
		newTrackerConfig := m.configManager.GetTrackerConfig(ctx)
		m.errorTracker.UpdateConfig(
			newTrackerConfig.WindowSize,
			newTrackerConfig.ErrorRateThreshold,
			newTrackerConfig.ConsecutiveThreshold,
		)
	}

	logger.Info("Recovery configuration refreshed from database",
		zap.Int("window_size", m.config.WindowSize),
		zap.Float64("error_rate_threshold", m.config.ErrorRateThreshold),
		zap.Int("consecutive_threshold", m.config.ConsecutiveThreshold),
	)

	return nil
}

// =====================================================
// INCIDENT REPORTING METHODS
// Creates reports for human investigation when recovery fails
// =====================================================

// CreateIncident creates an incident report for human investigation
// Call this when all recovery attempts have failed
func (m *RecoveryManager) CreateIncident(
	ctx context.Context,
	taskID, executionID, workflowID, url string,
	detected *DetectedError,
	aiReasoning, aiFailure string,
	snapshot *PageSnapshot,
) (*IncidentReport, error) {
	if m.incidentReporter == nil {
		logger.Warn("Incident reporter not initialized")
		return nil, nil
	}

	// Get recovery history for this task
	attempts := m.getHistory(taskID)

	return m.incidentReporter.CreateFromRecoveryFailure(
		ctx, taskID, executionID, workflowID, url,
		detected, attempts, aiReasoning, aiFailure, snapshot,
	)
}

// GetIncidentReporter returns the incident reporter for direct access
func (m *RecoveryManager) GetIncidentReporter() *IncidentReporter {
	return m.incidentReporter
}

// GetIncidentStats returns incident statistics
func (m *RecoveryManager) GetIncidentStats(ctx context.Context) (map[string]interface{}, error) {
	if m.incidentReporter == nil {
		return nil, nil
	}
	return m.incidentReporter.GetIncidentStats(ctx)
}

// GetOpenIncidents returns all open incidents for dashboard
func (m *RecoveryManager) GetOpenIncidents(ctx context.Context) ([]IncidentReport, error) {
	if m.incidentReporter == nil {
		return nil, nil
	}
	return m.incidentReporter.GetOpenIncidents(ctx)
}

// ResolveIncident marks an incident as resolved
func (m *RecoveryManager) ResolveIncident(ctx context.Context, incidentID, resolution string) error {
	if m.incidentReporter == nil {
		return nil
	}
	return m.incidentReporter.UpdateStatus(ctx, incidentID, IncidentStatusResolved, resolution)
}
