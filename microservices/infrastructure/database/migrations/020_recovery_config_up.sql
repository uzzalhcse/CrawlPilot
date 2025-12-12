-- Add recovery system configurable settings to system_config table
-- These can be managed from frontend

-- Tiered Proxy Settings
INSERT INTO system_config (key, value, description, category, editable) VALUES
('tiered_proxy.escalate_after_failures', '2', 'Failures before escalating to next proxy tier', 'recovery', true),
('tiered_proxy.tier_cooldown_minutes', '10', 'Cooldown before retrying lower tier', 'recovery', true),
('tiered_proxy.min_samples_for_confidence', '20', 'Min requests before considering tier stable', 'recovery', true),
('tiered_proxy.success_rate_threshold', '0.8', 'Success rate to consider tier working', 'recovery', true)
ON CONFLICT (key) DO NOTHING;

-- Protected Domains (JSON map: domain -> minimum tier)
-- Tier values: 0=Direct, 1=Datacenter, 2=Residential, 3=Mobile
INSERT INTO system_config (key, value, description, category, editable) VALUES
('protected_domains', '{
  "amazon.com": 2,
  "amazon.co.uk": 2,
  "amazon.de": 2,
  "amazon.co.jp": 2,
  "ebay.com": 1,
  "walmart.com": 2,
  "target.com": 2,
  "bestbuy.com": 2,
  "linkedin.com": 2,
  "facebook.com": 2,
  "instagram.com": 2,
  "twitter.com": 2,
  "x.com": 2,
  "tiktok.com": 3,
  "google.com": 2,
  "booking.com": 2,
  "expedia.com": 2,
  "airbnb.com": 2,
  "tripadvisor.com": 2,
  "zillow.com": 2,
  "indeed.com": 2,
  "glassdoor.com": 2,
  "cloudflare.com": 2
}', 'Domains that require proxies (0=Direct, 1=Datacenter, 2=Residential, 3=Mobile)', 'recovery', true)
ON CONFLICT (key) DO NOTHING;
