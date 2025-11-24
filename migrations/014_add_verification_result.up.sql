-- Add verification_result column to fix_suggestions table
ALTER TABLE fix_suggestions ADD COLUMN IF NOT EXISTS verification_result JSONB;
