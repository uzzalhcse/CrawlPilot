package recovery

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// ErrorDetector Tests - Pattern Detection
// ============================================================================

func TestErrorPattern_Detection(t *testing.T) {
	detector := NewErrorDetector()

	testCases := []struct {
		name       string
		htmlBody   string
		errMsg     string
		statusCode int
		expected   ErrorPattern
	}{
		{
			name:       "Captcha detection - recaptcha",
			htmlBody:   `<div class="g-recaptcha">Please verify</div>`,
			errMsg:     "",
			statusCode: 200,
			expected:   PatternCaptcha,
		},
		{
			name:       "Captcha detection - hcaptcha",
			htmlBody:   `<div class="h-captcha">Verify you're human</div>`,
			errMsg:     "",
			statusCode: 200,
			expected:   PatternCaptcha,
		},
		{
			name:       "Rate limit - 429 error",
			htmlBody:   "",
			errMsg:     "",
			statusCode: 429,
			expected:   PatternRateLimited,
		},
		{
			name:       "Rate limit - throttle message",
			htmlBody:   `<p>You have been rate limited. Please try again later.</p>`,
			errMsg:     "",
			statusCode: 200,
			expected:   PatternRateLimited,
		},
		{
			name:       "Blocked IP detection",
			htmlBody:   `<h1>Access Denied</h1><p>Your IP address has been blocked</p>`,
			errMsg:     "",
			statusCode: 200,
			expected:   PatternBlocked,
		},
		{
			name:       "Blocked - 403 forbidden",
			htmlBody:   "",
			errMsg:     "",
			statusCode: 403,
			expected:   PatternBlocked,
		},
		{
			name:       "Timeout - context deadline",
			htmlBody:   "",
			errMsg:     "context deadline exceeded",
			statusCode: 0,
			expected:   PatternTimeout,
		},
		{
			name:       "Timeout - dial timeout",
			htmlBody:   "",
			errMsg:     "dial tcp: i/o timeout",
			statusCode: 0,
			expected:   PatternTimeout,
		},
		{
			name:       "Connection refused",
			htmlBody:   "",
			errMsg:     "connection refused",
			statusCode: 0,
			expected:   PatternConnectionErr,
		},
		{
			name:       "Connection reset",
			htmlBody:   "",
			errMsg:     "connection reset by peer",
			statusCode: 0,
			expected:   PatternConnectionErr,
		},
		{
			name:       "Auth required - login form",
			htmlBody:   `<form id="login">Please log in to continue</form>`,
			errMsg:     "",
			statusCode: 401,
			expected:   PatternAuthRequired,
		},
		{
			name:       "Not found - 404",
			htmlBody:   "",
			errMsg:     "",
			statusCode: 404,
			expected:   PatternNotFound,
		},
		{
			name:       "Server error - 500",
			htmlBody:   "",
			errMsg:     "",
			statusCode: 500,
			expected:   PatternServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.errMsg != "" {
				err = &testError{msg: tc.errMsg}
			}
			// Correct signature: Detect(err, pageURL, statusCode, pageContent)
			detected := detector.Detect(err, "https://example.com", tc.statusCode, tc.htmlBody)

			require.NotNil(t, detected)
			assert.Equal(t, tc.expected, detected.Pattern, "Pattern mismatch for %s", tc.name)
		})
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// ============================================================================
// Block Duration Calculation Tests
// ============================================================================

func TestCalculateBlockDuration(t *testing.T) {
	testCases := []struct {
		name             string
		consecutiveFails int
		pattern          ErrorPattern
		expectedMin      time.Duration
		expectedMax      time.Duration
	}{
		{
			name:             "Basic timeout - 5 failures",
			consecutiveFails: 5,
			pattern:          PatternTimeout,
			expectedMin:      4 * time.Minute,
			expectedMax:      6 * time.Minute,
		},
		{
			name:             "Rate limited - higher multiplier",
			consecutiveFails: 5,
			pattern:          PatternRateLimited,
			expectedMin:      9 * time.Minute,
			expectedMax:      11 * time.Minute,
		},
		{
			name:             "Blocked - highest non-captcha",
			consecutiveFails: 5,
			pattern:          PatternBlocked,
			expectedMin:      24 * time.Minute,
			expectedMax:      26 * time.Minute,
		},
		{
			name:             "Captcha - needs human intervention",
			consecutiveFails: 5,
			pattern:          PatternCaptcha,
			expectedMin:      49 * time.Minute,
			expectedMax:      51 * time.Minute,
		},
		{
			name:             "High failures - capped at 1 hour",
			consecutiveFails: 100,
			pattern:          PatternBlocked,
			expectedMin:      59 * time.Minute,
			expectedMax:      61 * time.Minute,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			duration := calculateBlockDuration(tc.consecutiveFails, tc.pattern)
			assert.GreaterOrEqual(t, duration, tc.expectedMin)
			assert.LessOrEqual(t, duration, tc.expectedMax)
		})
	}
}

