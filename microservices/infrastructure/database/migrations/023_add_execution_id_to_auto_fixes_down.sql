DROP INDEX IF EXISTS idx_workflow_auto_fixes_execution_id;

ALTER TABLE workflow_auto_fixes
DROP COLUMN IF EXISTS execution_id;
