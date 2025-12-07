-- Create workflow_auto_fixes table for tracking AI-applied fixes
CREATE TABLE IF NOT EXISTS workflow_auto_fixes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL,
    fix_type TEXT NOT NULL,  -- 'update_selector', 'skip_node'
    old_selector TEXT,
    new_selector TEXT,
    reasoning TEXT,
    confidence DOUBLE PRECISION DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'applied',  -- 'applied', 'pending', 'rejected'
    auto_applied BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_workflow_auto_fixes_workflow_id ON workflow_auto_fixes(workflow_id);
CREATE INDEX IF NOT EXISTS idx_workflow_auto_fixes_status ON workflow_auto_fixes(status);
CREATE INDEX IF NOT EXISTS idx_workflow_auto_fixes_created_at ON workflow_auto_fixes(created_at DESC);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_workflow_auto_fixes_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_workflow_auto_fixes_updated_at
    BEFORE UPDATE ON workflow_auto_fixes
    FOR EACH ROW
    EXECUTE FUNCTION update_workflow_auto_fixes_updated_at();
