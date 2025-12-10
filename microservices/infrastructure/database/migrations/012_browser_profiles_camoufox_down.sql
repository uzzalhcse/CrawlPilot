-- Remove camoufox-specific fields from browser_profiles
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS geo_ip;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS virtual_headless;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS force_scope_access;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS auto_captcha_solve;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS block_images;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS block_webgl;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS humanize;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS include_default_addons;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS enable_cache;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS user_data_dir;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS target_os;
