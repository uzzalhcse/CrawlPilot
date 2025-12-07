-- Migration: Remove probe execution support
-- Down migration for probe system

-- Drop probe tables
DROP TABLE IF EXISTS probe_sample_urls;
DROP TABLE IF EXISTS probe_results;

-- Drop index
DROP INDEX IF EXISTS idx_executions_is_probe;

-- Remove probe columns from workflow_executions table
ALTER TABLE workflow_executions DROP COLUMN IF EXISTS probe_status;
ALTER TABLE workflow_executions DROP COLUMN IF EXISTS is_probe;
