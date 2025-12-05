-- Recovery System Tables Migration
-- Adds tables for system config, proxies, recovery rules, and learned AI actions

-- =====================================================
-- SYSTEM CONFIG TABLE
-- Stores all configurable settings (manageable from frontend)
-- =====================================================
CREATE TABLE IF NOT EXISTS system_config (
    key VARCHAR(128) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    category VARCHAR(64) NOT NULL,
    editable BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_system_config_category ON system_config(category);

-- Insert default recovery settings (all manageable from frontend)
INSERT INTO system_config (key, value, description, category) VALUES
    -- Smart Triggering Settings
    ('recovery.enabled', 'true', 'Enable/disable the entire error recovery system', 'recovery'),
    ('recovery.window_size', '100', 'Number of recent results to track per domain for error rate calculation', 'recovery'),
    ('recovery.error_rate_threshold', '0.10', 'Trigger recovery when error rate exceeds this (0.10 = 10%)', 'recovery'),
    ('recovery.consecutive_threshold', '3', 'Trigger recovery after this many consecutive errors', 'recovery'),
    ('recovery.max_attempts', '3', 'Maximum recovery attempts per task before sending to DLQ', 'recovery'),
    
    -- AI Agent Settings
    ('ai.enabled', 'true', 'Enable AI agent as fallback when no rules match', 'ai'),
    ('ai.provider', '"ollama"', 'LLM provider: "ollama" or "openai"', 'ai'),
    ('ai.model', '"qwen2.5"', 'Model to use for AI analysis', 'ai'),
    ('ai.endpoint', '"http://localhost:11434"', 'LLM API endpoint', 'ai'),
    ('ai.timeout', '30', 'Timeout in seconds for AI requests', 'ai'),
    
    -- Learning System
    ('learning.enabled', 'true', 'Enable learning from successful AI recoveries', 'learning'),
    ('learning.promotion_threshold', '3', 'Promote to rule after N successful uses', 'learning'),
    ('learning.cleanup_days', '7', 'Delete failed learned actions older than N days', 'learning'),
    
    -- Proxy Settings
    ('proxy.enabled', 'true', 'Enable proxy rotation', 'proxy'),
    ('proxy.rotation_strategy', '"round_robin"', 'Strategy: "round_robin", "random", "least_failed"', 'proxy'),
    ('proxy.health_check_interval', '300', 'Seconds between proxy health checks', 'proxy'),
    ('proxy.max_failures_before_disable', '5', 'Disable proxy after N consecutive failures', 'proxy'),
    
    -- Domain Health Settings
    ('domain.block_duration_base', '60', 'Base block duration in seconds', 'domain'),
    ('domain.consecutive_fails_to_block', '5', 'Block domain after N consecutive failures', 'domain'),
    ('domain.max_block_duration', '3600', 'Maximum block duration in seconds (1 hour)', 'domain')
ON CONFLICT (key) DO NOTHING;

-- =====================================================
-- PROXIES TABLE
-- Stores proxy configurations loaded from JSON/DB
-- =====================================================
CREATE TABLE IF NOT EXISTS proxies (
    id VARCHAR(64) PRIMARY KEY,
    proxy_id VARCHAR(64) UNIQUE NOT NULL,
    server VARCHAR(255) NOT NULL,
    username VARCHAR(128),
    password VARCHAR(128),
    proxy_address VARCHAR(64) NOT NULL,
    port INTEGER NOT NULL,
    valid BOOLEAN DEFAULT true,
    last_verified TIMESTAMPTZ,
    country_code VARCHAR(8),
    city_name VARCHAR(128),
    asn_name VARCHAR(128),
    asn_number INTEGER,
    confidence_high BOOLEAN DEFAULT false,
    proxy_type VARCHAR(32) DEFAULT 'static',
    failure_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    last_used TIMESTAMPTZ,
    is_healthy BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_proxies_valid_healthy ON proxies(valid, is_healthy);
CREATE INDEX IF NOT EXISTS idx_proxies_country ON proxies(country_code);
CREATE INDEX IF NOT EXISTS idx_proxies_failure_count ON proxies(failure_count);

-- =====================================================
-- RECOVERY RULES TABLE
-- Stores configurable recovery rules (from frontend + learned)
-- =====================================================
CREATE TABLE IF NOT EXISTS recovery_rules (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    priority INTEGER DEFAULT 100,
    enabled BOOLEAN DEFAULT true,
    pattern VARCHAR(64) NOT NULL,
    conditions JSONB DEFAULT '[]'::jsonb,
    action VARCHAR(64) NOT NULL,
    action_params JSONB DEFAULT '{}'::jsonb,
    max_retries INTEGER DEFAULT 3,
    retry_delay INTEGER DEFAULT 5,
    is_learned BOOLEAN DEFAULT false,
    learned_from VARCHAR(64),
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recovery_rules_enabled ON recovery_rules(enabled);
CREATE INDEX IF NOT EXISTS idx_recovery_rules_priority ON recovery_rules(priority);
CREATE INDEX IF NOT EXISTS idx_recovery_rules_pattern ON recovery_rules(pattern);
CREATE INDEX IF NOT EXISTS idx_recovery_rules_learned ON recovery_rules(is_learned);

-- =====================================================
-- LEARNED ACTIONS TABLE
-- Tracks AI-suggested actions for learning and promotion
-- =====================================================
CREATE TABLE IF NOT EXISTS learned_actions (
    id VARCHAR(64) PRIMARY KEY,
    execution_id VARCHAR(64) NOT NULL,
    task_id VARCHAR(64) NOT NULL,
    error_pattern VARCHAR(64) NOT NULL,
    error_signature VARCHAR(64) NOT NULL,
    domain VARCHAR(255) NOT NULL,
    action VARCHAR(64) NOT NULL,
    action_params JSONB DEFAULT '{}'::jsonb,
    ai_reasoning TEXT,
    success BOOLEAN DEFAULT false,
    promoted_to_rule BOOLEAN DEFAULT false,
    rule_id VARCHAR(64) REFERENCES recovery_rules(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_learned_actions_signature ON learned_actions(error_signature);
CREATE INDEX IF NOT EXISTS idx_learned_actions_success ON learned_actions(success);
CREATE INDEX IF NOT EXISTS idx_learned_actions_domain ON learned_actions(domain);
CREATE INDEX IF NOT EXISTS idx_learned_actions_promoted ON learned_actions(promoted_to_rule);

-- =====================================================
-- SAMPLE RULES (Common patterns)
-- =====================================================
INSERT INTO recovery_rules (id, name, description, priority, enabled, pattern, conditions, action, action_params, max_retries)
VALUES
    ('rule-blocked-default', 'Default Blocked Handler', 'Switch proxy when blocked is detected', 10, true, 'blocked', '[]', 'switch_proxy', '{"reason": "IP blocked by target site"}', 3),
    ('rule-rate-limit-default', 'Default Rate Limit Handler', 'Add delay when rate limited', 10, true, 'rate_limited', '[]', 'add_delay', '{"seconds": 30, "reason": "Rate limit detected"}', 3),
    ('rule-captcha-default', 'Default Captcha Handler', 'Send to DLQ on captcha', 10, true, 'captcha', '[]', 'send_to_dlq', '{"category": "captcha", "reason": "CAPTCHA detected, requires human intervention"}', 1),
    ('rule-timeout-default', 'Default Timeout Handler', 'Retry on timeout', 20, true, 'timeout', '[]', 'retry', '{"reason": "Timeout - transient error"}', 3),
    ('rule-server-error', 'Default Server Error Handler', 'Retry on server error', 20, true, 'server_error', '[]', 'retry', '{"reason": "Server error - will likely resolve"}', 3)
ON CONFLICT (id) DO NOTHING;
