-- Rollback multi-provider support
ALTER TABLE ai_api_keys RENAME TO gemini_api_keys;
ALTER TABLE gemini_api_keys DROP COLUMN IF EXISTS provider;

DROP INDEX IF EXISTS idx_ai_keys_active;
DROP INDEX IF EXISTS idx_ai_keys_usage;

CREATE INDEX IF NOT EXISTS idx_gemini_keys_active ON gemini_api_keys(is_active, is_rate_limited, cooldown_until);
CREATE INDEX IF NOT EXISTS idx_gemini_keys_usage ON gemini_api_keys(total_requests ASC);

DROP TRIGGER IF EXISTS ai_keys_updated_at ON gemini_api_keys;
ALTER FUNCTION update_ai_keys_updated_at() RENAME TO update_gemini_keys_updated_at;

CREATE TRIGGER gemini_keys_updated_at
    BEFORE UPDATE ON gemini_api_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_gemini_keys_updated_at();