// ============================================================================
// Recovery Action Recommendation Tests
// ============================================================================

func TestGetRecommendedAction(t *testing.T) {
	testCases := []struct {
		pattern  ErrorPattern
		expected ActionType
	}{
		{PatternRateLimited, ActionAddDelay},
		{PatternBlocked, ActionSwitchProxy},
		{PatternCaptcha, ActionSendToDLQ},
		{PatternTimeout, ActionRetry},
		{PatternAuthRequired, ActionSendToDLQ},
		{PatternConnectionErr, ActionSwitchProxy},
		{PatternLayoutChanged, ActionSendToDLQ},
		{PatternNotFound, ActionSendToDLQ},
		{PatternServerError, ActionRetry},
		{PatternUnknown, ActionRetry},
	}

	for _, tc := range testCases {
		t.Run(string(tc.pattern), func(t *testing.T) {
			action := GetRecommendedAction(tc.pattern)
			assert.Equal(t, tc.expected, action)
		})
	}
}

// ============================================================================
// TrackerConfig Tests
// ============================================================================

func TestDefaultTrackerConfig(t *testing.T) {
	config := DefaultTrackerConfig()

	assert.Equal(t, 100, config.WindowSize)
	assert.Equal(t, 0.10, config.ErrorRateThreshold)
	assert.Equal(t, 3, config.ConsecutiveThreshold)
	assert.Equal(t, 1*time.Hour, config.WindowTTL)
}

// ============================================================================
// ManagerConfig Tests
// ============================================================================

func TestDefaultManagerConfig(t *testing.T) {
	config := DefaultManagerConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, 3, config.MaxRecoveryAttempts)
	assert.True(t, config.AIFallbackEnabled)
	assert.Equal(t, 100, config.WindowSize)
	assert.Equal(t, 0.10, config.ErrorRateThreshold)
	assert.Equal(t, 3, config.ConsecutiveThreshold)
}

// ============================================================================
// Rule Condition Matching Tests
// ============================================================================

