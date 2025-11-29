-- Add browser_profile_id to workflows table for profile-based execution
ALTER TABLE workflows
ADD COLUMN IF NOT EXISTS browser_profile_id UUID REFERENCES browser_profiles(id) ON DELETE SET NULL;

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_workflows_browser_profile_id ON workflows(browser_profile_id);

-- Comment
COMMENT ON COLUMN workflows.browser_profile_id IS 'Browser profile to use for this workflow execution';
