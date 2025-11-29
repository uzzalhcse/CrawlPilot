-- Create browser_profiles table for managing browser profiles with fingerprinting
CREATE TABLE IF NOT EXISTS browser_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, inactive, archived
    folder VARCHAR(255), -- For organization (e.g., "Production", "Testing")
    tags TEXT[], -- Array of tags for categorization
    
    -- Browser Configuration
    browser_type VARCHAR(50) NOT NULL DEFAULT 'chromium', -- chromium, firefox, webkit
    executable_path TEXT, -- Custom browser executable path
    cdp_endpoint TEXT, -- WebSocket CDP endpoint (e.g., ws://localhost:9222/devtools/...)
    launch_args TEXT[], -- Additional launch arguments
    
    -- Fingerprint Configuration
    user_agent TEXT,
    platform VARCHAR(100), -- Windows, macOS, Linux
    screen_width INTEGER DEFAULT 1920,
    screen_height INTEGER DEFAULT 1080,
    timezone VARCHAR(100), -- e.g., "America/New_York"
    locale VARCHAR(20) DEFAULT 'en-US',
    languages TEXT[] DEFAULT ARRAY['en-US', 'en'], -- Accept-Language header
    
    -- Advanced Fingerprinting
    webgl_vendor TEXT, -- e.g., "Intel Inc."
    webgl_renderer TEXT, -- e.g., "Intel Iris OpenGL Engine"
    canvas_noise BOOLEAN DEFAULT false, -- Enable canvas fingerprint noise
    hardware_concurrency INTEGER DEFAULT 4, -- CPU cores
    device_memory INTEGER DEFAULT 8, -- RAM in GB
    fonts TEXT[], -- Custom font list
    
    -- Privacy & Security
    do_not_track BOOLEAN DEFAULT false,
    disable_webrtc BOOLEAN DEFAULT false,
    geolocation_latitude DECIMAL(10, 8),
    geolocation_longitude DECIMAL(11, 8),
    geolocation_accuracy INTEGER,
    
    -- Proxy Configuration (basic, detailed proxy rotation in separate table)
    proxy_enabled BOOLEAN DEFAULT false,
    proxy_type VARCHAR(20), -- http, https, socks5
    proxy_server TEXT, -- host:port
    proxy_username TEXT,
    proxy_password TEXT,
    
    -- Cookies & Storage
    cookies JSONB, -- Stored cookies
    local_storage JSONB, -- LocalStorage data
    session_storage JSONB, -- SessionStorage data
    indexed_db JSONB, -- IndexedDB data
    clear_on_close BOOLEAN DEFAULT true, -- Clear cookies/storage after session
    
    -- Team & Sharing (for future collaboration features)
    owner_id UUID, -- User who created this profile
    shared_with UUID[], -- Array of user IDs
    permissions JSONB, -- Fine-grained permissions
    
    -- Statistics
    usage_count INTEGER DEFAULT 0, -- Number of times used
    last_used_at TIMESTAMP,
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT valid_status CHECK (status IN ('active', 'inactive', 'archived')),
    CONSTRAINT valid_browser_type CHECK (browser_type IN ('chromium', 'firefox', 'webkit')),
    CONSTRAINT valid_proxy_type CHECK (proxy_type IS NULL OR proxy_type IN ('http', 'https', 'socks5'))
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_browser_profiles_status ON browser_profiles(status);
CREATE INDEX IF NOT EXISTS idx_browser_profiles_folder ON browser_profiles(folder);
CREATE INDEX IF NOT EXISTS idx_browser_profiles_browser_type ON browser_profiles(browser_type);
CREATE INDEX IF NOT EXISTS idx_browser_profiles_created_at ON browser_profiles(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_browser_profiles_last_used_at ON browser_profiles(last_used_at DESC);
CREATE INDEX IF NOT EXISTS idx_browser_profiles_tags ON browser_profiles USING GIN(tags);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_browser_profiles_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER browser_profiles_updated_at
    BEFORE UPDATE ON browser_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_browser_profiles_updated_at();

-- Comments
COMMENT ON TABLE browser_profiles IS 'Browser profiles with fingerprinting configuration';
COMMENT ON COLUMN browser_profiles.browser_type IS 'Type of browser: chromium, firefox, or webkit';
COMMENT ON COLUMN browser_profiles.executable_path IS 'Path to custom browser executable';
COMMENT ON COLUMN browser_profiles.cdp_endpoint IS 'Chrome DevTools Protocol WebSocket endpoint';
COMMENT ON COLUMN browser_profiles.canvas_noise IS 'Add noise to canvas fingerprint to avoid detection';
