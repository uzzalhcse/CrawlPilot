-- Remove parent_node_execution_id from url_queue
DROP INDEX IF EXISTS idx_url_queue_parent_node_exec;
ALTER TABLE url_queue DROP COLUMN IF EXISTS parent_node_execution_id;
