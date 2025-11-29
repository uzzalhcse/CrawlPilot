-- Drop browser profile proxies table and related objects
DROP TRIGGER IF EXISTS browser_profile_proxies_updated_at ON browser_profile_proxies;
DROP FUNCTION IF EXISTS update_browser_profile_proxies_updated_at();
DROP TABLE IF EXISTS browser_profile_proxies CASCADE;
