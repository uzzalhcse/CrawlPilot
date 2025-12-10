// Example: Solving Cloudflare Interstitial Challenge with Camoufox
//
// This example uses camoufox-go for best CAPTCHA solving compatibility.
// ForceScopeAccess and DisableCOOP are required for Shadow DOM access.
package main

import (
	"log"
	"time"

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

	// Navigate to a site with Cloudflare interstitial protection
	log.Println("Navigating to Cloudflare demo page...")
	if _, err := page.Goto("https://www.mister-auto.es/filtro-de-aire-8-g/"); err != nil {
		log.Fatalf("Failed to navigate: %v", err)
	}

	// Wait for the page to load and Cloudflare challenge to appear
	log.Println("Waiting for page to load...")
	time.Sleep(5 * time.Second)

	// Solve the CAPTCHA
	log.Println("Solving Cloudflare interstitial challenge...")
	success, err := captcha.SolveCaptchaPage(page, captcha.SolveOptions{
		CaptchaType:   captcha.CaptchaCloudflare,
		ChallengeType: captcha.ChallengeInterstitial,
		Debug:         true,
	})
	if err != nil {
		log.Fatalf("Error solving CAPTCHA: %v", err)
	}

	if success {
		log.Println("✅ Successfully solved Cloudflare challenge!")
	} else {
		log.Println("❌ Failed to solve Cloudflare challenge")
	}

	// Keep browser open for inspection
	log.Println("Keeping browser open for 10 seconds...")
	time.Sleep(10 * time.Minute)
}
