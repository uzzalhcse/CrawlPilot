-- API Key rotation table for Gemini API keys
CREATE TABLE IF NOT EXISTS gemini_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key VARCHAR(500) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    
    -- Usage tracking
    total_requests INT DEFAULT 0,
    successful_requests INT DEFAULT 0,
    failed_requests INT DEFAULT 0,
    
    -- Rate limit tracking
    last_used_at TIMESTAMP,
    last_error_at TIMESTAMP,
    last_error_message TEXT,
    cooldown_until TIMESTAMP,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    is_rate_limited BOOLEAN DEFAULT false,
    
    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Index for fast key selection
CREATE INDEX IF NOT EXISTS idx_gemini_keys_active ON gemini_api_keys(is_active, is_rate_limited, cooldown_until);
CREATE INDEX IF NOT EXISTS idx_gemini_keys_usage ON gemini_api_keys(total_requests ASC);

-- Auto-update trigger
CREATE OR REPLACE FUNCTION update_gemini_keys_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER gemini_keys_updated_at
    BEFORE UPDATE ON gemini_api_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_gemini_keys_updated_at();
