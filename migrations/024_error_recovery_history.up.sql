-- Create error_recovery_history table for tracking all recovery events
CREATE TABLE error_recovery_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID,
    workflow_id UUID,
    
    -- Error Details
    error_type VARCHAR(255) NOT NULL,
    error_message TEXT NOT NULL,
    status_code INTEGER,
    url TEXT NOT NULL,
    domain VARCHAR(255),
    node_id VARCHAR(255),
    phase_id VARCHAR(255),
    
    -- Pattern Analysis
    pattern_detected BOOLEAN DEFAULT FALSE,
    pattern_type VARCHAR(50),  -- 'rate_spike', 'consecutive', 'systematic', 'critical'
    activation_reason TEXT,
    error_rate DECIMAL(5,4),   -- e.g., 0.1523 for 15.23%
    
    -- Recovery Solution
    rule_id UUID,
    rule_name VARCHAR(255),
    solution_type VARCHAR(50) NOT NULL,  -- 'rule', 'ai', 'none'
    confidence DECIMAL(3,2),
    
    -- Actions Applied
    actions_applied JSONB,  -- Array of actions with parameters
    
    -- Outcome
    recovery_attempted BOOLEAN DEFAULT TRUE,
    recovery_successful BOOLEAN,
    retry_count INTEGER DEFAULT 1,
    time_to_recovery_ms INTEGER,
    
    -- Full Context (for debugging)
    request_context JSONB,
    
    -- Timestamps
    detected_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    recovered_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for efficient querying
CREATE INDEX idx_recovery_history_execution ON error_recovery_history(execution_id);
CREATE INDEX idx_recovery_history_workflow ON error_recovery_history(workflow_id);
CREATE INDEX idx_recovery_history_domain ON error_recovery_history(domain);
CREATE INDEX idx_recovery_history_rule ON error_recovery_history(rule_id);
CREATE INDEX idx_recovery_history_detected_at ON error_recovery_history(detected_at DESC);
CREATE INDEX idx_recovery_history_successful ON error_recovery_history(recovery_successful);
CREATE INDEX idx_recovery_history_error_type ON error_recovery_history(error_type);
CREATE INDEX idx_recovery_history_solution_type ON error_recovery_history(solution_type);

-- Composite index for common queries
CREATE INDEX idx_recovery_history_workflow_detected ON error_recovery_history(workflow_id, detected_at DESC);

-- View for aggregated statistics
CREATE VIEW error_recovery_stats AS
SELECT 
    DATE_TRUNC('hour', detected_at) as time_bucket,
    domain,
    error_type,
    rule_name,
    solution_type,
    COUNT(*) as total_attempts,
    SUM(CASE WHEN recovery_successful THEN 1 ELSE 0 END) as successful_recoveries,
    ROUND(AVG(time_to_recovery_ms), 2) as avg_recovery_time_ms,
    ROUND(AVG(retry_count), 2) as avg_retries,
    ROUND(AVG(confidence), 3) as avg_confidence
FROM error_recovery_history
WHERE recovery_attempted = true
GROUP BY DATE_TRUNC('hour', detected_at), domain, error_type, rule_name, solution_type;

-- View for rule performance  
CREATE VIEW rule_performance AS
SELECT 
    h.rule_id,
    h.rule_name,
    COUNT(h.id) as times_used,
    SUM(CASE WHEN h.recovery_successful THEN 1 ELSE 0 END) as successful_uses,
    ROUND(
        SUM(CASE WHEN h.recovery_successful THEN 1 ELSE 0 END)::DECIMAL / NULLIF(COUNT(h.id), 0),
        4
    ) as actual_success_rate,
    ROUND(AVG(h.time_to_recovery_ms), 2) as avg_recovery_time_ms,
    MAX(h.detected_at) as last_used_at
FROM error_recovery_history h
WHERE h.rule_name IS NOT NULL
GROUP BY h.rule_id, h.rule_name;

COMMENT ON TABLE error_recovery_history IS 'Tracks all error recovery attempts with full context and outcomes';
COMMENT ON COLUMN error_recovery_history.pattern_detected IS 'Whether pattern analyzer triggered recovery';
COMMENT ON COLUMN error_recovery_history.solution_type IS 'How error was resolved: rule-based, AI, or no solution';
COMMENT ON COLUMN error_recovery_history.time_to_recovery_ms IS 'Time from error detection to recovery completion';
COMMENT ON VIEW error_recovery_stats IS 'Hourly aggregated statistics for recovery performance';
COMMENT ON VIEW rule_performance IS 'Performance metrics for each recovery rule';
