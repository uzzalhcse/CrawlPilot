-- Create workflows table
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    config JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT workflows_status_check CHECK (status IN ('draft', 'active', 'paused', 'archived'))
);

-- Create workflow_executions table
CREATE TABLE IF NOT EXISTS workflow_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    error TEXT,
    stats JSONB NOT NULL DEFAULT '{}',
    context JSONB NOT NULL DEFAULT '{}',
    CONSTRAINT executions_status_check CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled'))
);

-- Create node_executions table
CREATE TABLE IF NOT EXISTS node_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    node_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    input JSONB,
    output JSONB,
    error TEXT,
    retry_count INTEGER DEFAULT 0,
    CONSTRAINT node_executions_status_check CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled'))
);

-- Create url_queue table
CREATE TABLE IF NOT EXISTS url_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    url_hash VARCHAR(64) NOT NULL,
    depth INTEGER NOT NULL DEFAULT 0,
    priority INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    retry_count INTEGER DEFAULT 0,
    error TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE,
    locked_at TIMESTAMP WITH TIME ZONE,
    locked_by VARCHAR(255),
    CONSTRAINT url_queue_status_check CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'skipped'))
);

-- Create extracted_data table
CREATE TABLE IF NOT EXISTS extracted_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    data JSONB NOT NULL,
    schema VARCHAR(255),
    extracted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_workflows_status ON workflows(status);
CREATE INDEX IF NOT EXISTS idx_workflows_created_at ON workflows(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_executions_workflow_id ON workflow_executions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_executions_status ON workflow_executions(status);
CREATE INDEX IF NOT EXISTS idx_executions_started_at ON workflow_executions(started_at DESC);

CREATE INDEX IF NOT EXISTS idx_node_executions_execution_id ON node_executions(execution_id);
CREATE INDEX IF NOT EXISTS idx_node_executions_status ON node_executions(status);

CREATE INDEX IF NOT EXISTS idx_url_queue_execution_id ON url_queue(execution_id);
CREATE INDEX IF NOT EXISTS idx_url_queue_status ON url_queue(status);
CREATE INDEX IF NOT EXISTS idx_url_queue_url_hash ON url_queue(url_hash);
CREATE INDEX IF NOT EXISTS idx_url_queue_priority_created ON url_queue(priority DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_url_queue_locked_at ON url_queue(locked_at) WHERE locked_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_extracted_data_execution_id ON extracted_data(execution_id);
CREATE INDEX IF NOT EXISTS idx_extracted_data_schema ON extracted_data(schema);
CREATE INDEX IF NOT EXISTS idx_extracted_data_extracted_at ON extracted_data(extracted_at DESC);

-- Create unique constraint for URL deduplication per execution
CREATE UNIQUE INDEX IF NOT EXISTS idx_url_queue_execution_url_hash ON url_queue(execution_id, url_hash);

-- Create updated_at trigger for workflows
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_workflows_updated_at BEFORE UPDATE ON workflows
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
