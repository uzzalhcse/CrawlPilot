-- Remove description column from workflows table
ALTER TABLE workflows DROP COLUMN IF EXISTS description;
