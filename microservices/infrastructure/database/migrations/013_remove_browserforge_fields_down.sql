-- Restore fingerprint fields if rollback is needed
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS user_agent TEXT DEFAULT '';
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS platform TEXT DEFAULT '';
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS hardware_concurrency INTEGER DEFAULT 4;
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS device_memory INTEGER DEFAULT 8;
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS fonts TEXT[] DEFAULT '{}';
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS webgl_vendor TEXT DEFAULT '';
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS webgl_renderer TEXT DEFAULT '';
ALTER TABLE browser_profiles ADD COLUMN IF NOT EXISTS canvas_noise BOOLEAN DEFAULT true;
