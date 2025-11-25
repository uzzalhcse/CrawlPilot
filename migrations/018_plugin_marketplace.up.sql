-- ============================================================================
-- PLUGIN MARKETPLACE SCHEMA
-- Migration 018: Add plugin marketplace tables
-- ============================================================================

BEGIN;

-- ============================================================================
-- Plugins table - Core plugin metadata
-- ============================================================================

CREATE TABLE IF NOT EXISTS plugins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL, -- URL-friendly identifier
    description TEXT,
    author_name VARCHAR(255),
    author_email VARCHAR(255),
    repository_url VARCHAR(500), -- Git repo for source code
    documentation_url VARCHAR(500),
    phase_type VARCHAR(50) NOT NULL, -- discovery, extraction, processing
    plugin_type VARCHAR(50) NOT NULL DEFAULT 'community', -- builtin, official, community, private
    category VARCHAR(100),
    tags JSONB DEFAULT '[]',
    is_verified BOOLEAN DEFAULT false,
    total_downloads INTEGER DEFAULT 0,
    total_installs INTEGER DEFAULT 0,
    average_rating DECIMAL(3,2) DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

COMMENT ON TABLE plugins IS 'Plugin marketplace - core plugin metadata';
COMMENT ON COLUMN plugins.slug IS 'URL-friendly unique identifier';
COMMENT ON COLUMN plugins.phase_type IS 'discovery, extraction, or processing';
COMMENT ON COLUMN plugins.plugin_type IS 'builtin, official, community, or private';

CREATE INDEX idx_plugins_slug ON plugins(slug);
CREATE INDEX idx_plugins_category ON plugins(category);
CREATE INDEX idx_plugins_type ON plugins(plugin_type, is_verified);
CREATE INDEX idx_plugins_phase_type ON plugins(phase_type);
CREATE INDEX idx_plugins_tags ON plugins USING gin(tags);
CREATE INDEX idx_plugins_downloads ON plugins(total_downloads DESC);
CREATE INDEX idx_plugins_rating ON plugins(average_rating DESC);

-- ============================================================================
-- Plugin versions table - Stores compiled binaries for each version
-- ============================================================================

CREATE TABLE IF NOT EXISTS plugin_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id UUID NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL, -- Semantic version: 1.2.3
    changelog TEXT,
    is_stable BOOLEAN DEFAULT true,
    min_crawlify_version VARCHAR(50), -- Minimum compatible Crawlify version
    
    -- Platform-specific binaries (paths to .so files)
    linux_amd64_binary_path VARCHAR(500),
    linux_arm64_binary_path VARCHAR(500),
    darwin_amd64_binary_path VARCHAR(500),
    darwin_arm64_binary_path VARCHAR(500),
    
    -- Binary metadata
    binary_hash VARCHAR(64), -- SHA-256 hash for verification
    binary_size_bytes BIGINT,
    
    -- Configuration schema (JSON Schema for plugin config)
    config_schema JSONB,
    
    downloads INTEGER DEFAULT 0,
    published_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(plugin_id, version)
);

COMMENT ON TABLE plugin_versions IS 'Plugin versions with platform-specific compiled binaries';
COMMENT ON COLUMN plugin_versions.binary_hash IS 'SHA-256 hash for binary integrity verification';
COMMENT ON COLUMN plugin_versions.config_schema IS 'JSON Schema defining plugin configuration options';

CREATE INDEX idx_plugin_versions_plugin ON plugin_versions(plugin_id, published_at DESC);
CREATE INDEX idx_plugin_versions_version ON plugin_versions(version);
CREATE INDEX idx_plugin_versions_stable ON plugin_versions(plugin_id, is_stable) WHERE is_stable = true;

-- ============================================================================
-- Plugin installations - Track who has which plugins installed
-- ============================================================================

CREATE TABLE IF NOT EXISTS plugin_installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id UUID NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    plugin_version_id UUID NOT NULL REFERENCES plugin_versions(id),
    workspace_id VARCHAR(255) NOT NULL, -- Organization/user identifier
    installed_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP,
    usage_count INTEGER DEFAULT 0,
    
    UNIQUE(plugin_id, workspace_id)
);

COMMENT ON TABLE plugin_installations IS 'Tracks plugin installations per workspace';

CREATE INDEX idx_plugin_installs_workspace ON plugin_installations(workspace_id);
CREATE INDEX idx_plugin_installs_plugin ON plugin_installations(plugin_id);
CREATE INDEX idx_plugin_installs_last_used ON plugin_installations(last_used_at DESC);

-- ============================================================================
-- Plugin reviews - User ratings and reviews
-- ============================================================================

