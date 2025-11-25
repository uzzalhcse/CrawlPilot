-- Rename health_check tables to monitoring tables for consistency

-- Rename the main reports table
ALTER TABLE health_check_reports RENAME TO monitoring_reports;

-- Rename the schedules table
ALTER TABLE health_check_schedules RENAME TO monitoring_schedules;

-- Rename the snapshots table
ALTER TABLE health_check_snapshots RENAME TO monitoring_snapshots;

-- Rename indexes for reports table
ALTER INDEX idx_health_checks_workflow RENAME TO idx_monitoring_workflow;
ALTER INDEX idx_health_checks_status RENAME TO idx_monitoring_status;
ALTER INDEX idx_health_check_baseline RENAME TO idx_monitoring_baseline;

-- Rename indexes for schedules table
ALTER INDEX idx_schedule_workflow RENAME TO idx_monitoring_schedule_workflow;
ALTER INDEX idx_schedule_enabled RENAME TO idx_monitoring_schedule_enabled;

-- Rename indexes for snapshots table
ALTER INDEX idx_snapshots_report RENAME TO idx_monitoring_snapshots_report;
ALTER INDEX idx_snapshots_node RENAME TO idx_monitoring_snapshots_node;
ALTER INDEX idx_snapshots_created RENAME TO idx_monitoring_snapshots_created;

-- Update table comments
COMMENT ON TABLE monitoring_reports IS 'Monitoring validation reports for workflows';
COMMENT ON COLUMN monitoring_reports.results IS 'Phase-by-phase validation results';
COMMENT ON COLUMN monitoring_reports.summary IS 'Aggregate summary of validation results';
COMMENT ON COLUMN monitoring_reports.config IS 'Monitoring check configuration used';
