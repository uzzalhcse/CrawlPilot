-- ============================================================================
-- COMPLETE DATABASE SCHEMA - DOWN MIGRATION
-- Rollback for consolidated schema
-- ============================================================================

BEGIN;

-- Drop views
DROP VIEW IF EXISTS execution_stats;

-- Drop triggers
DROP TRIGGER IF EXISTS fix_suggestions_updated_at ON fix_suggestions;
DROP TRIGGER IF EXISTS ai_keys_updated_at ON ai_api_keys;

-- Drop functions
DROP FUNCTION IF EXISTS update_fix_suggestions_updated_at();
DROP FUNCTION IF EXISTS update_ai_keys_updated_at();

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS fix_suggestions CASCADE;
DROP TABLE IF EXISTS ai_api_keys CASCADE;
DROP TABLE IF EXISTS monitoring_snapshots CASCADE;
DROP TABLE IF EXISTS monitoring_schedules CASCADE;
DROP TABLE IF EXISTS monitoring_reports CASCADE;
DROP TABLE IF EXISTS extracted_items CASCADE;
DROP TABLE IF EXISTS node_executions CASCADE;
DROP TABLE IF EXISTS url_queue CASCADE;
DROP TABLE IF EXISTS workflow_executions CASCADE;
DROP TABLE IF EXISTS workflow_versions CASCADE;
DROP TABLE IF EXISTS workflows CASCADE;

COMMIT;
