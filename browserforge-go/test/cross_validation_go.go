package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/uzzalhcse/browserforge-go/fingerprints"
	"github.com/uzzalhcse/browserforge-go/headers"
)

func main() {
	numSamples := 50

	fmt.Printf("Generating %d fingerprints with Go browserforge-go...\n", numSamples)
	fmt.Println()

	fpGen, err := fingerprints.NewFingerprintGenerator(nil)
	if err != nil {
		fmt.Printf("Error creating fingerprint generator: %v\n", err)
		return
	}

	headerGen, err := headers.NewHeaderGenerator(nil)
	if err != nil {
		fmt.Printf("Error creating header generator: %v\n", err)
		return
	}

	// Collect statistics
	browsers := make(map[string]int)
	platforms := make(map[string]int)
	screenWidths := make(map[int]int)
	hardwareConcurrencies := make(map[int]int)
	userAgents := make([]string, 0)

	for i := 0; i < numSamples; i++ {
		fp, err := fpGen.Generate(nil)
		if err != nil {
			fmt.Printf("Error generating fingerprint %d: %v\n", i, err)
			continue
		}

		ua := fp.Navigator.UserAgent
		if len(userAgents) < 5 {
			userAgents = append(userAgents, ua)
		}

		// Extract browser from user agent
		if strings.Contains(ua, "Chrome") || strings.Contains(ua, "CriOS") {
			browsers["chrome"]++
		} else if strings.Contains(ua, "Firefox") || strings.Contains(ua, "FxiOS") {
			browsers["firefox"]++
		} else if strings.Contains(ua, "Safari") && !strings.Contains(ua, "Chrome") {
			browsers["safari"]++
		} else if strings.Contains(ua, "Edge") || strings.Contains(ua, "Edg") {
			browsers["edge"]++
		} else {
			browsers["other"]++
		}

		// Platform
		platform := fp.Navigator.Platform
		if strings.Contains(platform, "Win") {
			platforms["windows"]++
		} else if strings.Contains(platform, "Mac") {
			platforms["macos"]++
		} else if strings.Contains(platform, "Linux") {
			platforms["linux"]++
		} else if strings.Contains(platform, "Android") || strings.Contains(ua, "Android") {
			platforms["android"]++
		} else if strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad") {
			platforms["ios"]++
		} else {
			platforms["other"]++
		}

		// Screen width
		screenWidths[fp.Screen.Width]++

		// Hardware concurrency
		hardwareConcurrencies[fp.Navigator.HardwareConcurrency]++
	}

	// Output results as JSON for comparison
	results := map[string]interface{}{
		"total_samples":          numSamples,
		"browsers":               browsers,
		"platforms":              platforms,
		"screen_widths":          screenWidths,
		"hardware_concurrencies": hardwareConcurrencies,
		"sample_user_agents":     userAgents,
	}

	fmt.Println("=== Go browserforge-go Statistics ===")
	jsonBytes, _ := json.MarshalIndent(results, "", "  ")
	fmt.Println(string(jsonBytes))

	// Also test headers
	fmt.Println()
	fmt.Println("=== Header Generation Test ===")
	hdrs, err := headerGen.Generate(nil)
	if err != nil {
		fmt.Printf("Error generating headers: %v\n", err)
		return
	}
	fmt.Printf("Generated %d headers\n", len(hdrs))
	ua := headers.GetUserAgent(hdrs)
	if len(ua) > 80 {
		ua = ua[:80]
	}
	fmt.Printf("User-Agent: %s...\n", ua)
}
