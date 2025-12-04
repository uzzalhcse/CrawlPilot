-- Crawlify Microservices Database Schema
-- Migration: 001_initial_schema (ROLLBACK)
-- Description: Drop all tables created in initial schema
-- Version: 1.0.0

-- Drop triggers first
DROP TRIGGER IF EXISTS update_workflows_updated_at ON workflows;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (respecting foreign key dependencies)
DROP TABLE IF EXISTS task_history CASCADE;
DROP TABLE IF EXISTS extracted_items_metadata CASCADE;
DROP TABLE IF EXISTS workflow_executions CASCADE;
DROP TABLE IF EXISTS workflows CASCADE;