CREATE TABLE IF NOT EXISTS plugin_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id UUID NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review_text TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(plugin_id, user_id)
);

COMMENT ON TABLE plugin_reviews IS 'User reviews and ratings for plugins';

CREATE INDEX idx_plugin_reviews_plugin ON plugin_reviews(plugin_id);
CREATE INDEX idx_plugin_reviews_rating ON plugin_reviews(rating);
CREATE INDEX idx_plugin_reviews_created ON plugin_reviews(created_at DESC);

-- ============================================================================
-- Plugin categories - Predefined categories for organization
-- ============================================================================

CREATE TABLE IF NOT EXISTS plugin_categories (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    icon VARCHAR(100),
    display_order INTEGER DEFAULT 0
);

COMMENT ON TABLE plugin_categories IS 'Predefined categories for plugin organization';

INSERT INTO plugin_categories (id, name, description, display_order) VALUES
('ecommerce', 'E-commerce', 'Plugins for e-commerce websites', 1),
('social-media', 'Social Media', 'Plugins for social media platforms', 2),
('news', 'News & Media', 'Plugins for news and media sites', 3),
('data-extraction', 'Data Extraction', 'Specialized data extraction plugins', 4),
('authentication', 'Authentication', 'Login and authentication handling', 5),
('pagination', 'Pagination', 'Advanced pagination patterns', 6),
('javascript-heavy', 'JavaScript-Heavy Sites', 'For SPA and dynamic sites', 7),
('api-integration', 'API Integration', 'REST/GraphQL API plugins', 8),
('general', 'General Purpose', 'General purpose plugins', 99)
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- Functions and Triggers
-- ============================================================================

-- Update plugins updated_at timestamp
CREATE OR REPLACE FUNCTION update_plugins_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER plugins_updated_at
    BEFORE UPDATE ON plugins
    FOR EACH ROW
    EXECUTE FUNCTION update_plugins_updated_at();

-- Update plugin reviews updated_at timestamp
CREATE OR REPLACE FUNCTION update_plugin_reviews_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER plugin_reviews_updated_at
    BEFORE UPDATE ON plugin_reviews
    FOR EACH ROW
    EXECUTE FUNCTION update_plugin_reviews_updated_at();

-- Update plugin average rating when reviews change
CREATE OR REPLACE FUNCTION update_plugin_average_rating()
RETURNS TRIGGER AS $$
DECLARE
    avg_rating DECIMAL(3,2);
BEGIN
    SELECT AVG(rating)::DECIMAL(3,2)
    INTO avg_rating
    FROM plugin_reviews
    WHERE plugin_id = COALESCE(NEW.plugin_id, OLD.plugin_id);
    
    UPDATE plugins
    SET average_rating = COALESCE(avg_rating, 0.00)
    WHERE id = COALESCE(NEW.plugin_id, OLD.plugin_id);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_rating_on_review_insert
    AFTER INSERT ON plugin_reviews
    FOR EACH ROW
    EXECUTE FUNCTION update_plugin_average_rating();

CREATE TRIGGER update_rating_on_review_update
    AFTER UPDATE ON plugin_reviews
    FOR EACH ROW
    EXECUTE FUNCTION update_plugin_average_rating();

CREATE TRIGGER update_rating_on_review_delete
    AFTER DELETE ON plugin_reviews
    FOR EACH ROW
    EXECUTE FUNCTION update_plugin_average_rating();

-- Increment download count when plugin version is downloaded
CREATE OR REPLACE FUNCTION increment_plugin_downloads()
RETURNS TRIGGER AS $$
BEGIN
    -- Increment version downloads
    UPDATE plugin_versions
    SET downloads = downloads + 1
    WHERE id = NEW.plugin_version_id;
    
    -- Increment total plugin downloads
    UPDATE plugins
    SET total_downloads = total_downloads + 1
    WHERE id = NEW.plugin_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER increment_downloads_on_install
    AFTER INSERT ON plugin_installations
    FOR EACH ROW
    EXECUTE FUNCTION increment_plugin_downloads();

-- Update total installs count
CREATE OR REPLACE FUNCTION update_plugin_install_count()
RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        UPDATE plugins
        SET total_installs = total_installs + 1
        WHERE id = NEW.plugin_id;
        RETURN NEW;
    ELSIF (TG_OP = 'DELETE') THEN
        UPDATE plugins
        SET total_installs = total_installs - 1
        WHERE id = OLD.plugin_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_install_count
    AFTER INSERT OR DELETE ON plugin_installations
    FOR EACH ROW
    EXECUTE FUNCTION update_plugin_install_count();

COMMIT;
