// Package captcha provides automatic CAPTCHA solving capabilities for playwright-go.
// Currently supports Cloudflare challenges (interstitial and turnstile).
//
// This package is designed to work best with Camoufox browser due to its
// forceScopeAccess and Shadow DOM unlocking features, but can work with any
// playwright-go browser that supports these capabilities.
//
// Basic usage:
//
//	success, err := captcha.SolveCaptcha(page, captcha.SolveOptions{
//	    CaptchaType:   captcha.CaptchaCloudflare,
//	    ChallengeType: captcha.ChallengeInterstitial,
//	})
package captcha

import (
	"fmt"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/camoufox-captcha-go/cloudflare"
	"github.com/uzzalhcse/camoufox-captcha-go/common"
)

// CaptchaType represents the type of CAPTCHA provider
type CaptchaType string

const (
	// CaptchaCloudflare is for Cloudflare challenges
	CaptchaCloudflare CaptchaType = "cloudflare"
)

// ChallengeType represents the specific challenge type for a CAPTCHA provider
type ChallengeType string

const (
	// ChallengeInterstitial is for full-page Cloudflare interstitial challenges
	ChallengeInterstitial ChallengeType = "interstitial"
	// ChallengeTurnstile is for embedded Cloudflare Turnstile widgets
	ChallengeTurnstile ChallengeType = "turnstile"
)

// SolveOptions configures the CAPTCHA solving behavior
type SolveOptions struct {
	// CaptchaType is the type of CAPTCHA provider (default: CaptchaCloudflare)
	CaptchaType CaptchaType

	// ChallengeType is the specific challenge type (default: ChallengeInterstitial)
	ChallengeType ChallengeType

	// ExpectedContentSelector is an optional CSS selector to verify page content is accessible after solving
	ExpectedContentSelector string

	// SolveAttempts is the maximum number of attempts to solve the challenge (default: 3)
	SolveAttempts int

	// SolveClickDelay is the delay after clicking the checkbox to allow processing (default: 6s)
	SolveClickDelay time.Duration

	// WaitCheckboxAttempts is the maximum attempts to find and wait for the checkbox (default: 10)
	WaitCheckboxAttempts int

	// WaitCheckboxDelay is the delay between checkbox availability checks (default: 6s)
	WaitCheckboxDelay time.Duration

	// CheckboxClickAttempts is the maximum attempts to click the checkbox (default: 3)
	CheckboxClickAttempts int

	// AttemptDelay is the delay between solve attempts (default: 5s)
	AttemptDelay time.Duration

	// Debug enables debug logging
	Debug bool
}

// DefaultOptions returns SolveOptions with sensible defaults
func DefaultOptions() SolveOptions {
	return SolveOptions{
		CaptchaType:           CaptchaCloudflare,
		ChallengeType:         ChallengeInterstitial,
		SolveAttempts:         3,
		SolveClickDelay:       6 * time.Second,
		WaitCheckboxAttempts:  10,
		WaitCheckboxDelay:     6 * time.Second,
		CheckboxClickAttempts: 3,
		AttemptDelay:          5 * time.Second,
	}
}

// applyDefaults fills in zero values with defaults
func applyDefaults(opts *SolveOptions) {
	if opts.CaptchaType == "" {
		opts.CaptchaType = CaptchaCloudflare
	}
	if opts.ChallengeType == "" {
		opts.ChallengeType = ChallengeInterstitial
	}
	if opts.SolveAttempts == 0 {
		opts.SolveAttempts = 3
	}
	if opts.SolveClickDelay == 0 {
		opts.SolveClickDelay = 6 * time.Second
	}
	if opts.WaitCheckboxAttempts == 0 {
		opts.WaitCheckboxAttempts = 10
	}
	if opts.WaitCheckboxDelay == 0 {
		opts.WaitCheckboxDelay = 6 * time.Second
	}
	if opts.CheckboxClickAttempts == 0 {
		opts.CheckboxClickAttempts = 3
	}
	if opts.AttemptDelay == 0 {
		opts.AttemptDelay = 5 * time.Second
	}
}

