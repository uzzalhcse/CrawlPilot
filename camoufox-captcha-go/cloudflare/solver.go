package cloudflare

import (
	"log"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/camoufox-captcha-go/common"
)

// CloudflareIframeSrcFilter is the URL pattern for Cloudflare challenge iframes
const CloudflareIframeSrcFilter = "https://challenges.cloudflare.com/cdn-cgi/challenge-platform/"

// SolveOptions configures the Cloudflare solver
type SolveOptions struct {
	// ChallengeType is the type of Cloudflare challenge
	ChallengeType ChallengeType

	// ExpectedContentSelector is a CSS selector to verify solving success
	ExpectedContentSelector string

	// SolveAttempts is the maximum attempts for solving (default: 3)
	SolveAttempts int

	// SolveClickDelay is the delay after clicking checkbox (default: 6s)
	SolveClickDelay time.Duration

	// WaitCheckboxAttempts is the max attempts to find checkbox (default: 10)
	WaitCheckboxAttempts int

	// WaitCheckboxDelay is the delay between checkbox checks (default: 6s)
	WaitCheckboxDelay time.Duration

	// CheckboxClickAttempts is the max attempts to click checkbox (default: 3)
	CheckboxClickAttempts int

	// AttemptDelay is the delay between solve attempts (default: 5s)
	AttemptDelay time.Duration

	// Debug enables debug logging
	Debug bool
}

// SolveByClick solves a Cloudflare challenge by clicking the checkbox.
//
// This function:
// 1. Detects if a Cloudflare challenge is present
// 2. Finds Cloudflare iframes in the Shadow DOM
// 3. Searches for the checkbox and waits until it's visible
// 4. Clicks the checkbox
// 5. Verifies success
//
// Parameters:
//   - queryable: PageOrFrame containing the challenge
//   - opts: Configuration options
//
// Returns:
//   - bool: true if solved successfully, false otherwise
//   - error: any error that occurred
func SolveByClick(queryable common.PageOrFrame, opts SolveOptions) (bool, error) {
	if opts.Debug {
		log.Printf("[Captcha] Starting Cloudflare %s challenge solving by click...", opts.ChallengeType)
	}

	for attempt := 0; attempt < opts.SolveAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(opts.AttemptDelay)
			if opts.Debug {
				log.Printf("[Captcha] Retrying to solve (%d/%d)...", attempt+1, opts.SolveAttempts)
			}
		}

		// 1. Check if Cloudflare challenge is present
		cloudflareDetected, err := DetectCloudflareChallenge(queryable, opts.ChallengeType)
		if err != nil && opts.Debug {
			log.Printf("[Captcha] Error detecting Cloudflare challenge: %v", err)
		}

		expectedContentDetected, err := common.DetectExpectedContent(queryable, opts.ExpectedContentSelector)
		if err != nil && opts.Debug {
			log.Printf("[Captcha] Error detecting expected content: %v", err)
		}

		if !cloudflareDetected || expectedContentDetected {
			if opts.Debug {
				log.Println("[Captcha] No Cloudflare challenge detected")
			}
			return true, nil
		}

		// 2. Find Cloudflare iframes
		cfIframes, err := common.FindAllIframesMultiple(queryable, CloudflareIframeSrcFilter)
		if err != nil {
			if opts.Debug {
				log.Printf("[Captcha] Error searching for Cloudflare iframes: %v", err)
			}
			continue
		}

		if opts.Debug {
			log.Printf("[Captcha] Found %d iframes matching Cloudflare filter", len(cfIframes))

			// Debug: List all iframes on the page regardless of filter
			allIframes, _ := common.FindAllIframesMultiple(queryable, "")
			log.Printf("[Captcha] Total iframes on page: %d", len(allIframes))
			for i, iframe := range allIframes {
				log.Printf("[Captcha]   iframe[%d]: %s", i, iframe.URL())
			}
		}

		if len(cfIframes) == 0 {
			if opts.Debug {
				log.Println("[Captcha] Cloudflare iframes not found")
			}
			continue
		}

		// 3. Find and wait for the checkbox
		checkboxData, err := GetReadyCheckbox(cfIframes, opts.WaitCheckboxDelay, opts.WaitCheckboxAttempts, opts.Debug)
		if err != nil {
			if opts.Debug {
				log.Printf("[Captcha] Error getting checkbox: %v", err)
			}
			continue
		}

		if checkboxData == nil {
			if opts.Debug {
				log.Println("[Captcha] Cloudflare checkbox not found or not ready")
			}
			continue
		}

		if opts.Debug {
			log.Println("[Captcha] Found checkbox in Cloudflare iframe")
		}

		// 4. Click the checkbox
		clickSuccess := false
		for clickAttempt := 0; clickAttempt < opts.CheckboxClickAttempts; clickAttempt++ {
			err := checkboxData.Checkbox.Click()
			if err != nil {
				if opts.Debug {
					log.Printf("[Captcha] Error clicking checkbox (%d/%d attempt): %v",
						clickAttempt+1, opts.CheckboxClickAttempts, err)
				}
				continue
			}

			if opts.Debug {
				log.Println("[Captcha] Checkbox clicked successfully")
			}
			clickSuccess = true
			break
		}

		if !clickSuccess {
			if opts.Debug {
				log.Println("[Captcha] Failed to click checkbox after maximum attempts")
			}
			continue
		}

		// Wait for Cloudflare to process the click
		time.Sleep(opts.SolveClickDelay)

		// 5. Verify success
		var challengeSolved bool
		if opts.ChallengeType == ChallengeTurnstile {
			// For turnstile, check for success element in the iframe
			wrappedFrame := common.WrapFrame(checkboxData.Frame)
			successElements, err := common.SearchShadowRootElements(wrappedFrame, `div[id="success"]`)
			if err == nil {
				challengeSolved = len(successElements) > 0
			}
		} else {
			// For interstitial, check if challenge is gone
			cloudflareDetected, err := DetectCloudflareChallenge(queryable, opts.ChallengeType)
			if err == nil {
				challengeSolved = !cloudflareDetected
			}
		}

		expectedContentDetected, err = common.DetectExpectedContent(queryable, opts.ExpectedContentSelector)
		if err != nil && opts.Debug {
			log.Printf("[Captcha] Error detecting expected content after solve: %v", err)
		}

		if challengeSolved || expectedContentDetected {
			if opts.Debug {
				log.Println("[Captcha] Solved successfully")
			}
			return true, nil
		}

		if opts.Debug {
			log.Println("[Captcha] Failed to solve Cloudflare challenge")
		}
	}

	if opts.Debug {
		log.Println("[Captcha] Max solving attempts reached, giving up")
	}
	return false, nil
}

// SolveByClickPage solves a Cloudflare challenge on a Page.
func SolveByClickPage(page playwright.Page, opts SolveOptions) (bool, error) {
	return SolveByClick(common.WrapPage(page), opts)
}

// SolveByClickFrame solves a Cloudflare challenge in a Frame.
func SolveByClickFrame(frame playwright.Frame, opts SolveOptions) (bool, error) {
	return SolveByClick(common.WrapFrame(frame), opts)
}

// SolveByClickElement solves a Cloudflare challenge on an Element.
func SolveByClickElement(element playwright.ElementHandle, opts SolveOptions) (bool, error) {
	return SolveByClick(common.WrapElement(element), opts)
}
