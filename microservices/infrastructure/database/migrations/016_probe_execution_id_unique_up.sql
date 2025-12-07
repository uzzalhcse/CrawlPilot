-- Add unique constraint on execution_id for probe UPSERT to work properly
-- This enables merging phases from separate phase executions into single probe record

-- First, remove any duplicate execution_id entries (keep the latest one)
DELETE FROM probe_results a
USING probe_results b
WHERE a.execution_id = b.execution_id 
  AND a.created_at < b.created_at;

-- Add unique constraint
ALTER TABLE probe_results ADD CONSTRAINT probe_results_execution_id_unique UNIQUE (execution_id);