// convertToCloudflareOpts converts SolveOptions to cloudflare.SolveOptions
func convertToCloudflareOpts(opts SolveOptions) cloudflare.SolveOptions {
	cfChallengeType := cloudflare.ChallengeInterstitial
	if opts.ChallengeType == ChallengeTurnstile {
		cfChallengeType = cloudflare.ChallengeTurnstile
	}

	return cloudflare.SolveOptions{
		ChallengeType:           cfChallengeType,
		ExpectedContentSelector: opts.ExpectedContentSelector,
		SolveAttempts:           opts.SolveAttempts,
		SolveClickDelay:         opts.SolveClickDelay,
		WaitCheckboxAttempts:    opts.WaitCheckboxAttempts,
		WaitCheckboxDelay:       opts.WaitCheckboxDelay,
		CheckboxClickAttempts:   opts.CheckboxClickAttempts,
		AttemptDelay:            opts.AttemptDelay,
		Debug:                   opts.Debug,
	}
}

// SolveCaptchaPage solves a CAPTCHA challenge on the given Page.
//
// This function provides a unified interface for solving different types of CAPTCHA
// on a playwright.Page. Currently supports Cloudflare challenges.
//
// Example:
//
//	success, err := captcha.SolveCaptchaPage(page, captcha.SolveOptions{
//	    CaptchaType:   captcha.CaptchaCloudflare,
//	    ChallengeType: captcha.ChallengeInterstitial,
//	})
func SolveCaptchaPage(page playwright.Page, opts SolveOptions) (bool, error) {
	applyDefaults(&opts)

	switch opts.CaptchaType {
	case CaptchaCloudflare:
		if opts.ChallengeType != ChallengeInterstitial && opts.ChallengeType != ChallengeTurnstile {
			return false, fmt.Errorf("unsupported Cloudflare challenge type: '%s'", opts.ChallengeType)
		}
		return cloudflare.SolveByClick(common.WrapPage(page), convertToCloudflareOpts(opts))

	default:
		return false, fmt.Errorf("unsupported CAPTCHA type: '%s'", opts.CaptchaType)
	}
}

// SolveCaptchaFrame solves a CAPTCHA challenge in the given Frame.
func SolveCaptchaFrame(frame playwright.Frame, opts SolveOptions) (bool, error) {
	applyDefaults(&opts)

	switch opts.CaptchaType {
	case CaptchaCloudflare:
		if opts.ChallengeType != ChallengeInterstitial && opts.ChallengeType != ChallengeTurnstile {
			return false, fmt.Errorf("unsupported Cloudflare challenge type: '%s'", opts.ChallengeType)
		}
		return cloudflare.SolveByClick(common.WrapFrame(frame), convertToCloudflareOpts(opts))

	default:
		return false, fmt.Errorf("unsupported CAPTCHA type: '%s'", opts.CaptchaType)
	}
}

// SolveCaptchaElement solves a CAPTCHA challenge on the given ElementHandle.
// Useful for Turnstile challenges where you target a specific container element.
func SolveCaptchaElement(element playwright.ElementHandle, opts SolveOptions) (bool, error) {
	applyDefaults(&opts)

	switch opts.CaptchaType {
	case CaptchaCloudflare:
		if opts.ChallengeType != ChallengeInterstitial && opts.ChallengeType != ChallengeTurnstile {
			return false, fmt.Errorf("unsupported Cloudflare challenge type: '%s'", opts.ChallengeType)
		}
		return cloudflare.SolveByClick(common.WrapElement(element), convertToCloudflareOpts(opts))

	default:
		return false, fmt.Errorf("unsupported CAPTCHA type: '%s'", opts.CaptchaType)
	}
}

// SolveCaptcha is a convenience function that accepts Page, Frame, or ElementHandle.
// It determines the type at runtime and calls the appropriate function.
// For better type safety, prefer using SolveCaptchaPage, SolveCaptchaFrame, or SolveCaptchaElement.
func SolveCaptcha(queryable interface{}, opts SolveOptions) (bool, error) {
	switch q := queryable.(type) {
	case playwright.Page:
		return SolveCaptchaPage(q, opts)
	case playwright.Frame:
		return SolveCaptchaFrame(q, opts)
	case playwright.ElementHandle:
		return SolveCaptchaElement(q, opts)
	default:
		return false, fmt.Errorf("unsupported queryable type: %T. Use playwright.Page, playwright.Frame, or playwright.ElementHandle", queryable)
	}
}
