-- Health Check Reports Table
CREATE TABLE IF NOT EXISTS health_check_reports (
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
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_health_checks_workflow ON health_check_reports(workflow_id, started_at DESC);
CREATE INDEX idx_health_checks_status ON health_check_reports(status);

COMMENT ON TABLE health_check_reports IS 'Health check validation reports for workflows';
COMMENT ON COLUMN health_check_reports.results IS 'Phase-by-phase validation results';
COMMENT ON COLUMN health_check_reports.summary IS 'Aggregate summary of validation results';
COMMENT ON COLUMN health_check_reports.config IS 'Health check configuration used';
