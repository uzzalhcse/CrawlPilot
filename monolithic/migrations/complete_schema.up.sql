-- ============================================================================
-- COMPLETE DATABASE SCHEMA - UP MIGRATION
-- Consolidated from all migrations (001-024) for Crawlify
-- Generated: 2025-12-05
-- ============================================================================

BEGIN;

-- ============================================================================
-- STEP 1: Create core workflow and execution tables
-- ============================================================================

-- Workflows table
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    config JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE workflows IS 'Workflow definitions and configurations';

CREATE INDEX idx_workflows_status ON workflows(status);

-- Workflow versions table
CREATE TABLE IF NOT EXISTS workflow_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    version INT NOT NULL,
    config JSONB NOT NULL,
    change_reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workflow_id, version)
);

CREATE INDEX idx_workflow_versions_workflow_id ON workflow_versions(workflow_id);
CREATE INDEX idx_workflow_versions_created_at ON workflow_versions(created_at DESC);

-- Workflow executions table
CREATE TABLE IF NOT EXISTS workflow_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    error TEXT,
    metadata JSONB
);

COMMENT ON TABLE workflow_executions IS 'Workflow execution tracking';
COMMENT ON COLUMN workflow_executions.status IS 'Status: pending, running, completed, failed, cancelled';

CREATE INDEX idx_workflow_executions_workflow_id ON workflow_executions(workflow_id);
CREATE INDEX idx_workflow_executions_status ON workflow_executions(status);
CREATE INDEX idx_workflow_executions_started_at ON workflow_executions(started_at DESC);

-- Node executions table (created before url_queue for FK reference)
CREATE TABLE IF NOT EXISTS node_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    node_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    url_id UUID,  -- Will add FK constraint after url_queue is created
    parent_node_execution_id UUID REFERENCES node_executions(id) ON DELETE SET NULL,
    node_type VARCHAR(50),
    urls_discovered INTEGER DEFAULT 0,
    items_extracted INTEGER DEFAULT 0,
    duration_ms INTEGER,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    input JSONB,
    output JSONB,
    error TEXT,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0
);

COMMENT ON TABLE node_executions IS 'Node execution tracking with complete debugging info';

CREATE INDEX idx_node_exec_execution_time ON node_executions(execution_id, started_at);
CREATE INDEX idx_node_exec_parent ON node_executions(parent_node_execution_id) WHERE parent_node_execution_id IS NOT NULL;
CREATE INDEX idx_node_exec_status ON node_executions(status) WHERE status != 'completed';
CREATE INDEX idx_node_exec_type ON node_executions(execution_id, node_type);
CREATE INDEX idx_node_exec_node_id ON node_executions(execution_id, node_id);

-- URL Queue table with hierarchy support
CREATE TABLE IF NOT EXISTS url_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    url_hash VARCHAR(64) NOT NULL,
    depth INTEGER NOT NULL DEFAULT 0,
    priority INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    parent_url_id UUID REFERENCES url_queue(id) ON DELETE SET NULL,
    parent_node_execution_id UUID REFERENCES node_executions(id) ON DELETE SET NULL,
    discovered_by_node VARCHAR(255),
    url_type VARCHAR(50) DEFAULT 'page',
    marker VARCHAR(100) DEFAULT '',
    phase_id VARCHAR(100) DEFAULT '',
    retry_count INTEGER DEFAULT 0,
    error TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE,
    locked_at TIMESTAMP WITH TIME ZONE,
    locked_by VARCHAR(255),
    UNIQUE(execution_id, url_hash)
);

COMMENT ON TABLE url_queue IS 'URL queue with hierarchy tracking';
COMMENT ON COLUMN url_queue.parent_node_execution_id IS 'Node execution that discovered this URL (cross-URL parent tracking)';

CREATE INDEX idx_url_queue_execution_status ON url_queue(execution_id, status) WHERE status = 'pending';
CREATE INDEX idx_url_queue_depth ON url_queue(execution_id, depth);
CREATE INDEX idx_url_queue_parent ON url_queue(parent_url_id) WHERE parent_url_id IS NOT NULL;
CREATE INDEX idx_url_queue_type ON url_queue(execution_id, url_type);
CREATE INDEX idx_url_queue_discovered_by ON url_queue(discovered_by_node) WHERE discovered_by_node IS NOT NULL;
CREATE INDEX idx_url_queue_url_hash ON url_queue(url_hash);
CREATE INDEX idx_url_queue_marker ON url_queue(marker);
CREATE INDEX idx_url_queue_phase_id ON url_queue(phase_id);
CREATE INDEX idx_url_queue_parent_node_exec ON url_queue(parent_node_execution_id) WHERE parent_node_execution_id IS NOT NULL;

