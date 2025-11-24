-- Create health check snapshots table for diagnostic data capture
CREATE TABLE IF NOT EXISTS health_check_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id UUID NOT NULL REFERENCES health_check_reports(id) ON DELETE CASCADE,
    node_id VARCHAR(255) NOT NULL,
    phase_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Page information
    url TEXT NOT NULL,
    page_title VARCHAR(500),
    status_code INT,
    
    -- Snapshot file paths (stored on filesystem)
    screenshot_path VARCHAR(500),
    dom_snapshot_path VARCHAR(500),
    
    -- Console logs (JSON array of log entries)
    console_logs JSONB,
    
    -- Selector details that failed
    selector_type VARCHAR(50),
    selector_value TEXT,
    elements_found INT DEFAULT 0,
    error_message TEXT,
    
    -- Additional metadata
    metadata JSONB
);

-- Create indexes for efficient queries
CREATE INDEX idx_snapshots_report ON health_check_snapshots(report_id);
CREATE INDEX idx_snapshots_node ON health_check_snapshots(node_id);
CREATE INDEX idx_snapshots_created ON health_check_snapshots(created_at DESC);
