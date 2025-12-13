-- Rollback detailed recovery tracking columns

ALTER TABLE recovery_attempts DROP COLUMN IF EXISTS proxy_id;
ALTER TABLE recovery_attempts DROP COLUMN IF EXISTS proxy_tier;
ALTER TABLE recovery_attempts DROP COLUMN IF EXISTS tier_from;
ALTER TABLE recovery_attempts DROP COLUMN IF EXISTS tier_to;
ALTER TABLE recovery_attempts DROP COLUMN IF EXISTS confidence;
ALTER TABLE recovery_attempts DROP COLUMN IF EXISTS trigger_reason;
ALTER TABLE recovery_attempts DROP COLUMN IF EXISTS retry_count;

DROP INDEX IF EXISTS idx_recovery_attempts_proxy;
DROP INDEX IF EXISTS idx_recovery_attempts_tier;
