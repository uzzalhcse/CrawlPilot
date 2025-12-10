-- Add camoufox-specific fields to browser_profiles
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS geo_ip VARCHAR(64);
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS virtual_headless BOOLEAN DEFAULT false;
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS force_scope_access BOOLEAN DEFAULT true;
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS auto_captcha_solve BOOLEAN DEFAULT true;
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS block_images BOOLEAN DEFAULT false;
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS block_webgl BOOLEAN DEFAULT false;
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS humanize DECIMAL(5,2) DEFAULT 1.0;
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS include_default_addons BOOLEAN DEFAULT true;
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS enable_cache BOOLEAN DEFAULT false;
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS user_data_dir VARCHAR(512);
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS target_os VARCHAR(32);
