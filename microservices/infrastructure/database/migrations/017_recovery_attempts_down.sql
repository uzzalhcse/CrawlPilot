-- Down migration for recovery_attempts table

DROP TRIGGER IF EXISTS trigger_recovery_attempts_updated_at ON recovery_attempts;
DROP FUNCTION IF EXISTS update_recovery_attempts_updated_at();
DROP TABLE IF EXISTS recovery_attempts CASCADE;
