-- Recovery System Tables Rollback
-- Drops tables for system config, proxies, recovery rules, and learned actions

DROP TABLE IF EXISTS learned_actions CASCADE;
DROP TABLE IF EXISTS recovery_rules CASCADE;
DROP TABLE IF EXISTS proxies CASCADE;
DROP TABLE IF EXISTS system_config CASCADE;

