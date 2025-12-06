package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/worker/internal/driver"
)

func main() {
	// 1. Initialize Driver
	drv := driver.NewHttpDriver()
	defer drv.Close()

	// 2. Configure JA3 (Default should be Chrome)
	// No config passed to context
	ctx := context.Background()

	// 3. Create Page
	page, err := drv.NewPage(ctx)
	if err != nil {
		log.Fatalf("Failed to create page: %v", err)
	}
	defer page.Close()

	// 4. Make Request to JA3 fingerprinting service
	targetURL := "https://tls.peet.ws/api/all"
	fmt.Printf("Navigating to %s with Default JA3 profile (expecting Chrome)\n", targetURL)

	start := time.Now()
	if err := page.Goto(targetURL); err != nil {
		log.Fatalf("Failed to goto url: %v", err)
	}
	fmt.Printf("Request took %v\n", time.Since(start))

	// 5. Get and Print Content
	content, err := page.Content()
	if err != nil {
		log.Fatalf("Failed to get content: %v", err)
	}

	fmt.Println("\n--- Response Body ---")
	fmt.Println(content)
	fmt.Println("---------------------")
}