-- Add FK constraint for node_executions.url_id now that url_queue exists
ALTER TABLE node_executions 
    ADD CONSTRAINT fk_node_exec_url 
    FOREIGN KEY (url_id) REFERENCES url_queue(id) ON DELETE SET NULL;

CREATE INDEX idx_node_exec_url ON node_executions(url_id) WHERE url_id IS NOT NULL;

-- Extracted items table
CREATE TABLE IF NOT EXISTS extracted_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    url_id UUID NOT NULL REFERENCES url_queue(id) ON DELETE CASCADE,
    node_execution_id UUID REFERENCES node_executions(id) ON DELETE SET NULL,
    schema_name VARCHAR(255),
    data JSONB NOT NULL,
    extracted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_extracted_item_per_url UNIQUE(execution_id, url_id, schema_name)
);

COMMENT ON TABLE extracted_items IS 'Dynamic extracted data from any source';

CREATE INDEX idx_extracted_items_execution ON extracted_items(execution_id);
CREATE INDEX idx_extracted_items_url ON extracted_items(url_id);
CREATE INDEX idx_extracted_items_node_exec ON extracted_items(node_execution_id) WHERE node_execution_id IS NOT NULL;
CREATE INDEX idx_extracted_items_schema ON extracted_items(schema_name) WHERE schema_name IS NOT NULL;
CREATE INDEX idx_extracted_items_data ON extracted_items USING gin(data);

-- ============================================================================
-- STEP 2: Browser Profiles
-- ============================================================================

-- Browser profiles table
CREATE TABLE IF NOT EXISTS browser_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    folder VARCHAR(255),
    tags TEXT[],
    
    -- Browser Configuration
    browser_type VARCHAR(50) NOT NULL DEFAULT 'chromium',
    executable_path TEXT,
    cdp_endpoint TEXT,
    launch_args TEXT[],
    
    -- Fingerprint Configuration
    user_agent TEXT,
    platform VARCHAR(100),
    screen_width INTEGER DEFAULT 1920,
    screen_height INTEGER DEFAULT 1080,
    timezone VARCHAR(100),
    locale VARCHAR(20) DEFAULT 'en-US',
    languages TEXT[] DEFAULT ARRAY['en-US', 'en'],
    
    -- Advanced Fingerprinting
    webgl_vendor TEXT,
    webgl_renderer TEXT,
    canvas_noise BOOLEAN DEFAULT false,
    hardware_concurrency INTEGER DEFAULT 4,
    device_memory INTEGER DEFAULT 8,
    fonts TEXT[],
    
    -- Privacy & Security
    do_not_track BOOLEAN DEFAULT false,
    disable_webrtc BOOLEAN DEFAULT false,
    geolocation_latitude DECIMAL(10, 8),
    geolocation_longitude DECIMAL(11, 8),
    geolocation_accuracy INTEGER,
    
    -- Proxy Configuration
    proxy_enabled BOOLEAN DEFAULT false,
    proxy_type VARCHAR(20),
    proxy_server TEXT,
    proxy_username TEXT,
    proxy_password TEXT,
    
    -- Cookies & Storage
    cookies JSONB,
    local_storage JSONB,
    session_storage JSONB,
    indexed_db JSONB,
    clear_on_close BOOLEAN DEFAULT true,
    
    -- Team & Sharing
    owner_id UUID,
    shared_with UUID[],
    permissions JSONB,
    
    -- Statistics
    usage_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMP,
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints (includes 'running' status from migration 022)
    CONSTRAINT valid_status CHECK (status IN ('active', 'inactive', 'archived', 'running')),
    CONSTRAINT valid_browser_type CHECK (browser_type IN ('chromium', 'firefox', 'webkit')),
    CONSTRAINT valid_proxy_type CHECK (proxy_type IS NULL OR proxy_type IN ('http', 'https', 'socks5'))
);

