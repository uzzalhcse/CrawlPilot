-- Add provider column to API keys table
ALTER TABLE gemini_api_keys 
  ADD COLUMN IF NOT EXISTS provider VARCHAR(50) DEFAULT 'gemini';

-- Rename table to be provider-agnostic
ALTER TABLE gemini_api_keys RENAME TO ai_api_keys;

-- Update indexes
DROP INDEX IF EXISTS idx_gemini_keys_active;
DROP INDEX IF EXISTS idx_gemini_keys_usage;

CREATE INDEX IF NOT EXISTS idx_ai_keys_active ON ai_api_keys(provider, is_active, is_rate_limited, cooldown_until);
CREATE INDEX IF NOT EXISTS idx_ai_keys_usage ON ai_api_keys(provider, total_requests ASC);

-- Rename function and trigger
DROP TRIGGER IF EXISTS gemini_keys_updated_at ON ai_api_keys;
ALTER FUNCTION update_gemini_keys_updated_at() RENAME TO update_ai_keys_updated_at;

CREATE TRIGGER ai_keys_updated_at
    BEFORE UPDATE ON ai_api_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_ai_keys_updated_at();
