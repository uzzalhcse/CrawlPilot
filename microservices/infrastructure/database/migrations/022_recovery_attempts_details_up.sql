-- Add detailed recovery tracking columns
-- Captures proxy tier, escalation, confidence, and trigger information

-- Add new columns for detailed recovery tracking
ALTER TABLE recovery_attempts ADD COLUMN IF NOT EXISTS proxy_id VARCHAR(64);
ALTER TABLE recovery_attempts ADD COLUMN IF NOT EXISTS proxy_tier INTEGER;
ALTER TABLE recovery_attempts ADD COLUMN IF NOT EXISTS tier_from INTEGER;
ALTER TABLE recovery_attempts ADD COLUMN IF NOT EXISTS tier_to INTEGER;
ALTER TABLE recovery_attempts ADD COLUMN IF NOT EXISTS confidence DECIMAL(3,2);
ALTER TABLE recovery_attempts ADD COLUMN IF NOT EXISTS trigger_reason TEXT;
ALTER TABLE recovery_attempts ADD COLUMN IF NOT EXISTS retry_count INTEGER DEFAULT 0;

-- Comments
COMMENT ON COLUMN recovery_attempts.proxy_id IS 'ID of proxy used for recovery retry';
COMMENT ON COLUMN recovery_attempts.proxy_tier IS 'Tier of proxy used (1=datacenter, 2=residential, 3=mobile)';
COMMENT ON COLUMN recovery_attempts.tier_from IS 'Original tier before escalation';
COMMENT ON COLUMN recovery_attempts.tier_to IS 'Target tier after escalation';
COMMENT ON COLUMN recovery_attempts.confidence IS 'Confidence score of error detection (0.00-1.00)';
COMMENT ON COLUMN recovery_attempts.trigger_reason IS 'Why recovery was triggered (e.g., error rate exceeded threshold)';
COMMENT ON COLUMN recovery_attempts.retry_count IS 'Number of retry attempts for this task';

-- Index for proxy analysis
CREATE INDEX IF NOT EXISTS idx_recovery_attempts_proxy ON recovery_attempts(proxy_id);
CREATE INDEX IF NOT EXISTS idx_recovery_attempts_tier ON recovery_attempts(proxy_tier);
