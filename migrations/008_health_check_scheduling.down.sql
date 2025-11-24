-- Reverse baseline and scheduling migrations
DROP INDEX IF EXISTS idx_schedule_enabled;
DROP INDEX IF EXISTS idx_schedule_workflow;
DROP TABLE IF EXISTS health_check_schedules;

DROP INDEX IF EXISTS idx_health_check_baseline;
ALTER TABLE health_check_reports 
DROP COLUMN IF EXISTS baseline_report_id,
DROP COLUMN IF EXISTS is_baseline;
