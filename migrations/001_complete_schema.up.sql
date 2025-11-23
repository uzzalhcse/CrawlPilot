-- Complete Database Schema with Optimizations
-- This migration creates the full schema with URL hierarchy tracking and optimized data structures

BEGIN;

-- ============================================================================
-- STEP 1: Create core tables
-- ============================================================================

-- Workflows table
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    config JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE workflows IS 'Workflow definitions and configurations';
COMMENT ON COLUMN workflows.config IS 'Workflow DAG configuration as JSONB';
COMMENT ON COLUMN workflows.status IS 'Status: active, inactive, archived';

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

    -- Hierarchy tracking fields
    parent_url_id UUID REFERENCES url_queue(id) ON DELETE SET NULL,
    discovered_by_node VARCHAR(255),
    url_type VARCHAR(50) DEFAULT 'page',
    
    -- Phase-based workflow fields
    marker VARCHAR(100) DEFAULT '',
    phase_id VARCHAR(100) DEFAULT '',

    -- Processing metadata
    retry_count INTEGER DEFAULT 0,
    error TEXT,
    metadata JSONB,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE,
    locked_at TIMESTAMP WITH TIME ZONE,
    locked_by VARCHAR(255),

    UNIQUE(execution_id, url_hash)
);

COMMENT ON TABLE url_queue IS 'URL queue with hierarchy tracking';
COMMENT ON COLUMN url_queue.parent_url_id IS 'Parent URL that discovered this URL (for hierarchy tracking)';
COMMENT ON COLUMN url_queue.discovered_by_node IS 'Workflow node name that discovered this URL';
COMMENT ON COLUMN url_queue.url_type IS 'Type of URL: seed, category, product, pagination, page';
COMMENT ON COLUMN url_queue.marker IS 'Marker for phase-based routing (e.g. category, product)';
COMMENT ON COLUMN url_queue.phase_id IS 'Assigned phase ID for this URL';
COMMENT ON COLUMN url_queue.status IS 'Status: pending, processing, completed, failed';

-- Optimized indexes for url_queue
CREATE INDEX idx_url_queue_execution_status ON url_queue(execution_id, status) WHERE status = 'pending';
CREATE INDEX idx_url_queue_depth ON url_queue(execution_id, depth);
CREATE INDEX idx_url_queue_parent ON url_queue(parent_url_id) WHERE parent_url_id IS NOT NULL;
CREATE INDEX idx_url_queue_type ON url_queue(execution_id, url_type);
CREATE INDEX idx_url_queue_discovered_by ON url_queue(discovered_by_node) WHERE discovered_by_node IS NOT NULL;
CREATE INDEX idx_url_queue_url_hash ON url_queue(url_hash);
CREATE INDEX idx_url_queue_marker ON url_queue(marker);
CREATE INDEX idx_url_queue_phase_id ON url_queue(phase_id);

-- Node executions table with enhanced debugging
CREATE TABLE IF NOT EXISTS node_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    node_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',

    -- Hierarchy and tracking
    url_id UUID REFERENCES url_queue(id) ON DELETE SET NULL,
    parent_node_execution_id UUID REFERENCES node_executions(id) ON DELETE SET NULL,
    node_type VARCHAR(50),

    -- Metrics
    urls_discovered INTEGER DEFAULT 0,
    items_extracted INTEGER DEFAULT 0,
    duration_ms INTEGER,

    -- Timing
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Data
    input JSONB,
    output JSONB,
    error TEXT,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0
);

COMMENT ON TABLE node_executions IS 'Node execution tracking with complete debugging info';
COMMENT ON COLUMN node_executions.url_id IS 'Link to the URL being processed by this node';
COMMENT ON COLUMN node_executions.parent_node_execution_id IS 'Parent node execution in the workflow flow';
COMMENT ON COLUMN node_executions.node_type IS 'Type: navigate, extract_items, discover_urls, wait, click, scroll';
COMMENT ON COLUMN node_executions.urls_discovered IS 'Count of URLs discovered by this node execution';
COMMENT ON COLUMN node_executions.items_extracted IS 'Count of items extracted by this node execution';
COMMENT ON COLUMN node_executions.duration_ms IS 'Execution duration in milliseconds';

-- Optimized indexes for node_executions
CREATE INDEX idx_node_exec_execution_time ON node_executions(execution_id, started_at);
CREATE INDEX idx_node_exec_url ON node_executions(url_id) WHERE url_id IS NOT NULL;
CREATE INDEX idx_node_exec_parent ON node_executions(parent_node_execution_id) WHERE parent_node_execution_id IS NOT NULL;
CREATE INDEX idx_node_exec_status ON node_executions(status) WHERE status != 'completed';
CREATE INDEX idx_node_exec_type ON node_executions(execution_id, node_type);
CREATE INDEX idx_node_exec_node_id ON node_executions(execution_id, node_id);

-- Extracted items table - structured data
CREATE TABLE IF NOT EXISTS extracted_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    url_id UUID NOT NULL REFERENCES url_queue(id) ON DELETE CASCADE,
    node_execution_id UUID REFERENCES node_executions(id) ON DELETE SET NULL,

    -- Item classification
    item_type VARCHAR(100) NOT NULL,
    schema_name VARCHAR(255),

    -- Common structured fields for fast queries
    title TEXT,
    price DECIMAL(10,2),
    currency VARCHAR(10) DEFAULT 'USD',
    availability VARCHAR(50),
    rating DECIMAL(3,2),
    review_count INTEGER,

    -- Flexible additional data
    attributes JSONB,

    -- Metadata
    extracted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_extracted_item_per_url_schema UNIQUE(execution_id, url_id, schema_name)
);

COMMENT ON TABLE extracted_items IS 'Structured extracted data (products, items, articles, etc.)';
COMMENT ON COLUMN extracted_items.item_type IS 'Type of item: book, product, article, listing, etc.';
COMMENT ON COLUMN extracted_items.schema_name IS 'Name of extraction schema used';
COMMENT ON COLUMN extracted_items.attributes IS 'Additional flexible attributes as JSONB';

-- Optimized indexes for extracted_items
CREATE INDEX idx_extracted_items_execution ON extracted_items(execution_id);
CREATE INDEX idx_extracted_items_url ON extracted_items(url_id);
CREATE INDEX idx_extracted_items_node_exec ON extracted_items(node_execution_id) WHERE node_execution_id IS NOT NULL;
CREATE INDEX idx_extracted_items_type ON extracted_items(item_type);
CREATE INDEX idx_extracted_items_schema ON extracted_items(schema_name);
CREATE INDEX idx_extracted_items_price ON extracted_items(price) WHERE price IS NOT NULL;
CREATE INDEX idx_extracted_items_rating ON extracted_items(rating) WHERE rating IS NOT NULL;
CREATE INDEX idx_extracted_items_title_search ON extracted_items USING gin(to_tsvector('english', title)) WHERE title IS NOT NULL;
CREATE INDEX idx_extracted_items_attrs ON extracted_items USING gin(attributes);

-- ============================================================================
-- STEP 2: Create helpful views
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