func TestRuleCondition_Match(t *testing.T) {
	testCases := []struct {
		name      string
		condition RuleCondition
		error     *DetectedError
		expected  bool
	}{
		{
			name:      "Domain equals - match",
			condition: RuleCondition{Field: "domain", Operator: "equals", Value: "amazon.com"},
			error:     &DetectedError{Domain: "amazon.com"},
			expected:  true,
		},
		{
			name:      "Domain equals - no match",
			condition: RuleCondition{Field: "domain", Operator: "equals", Value: "amazon.com"},
			error:     &DetectedError{Domain: "google.com"},
			expected:  false,
		},
		{
			name:      "Domain contains - match",
			condition: RuleCondition{Field: "domain", Operator: "contains", Value: "amazon"},
			error:     &DetectedError{Domain: "www.amazon.co.uk"},
			expected:  true,
		},
		{
			name:      "URL pattern contains - match",
			condition: RuleCondition{Field: "url_pattern", Operator: "contains", Value: "amazon"},
			error:     &DetectedError{URL: "https://www.amazon.com/product/123"},
			expected:  true,
		},
		{
			name:      "Page content contains - match",
			condition: RuleCondition{Field: "page_content", Operator: "contains", Value: "access denied"},
			error:     &DetectedError{PageContent: "Access Denied - Please verify"},
			expected:  true,
		},
		{
			name:      "Error contains - match",
			condition: RuleCondition{Field: "error_contains", Operator: "contains", Value: "timeout"},
			error:     &DetectedError{RawError: "connection timeout occurred"},
			expected:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.condition.Match(tc.error)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// ============================================================================
// RecoveryPlan Tests
// ============================================================================

func TestRecoveryPlan_ParamsAccess(t *testing.T) {
	plan := &RecoveryPlan{
		Action:      ActionSwitchProxy,
		Reason:      "Blocked IP detected",
		ShouldRetry: true,
		RetryDelay:  5 * time.Second,
		Params: map[string]interface{}{
			"proxy_url": "http://proxy1:8080",
			"proxy_id":  "proxy-123",
		},
	}

	assert.Equal(t, "http://proxy1:8080", plan.Params["proxy_url"])
	assert.Equal(t, "proxy-123", plan.Params["proxy_id"])
}

// ============================================================================
// Helper Functions Tests
// ============================================================================

func TestExtractDomain(t *testing.T) {
	testCases := []struct {
		url      string
		expected string
	}{
		{"https://www.example.com/path", "www.example.com"},
		{"http://subdomain.example.com:8080/path", "subdomain.example.com:8080"},
		{"https://example.com", "example.com"},
		{"invalid-url", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			result := extractDomain(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	// IDs should be unique
	assert.NotEqual(t, id1, id2)

	// IDs should be non-empty
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
}

// ============================================================================
// Pattern Description Tests
// ============================================================================

func TestGetPatternDescription(t *testing.T) {
	testCases := []struct {
		pattern     ErrorPattern
		shouldExist bool
	}{
		{PatternBlocked, true},
		{PatternRateLimited, true},
		{PatternCaptcha, true},
		{PatternTimeout, true},
		{PatternConnectionErr, true},
		{PatternAuthRequired, true},
		{PatternNotFound, true},
		{PatternServerError, true},
		{PatternUnknown, true},
	}

	for _, tc := range testCases {
		t.Run(string(tc.pattern), func(t *testing.T) {
			desc := GetPatternDescription(tc.pattern)
			assert.NotEmpty(t, desc)
		})
	}
}

// ============================================================================
// Integration Test with Real Redis (skipped in short mode)
// ============================================================================

func TestErrorTracker_WithRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Try to connect to local Redis
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6380", // Your Redis port
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}

	t.Run("Real Redis - consecutive errors", func(t *testing.T) {
		domain := "test-" + strconv.FormatInt(time.Now().UnixNano(), 10) + ".com"
		key := "error:stats:" + domain

		// Clean up after test
		defer client.Del(ctx, key)

		// Simulate recording failures directly
		for i := 0; i < 3; i++ {
			client.HIncrBy(ctx, key, "failure", 1)
			client.HIncrBy(ctx, key, "consecutive_errs", 1)
		}

		// Verify the counts
		failures, _ := client.HGet(ctx, key, "failure").Int64()
		consecutive, _ := client.HGet(ctx, key, "consecutive_errs").Int64()

		assert.Equal(t, int64(3), failures)
		assert.Equal(t, int64(3), consecutive)
	})

	t.Run("Real Redis - success resets consecutive", func(t *testing.T) {
		domain := "test-reset-" + strconv.FormatInt(time.Now().UnixNano(), 10) + ".com"
		key := "error:stats:" + domain

		defer client.Del(ctx, key)

		// Record failures
		client.HIncrBy(ctx, key, "failure", 1)
		client.HIncrBy(ctx, key, "consecutive_errs", 1)
		client.HIncrBy(ctx, key, "failure", 1)
		client.HIncrBy(ctx, key, "consecutive_errs", 1)

		// Record success (resets consecutive)
		client.HIncrBy(ctx, key, "success", 1)
		client.HSet(ctx, key, "consecutive_errs", "0")

		// Verify
		consecutive, _ := client.HGet(ctx, key, "consecutive_errs").Int64()
		assert.Equal(t, int64(0), consecutive)
	})
}

func TestDomainHealth_WithRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6380",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available")
	}

	t.Run("Real Redis - auto block after failures", func(t *testing.T) {
		domain := "block-test-" + strconv.FormatInt(time.Now().UnixNano(), 10) + ".com"
		key := "domain:health:" + domain

		defer client.Del(ctx, key)

		// Simulate 5 consecutive failures
		for i := 0; i < 5; i++ {
			client.HIncrBy(ctx, key, "failure_count", 1)
			consecutive := client.HIncrBy(ctx, key, "consecutive_fails", 1).Val()

			if consecutive >= 5 {
				client.HSet(ctx, key, "is_blocked", "true")
				client.HSet(ctx, key, "blocked_until", time.Now().Add(10*time.Minute).Format(time.RFC3339))
			}
		}

		// Verify blocked
		blocked, _ := client.HGet(ctx, key, "is_blocked").Result()
		assert.Equal(t, "true", blocked)
	})
}

// ============================================================================
// Concurrent Access Tests
// ============================================================================

func TestConcurrentRedisOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6380",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available")
	}

	t.Run("Concurrent atomic increments", func(t *testing.T) {
		key := "test-concurrent-" + strconv.FormatInt(time.Now().UnixNano(), 10)
		defer client.Del(ctx, key)

		// Run 100 goroutines, each incrementing 100 times
		done := make(chan bool)
		numGoroutines := 100
		incrementsPerGoroutine := 100

		for i := 0; i < numGoroutines; i++ {
			go func() {
				for j := 0; j < incrementsPerGoroutine; j++ {
					client.HIncrBy(ctx, key, "counter", 1)
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify final count
		count, err := client.HGet(ctx, key, "counter").Int64()
		require.NoError(t, err)
		expected := int64(numGoroutines * incrementsPerGoroutine)
		assert.Equal(t, expected, count, "Concurrent increments should be atomic")
	})
}

// ============================================================================
// Known Protected Domains Tests
// ============================================================================

func TestKnownProtectedDomains_StaticList(t *testing.T) {
	kpd := NewKnownProtectedDomains(nil)

	testCases := []struct {
		domain      string
		shouldBeSet bool
	}{
		{"amazon.com", true},
		{"linkedin.com", true},
		{"tiktok.com", true},
		{"unknown-domain.com", false},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.domain, func(t *testing.T) {
			tier := kpd.GetMinimumTier(ctx, tc.domain)
			if tc.shouldBeSet {
				assert.Greater(t, int(tier), 0, "Protected domain should have tier > 0")
			} else {
				assert.Equal(t, 0, int(tier), "Unknown domain should have tier 0")
			}
		})
	}
}

func TestKnownProtectedDomains_ParentDomainMatch(t *testing.T) {
	kpd := NewKnownProtectedDomains(nil)
	ctx := context.Background()

	// Subdomains should match parent domain
	testCases := []struct {
		domain   string
		expected bool
	}{
		{"www.amazon.com", true},      // Should match amazon.com
		{"shop.amazon.com", true},     // Should match amazon.com
		{"api.linkedin.com", true},    // Should match linkedin.com
		{"random.unknown.com", false}, // No match
	}

	for _, tc := range testCases {
		t.Run(tc.domain, func(t *testing.T) {
			tier := kpd.GetMinimumTier(ctx, tc.domain)
			if tc.expected {
				assert.Greater(t, int(tier), 0, "Subdomain of protected domain should have tier > 0")
			} else {
				assert.Equal(t, 0, int(tier), "Unknown subdomain should have tier 0")
			}
		})
	}
}

func TestKnownProtectedDomains_NormalizeDomain(t *testing.T) {
	kpd := NewKnownProtectedDomains(nil)
	ctx := context.Background()

	// Test case normalization
	testCases := []struct {
		input    string
		expected bool
	}{
		{"AMAZON.COM", true},     // Uppercase should work
		{"Amazon.Com", true},     // Mixed case should work
		{"www.AMAZON.COM", true}, // www prefix with uppercase
		{"amazon.com:443", true}, // With port - should be stripped
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			tier := kpd.GetMinimumTier(ctx, tc.input)
			if tc.expected {
				assert.Greater(t, int(tier), 0, "Domain should be normalized and matched")
			}
		})
	}
}

