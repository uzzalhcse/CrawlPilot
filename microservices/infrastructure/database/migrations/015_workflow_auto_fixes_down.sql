-- Drop workflow_auto_fixes table and related objects
DROP TRIGGER IF EXISTS trigger_workflow_auto_fixes_updated_at ON workflow_auto_fixes;
DROP FUNCTION IF EXISTS update_workflow_auto_fixes_updated_at();
DROP INDEX IF EXISTS idx_workflow_auto_fixes_workflow_id;
DROP INDEX IF EXISTS idx_workflow_auto_fixes_status;
DROP INDEX IF EXISTS idx_workflow_auto_fixes_created_at;
DROP TABLE IF EXISTS workflow_auto_fixes;
