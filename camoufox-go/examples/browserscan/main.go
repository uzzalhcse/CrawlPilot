package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	camoufox "github.com/uzzalhcse/camoufox-go"
)

// This test navigates to browserscan.net and checks the detection score
func main() {
	fmt.Println("🔍 BrowserScan Detection Test")
	fmt.Println(strings.Repeat("=", 50))

	// Launch with full stealth settings
	browser, err := camoufox.NewBrowser(camoufox.Options{
		Headless:    false,
		GeoIP:       "auto", // Auto-detect IP and spoof geolocation
		BlockWebRTC: false,  // Don't block - spoof instead
		Debug:       true,
	})
	if err != nil {
		fmt.Printf("❌ Failed to launch: %v\n", err)
		return
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		fmt.Printf("❌ Failed to create page: %v\n", err)
		return
	}

	fmt.Println("\n📱 Browser launched successfully")
	fmt.Printf("   Fingerprint OS: %s\n", browser.Fingerprint().Navigator.Platform)
	fmt.Printf("   UserAgent: %s\n", browser.Fingerprint().Navigator.UserAgent)

	// Navigate to BrowserScan
	fmt.Println("\n🌐 Navigating to browserscan.net...")
	_, err = page.Goto("https://www.browserscan.net/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(60000),
	})
	if err != nil {
		fmt.Printf("❌ Navigation failed: %v\n", err)
		return
	}

	fmt.Println("⏳ Waiting for analysis...")
	time.Sleep(10 * time.Second)

	// Extract key information
	extractInfo(page)

	// Keep browser open for manual inspection
	fmt.Println("\n📌 Browser will stay open for manual inspection")
	fmt.Println("   Press Ctrl+C to close and exit")

	// Wait indefinitely
	select {}
}

func extractInfo(page playwright.Page) {
	fmt.Println("\n📊 Detection Results:")
	fmt.Println(strings.Repeat("-", 40))

	// Try to get various scores and info
	checks := []struct {
		name     string
		selector string
	}{
		{"Overall Score", ".score-value, .total-score, [class*='percentage']"},
		{"UserAgent", "text=UserAgent >> .."},
		{"IP Address", "text=IP Address >> .."},
		{"Timezone", "text=Timezone >> .."},
		{"WebRTC", "text=WebRTC >> .."},
		{"Browser", "text=Browser >> .."},
	}

	for _, check := range checks {
		el := page.Locator(check.selector).First()
		if el != nil {
			text, err := el.TextContent()
			if err == nil && text != "" {
				text = strings.TrimSpace(text)
				if len(text) > 80 {
					text = text[:80] + "..."
				}
				fmt.Printf("   %s: %s\n", check.name, text)
			}
		}
	}

	// Check for warnings/errors
	warns, err := page.Locator(".warning, .error, [class*='fail']").All()
	if err == nil && len(warns) > 0 {
		fmt.Println("\n⚠️ Warnings detected:")
		for i, w := range warns {
			if i >= 5 {
				fmt.Println("   ...")
				break
			}
			text, _ := w.TextContent()
			fmt.Printf("   - %s\n", strings.TrimSpace(text))
		}
	}
}
