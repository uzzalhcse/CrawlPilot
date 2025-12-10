// Example: Basic usage of camoufox-go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/uzzalhcse/camoufox-go"
)

func main() {
	// Launch Camoufox browser with default options
	browser, err := camoufox.NewBrowser(camoufox.Options{
		// Target OS for fingerprint (windows, macos, linux)
		OS: "linux",
		// Run in headed mode (browser window visible)
		Headless: false,
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

	// Navigate to a bot detection test site
	fmt.Println("Navigating to bot detection test...")
	if _, err := page.Goto("https://browserscan.net/"); err != nil {
		log.Fatalf("Failed to navigate: %v", err)
	}

	// Wait for page to load
	page.WaitForLoadState()

	// Take a screenshot
	screenshot, err := page.Screenshot()
	if err != nil {
		log.Printf("Failed to take screenshot: %v", err)
	} else {
		fmt.Printf("Screenshot captured: %d bytes\n", len(screenshot))
	}

	// Print the page title
	title, _ := page.Title()
	fmt.Printf("Page title: %s\n", title)

	fmt.Println("✅ Test completed successfully!")
	time.Sleep(5 * time.Minute)
}
