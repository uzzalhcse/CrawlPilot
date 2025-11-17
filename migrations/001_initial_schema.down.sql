-- Drop triggers
DROP TRIGGER IF EXISTS update_workflows_updated_at ON workflows;
DROP FUNCTION IF EXISTS update_updated_at_column;

-- Drop indexes
DROP INDEX IF EXISTS idx_extracted_data_extracted_at;
DROP INDEX IF EXISTS idx_extracted_data_schema;
DROP INDEX IF EXISTS idx_extracted_data_execution_id;

DROP INDEX IF EXISTS idx_url_queue_execution_url_hash;
DROP INDEX IF EXISTS idx_url_queue_locked_at;
DROP INDEX IF EXISTS idx_url_queue_priority_created;
DROP INDEX IF EXISTS idx_url_queue_url_hash;
DROP INDEX IF EXISTS idx_url_queue_status;
DROP INDEX IF EXISTS idx_url_queue_execution_id;

DROP INDEX IF EXISTS idx_node_executions_status;
DROP INDEX IF EXISTS idx_node_executions_execution_id;

DROP INDEX IF EXISTS idx_executions_started_at;
DROP INDEX IF EXISTS idx_executions_status;
DROP INDEX IF EXISTS idx_executions_workflow_id;

DROP INDEX IF EXISTS idx_workflows_created_at;
DROP INDEX IF EXISTS idx_workflows_status;

-- Drop tables
DROP TABLE IF EXISTS extracted_data;
DROP TABLE IF EXISTS url_queue;
DROP TABLE IF EXISTS node_executions;
DROP TABLE IF EXISTS workflow_executions;
DROP TABLE IF EXISTS workflows;
