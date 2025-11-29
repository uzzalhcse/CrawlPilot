-- Add 'running' status to browser_profiles
-- This allows tracking when a browser is actually running

-- Drop the old constraint
ALTER TABLE browser_profiles DROP CONSTRAINT IF EXISTS valid_status;

-- Add new constraint with 'running' status
ALTER TABLE browser_profiles ADD CONSTRAINT valid_status 
    CHECK (status IN ('active', 'inactive', 'archived', 'running'));

-- Create index for running profiles (for quick lookups)
CREATE INDEX IF NOT EXISTS idx_browser_profiles_running ON browser_profiles(status) WHERE status = 'running';
