-- Remove recovery system config from system_config table

DELETE FROM system_config WHERE key IN (
    'tiered_proxy.escalate_after_failures',
    'tiered_proxy.tier_cooldown_minutes',
    'tiered_proxy.min_samples_for_confidence',
    'tiered_proxy.success_rate_threshold'
);
