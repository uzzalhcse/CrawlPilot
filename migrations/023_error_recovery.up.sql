CREATE TABLE IF NOT EXISTS error_recovery_configs (
    config_key VARCHAR(255) PRIMARY KEY,
    config_value JSONB NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS context_aware_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    priority INTEGER NOT NULL DEFAULT 0,
    conditions JSONB NOT NULL,
    context JSONB NOT NULL,
    actions JSONB NOT NULL,
    confidence DOUBLE PRECISION DEFAULT 0,
    success_rate DOUBLE PRECISION DEFAULT 0,
    usage_count INTEGER DEFAULT 0,
    created_by VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS error_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_type VARCHAR(50) NOT NULL,
    pattern_data JSONB NOT NULL,
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index for faster rule matching
CREATE INDEX idx_context_aware_rules_priority ON context_aware_rules(priority DESC);
