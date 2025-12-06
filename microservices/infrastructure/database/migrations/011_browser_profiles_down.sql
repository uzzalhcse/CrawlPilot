-- Rollback browser profiles migration

-- Remove trigger
DROP TRIGGER IF EXISTS update_browser_profiles_updated_at ON browser_profiles;

-- Remove index and column from workflows
DROP INDEX IF EXISTS idx_workflows_browser_profile;
ALTER TABLE workflows DROP COLUMN IF EXISTS browser_profile_id;

-- Remove indexes
DROP INDEX IF EXISTS idx_browser_profiles_name;
DROP INDEX IF EXISTS idx_browser_profiles_folder;
DROP INDEX IF EXISTS idx_browser_profiles_driver;
DROP INDEX IF EXISTS idx_browser_profiles_status;

-- Drop table
DROP TABLE IF EXISTS browser_profiles;
