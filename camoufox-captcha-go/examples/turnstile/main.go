// Example: Solving Cloudflare Turnstile Challenge with Camoufox
//
// This example uses camoufox-go for best CAPTCHA solving compatibility.
// ForceScopeAccess and DisableCOOP are required for Shadow DOM access.
package main

import (
	"log"
	"time"

	"github.com/playwright-community/playwright-go"
	captcha "github.com/uzzalhcse/camoufox-captcha-go"
	camoufox "github.com/uzzalhcse/camoufox-go"
)

func main() {
	// Launch Camoufox browser with required settings for CAPTCHA solving
	browser, err := camoufox.NewBrowser(camoufox.Options{
		OS:               "linux",
		Headless:         false, // Set to true for headless mode
		DisableCOOP:      true,  // REQUIRED: Enables iframe access
		ForceScopeAccess: true,  // REQUIRED: Enables closed Shadow DOM access via shadowRootUnl
		Debug:            true,  // Optional: Enable debug output
	})
	if err != nil {
		log.Fatalf("Failed to launch browser: %v", err)
	}
	defer browser.Close()

	// Create a new page
	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("Failed to create page: %v", err)
	}

	// Navigate to a site with Cloudflare Turnstile
	log.Println("Navigating to Turnstile demo page...")
	if _, err := page.Goto("https://nopecha.com/demo/turnstile"); err != nil {
		log.Fatalf("Failed to navigate: %v", err)
	}

	// Wait for the page to load
	log.Println("Waiting for page to load...")
	time.Sleep(5 * time.Second)

	// Find the turnstile container element
	log.Println("Looking for Turnstile container...")
	turnstileContainer, err := page.WaitForSelector(".turnstile_container", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(10000),
	})
	if err != nil {
		log.Fatalf("Turnstile container not found: %v", err)
	}

	// Solve the Turnstile CAPTCHA on the container element
	log.Println("Solving Cloudflare Turnstile challenge...")
	success, err := captcha.SolveCaptchaElement(turnstileContainer, captcha.SolveOptions{
		CaptchaType:   captcha.CaptchaCloudflare,
		ChallengeType: captcha.ChallengeTurnstile,
		Debug:         true,
	})
	if err != nil {
		log.Fatalf("Error solving CAPTCHA: %v", err)
	}

	if success {
		log.Println("✅ Successfully solved Turnstile challenge!")
	} else {
		log.Println("❌ Failed to solve Turnstile challenge")
	}

	// Keep browser open for inspection
	log.Println("Keeping browser open for 10 seconds...")
	time.Sleep(10 * time.Second)
}
