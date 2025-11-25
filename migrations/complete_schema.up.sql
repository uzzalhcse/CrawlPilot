-- ============================================================================
-- COMPLETE DATABASE SCHEMA - UP MIGRATION
-- Consolidated from all migrations for Crawlify
-- ============================================================================

BEGIN;

-- ============================================================================
-- STEP 1: Create core workflow and execution tables
-- ============================================================================

-- Workflows table
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    config JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE workflows IS 'Workflow definitions and configurations';

CREATE INDEX idx_workflows_status ON workflows(status);

-- Workflow versions table
CREATE TABLE IF NOT EXISTS workflow_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    version INT NOT NULL,
    config JSONB NOT NULL,
    change_reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workflow_id, version)
);

CREATE INDEX idx_workflow_versions_workflow_id ON workflow_versions(workflow_id);
CREATE INDEX idx_workflow_versions_created_at ON workflow_versions(created_at DESC);

-- Workflow executions table
CREATE TABLE IF NOT EXISTS workflow_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    error TEXT,
    metadata JSONB
);

COMMENT ON TABLE workflow_executions IS 'Workflow execution tracking';
COMMENT ON COLUMN workflow_executions.status IS 'Status: pending, running, completed, failed, cancelled';

CREATE INDEX idx_workflow_executions_workflow_id ON workflow_executions(workflow_id);
CREATE INDEX idx_workflow_executions_status ON workflow_executions(status);
CREATE INDEX idx_workflow_executions_started_at ON workflow_executions(started_at DESC);

-- URL Queue table with hierarchy support
CREATE TABLE IF NOT EXISTS url_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    url_hash VARCHAR(64) NOT NULL,
    depth INTEGER NOT NULL DEFAULT 0,
    priority INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    parent_url_id UUID REFERENCES url_queue(id) ON DELETE SET NULL,
    discovered_by_node VARCHAR(255),
    url_type VARCHAR(50) DEFAULT 'page',
    marker VARCHAR(100) DEFAULT '',
    phase_id VARCHAR(100) DEFAULT '',
    retry_count INTEGER DEFAULT 0,
    error TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE,
    locked_at TIMESTAMP WITH TIME ZONE,
    locked_by VARCHAR(255),
    UNIQUE(execution_id, url_hash)
);

COMMENT ON TABLE url_queue IS 'URL queue with hierarchy tracking';

CREATE INDEX idx_url_queue_execution_status ON url_queue(execution_id, status) WHERE status = 'pending';
CREATE INDEX idx_url_queue_depth ON url_queue(execution_id, depth);
CREATE INDEX idx_url_queue_parent ON url_queue(parent_url_id) WHERE parent_url_id IS NOT NULL;
CREATE INDEX idx_url_queue_type ON url_queue(execution_id, url_type);
CREATE INDEX idx_url_queue_discovered_by ON url_queue(discovered_by_node) WHERE discovered_by_node IS NOT NULL;
CREATE INDEX idx_url_queue_url_hash ON url_queue(url_hash);
CREATE INDEX idx_url_queue_marker ON url_queue(marker);
CREATE INDEX idx_url_queue_phase_id ON url_queue(phase_id);

-- Node executions table
CREATE TABLE IF NOT EXISTS node_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    node_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    url_id UUID REFERENCES url_queue(id) ON DELETE SET NULL,
    parent_node_execution_id UUID REFERENCES node_executions(id) ON DELETE SET NULL,
    node_type VARCHAR(50),
    urls_discovered INTEGER DEFAULT 0,
    items_extracted INTEGER DEFAULT 0,
    duration_ms INTEGER,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    input JSONB,
    output JSONB,
    error TEXT,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0
);

COMMENT ON TABLE node_executions IS 'Node execution tracking with complete debugging info';

CREATE INDEX idx_node_exec_execution_time ON node_executions(execution_id, started_at);
CREATE INDEX idx_node_exec_url ON node_executions(url_id) WHERE url_id IS NOT NULL;
CREATE INDEX idx_node_exec_parent ON node_executions(parent_node_execution_id) WHERE parent_node_execution_id IS NOT NULL;
CREATE INDEX idx_node_exec_status ON node_executions(status) WHERE status != 'completed';
CREATE INDEX idx_node_exec_type ON node_executions(execution_id, node_type);
CREATE INDEX idx_node_exec_node_id ON node_executions(execution_id, node_id);

