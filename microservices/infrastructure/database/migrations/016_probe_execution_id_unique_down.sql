-- Remove unique constraint on execution_id
ALTER TABLE probe_results DROP CONSTRAINT IF EXISTS probe_results_execution_id_unique;
