-- Add phase_stats to workflow_executions
ALTER TABLE workflow_executions 
  ADD COLUMN IF NOT EXISTS phase_stats JSONB DEFAULT '{}';

-- Add triggered_by column to track how execution was started
ALTER TABLE workflow_executions 
  ADD COLUMN IF NOT EXISTS triggered_by VARCHAR(50) DEFAULT 'manual';

-- Execution errors table for batched error logging
CREATE TABLE IF NOT EXISTS execution_errors (
    id BIGSERIAL PRIMARY KEY,
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    error_type VARCHAR(50),      -- timeout, blocked, parse_error, network, extraction
    message TEXT,
    phase_id VARCHAR(100),
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_execution_errors_execution_id ON execution_errors(execution_id);
CREATE INDEX idx_execution_errors_created_at ON execution_errors(created_at DESC);
CREATE INDEX idx_execution_errors_type ON execution_errors(error_type);

-- Comment
COMMENT ON TABLE execution_errors IS 'Stores error logs for workflow executions, batched writes for performance';
COMMENT ON COLUMN workflow_executions.phase_stats IS 'JSON object with per-phase statistics: {phase_id: {processed: N, errors: N, duration_ms: N}}';
