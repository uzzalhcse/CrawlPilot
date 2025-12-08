-- Recovery Attempts Tracking Table
-- Stores each error recovery attempt with immediate recording on detection

CREATE TABLE IF NOT EXISTS recovery_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id VARCHAR(64) NOT NULL,
    task_id VARCHAR(64) NOT NULL,
    workflow_id UUID,  -- Can be NULL if not available during recovery
    url TEXT NOT NULL,
    domain VARCHAR(255) NOT NULL,
    
    -- Error Details
    error_pattern VARCHAR(64) NOT NULL,
    error_message TEXT,
    status_code INTEGER,
    
    -- Recovery Action
    action VARCHAR(64),  -- retry, switch_proxy, add_delay, send_to_dlq, etc.
    source VARCHAR(32) DEFAULT 'pending',  -- rule, ai, default, none, pending
    rule_id VARCHAR(64),
    ai_reasoning TEXT,
    
    -- Status: pending (just detected), success, failed
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    
    -- Timing
    retry_delay_ms INTEGER,
    duration_ms INTEGER,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_recovery_attempts_execution ON recovery_attempts(execution_id);
CREATE INDEX IF NOT EXISTS idx_recovery_attempts_workflow ON recovery_attempts(workflow_id);
CREATE INDEX IF NOT EXISTS idx_recovery_attempts_status ON recovery_attempts(status);
CREATE INDEX IF NOT EXISTS idx_recovery_attempts_pattern ON recovery_attempts(error_pattern);
CREATE INDEX IF NOT EXISTS idx_recovery_attempts_created ON recovery_attempts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_recovery_attempts_domain ON recovery_attempts(domain);

-- Compound index for dashboard (recent pending/failed)
CREATE INDEX IF NOT EXISTS idx_recovery_attempts_status_created 
ON recovery_attempts(status, created_at DESC) 
WHERE status IN ('pending', 'failed');

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_recovery_attempts_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_recovery_attempts_updated_at
    BEFORE UPDATE ON recovery_attempts
    FOR EACH ROW
    EXECUTE FUNCTION update_recovery_attempts_updated_at();

-- Comments
COMMENT ON TABLE recovery_attempts IS 'Stores error recovery attempts with immediate recording on detection';
COMMENT ON COLUMN recovery_attempts.status IS 'pending = just detected, success = recovered, failed = recovery failed';
COMMENT ON COLUMN recovery_attempts.source IS 'Where recovery came from: rule, ai, default, none (when no recovery), pending (initial)';
