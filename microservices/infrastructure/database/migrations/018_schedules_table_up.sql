-- Create schedules table for scheduled workflow executions
CREATE TABLE IF NOT EXISTS schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    cron_expression VARCHAR(100) NOT NULL,
    timezone VARCHAR(50) DEFAULT 'UTC',
    is_enabled BOOLEAN DEFAULT true,
    next_run_at TIMESTAMPTZ,
    last_run_at TIMESTAMPTZ,
    last_execution_id UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for efficient queries
CREATE INDEX IF NOT EXISTS idx_schedules_workflow_id ON schedules(workflow_id);
CREATE INDEX IF NOT EXISTS idx_schedules_next_run ON schedules(next_run_at) WHERE is_enabled = true;
CREATE INDEX IF NOT EXISTS idx_schedules_enabled ON schedules(is_enabled);

-- Comment on table
COMMENT ON TABLE schedules IS 'Stores scheduled workflow execution configurations';
COMMENT ON COLUMN schedules.cron_expression IS 'Standard cron expression: minute hour day month weekday';
COMMENT ON COLUMN schedules.timezone IS 'IANA timezone for schedule (e.g., America/New_York)';