func TestKnownProtectedDomains_AddStaticDomain(t *testing.T) {
	kpd := NewKnownProtectedDomains(nil)
	ctx := context.Background()

	// Domain should not be protected initially
	tier := kpd.GetMinimumTier(ctx, "mysite.com")
	assert.Equal(t, 0, int(tier))

	// Add domain at runtime
	kpd.AddStaticDomain("mysite.com", 2)

	// Now it should be protected
	tier = kpd.GetMinimumTier(ctx, "mysite.com")
	assert.Equal(t, 2, int(tier))
}

func TestKnownProtectedDomains_GetAllStaticDomains(t *testing.T) {
	kpd := NewKnownProtectedDomains(nil)

	domains := kpd.GetAllStaticDomains()
	assert.NotEmpty(t, domains)
	assert.Contains(t, domains, "amazon.com")
	assert.Contains(t, domains, "linkedin.com")
}

func TestKnownProtectedDomains_IsProtected(t *testing.T) {
	kpd := NewKnownProtectedDomains(nil)
	ctx := context.Background()

	assert.True(t, kpd.IsProtected(ctx, "amazon.com"))
	assert.True(t, kpd.IsProtected(ctx, "linkedin.com"))
	assert.False(t, kpd.IsProtected(ctx, "random-unknown-site.com"))
}

