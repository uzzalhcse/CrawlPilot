-- Remove browser_profile_id from workflows table
DROP INDEX IF EXISTS idx_workflows_browser_profile_id;
ALTER TABLE workflows DROP COLUMN IF EXISTS browser_profile_id;