-- Extracted items table
CREATE TABLE IF NOT EXISTS extracted_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    url_id UUID NOT NULL REFERENCES url_queue(id) ON DELETE CASCADE,
    node_execution_id UUID REFERENCES node_executions(id) ON DELETE SET NULL,
    schema_name VARCHAR(255),
    data JSONB NOT NULL,
    extracted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_extracted_item_per_url UNIQUE(execution_id, url_id, schema_name)
);

COMMENT ON TABLE extracted_items IS 'Dynamic extracted data from any source';

CREATE INDEX idx_extracted_items_execution ON extracted_items(execution_id);
CREATE INDEX idx_extracted_items_url ON extracted_items(url_id);
CREATE INDEX idx_extracted_items_node_exec ON extracted_items(node_execution_id) WHERE node_execution_id IS NOT NULL;
CREATE INDEX idx_extracted_items_schema ON extracted_items(schema_name) WHERE schema_name IS NOT NULL;
CREATE INDEX idx_extracted_items_data ON extracted_items USING gin(data);

-- ============================================================================
-- STEP 2: Monitoring (Health Check) Tables
-- ============================================================================

-- Monitoring Reports Table
CREATE TABLE IF NOT EXISTS monitoring_reports (
    id UUID PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    execution_id UUID REFERENCES workflow_executions(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    duration_ms BIGINT,
    results JSONB,
    summary JSONB,
    config JSONB,
    is_baseline BOOLEAN DEFAULT false,
    baseline_report_id UUID REFERENCES monitoring_reports(id),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_monitoring_workflow ON monitoring_reports(workflow_id, started_at DESC);
CREATE INDEX idx_monitoring_status ON monitoring_reports(status);
CREATE INDEX idx_monitoring_baseline ON monitoring_reports(workflow_id, is_baseline) WHERE is_baseline = true;

COMMENT ON TABLE monitoring_reports IS 'Monitoring validation reports for workflows';
COMMENT ON COLUMN monitoring_reports.results IS 'Phase-by-phase validation results';
COMMENT ON COLUMN monitoring_reports.summary IS 'Aggregate summary of validation results';
COMMENT ON COLUMN monitoring_reports.config IS 'Monitoring check configuration used';

-- Monitoring schedules table
CREATE TABLE monitoring_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    schedule VARCHAR(100) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    last_run_at TIMESTAMP,
    next_run_at TIMESTAMP,
    notification_config JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_monitoring_schedule_workflow ON monitoring_schedules(workflow_id);
CREATE INDEX idx_monitoring_schedule_enabled ON monitoring_schedules(enabled, next_run_at);

-- Monitoring snapshots table
CREATE TABLE IF NOT EXISTS monitoring_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id UUID NOT NULL REFERENCES monitoring_reports(id) ON DELETE CASCADE,
    node_id VARCHAR(255) NOT NULL,
    phase_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    url TEXT NOT NULL,
    page_title VARCHAR(500),
    status_code INT,
    screenshot_path VARCHAR(500),
    dom_snapshot_path VARCHAR(500),
    console_logs JSONB,
    selector_type VARCHAR(50),
    selector_value TEXT,
    elements_found INT DEFAULT 0,
    error_message TEXT,
    field_required BOOLEAN DEFAULT true,
    metadata JSONB
);

CREATE INDEX idx_monitoring_snapshots_report ON monitoring_snapshots(report_id);
CREATE INDEX idx_monitoring_snapshots_node ON monitoring_snapshots(node_id);
CREATE INDEX idx_monitoring_snapshots_created ON monitoring_snapshots(created_at DESC);

-- ============================================================================
-- STEP 3: AI and Fix Suggestions
-- ============================================================================

-- AI API Keys table (for rotation)
CREATE TABLE IF NOT EXISTS ai_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key VARCHAR(500) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    provider VARCHAR(50) DEFAULT 'gemini',
    total_requests INT DEFAULT 0,
    successful_requests INT DEFAULT 0,
    failed_requests INT DEFAULT 0,
    last_used_at TIMESTAMP,
    last_error_at TIMESTAMP,
    last_error_message TEXT,
    cooldown_until TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    is_rate_limited BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_keys_active ON ai_api_keys(provider, is_active, is_rate_limited, cooldown_until);
CREATE INDEX IF NOT EXISTS idx_ai_keys_usage ON ai_api_keys(provider, total_requests ASC);

-- Fix suggestions table
CREATE TABLE IF NOT EXISTS fix_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id UUID NOT NULL REFERENCES monitoring_snapshots(id) ON DELETE CASCADE,
    workflow_id UUID NOT NULL,
    node_id VARCHAR(255) NOT NULL,
    suggested_selector VARCHAR(500) NOT NULL,
    alternative_selectors JSONB,
    suggested_node_config JSONB,
    fix_explanation TEXT NOT NULL,
    confidence_score FLOAT NOT NULL CHECK (confidence_score >= 0 AND confidence_score <= 1),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    reviewed_by VARCHAR(255),
    reviewed_at TIMESTAMP,
    applied_at TIMESTAMP,
    reverted_at TIMESTAMP,
    verification_result JSONB,
    ai_model VARCHAR(100) NOT NULL,
    ai_prompt_tokens INT,
    ai_response_tokens INT,
    ai_response_raw TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_fix_suggestions_snapshot ON fix_suggestions(snapshot_id);
CREATE INDEX idx_fix_suggestions_workflow ON fix_suggestions(workflow_id);
CREATE INDEX idx_fix_suggestions_node ON fix_suggestions(node_id);
CREATE INDEX idx_fix_suggestions_status ON fix_suggestions(status);
CREATE INDEX idx_fix_suggestions_created ON fix_suggestions(created_at DESC);

-- ============================================================================
-- STEP 4: Create Functions and Triggers
-- ============================================================================

-- Update AI keys timestamp
CREATE OR REPLACE FUNCTION update_ai_keys_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ai_keys_updated_at
    BEFORE UPDATE ON ai_api_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_ai_keys_updated_at();

-- Update fix suggestions timestamp
CREATE OR REPLACE FUNCTION update_fix_suggestions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER fix_suggestions_updated_at
    BEFORE UPDATE ON fix_suggestions
    FOR EACH ROW
    EXECUTE FUNCTION update_fix_suggestions_updated_at();

-- ============================================================================
-- STEP 5: Create helpful views
-- ============================================================================

-- View for execution statistics
CREATE OR REPLACE VIEW execution_stats AS
SELECT
    we.id as execution_id,
    we.workflow_id,
    w.name as workflow_name,
    we.status,
    we.started_at,
    we.completed_at,
    EXTRACT(EPOCH FROM (COALESCE(we.completed_at, NOW()) - we.started_at)) as duration_seconds,
    COUNT(DISTINCT uq.id) as total_urls,
    COUNT(DISTINCT uq.id) FILTER (WHERE uq.status = 'completed') as completed_urls,
    COUNT(DISTINCT uq.id) FILTER (WHERE uq.status = 'failed') as failed_urls,
    COUNT(DISTINCT uq.id) FILTER (WHERE uq.status = 'pending') as pending_urls,
    COUNT(DISTINCT ei.id) as total_items_extracted,
    COUNT(DISTINCT ne.id) as total_node_executions,
    COUNT(DISTINCT ne.id) FILTER (WHERE ne.status = 'failed') as failed_node_executions
FROM workflow_executions we
JOIN workflows w ON w.id = we.workflow_id
LEFT JOIN url_queue uq ON uq.execution_id = we.id
LEFT JOIN extracted_items ei ON ei.execution_id = we.id
LEFT JOIN node_executions ne ON ne.execution_id = we.id
GROUP BY we.id, we.workflow_id, w.name, we.status, we.started_at, we.completed_at;

COMMENT ON VIEW execution_stats IS 'Aggregated statistics for workflow executions';

COMMIT;
