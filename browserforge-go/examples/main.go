package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/uzzalhcse/browserforge-go/fingerprints"
	"github.com/uzzalhcse/browserforge-go/headers"
)

func main() {
	fmt.Println("=== BrowserForge-Go Example ===")
	fmt.Println()

	// Example 1: Generate Headers
	fmt.Println("1. Generating HTTP Headers...")
	headerGen, err := headers.NewHeaderGenerator(nil)
	if err != nil {
		log.Fatalf("Failed to create header generator: %v", err)
	}

	generatedHeaders, err := headerGen.Generate(nil)
	if err != nil {
		log.Fatalf("Failed to generate headers: %v", err)
	}

	fmt.Println("Generated Headers:")
	for key, value := range generatedHeaders {
		fmt.Printf("  %s: %s\n", key, value)
	}
	fmt.Println()

	// Example 2: Generate Headers with Browser Constraints
	fmt.Println("2. Generating Headers for Chrome on Windows...")
	chromeHeaders, err := headerGen.Generate(&headers.GenerateOptions{
		Browsers: []interface{}{"chrome"},
		OS:       []string{"windows"},
		Devices:  []string{"desktop"},
	})
	if err != nil {
		log.Fatalf("Failed to generate Chrome headers: %v", err)
	}

	fmt.Println("Chrome/Windows Headers:")
	fmt.Printf("  User-Agent: %s\n", headers.GetUserAgent(chromeHeaders))
	fmt.Println()

	// Example 3: Generate Fingerprint
	fmt.Println("3. Generating Browser Fingerprint...")
	fpGen, err := fingerprints.NewFingerprintGenerator(nil)
	if err != nil {
		log.Fatalf("Failed to create fingerprint generator: %v", err)
	}

	fp, err := fpGen.Generate(nil)
	if err != nil {
		log.Fatalf("Failed to generate fingerprint: %v", err)
	}

	fmt.Println("Generated Fingerprint:")
	fmt.Printf("  User-Agent: %s\n", fp.Navigator.UserAgent)
	fmt.Printf("  Platform: %s\n", fp.Navigator.Platform)
	fmt.Printf("  Languages: %v\n", fp.Navigator.Languages)
	fmt.Printf("  Screen: %dx%d\n", fp.Screen.Width, fp.Screen.Height)
	fmt.Printf("  Hardware Concurrency: %d\n", fp.Navigator.HardwareConcurrency)
	if fp.VideoCard != nil {
		fmt.Printf("  Video Card: %s\n", fp.VideoCard.Renderer)
	}
	fmt.Println()

	// Example 4: Generate Fingerprint with Screen Constraints
	fmt.Println("4. Generating Fingerprint with Screen Constraints...")
	minWidth := 1920
	minHeight := 1080
	fpWithScreen, err := fpGen.Generate(&fingerprints.GenerateFingerprintOptions{
		Screen: &fingerprints.Screen{
			MinWidth:  &minWidth,
			MinHeight: &minHeight,
		},
	})
	if err != nil {
		log.Printf("Note: Screen constrained fingerprint generation: %v", err)
	} else {
		fmt.Printf("  Screen: %dx%d\n", fpWithScreen.Screen.Width, fpWithScreen.Screen.Height)
	}
	fmt.Println()

	// Example 5: Serialize Fingerprint to JSON
	fmt.Println("5. Fingerprint as JSON (first 500 chars):")
	jsonStr, err := fp.Dumps()
	if err != nil {
		log.Fatalf("Failed to serialize fingerprint: %v", err)
	}
	if len(jsonStr) > 500 {
		jsonStr = jsonStr[:500] + "..."
	}
	fmt.Println(jsonStr)
	fmt.Println()

	// Compact output for verification
	fmt.Println("=== Summary ===")
	summary := map[string]interface{}{
		"headers_generated":     len(generatedHeaders) > 0,
		"user_agent_present":    headers.GetUserAgent(generatedHeaders) != "",
		"fingerprint_generated": fp != nil,
		"screen_width":          fp.Screen.Width,
		"screen_height":         fp.Screen.Height,
		"platform":              fp.Navigator.Platform,
	}
	summaryJSON, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Println(string(summaryJSON))
}
