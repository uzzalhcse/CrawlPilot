-- Migration: 003_partition_extracted_items (DOWN)
-- Description: Rollback partitioned extracted_items to regular table
-- WARNING: This will drop all data in the partitioned table!

-- Step 1: Drop cascade delete triggers
DROP TRIGGER IF EXISTS trg_cleanup_extracted_items_execution ON workflow_executions;
DROP TRIGGER IF EXISTS trg_cleanup_extracted_items_workflow ON workflows;
DROP FUNCTION IF EXISTS cleanup_extracted_items_on_execution_delete();
DROP FUNCTION IF EXISTS cleanup_extracted_items_on_workflow_delete();

-- Step 2: Drop partition management functions
DROP FUNCTION IF EXISTS drop_old_extracted_items_partitions(INTEGER);
DROP FUNCTION IF EXISTS create_extracted_items_partition();

-- Step 2: Drop the partitioned table (cascades to all partitions)
DROP TABLE IF EXISTS extracted_items CASCADE;

-- Step 3: Recreate original non-partitioned table
CREATE TABLE extracted_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    task_id VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    data JSONB NOT NULL,
    extracted_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_extracted_items_execution_id ON extracted_items(execution_id);
CREATE INDEX IF NOT EXISTS idx_extracted_items_workflow_id ON extracted_items(workflow_id);
CREATE INDEX IF NOT EXISTS idx_extracted_items_task_id ON extracted_items(task_id);
CREATE INDEX IF NOT EXISTS idx_extracted_items_extracted_at ON extracted_items(extracted_at DESC);

-- Table comment
COMMENT ON TABLE extracted_items IS 'Stores extracted data directly in database (non-partitioned version)';
