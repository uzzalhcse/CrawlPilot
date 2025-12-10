// Example demonstrating the limitation of regular playwright with Cloudflare
// This will NOT work because Cloudflare uses closed shadow roots
// which are only accessible with Camoufox's forceScopeAccess feature
package main

import (
	"log"
	"time"

	captcha "github.com/uzzalhcse/camoufox-captcha-go"

	"github.com/playwright-community/playwright-go"
)

func main() {
	// Install playwright if needed
	err := playwright.Install()
	if err != nil {
		log.Fatalf("Could not install playwright: %v", err)
	}

	// Start playwright
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("Could not start playwright: %v", err)
	}
	defer pw.Stop()

	// Launch Chrome browser
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	if err != nil {
		log.Fatalf("Could not launch browser: %v", err)
	}
	defer browser.Close()

	// Create a new page
	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("Could not create page: %v", err)
	}

	// Navigate to Cloudflare demo page
	log.Println("Navigating to Cloudflare demo page...")
	if _, err := page.Goto("https://nopecha.com/demo/cloudflare"); err != nil {
		log.Fatalf("Failed to navigate: %v", err)
	}

	// Wait for page to load
	log.Println("Waiting for page to load...")
	time.Sleep(5 * time.Second)

	// Try to solve the CAPTCHA (this will likely fail)
	log.Println("Attempting to solve Cloudflare challenge...")
	log.Println("⚠️  Note: This will likely fail because Chrome cannot access closed shadow roots!")

	opts := captcha.DefaultOptions()
	opts.ChallengeType = captcha.ChallengeInterstitial
	opts.Debug = true

	success, err := captcha.SolveCaptchaPage(page, opts)
	if err != nil {
		log.Printf("Error: %v", err)
	}

	if success {
		log.Println("✅ Successfully solved Cloudflare challenge!")
	} else {
		log.Println("❌ Failed to solve Cloudflare challenge")
		log.Println("")
		log.Println("This is expected! Regular Chrome/Chromium cannot access closed shadow roots.")
		log.Println("Cloudflare specifically uses closed shadow roots to prevent automation.")
		log.Println("")
		log.Println("To solve Cloudflare CAPTCHAs, you must use Camoufox with:")
		log.Println("  - DisableCOOP: true")
		log.Println("  - ForceScopeAccess: true")
	}

	// Keep browser open for inspection
	log.Println("")
	log.Println("Keeping browser open for 30 seconds for inspection...")
	time.Sleep(30 * time.Second)
}
