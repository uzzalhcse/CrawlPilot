-- Rollback Gemini API keys table
DROP TRIGGER IF EXISTS gemini_keys_updated_at ON gemini_api_keys;
DROP FUNCTION IF EXISTS update_gemini_keys_updated_at();
DROP TABLE IF EXISTS gemini_api_keys;