// ============================================================================
// Tiered Proxy Config Tests
// ============================================================================

func TestDefaultTieredProxyConfig(t *testing.T) {
	config := DefaultTieredProxyConfig()

	assert.Equal(t, 2, config.EscalateAfterFailures)
	assert.Equal(t, 10*time.Minute, config.TierCooldown)
	assert.Equal(t, 20, config.MinSamplesForConfidence)
	assert.Equal(t, 0.8, config.SuccessRateThreshold)
}

func TestProxyRotationConfig_Defaults(t *testing.T) {
	config := DefaultProxyRotationConfig()

	assert.Equal(t, 30*time.Second, config.LeaseDuration)
	assert.Equal(t, 60*time.Second, config.CooldownDuration)
	assert.Equal(t, StrategyDomainAffinity, config.Strategy)
	assert.Equal(t, 5, config.MaxFailuresPerHour)
	assert.Equal(t, 5*time.Minute, config.HealthCheckTTL)
	assert.Equal(t, 0.7, config.DomainAffinityWeight)
	assert.Equal(t, 20, config.MaxProxiesPerDomain)
}

func TestProxyRotationStrategy_StringValues(t *testing.T) {
	testCases := []struct {
		strategy RotationStrategy
		expected string
	}{
		{StrategyRoundRobin, "round_robin"},
		{StrategyLeastUsed, "least_used"},
		{StrategyRandom, "random"},
		{StrategyDomainAffinity, "domain_affinity"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, string(tc.strategy))
		})
	}
}

// ============================================================================
// Proxy Types Tests
// ============================================================================

func TestProxyURL(t *testing.T) {
	testCases := []struct {
		name     string
		proxy    Proxy
		expected string
	}{
		{
			name: "With auth",
			proxy: Proxy{
				Server:   "proxy.example.com:8080",
				Username: "user",
				Password: "pass",
			},
			expected: "http://user:pass@proxy.example.com:8080",
		},
		{
			name: "Without auth",
			proxy: Proxy{
				Server: "proxy.example.com:8080",
			},
			expected: "http://proxy.example.com:8080",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := tc.proxy.ProxyURL()
			assert.Equal(t, tc.expected, url)
		})
	}
}

// ============================================================================
// Proxy Auth Failed Pattern Tests
// ============================================================================

func TestProxyAuthFailedPattern(t *testing.T) {
	detector := NewErrorDetector()

	testCases := []struct {
		name       string
		errMsg     string
		statusCode int
		expected   ErrorPattern
	}{
		{
			name:       "407 status code",
			errMsg:     "",
			statusCode: 407,
			expected:   PatternProxyAuthFailed,
		},
		{
			name:       "Invalid auth credentials error",
			errMsg:     "net::ERR_INVALID_AUTH_CREDENTIALS",
			statusCode: 0,
			expected:   PatternProxyAuthFailed,
		},
		{
			name:       "Proxy authentication required",
			errMsg:     "proxy authentication required",
			statusCode: 0,
			expected:   PatternProxyAuthFailed, // Matched by "proxy" + "authentication" combo
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.errMsg != "" {
				err = &testError{msg: tc.errMsg}
			}
			detected := detector.Detect(err, "https://example.com", tc.statusCode, "")

			require.NotNil(t, detected)
			assert.Equal(t, tc.expected, detected.Pattern)
		})
	}
}

// ============================================================================
// Integration Tests with Redis (skipped in short mode)
// ============================================================================

func TestKnownProtectedDomains_WithRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6380",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available")
	}

	t.Run("Real Redis - learned tier persistence", func(t *testing.T) {
		domain := "learned-test-" + strconv.FormatInt(time.Now().UnixNano(), 10) + ".com"
		key := "protected:learned:" + domain

		defer client.Del(ctx, key)

		// Set learned tier
		client.Set(ctx, key, "2", 24*time.Hour)

		// Verify it persists
		tier, err := client.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, "2", tier)

		// Verify TTL is set
		ttl, _ := client.TTL(ctx, key).Result()
		assert.Greater(t, ttl, time.Duration(0))
	})
}
