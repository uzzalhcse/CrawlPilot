-- Migration: Add probe execution support
-- Up migration for probe system

-- Add probe columns to workflow_executions table
ALTER TABLE workflow_executions ADD COLUMN IF NOT EXISTS is_probe BOOLEAN DEFAULT FALSE;
ALTER TABLE workflow_executions ADD COLUMN IF NOT EXISTS probe_status VARCHAR(20);

-- Index for filtering probe executions
CREATE INDEX IF NOT EXISTS idx_executions_is_probe ON workflow_executions(is_probe) WHERE is_probe = TRUE;

-- Probe results table stores detailed probe outcome
CREATE TABLE IF NOT EXISTS probe_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL, -- healthy, degraded, broken
    duration_ms INTEGER,
    phases JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_probe_results_workflow ON probe_results(workflow_id);
CREATE INDEX IF NOT EXISTS idx_probe_results_created ON probe_results(created_at DESC);

-- Sample URLs table for probe execution (stores last successful URL per phase)
CREATE TABLE IF NOT EXISTS probe_sample_urls (
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    phase_id VARCHAR(100) NOT NULL,
    sample_url TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (workflow_id, phase_id)
);
