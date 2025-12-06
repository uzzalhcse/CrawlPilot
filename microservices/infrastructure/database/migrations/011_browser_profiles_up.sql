-- Browser profiles table for advanced fingerprint management
-- Migration: 011_browser_profiles
-- Description: Create browser_profiles table and add profile reference to workflows

CREATE TABLE IF NOT EXISTS browser_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    folder VARCHAR(128),
    tags JSONB DEFAULT '[]',
    
    -- Driver Configuration
    driver_type VARCHAR(50) NOT NULL DEFAULT 'playwright',
    browser_type VARCHAR(50) NOT NULL DEFAULT 'chromium',
    executable_path TEXT,
    cdp_endpoint TEXT,
    launch_args JSONB DEFAULT '[]',
    
    -- Fingerprint settings
    user_agent TEXT,
    platform VARCHAR(64) DEFAULT 'Win32',
    screen_width INTEGER DEFAULT 1920,
    screen_height INTEGER DEFAULT 1080,
    timezone VARCHAR(64),
    locale VARCHAR(32),
    languages JSONB DEFAULT '["en-US", "en"]',
    webgl_vendor VARCHAR(255),
    webgl_renderer VARCHAR(255),
    canvas_noise BOOLEAN DEFAULT true,
    hardware_concurrency INTEGER DEFAULT 4,
    device_memory INTEGER DEFAULT 8,
    fonts JSONB,
    do_not_track BOOLEAN DEFAULT false,
    disable_webrtc BOOLEAN DEFAULT false,
    
    -- Geolocation
    geolocation_latitude DECIMAL(10, 7),
    geolocation_longitude DECIMAL(10, 7),
    geolocation_accuracy INTEGER,
    
    -- Proxy
    proxy_enabled BOOLEAN DEFAULT false,
    proxy_type VARCHAR(32),
    proxy_server TEXT,
    proxy_username VARCHAR(255),
    proxy_password VARCHAR(255),
    
    -- Metadata
    usage_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    
    CONSTRAINT browser_profiles_status_check 
        CHECK (status IN ('active', 'inactive', 'archived', 'running')),
    CONSTRAINT browser_profiles_driver_check 
        CHECK (driver_type IN ('playwright', 'chromedp', 'http')),
    CONSTRAINT browser_profiles_browser_check 
        CHECK (browser_type IN ('chromium', 'firefox', 'webkit')),
    -- Validation: chromedp only works with chromium
    CONSTRAINT browser_profiles_chromedp_chromium_check
        CHECK (driver_type != 'chromedp' OR browser_type = 'chromium')
);

-- Indexes for performance
CREATE INDEX idx_browser_profiles_status ON browser_profiles(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_browser_profiles_driver ON browser_profiles(driver_type);
CREATE INDEX idx_browser_profiles_folder ON browser_profiles(folder) WHERE folder IS NOT NULL;
CREATE INDEX idx_browser_profiles_name ON browser_profiles(name);

-- Add browser_profile_id to workflows
ALTER TABLE workflows ADD COLUMN browser_profile_id UUID REFERENCES browser_profiles(id);
CREATE INDEX idx_workflows_browser_profile ON workflows(browser_profile_id) WHERE browser_profile_id IS NOT NULL;

-- Trigger for updated_at
CREATE TRIGGER update_browser_profiles_updated_at 
    BEFORE UPDATE ON browser_profiles 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE browser_profiles IS 'Browser profiles with driver configuration and fingerprint settings';
COMMENT ON COLUMN browser_profiles.driver_type IS 'Driver type: playwright (all browsers), chromedp (chromium only), http (no browser)';
COMMENT ON COLUMN browser_profiles.browser_type IS 'Browser type: chromium, firefox, webkit. Chromedp requires chromium.';