CREATE INDEX IF NOT EXISTS idx_browser_profiles_status ON browser_profiles(status);
CREATE INDEX IF NOT EXISTS idx_browser_profiles_folder ON browser_profiles(folder);
CREATE INDEX IF NOT EXISTS idx_browser_profiles_browser_type ON browser_profiles(browser_type);
CREATE INDEX IF NOT EXISTS idx_browser_profiles_created_at ON browser_profiles(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_browser_profiles_last_used_at ON browser_profiles(last_used_at DESC);
CREATE INDEX IF NOT EXISTS idx_browser_profiles_tags ON browser_profiles USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_browser_profiles_running ON browser_profiles(status) WHERE status = 'running';

COMMENT ON TABLE browser_profiles IS 'Browser profiles with fingerprinting configuration';
COMMENT ON COLUMN browser_profiles.browser_type IS 'Type of browser: chromium, firefox, or webkit';
COMMENT ON COLUMN browser_profiles.executable_path IS 'Path to custom browser executable';
COMMENT ON COLUMN browser_profiles.cdp_endpoint IS 'Chrome DevTools Protocol WebSocket endpoint';
COMMENT ON COLUMN browser_profiles.canvas_noise IS 'Add noise to canvas fingerprint to avoid detection';

-- Browser profile proxies table
CREATE TABLE IF NOT EXISTS browser_profile_proxies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES browser_profiles(id) ON DELETE CASCADE,
    
    -- Proxy Configuration
    proxy_type VARCHAR(20) NOT NULL,
    proxy_server TEXT NOT NULL,
    proxy_username TEXT,
    proxy_password TEXT,
    
    -- Rotation Configuration
    priority INTEGER DEFAULT 0,
    rotation_strategy VARCHAR(50) DEFAULT 'round-robin',
    is_active BOOLEAN DEFAULT true,
    
    -- Health Check
    last_health_check TIMESTAMP,
    health_status VARCHAR(50) DEFAULT 'unknown',
    failure_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    average_response_time INTEGER,
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT valid_proxy_type_proxies CHECK (proxy_type IN ('http', 'https', 'socks5')),
    CONSTRAINT valid_rotation_strategy CHECK (rotation_strategy IN ('round-robin', 'random', 'sticky')),
    CONSTRAINT valid_health_status CHECK (health_status IN ('healthy', 'unhealthy', 'unknown'))
);

CREATE INDEX IF NOT EXISTS idx_browser_profile_proxies_profile_id ON browser_profile_proxies(profile_id);
CREATE INDEX IF NOT EXISTS idx_browser_profile_proxies_is_active ON browser_profile_proxies(is_active);
CREATE INDEX IF NOT EXISTS idx_browser_profile_proxies_health_status ON browser_profile_proxies(health_status);
CREATE INDEX IF NOT EXISTS idx_browser_profile_proxies_priority ON browser_profile_proxies(priority DESC);

COMMENT ON TABLE browser_profile_proxies IS 'Proxy rotation configuration for browser profiles';
COMMENT ON COLUMN browser_profile_proxies.rotation_strategy IS 'Strategy for selecting proxy: round-robin, random, or sticky';

