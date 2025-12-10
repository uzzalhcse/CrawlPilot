// Package cloudflare provides Cloudflare CAPTCHA solving capabilities.
package cloudflare

import (
	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/camoufox-captcha-go/common"
)

// ChallengeType represents the type of Cloudflare challenge
type ChallengeType string

const (
	// ChallengeInterstitial is for full-page Cloudflare interstitial challenges
	ChallengeInterstitial ChallengeType = "interstitial"
	// ChallengeTurnstile is for embedded Cloudflare Turnstile widgets
	ChallengeTurnstile ChallengeType = "turnstile"
)

// Selectors for detecting Cloudflare interstitial challenge (full page)
var CFInterstitialSelectors = []string{
	`script[src*="/cdn-cgi/challenge-platform/"]`,
}

// Selectors for detecting Cloudflare turnstile challenge (embedded widget)
var CFTurnstileSelectors = []string{
	`input[name="cf-turnstile-response"]`,
	`script[src*="challenges.cloudflare.com/turnstile/v0"]`,
}

// DetectCloudflareChallenge checks if a Cloudflare challenge is present.
//
// Parameters:
//   - queryable: PageOrFrame to search in
//   - challengeType: Type of challenge to detect
//
// Returns:
//   - bool: true if Cloudflare challenge is detected, false otherwise
//   - error: any error that occurred
func DetectCloudflareChallenge(queryable common.PageOrFrame, challengeType ChallengeType) (bool, error) {
	var selectors []string
	if challengeType == ChallengeTurnstile {
		selectors = CFTurnstileSelectors
	} else {
		selectors = CFInterstitialSelectors
	}

	for _, selector := range selectors {
		element, err := queryable.QuerySelector(selector)
		if err != nil {
			continue
		}
		if element != nil {
			return true, nil
		}
	}

	return false, nil
}

// DetectCloudflareChallengePage checks if a Cloudflare challenge is present on a Page.
func DetectCloudflareChallengePage(page playwright.Page, challengeType ChallengeType) (bool, error) {
	return DetectCloudflareChallenge(common.WrapPage(page), challengeType)
}

// DetectCloudflareChallengeFrame checks if a Cloudflare challenge is present in a Frame.
func DetectCloudflareChallengeFrame(frame playwright.Frame, challengeType ChallengeType) (bool, error) {
	return DetectCloudflareChallenge(common.WrapFrame(frame), challengeType)
}

// DetectCloudflareChallengeElement checks if a Cloudflare challenge is present in an Element.
func DetectCloudflareChallengeElement(element playwright.ElementHandle, challengeType ChallengeType) (bool, error) {
	return DetectCloudflareChallenge(common.WrapElement(element), challengeType)
}
