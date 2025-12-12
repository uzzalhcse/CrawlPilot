-- Add recovery system configurable settings to system_config table
-- These can be managed from frontend

-- Tiered Proxy Settings
INSERT INTO system_config (key, value, description, category, editable) VALUES
('tiered_proxy.escalate_after_failures', '2', 'Failures before escalating to next proxy tier', 'recovery', true),
('tiered_proxy.tier_cooldown_minutes', '10', 'Cooldown before retrying lower tier', 'recovery', true),
('tiered_proxy.min_samples_for_confidence', '20', 'Min requests before considering tier stable', 'recovery', true),
('tiered_proxy.success_rate_threshold', '0.8', 'Success rate to consider tier working', 'recovery', true)
ON CONFLICT (key) DO NOTHING;

-- NOTE: Protected domains are now managed via the domain_strategies table
-- See migration 019_domain_strategies for the table structure
-- The domain_strategies table serves as the single source of truth for learned anti-bot strategies