-- Add browser_profile_id to workflows (after browser_profiles table exists)
ALTER TABLE workflows
ADD COLUMN IF NOT EXISTS browser_profile_id UUID REFERENCES browser_profiles(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_workflows_browser_profile_id ON workflows(browser_profile_id);

COMMENT ON COLUMN workflows.browser_profile_id IS 'Browser profile to use for this workflow execution';

-- ============================================================================
-- STEP 3: Monitoring (Health Check) Tables
-- ============================================================================

-- Monitoring Reports Table
CREATE TABLE IF NOT EXISTS monitoring_reports (
    id UUID PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    execution_id UUID REFERENCES workflow_executions(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    duration_ms BIGINT,
    results JSONB,
    summary JSONB,
    config JSONB,
    is_baseline BOOLEAN DEFAULT false,
    baseline_report_id UUID REFERENCES monitoring_reports(id),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_monitoring_workflow ON monitoring_reports(workflow_id, started_at DESC);
CREATE INDEX idx_monitoring_status ON monitoring_reports(status);
CREATE INDEX idx_monitoring_baseline ON monitoring_reports(workflow_id, is_baseline) WHERE is_baseline = true;

COMMENT ON TABLE monitoring_reports IS 'Monitoring validation reports for workflows';
COMMENT ON COLUMN monitoring_reports.results IS 'Phase-by-phase validation results';
COMMENT ON COLUMN monitoring_reports.summary IS 'Aggregate summary of validation results';
COMMENT ON COLUMN monitoring_reports.config IS 'Monitoring check configuration used';

-- Monitoring schedules table
CREATE TABLE monitoring_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    schedule VARCHAR(100) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    last_run_at TIMESTAMP,
    next_run_at TIMESTAMP,
    notification_config JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_monitoring_schedule_workflow ON monitoring_schedules(workflow_id);
CREATE INDEX idx_monitoring_schedule_enabled ON monitoring_schedules(enabled, next_run_at);

-- Monitoring snapshots table
CREATE TABLE IF NOT EXISTS monitoring_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id UUID NOT NULL REFERENCES monitoring_reports(id) ON DELETE CASCADE,
    node_id VARCHAR(255) NOT NULL,
    phase_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    url TEXT NOT NULL,
    page_title VARCHAR(500),
    status_code INT,
    screenshot_path VARCHAR(500),
    dom_snapshot_path VARCHAR(500),
    console_logs JSONB,
    selector_type VARCHAR(50),
    selector_value TEXT,
    elements_found INT DEFAULT 0,
    error_message TEXT,
    field_required BOOLEAN DEFAULT true,
    metadata JSONB
);

CREATE INDEX idx_monitoring_snapshots_report ON monitoring_snapshots(report_id);
CREATE INDEX idx_monitoring_snapshots_node ON monitoring_snapshots(node_id);
CREATE INDEX idx_monitoring_snapshots_created ON monitoring_snapshots(created_at DESC);

-- ============================================================================
-- STEP 4: AI and Fix Suggestions
-- ============================================================================

-- AI API Keys table (for rotation)
CREATE TABLE IF NOT EXISTS ai_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key VARCHAR(500) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    provider VARCHAR(50) DEFAULT 'gemini',
    total_requests INT DEFAULT 0,
    successful_requests INT DEFAULT 0,
    failed_requests INT DEFAULT 0,
    last_used_at TIMESTAMP,
    last_error_at TIMESTAMP,
    last_error_message TEXT,
    cooldown_until TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    is_rate_limited BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_keys_active ON ai_api_keys(provider, is_active, is_rate_limited, cooldown_until);
CREATE INDEX IF NOT EXISTS idx_ai_keys_usage ON ai_api_keys(provider, total_requests ASC);

-- Fix suggestions table
CREATE TABLE IF NOT EXISTS fix_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id UUID NOT NULL REFERENCES monitoring_snapshots(id) ON DELETE CASCADE,
    workflow_id UUID NOT NULL,
    node_id VARCHAR(255) NOT NULL,
    suggested_selector VARCHAR(500) NOT NULL,
    alternative_selectors JSONB,
    suggested_node_config JSONB,
    fix_explanation TEXT NOT NULL,
    confidence_score FLOAT NOT NULL CHECK (confidence_score >= 0 AND confidence_score <= 1),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    reviewed_by VARCHAR(255),
    reviewed_at TIMESTAMP,
    applied_at TIMESTAMP,
    reverted_at TIMESTAMP,
    verification_result JSONB,
    ai_model VARCHAR(100) NOT NULL,
    ai_prompt_tokens INT,
    ai_response_tokens INT,
    ai_response_raw TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_fix_suggestions_snapshot ON fix_suggestions(snapshot_id);
CREATE INDEX idx_fix_suggestions_workflow ON fix_suggestions(workflow_id);
CREATE INDEX idx_fix_suggestions_node ON fix_suggestions(node_id);
CREATE INDEX idx_fix_suggestions_status ON fix_suggestions(status);
CREATE INDEX idx_fix_suggestions_created ON fix_suggestions(created_at DESC);

-- ============================================================================
-- STEP 5: Plugin Marketplace Tables
-- ============================================================================

-- Plugins table
CREATE TABLE IF NOT EXISTS plugins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    author_name VARCHAR(255),
    author_email VARCHAR(255),
    repository_url VARCHAR(500),
    documentation_url VARCHAR(500),
    phase_type VARCHAR(50) NOT NULL,
    plugin_type VARCHAR(50) NOT NULL DEFAULT 'community',
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

-- Plugin versions table
CREATE TABLE IF NOT EXISTS plugin_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id UUID NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    changelog TEXT,
    is_stable BOOLEAN DEFAULT true,
    min_crawlify_version VARCHAR(50),
    linux_amd64_binary_path VARCHAR(500),
    linux_arm64_binary_path VARCHAR(500),
    darwin_amd64_binary_path VARCHAR(500),
    darwin_arm64_binary_path VARCHAR(500),
    binary_hash VARCHAR(64),
    binary_size_bytes BIGINT,
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

-- Plugin installations table
CREATE TABLE IF NOT EXISTS plugin_installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id UUID NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    plugin_version_id UUID NOT NULL REFERENCES plugin_versions(id),
    workspace_id VARCHAR(255) NOT NULL,
    installed_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP,
    usage_count INTEGER DEFAULT 0,
    UNIQUE(plugin_id, workspace_id)
);

COMMENT ON TABLE plugin_installations IS 'Tracks plugin installations per workspace';

CREATE INDEX idx_plugin_installs_workspace ON plugin_installations(workspace_id);
CREATE INDEX idx_plugin_installs_plugin ON plugin_installations(plugin_id);
CREATE INDEX idx_plugin_installs_last_used ON plugin_installations(last_used_at DESC);

-- Plugin reviews table
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

-- Plugin categories table
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
-- STEP 6: Error Recovery Tables
-- ============================================================================

-- Error recovery configs table
CREATE TABLE IF NOT EXISTS error_recovery_configs (
    config_key VARCHAR(255) PRIMARY KEY,
    config_value JSONB NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Context aware rules table
CREATE TABLE IF NOT EXISTS context_aware_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    priority INTEGER NOT NULL DEFAULT 0,
    conditions JSONB NOT NULL,
    context JSONB NOT NULL,
    actions JSONB NOT NULL,
    confidence DOUBLE PRECISION DEFAULT 0,
    success_rate DOUBLE PRECISION DEFAULT 0,
    usage_count INTEGER DEFAULT 0,
    created_by VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_context_aware_rules_priority ON context_aware_rules(priority DESC);

-- Error patterns table
CREATE TABLE IF NOT EXISTS error_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_type VARCHAR(50) NOT NULL,
    pattern_data JSONB NOT NULL,
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Error recovery history table
CREATE TABLE error_recovery_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID,
    workflow_id UUID,
    
    -- Error Details
    error_type VARCHAR(255) NOT NULL,
    error_message TEXT NOT NULL,
    status_code INTEGER,
    url TEXT NOT NULL,
    domain VARCHAR(255),
    node_id VARCHAR(255),
    phase_id VARCHAR(255),
    
    -- Pattern Analysis
    pattern_detected BOOLEAN DEFAULT FALSE,
    pattern_type VARCHAR(50),
    activation_reason TEXT,
    error_rate DECIMAL(5,4),
    
    -- Recovery Solution
    rule_id UUID,
    rule_name VARCHAR(255),
    solution_type VARCHAR(50) NOT NULL,
    confidence DECIMAL(3,2),
    
    -- Actions Applied
    actions_applied JSONB,
    
    -- Outcome
    recovery_attempted BOOLEAN DEFAULT TRUE,
    recovery_successful BOOLEAN,
    retry_count INTEGER DEFAULT 1,
    time_to_recovery_ms INTEGER,
    
    -- Full Context
    request_context JSONB,
    
    -- Timestamps
    detected_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    recovered_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_recovery_history_execution ON error_recovery_history(execution_id);
CREATE INDEX idx_recovery_history_workflow ON error_recovery_history(workflow_id);
CREATE INDEX idx_recovery_history_domain ON error_recovery_history(domain);
CREATE INDEX idx_recovery_history_rule ON error_recovery_history(rule_id);
CREATE INDEX idx_recovery_history_detected_at ON error_recovery_history(detected_at DESC);
CREATE INDEX idx_recovery_history_successful ON error_recovery_history(recovery_successful);
CREATE INDEX idx_recovery_history_error_type ON error_recovery_history(error_type);
CREATE INDEX idx_recovery_history_solution_type ON error_recovery_history(solution_type);
CREATE INDEX idx_recovery_history_workflow_detected ON error_recovery_history(workflow_id, detected_at DESC);

COMMENT ON TABLE error_recovery_history IS 'Tracks all error recovery attempts with full context and outcomes';
COMMENT ON COLUMN error_recovery_history.pattern_detected IS 'Whether pattern analyzer triggered recovery';
COMMENT ON COLUMN error_recovery_history.solution_type IS 'How error was resolved: rule-based, AI, or no solution';
COMMENT ON COLUMN error_recovery_history.time_to_recovery_ms IS 'Time from error detection to recovery completion';

-- ============================================================================
-- STEP 7: Create Functions and Triggers
-- ============================================================================

-- Update AI keys timestamp
CREATE OR REPLACE FUNCTION update_ai_keys_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ai_keys_updated_at
    BEFORE UPDATE ON ai_api_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_ai_keys_updated_at();

-- Update fix suggestions timestamp
CREATE OR REPLACE FUNCTION update_fix_suggestions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER fix_suggestions_updated_at
    BEFORE UPDATE ON fix_suggestions
    FOR EACH ROW
    EXECUTE FUNCTION update_fix_suggestions_updated_at();

-- Update browser profiles timestamp
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

-- Update browser profile proxies timestamp
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

-- Update plugins timestamp
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

-- Update plugin reviews timestamp
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

-- Update plugin average rating
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

-- Increment download count
CREATE OR REPLACE FUNCTION increment_plugin_downloads()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE plugin_versions
    SET downloads = downloads + 1
    WHERE id = NEW.plugin_version_id;
    
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

-- Update plugin install count
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

-- ============================================================================
-- STEP 8: Create helpful views
-- ============================================================================

-- View for execution statistics
CREATE OR REPLACE VIEW execution_stats AS
SELECT
    we.id as execution_id,
    we.workflow_id,
    w.name as workflow_name,
    we.status,
    we.started_at,
    we.completed_at,
    EXTRACT(EPOCH FROM (COALESCE(we.completed_at, NOW()) - we.started_at)) as duration_seconds,
    COUNT(DISTINCT uq.id) as total_urls,
    COUNT(DISTINCT uq.id) FILTER (WHERE uq.status = 'completed') as completed_urls,
    COUNT(DISTINCT uq.id) FILTER (WHERE uq.status = 'failed') as failed_urls,
    COUNT(DISTINCT uq.id) FILTER (WHERE uq.status = 'pending') as pending_urls,
    COUNT(DISTINCT ei.id) as total_items_extracted,
    COUNT(DISTINCT ne.id) as total_node_executions,
    COUNT(DISTINCT ne.id) FILTER (WHERE ne.status = 'failed') as failed_node_executions
FROM workflow_executions we
JOIN workflows w ON w.id = we.workflow_id
LEFT JOIN url_queue uq ON uq.execution_id = we.id
LEFT JOIN extracted_items ei ON ei.execution_id = we.id
LEFT JOIN node_executions ne ON ne.execution_id = we.id
GROUP BY we.id, we.workflow_id, w.name, we.status, we.started_at, we.completed_at;

COMMENT ON VIEW execution_stats IS 'Aggregated statistics for workflow executions';

-- View for error recovery statistics
CREATE VIEW error_recovery_stats AS
SELECT 
    DATE_TRUNC('hour', detected_at) as time_bucket,
    domain,
    error_type,
    rule_name,
    solution_type,
    COUNT(*) as total_attempts,
    SUM(CASE WHEN recovery_successful THEN 1 ELSE 0 END) as successful_recoveries,
    ROUND(AVG(time_to_recovery_ms), 2) as avg_recovery_time_ms,
    ROUND(AVG(retry_count), 2) as avg_retries,
    ROUND(AVG(confidence), 3) as avg_confidence
FROM error_recovery_history
WHERE recovery_attempted = true
GROUP BY DATE_TRUNC('hour', detected_at), domain, error_type, rule_name, solution_type;

COMMENT ON VIEW error_recovery_stats IS 'Hourly aggregated statistics for recovery performance';

-- View for rule performance
CREATE VIEW rule_performance AS
SELECT 
    h.rule_id,
    h.rule_name,
    COUNT(h.id) as times_used,
    SUM(CASE WHEN h.recovery_successful THEN 1 ELSE 0 END) as successful_uses,
    ROUND(
        SUM(CASE WHEN h.recovery_successful THEN 1 ELSE 0 END)::DECIMAL / NULLIF(COUNT(h.id), 0),
        4
    ) as actual_success_rate,
    ROUND(AVG(h.time_to_recovery_ms), 2) as avg_recovery_time_ms,
    MAX(h.detected_at) as last_used_at
FROM error_recovery_history h
WHERE h.rule_name IS NOT NULL
GROUP BY h.rule_id, h.rule_name;

COMMENT ON VIEW rule_performance IS 'Performance metrics for each recovery rule';

COMMIT;
