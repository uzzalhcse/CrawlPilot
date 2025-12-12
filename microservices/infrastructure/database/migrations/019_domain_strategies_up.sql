-- Smart Unblocker: Domain Strategies Table
-- Stores learned anti-bot strategies per domain, persisted for future executions
-- Migration: 019_domain_strategies

CREATE TABLE IF NOT EXISTS domain_strategies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain VARCHAR(255) NOT NULL UNIQUE,
    
    -- Tier Settings (Crawlee pattern)
    recommended_tier INTEGER DEFAULT 0,           -- 0=direct, 1=datacenter, 2=residential, 3=mobile
    tier_confidence DECIMAL(3,2) DEFAULT 0,       -- 0.00-1.00 (how sure we are about the tier)
    
    -- Session Settings (anti-bot cookie detection)
    session_requirement INTEGER DEFAULT 0,        -- 0=unknown, 1=required, 2=optional, 3=not_needed
    detected_cookies TEXT[],                      -- Array of detected anti-bot cookie names
    
    -- Rotation Settings
    optimal_rotation_count INTEGER,               -- Rotate proxy every N requests (NULL = no rotation)
    rotation_confidence DECIMAL(3,2) DEFAULT 0,
    
    -- Throttling
    adaptive_delay_ms INTEGER DEFAULT 0,          -- Delay between requests in ms
    max_concurrent_requests INTEGER,              -- Detected concurrency limit (NULL = unknown)
    
    -- Success Metrics (used for learning)
    total_requests BIGINT DEFAULT 0,
    total_successes BIGINT DEFAULT 0,
    total_failures BIGINT DEFAULT 0,
    success_rate DECIMAL(5,4) GENERATED ALWAYS AS (
        CASE WHEN total_requests > 0 THEN total_successes::decimal / total_requests 
        ELSE 0 END
    ) STORED,
    
    -- Learning Status
    learning_status VARCHAR(32) DEFAULT 'new',    -- new, learning, stable, needs_review
    sample_size INTEGER DEFAULT 0,                -- Total requests used for learning
    last_block_at TIMESTAMPTZ,                    -- When was the last block detected
    last_success_at TIMESTAMPTZ,                  -- When was the last success
    
    -- Tier History (what we tried)
    tier_attempts JSONB DEFAULT '{}',             -- {"0": {"attempts": 10, "successes": 2}, "1": {...}}
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_domain_strategies_domain ON domain_strategies(domain);
CREATE INDEX idx_domain_strategies_tier ON domain_strategies(recommended_tier);
CREATE INDEX idx_domain_strategies_status ON domain_strategies(learning_status);

-- Auto-update timestamp
CREATE TRIGGER update_domain_strategies_updated_at
    BEFORE UPDATE ON domain_strategies
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add tier column to proxies table if not exists
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS tier INTEGER DEFAULT 1;

-- Update tier based on existing proxy_type
-- datacenter = 1, residential = 2, mobile = 3
UPDATE proxies SET tier = CASE
    WHEN proxy_type = 'datacenter' THEN 1
    WHEN proxy_type = 'residential' THEN 2
    WHEN proxy_type = 'mobile' THEN 3
    ELSE 1
END WHERE tier IS NULL OR tier = 1;

CREATE INDEX IF NOT EXISTS idx_proxies_tier ON proxies(tier);

-- Comments
COMMENT ON TABLE domain_strategies IS 'Learned anti-bot strategies per domain - persisted for future executions';
COMMENT ON COLUMN domain_strategies.recommended_tier IS 'Proxy tier that works: 0=direct, 1=datacenter, 2=residential, 3=mobile';
COMMENT ON COLUMN domain_strategies.learning_status IS 'new: no data, learning: gathering data, stable: confident, needs_review: issues detected';
COMMENT ON COLUMN proxies.tier IS 'Proxy tier for escalation: 1=datacenter, 2=residential, 3=mobile/premium';
