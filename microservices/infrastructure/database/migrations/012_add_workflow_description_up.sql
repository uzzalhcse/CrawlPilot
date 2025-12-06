-- Add description column to workflows table
ALTER TABLE workflows ADD COLUMN IF NOT EXISTS description TEXT DEFAULT '';
