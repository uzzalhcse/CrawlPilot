-- Remove phase_stats column from workflow_executions
ALTER TABLE workflow_executions DROP COLUMN IF EXISTS phase_stats;
ALTER TABLE workflow_executions DROP COLUMN IF EXISTS triggered_by;

-- Drop execution_errors table
DROP INDEX IF EXISTS idx_execution_errors_execution_id;
DROP INDEX IF EXISTS idx_execution_errors_created_at;
DROP INDEX IF EXISTS idx_execution_errors_type;
DROP TABLE IF EXISTS execution_errors;
