-- Rollback fix suggestions table
DROP TRIGGER IF EXISTS fix_suggestions_updated_at ON fix_suggestions;
DROP FUNCTION IF EXISTS update_fix_suggestions_updated_at();
DROP TABLE IF EXISTS fix_suggestions;
