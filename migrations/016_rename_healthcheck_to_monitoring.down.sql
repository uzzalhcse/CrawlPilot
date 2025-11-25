-- Rollback migration: Rename monitoring tables back to health_check tables

-- Rename the main reports table
ALTER TABLE monitoring_reports RENAME TO health_check_reports;

-- Rename the schedules table
ALTER TABLE monitoring_schedules RENAME TO health_check_schedules;

-- Rename the snapshots table
ALTER TABLE monitoring_snapshots RENAME TO health_check_snapshots;

-- Rename indexes for reports table
ALTER INDEX idx_monitoring_workflow RENAME TO idx_health_checks_workflow;
ALTER INDEX idx_monitoring_status RENAME TO idx_health_checks_status;
ALTER INDEX idx_monitoring_baseline RENAME TO idx_health_check_baseline;

-- Rename indexes for schedules table
ALTER INDEX idx_monitoring_schedule_workflow RENAME TO idx_schedule_workflow;
ALTER INDEX idx_monitoring_schedule_enabled RENAME TO idx_schedule_enabled;

-- Rename indexes for snapshots table
ALTER INDEX idx_monitoring_snapshots_report RENAME TO idx_snapshots_report;
ALTER INDEX idx_monitoring_snapshots_node RENAME TO idx_snapshots_node;
ALTER INDEX idx_monitoring_snapshots_created RENAME TO idx_snapshots_created;

-- Restore table comments
COMMENT ON TABLE health_check_reports IS 'Health check validation reports for workflows';
COMMENT ON COLUMN health_check_reports.results IS 'Phase-by-phase validation results';
COMMENT ON COLUMN health_check_reports.summary IS 'Aggregate summary of validation results';
COMMENT ON COLUMN health_check_reports.config IS 'Health check configuration used';
