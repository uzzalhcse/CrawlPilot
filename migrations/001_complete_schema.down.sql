-- Rollback Complete Schema
-- This drops all tables and objects created by the up migration

BEGIN;

-- Drop views
DROP VIEW IF EXISTS execution_stats;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS extracted_items CASCADE;
DROP TABLE IF EXISTS node_executions CASCADE;
DROP TABLE IF EXISTS url_queue CASCADE;
DROP TABLE IF EXISTS workflow_executions CASCADE;
DROP TABLE IF EXISTS workflows CASCADE;

COMMIT;
