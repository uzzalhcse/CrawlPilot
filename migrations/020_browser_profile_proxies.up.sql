-- Create table for proxy rotation support (multiple proxies per profile)
CREATE TABLE IF NOT EXISTS browser_profile_proxies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES browser_profiles(id) ON DELETE CASCADE,
    
    -- Proxy Configuration
    proxy_type VARCHAR(20) NOT NULL, -- http, https, socks5
    proxy_server TEXT NOT NULL, -- host:port
    proxy_username TEXT,
    proxy_password TEXT,
    
    -- Rotation Configuration
    priority INTEGER DEFAULT 0, -- Higher priority proxies used first
    rotation_strategy VARCHAR(50) DEFAULT 'round-robin', -- round-robin, random, sticky
    is_active BOOLEAN DEFAULT true,
    
    -- Health Check
    last_health_check TIMESTAMP,
    health_status VARCHAR(50) DEFAULT 'unknown', -- healthy, unhealthy, unknown
    failure_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    average_response_time INTEGER, -- in milliseconds
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT valid_proxy_type CHECK (proxy_type IN ('http', 'https', 'socks5')),
    CONSTRAINT valid_rotation_strategy CHECK (rotation_strategy IN ('round-robin', 'random', 'sticky')),
    CONSTRAINT valid_health_status CHECK (health_status IN ('healthy', 'unhealthy', 'unknown'))
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_browser_profile_proxies_profile_id ON browser_profile_proxies(profile_id);
CREATE INDEX IF NOT EXISTS idx_browser_profile_proxies_is_active ON browser_profile_proxies(is_active);
CREATE INDEX IF NOT EXISTS idx_browser_profile_proxies_health_status ON browser_profile_proxies(health_status);
CREATE INDEX IF NOT EXISTS idx_browser_profile_proxies_priority ON browser_profile_proxies(priority DESC);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_browser_profile_proxies_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER browser_profile_proxies_updated_at
    BEFORE UPDATE ON browser_profile_proxies
    FOR EACH ROW
    EXECUTE FUNCTION update_browser_profile_proxies_updated_at();

-- Comments
COMMENT ON TABLE browser_profile_proxies IS 'Proxy rotation configuration for browser profiles';
COMMENT ON COLUMN browser_profile_proxies.rotation_strategy IS 'Strategy for selecting proxy: round-robin, random, or sticky';
