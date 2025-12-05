-- Incident Reports Migration
-- Stores detailed reports when automated recovery fails and human intervention is needed

CREATE TABLE IF NOT EXISTS incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id VARCHAR(64) NOT NULL,
    task_id VARCHAR(64) NOT NULL,
    workflow_id VARCHAR(64) NOT NULL,
    url TEXT NOT NULL,
    domain VARCHAR(255) NOT NULL,
    
    -- Error Details
    error_pattern VARCHAR(64) NOT NULL,
    error_message TEXT,
    status_code INTEGER,
    
    -- Recovery History
    recovery_attempts JSONB DEFAULT '[]'::jsonb,
    total_attempts INTEGER DEFAULT 0,
    
    -- AI Agent Details
    ai_enabled BOOLEAN DEFAULT false,
    ai_provider VARCHAR(32),
    ai_reasoning TEXT,
    ai_failure_reason TEXT,
    
    -- Snapshots for Investigation
    screenshot TEXT,          -- GCS path or base64
    dom_snapshot TEXT,        -- Full HTML (can be large)
    page_title VARCHAR(512),
    page_url TEXT,            -- Final URL after redirects
    
    -- Context
    browser_profile VARCHAR(128),
    proxy_used VARCHAR(255),
    cookies JSONB,
    request_headers JSONB,
    response_headers JSONB,
    
    -- Suggested Actions
    suggested_actions JSONB DEFAULT '[]'::jsonb,
    
    -- Status Tracking
    status VARCHAR(32) NOT NULL DEFAULT 'open',
    priority VARCHAR(16) NOT NULL DEFAULT 'medium',
    assigned_to VARCHAR(128),
    resolution TEXT,
    resolved_at TIMESTAMPTZ,
    
    -- Timestamps
    first_error_at TIMESTAMPTZ NOT NULL,
    last_error_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);
CREATE INDEX IF NOT EXISTS idx_incidents_priority ON incidents(priority);
CREATE INDEX IF NOT EXISTS idx_incidents_domain ON incidents(domain);
CREATE INDEX IF NOT EXISTS idx_incidents_execution ON incidents(execution_id);
CREATE INDEX IF NOT EXISTS idx_incidents_workflow ON incidents(workflow_id);
CREATE INDEX IF NOT EXISTS idx_incidents_created ON incidents(created_at DESC);

-- Compound index for dashboard view (open incidents by priority)
CREATE INDEX IF NOT EXISTS idx_incidents_open_priority ON incidents(status, priority, created_at DESC) 
WHERE status IN ('open', 'in_progress');

-- Index for domain pattern analysis
CREATE INDEX IF NOT EXISTS idx_incidents_domain_pattern ON incidents(domain, error_pattern);

-- Comments
COMMENT ON TABLE incidents IS 'Stores incident reports when automated recovery fails';
COMMENT ON COLUMN incidents.screenshot IS 'Base64 encoded screenshot or GCS path';
COMMENT ON COLUMN incidents.dom_snapshot IS 'Full HTML DOM at time of failure';
COMMENT ON COLUMN incidents.recovery_attempts IS 'JSON array of recovery attempt summaries';
COMMENT ON COLUMN incidents.suggested_actions IS 'AI-generated suggestions for human investigation';
