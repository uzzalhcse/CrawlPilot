-- Migration: 002_add_extracted_items_table (ROLLBACK)
-- Description: Drop extracted_items table
-- Version: 1.0.0

-- Drop table
DROP TABLE IF EXISTS extracted_items CASCADE;
