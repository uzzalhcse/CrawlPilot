-- Migration: 021_remove_task_history
-- Description: Remove unused task_history table (never implemented)
-- Date: 2025-12-13

-- Drop triggers and indexes first
DROP INDEX IF EXISTS idx_task_history_execution_id;
DROP INDEX IF EXISTS idx_task_history_status;
DROP INDEX IF EXISTS idx_task_history_started_at;

-- Drop the table
DROP TABLE IF EXISTS task_history;

-- Note: task_history was designed in 001_initial_schema but never had INSERT code implemented
-- Removing it to keep schema clean
