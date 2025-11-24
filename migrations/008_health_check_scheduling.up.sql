-- Add baseline tracking to health check reports
ALTER TABLE health_check_reports 
ADD COLUMN is_baseline BOOLEAN DEFAULT false,
ADD COLUMN baseline_report_id UUID REFERENCES health_check_reports(id);

CREATE INDEX idx_health_check_baseline ON health_check_reports(workflow_id, is_baseline) WHERE is_baseline = true;

-- Create health check schedules table
CREATE TABLE health_check_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    schedule VARCHAR(100) NOT NULL, -- cron format: "0 */6 * * *"
    enabled BOOLEAN DEFAULT true,
    last_run_at TIMESTAMP,
    next_run_at TIMESTAMP,
    notification_config JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_schedule_workflow ON health_check_schedules(workflow_id);
CREATE INDEX idx_schedule_enabled ON health_check_schedules(enabled, next_run_at);
