-- Rollback: Remove 'running' status from browser_profiles

-- Drop the running status index
DROP INDEX IF EXISTS idx_browser_profiles_running;

-- Drop the constraint
ALTER TABLE browser_profiles DROP CONSTRAINT IF EXISTS valid_status;

-- Restore original constraint
ALTER TABLE browser_profiles ADD CONSTRAINT valid_status 
    CHECK (status IN ('active', 'inactive', 'archived'));
