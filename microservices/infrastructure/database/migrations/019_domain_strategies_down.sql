-- Down migration for domain_strategies
DROP TRIGGER IF EXISTS update_domain_strategies_updated_at ON domain_strategies;
DROP INDEX IF EXISTS idx_domain_strategies_domain;
DROP INDEX IF EXISTS idx_domain_strategies_tier;
DROP INDEX IF EXISTS idx_domain_strategies_status;
DROP TABLE IF EXISTS domain_strategies;

-- Remove tier from proxies (keep the column but reset to default)
-- Note: We don't drop the column to avoid data loss
ALTER TABLE proxies ALTER COLUMN tier SET DEFAULT 1;
DROP INDEX IF EXISTS idx_proxies_tier;
