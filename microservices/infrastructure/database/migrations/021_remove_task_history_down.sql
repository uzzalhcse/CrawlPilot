-- Down migration for remove_task_history
-- Recreates the task_history table if rolled back

CREATE TABLE IF NOT EXISTS task_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id VARCHAR(255) NOT NULL,
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    status VARCHAR(50) NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    
    CONSTRAINT task_status_check CHECK (status IN ('pending', 'processing', 'completed', 'failed'))
);

CREATE INDEX idx_task_history_execution_id ON task_history(execution_id);
CREATE INDEX idx_task_history_status ON task_history(status);
CREATE INDEX idx_task_history_started_at ON task_history(started_at DESC);

COMMENT ON TABLE task_history IS 'Optional task-level tracking for debugging and monitoring';
