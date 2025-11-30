-- Drop views first
DROP VIEW IF EXISTS rule_performance;
DROP VIEW IF EXISTS error_recovery_stats;

-- Drop indexes
DROP INDEX IF EXISTS idx_recovery_history_workflow_detected;
DROP INDEX IF EXISTS idx_recovery_history_solution_type;
DROP INDEX IF EXISTS idx_recovery_history_error_type;
DROP INDEX IF EXISTS idx_recovery_history_successful;
DROP INDEX IF EXISTS idx_recovery_history_detected_at;
DROP INDEX IF EXISTS idx_recovery_history_rule;
DROP INDEX IF EXISTS idx_recovery_history_domain;
DROP INDEX IF EXISTS idx_recovery_history_workflow;
DROP INDEX IF EXISTS idx_recovery_history_execution;

-- Drop table
DROP TABLE IF EXISTS error_recovery_history;
