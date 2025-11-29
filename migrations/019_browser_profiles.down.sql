-- Drop browser profiles table and related objects
DROP TRIGGER IF EXISTS browser_profiles_updated_at ON browser_profiles;
DROP FUNCTION IF EXISTS update_browser_profiles_updated_at();
DROP TABLE IF EXISTS browser_profiles CASCADE;
