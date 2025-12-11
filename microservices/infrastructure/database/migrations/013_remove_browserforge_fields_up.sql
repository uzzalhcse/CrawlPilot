-- Remove fingerprint fields managed by BrowserForge
-- These fields will be generated dynamically by BrowserForge for all drivers
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS user_agent;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS platform;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS hardware_concurrency;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS device_memory;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS fonts;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS webgl_vendor;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS webgl_renderer;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS canvas_noise;
