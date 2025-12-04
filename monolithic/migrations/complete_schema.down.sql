-- ============================================================================
-- COMPLETE DATABASE SCHEMA - DOWN MIGRATION
-- Rollback for consolidated schema - Version 2.0
-- ============================================================================

BEGIN;

-- Drop views
DROP VIEW IF EXISTS rule_performance;
DROP VIEW IF EXISTS error_recovery_stats;
DROP VIEW IF EXISTS execution_stats;

-- Drop triggers
DROP TRIGGER IF EXISTS browser_profile_proxies_updated_at ON browser_profile_proxies;
DROP TRIGGER IF EXISTS browser_profiles_updated_at ON browser_profiles;
DROP TRIGGER IF EXISTS update_install_count ON plugin_installations;
DROP TRIGGER IF EXISTS increment_downloads_on_install ON plugin_installations;
DROP TRIGGER IF EXISTS update_rating_on_review_delete ON plugin_reviews;
DROP TRIGGER IF EXISTS update_rating_on_review_update ON plugin_reviews;
DROP TRIGGER IF EXISTS update_rating_on_review_insert ON plugin_reviews;
DROP TRIGGER IF EXISTS plugin_reviews_updated_at ON plugin_reviews;
DROP TRIGGER IF EXISTS plugins_updated_at ON plugins;
DROP TRIGGER IF EXISTS fix_suggestions_updated_at ON fix_suggestions;
DROP TRIGGER IF EXISTS ai_keys_updated_at ON ai_api_keys;

-- Drop functions
DROP FUNCTION IF EXISTS update_browser_profile_proxies_updated_at();
DROP FUNCTION IF EXISTS update_browser_profiles_updated_at();
DROP FUNCTION IF EXISTS update_plugin_install_count();
DROP FUNCTION IF EXISTS increment_plugin_downloads();
DROP FUNCTION IF EXISTS update_plugin_average_rating();
DROP FUNCTION IF EXISTS update_plugin_reviews_updated_at();
DROP FUNCTION IF EXISTS update_plugins_updated_at();
DROP FUNCTION IF EXISTS update_fix_suggestions_updated_at();
DROP FUNCTION IF EXISTS update_ai_keys_updated_at();

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS error_recovery_history CASCADE;
DROP TABLE IF EXISTS error_patterns CASCADE;
DROP TABLE IF EXISTS context_aware_rules CASCADE;
DROP TABLE IF EXISTS error_recovery_configs CASCADE;
DROP TABLE IF EXISTS browser_profile_proxies CASCADE;
DROP TABLE IF EXISTS browser_profiles CASCADE;
DROP TABLE IF EXISTS plugin_categories CASCADE;
DROP TABLE IF EXISTS plugin_reviews CASCADE;
DROP TABLE IF EXISTS plugin_installations CASCADE;
DROP TABLE IF EXISTS plugin_versions CASCADE;
DROP TABLE IF EXISTS plugins CASCADE;
DROP TABLE IF EXISTS fix_suggestions CASCADE;
DROP TABLE IF EXISTS ai_api_keys CASCADE;
DROP TABLE IF EXISTS monitoring_snapshots CASCADE;
DROP TABLE IF EXISTS monitoring_schedules CASCADE;
DROP TABLE IF EXISTS monitoring_reports CASCADE;
DROP TABLE IF EXISTS extracted_items_metadata CASCADE;
DROP TABLE IF EXISTS extracted_items CASCADE;
DROP TABLE IF EXISTS node_executions CASCADE;
DROP TABLE IF EXISTS url_queue CASCADE;
DROP TABLE IF EXISTS workflow_executions CASCADE;
DROP TABLE IF EXISTS workflow_versions CASCADE;
DROP TABLE IF EXISTS workflows CASCADE;

COMMIT;
