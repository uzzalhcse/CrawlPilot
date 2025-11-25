-- Add parent_node_execution_id to url_queue for tracking cross-URL node relationships
ALTER TABLE url_queue ADD COLUMN parent_node_execution_id UUID REFERENCES node_executions(id) ON DELETE SET NULL;

-- Add index for efficient queries
CREATE INDEX idx_url_queue_parent_node_exec ON url_queue(parent_node_execution_id) WHERE parent_node_execution_id IS NOT NULL;

COMMENT ON COLUMN url_queue.parent_node_execution_id IS 'Node execution that discovered this URL (cross-URL parent tracking)';
