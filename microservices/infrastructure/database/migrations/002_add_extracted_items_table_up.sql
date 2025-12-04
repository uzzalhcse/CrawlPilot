-- Migration: 002_add_extracted_items_table
-- Description: Add table to store extracted data directly in database (for local dev when GCS is disabled)
-- Version: 1.0.0

-- Extracted items table (stores actual data in database)
CREATE TABLE IF NOT EXISTS extracted_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    task_id VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    data JSONB NOT NULL,  -- The actual extracted item data
    extracted_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_extracted_items_execution_id ON extracted_items(execution_id);
CREATE INDEX idx_extracted_items_workflow_id ON extracted_items(workflow_id);
CREATE INDEX idx_extracted_items_task_id ON extracted_items(task_id);
CREATE INDEX idx_extracted_items_extracted_at ON extracted_items(extracted_at DESC);

-- Table comment
COMMENT ON TABLE extracted_items IS 'Stores extracted data directly in database (used when GCS is disabled for local development)';
COMMENT ON COLUMN extracted_items.data IS 'JSONB containing the extracted item fields and values';
