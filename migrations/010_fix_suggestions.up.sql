-- Create fix suggestions table for AI-generated selector fixes
CREATE TABLE IF NOT EXISTS fix_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id UUID NOT NULL REFERENCES health_check_snapshots(id) ON DELETE CASCADE,
    workflow_id UUID NOT NULL,
    node_id VARCHAR(255) NOT NULL,
    
    -- Suggested fixes
    suggested_selector VARCHAR(500) NOT NULL,
    alternative_selectors JSONB,
    suggested_node_config JSONB,
    fix_explanation TEXT NOT NULL,
    confidence_score FLOAT NOT NULL CHECK (confidence_score >= 0 AND confidence_score <= 1),
    
    -- Review status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, approved, rejected, applied, reverted
    reviewed_by VARCHAR(255),
    reviewed_at TIMESTAMP,
    applied_at TIMESTAMP,
    reverted_at TIMESTAMP,
    
    -- AI metadata
    ai_model VARCHAR(100) NOT NULL,
    ai_prompt_tokens INT,
    ai_response_tokens INT,
    ai_response_raw TEXT,
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient queries
CREATE INDEX idx_fix_suggestions_snapshot ON fix_suggestions(snapshot_id);
CREATE INDEX idx_fix_suggestions_workflow ON fix_suggestions(workflow_id);
CREATE INDEX idx_fix_suggestions_node ON fix_suggestions(node_id);
CREATE INDEX idx_fix_suggestions_status ON fix_suggestions(status);
CREATE INDEX idx_fix_suggestions_created ON fix_suggestions(created_at DESC);

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_fix_suggestions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER fix_suggestions_updated_at
    BEFORE UPDATE ON fix_suggestions
    FOR EACH ROW
    EXECUTE FUNCTION update_fix_suggestions_updated_at();
