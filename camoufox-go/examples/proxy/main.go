// Example: Using camoufox-go with a proxy
package main

import (
	"fmt"
	"log"

	"github.com/uzzalhcse/camoufox-go"
)

func main() {
	// Launch Camoufox with proxy configuration
	browser, err := camoufox.NewBrowser(camoufox.Options{
		OS:       "windows",
		Headless: false,
		// Configure proxy
		Proxy: &camoufox.ProxyConfig{
			Server:   "http://82.22.93.243:7950",
			Username: "lnvmpyru",
			Password: "5un1tb1azapa",
		},
		// Enable human-like mouse movement
		Humanize: 1.5, // Max 1.5 seconds for cursor movements
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

	// Check your IP
	fmt.Println("Checking IP address via proxy...")
	if _, err := page.Goto("https://www.browserscan.net/"); err != nil {
		log.Fatalf("Failed to navigate: %v", err)
	}

	fmt.Println("Press Enter to close...")
	fmt.Scanln()
}
