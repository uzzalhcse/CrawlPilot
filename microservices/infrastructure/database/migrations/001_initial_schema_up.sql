-- Crawlify Microservices Database Schema
-- Migration: 001_initial_schema
-- Description: Create initial tables for workflows, executions, and task tracking
-- Version: 1.0.0

-- Workflows table
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    config JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT workflows_status_check CHECK (status IN ('active', 'inactive'))
);

CREATE INDEX idx_workflows_status ON workflows(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_workflows_created_at ON workflows(created_at DESC);

-- Workflow executions table
CREATE TABLE IF NOT EXISTS workflow_executions (
    id UUID PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'running',
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP NULL,
    metadata JSONB DEFAULT '{}',
    
    -- Stats columns
    urls_processed INTEGER DEFAULT 0,
    urls_discovered INTEGER DEFAULT 0,
    items_extracted INTEGER DEFAULT 0,
    errors INTEGER DEFAULT 0,
    
    CONSTRAINT executions_status_check CHECK (status IN ('running', 'completed', 'failed', 'stopped'))
);

CREATE INDEX idx_executions_workflow_id ON workflow_executions(workflow_id);
CREATE INDEX idx_executions_status ON workflow_executions(status);
CREATE INDEX idx_executions_started_at ON workflow_executions(started_at DESC);

-- Extracted items metadata table (GCS references)
CREATE TABLE IF NOT EXISTS extracted_items_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    gcs_path TEXT NOT NULL,
    item_count INTEGER NOT NULL,
    uploaded_at TIMESTAMP NOT NULL DEFAULT NOW(),
    file_size_bytes BIGINT,
    
    CONSTRAINT gcs_path_format CHECK (gcs_path LIKE 'gs://%')
);

CREATE INDEX idx_extracted_items_execution_id ON extracted_items_metadata(execution_id);
CREATE INDEX idx_extracted_items_workflow_id ON extracted_items_metadata(workflow_id);

-- Task tracking table (optional, for debugging)
CREATE TABLE IF NOT EXISTS task_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id VARCHAR(255) NOT NULL,
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    status VARCHAR(50) NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    
    CONSTRAINT task_status_check CHECK (status IN ('pending', 'processing', 'completed', 'failed'))
);

CREATE INDEX idx_task_history_execution_id ON task_history(execution_id);
CREATE INDEX idx_task_history_status ON task_history(status);
CREATE INDEX idx_task_history_started_at ON task_history(started_at DESC);

-- Functions and triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_workflows_updated_at 
    BEFORE UPDATE ON workflows 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Table comments
COMMENT ON TABLE workflows IS 'Stores workflow definitions and configurations';
COMMENT ON TABLE workflow_executions IS 'Tracks workflow execution runs with aggregated statistics';
COMMENT ON TABLE extracted_items_metadata IS 'References to Cloud Storage files containing extracted data in JSONL format';
COMMENT ON TABLE task_history IS 'Optional task-level tracking for debugging and monitoring';

COMMENT ON COLUMN workflows.config IS 'JSONB structure: {
  "start_urls": ["https://example.com"],
  "phases": [{
    "id": "phase-1",
    "name": "List",
    "nodes": ["node-1", "node-2"]
  }],
  "nodes": [{
    "id": "node-1",
    "type": "navigate",
    "config": {}
  }]
}';
