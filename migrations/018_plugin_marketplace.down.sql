-- Down migration for plugin marketplace

BEGIN;

DROP TRIGGER IF EXISTS update_install_count ON plugin_installations;
DROP TRIGGER IF EXISTS increment_downloads_on_install ON plugin_installations;
DROP TRIGGER IF EXISTS update_rating_on_review_delete ON plugin_reviews;
DROP TRIGGER IF EXISTS update_rating_on_review_update ON plugin_reviews;
DROP TRIGGER IF EXISTS update_rating_on_review_insert ON plugin_reviews;
DROP TRIGGER IF EXISTS plugin_reviews_updated_at ON plugin_reviews;
DROP TRIGGER IF EXISTS plugins_updated_at ON plugins;

DROP FUNCTION IF EXISTS update_plugin_install_count();
DROP FUNCTION IF EXISTS increment_plugin_downloads();
DROP FUNCTION IF EXISTS update_plugin_average_rating();
DROP FUNCTION IF EXISTS update_plugin_reviews_updated_at();
DROP FUNCTION IF EXISTS update_plugins_updated_at();

DROP TABLE IF EXISTS plugin_categories;
DROP TABLE IF EXISTS plugin_reviews;
DROP TABLE IF EXISTS plugin_installations;
DROP TABLE IF EXISTS plugin_versions;
DROP TABLE IF EXISTS plugins;

COMMIT;
