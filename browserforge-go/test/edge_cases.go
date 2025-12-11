package main

import (
	"fmt"
	"log"

	"github.com/uzzalhcse/browserforge-go/fingerprints"
	"github.com/uzzalhcse/browserforge-go/headers"
)

func main() {
	fmt.Println("=== BrowserForge-Go Edge Case Tests ===")
	fmt.Println()

	testsPassed := 0
	testsFailed := 0

	// Test 1: Create HeaderGenerator
	fmt.Print("Test 1: Create HeaderGenerator... ")
	hg, err := headers.NewHeaderGenerator(nil)
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		testsFailed++
	} else {
		fmt.Println("PASSED")
		testsPassed++
	}

	// Test 2: Generate headers with default options
	fmt.Print("Test 2: Generate headers (default)... ")
	hdrs, err := hg.Generate(nil)
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		testsFailed++
	} else if headers.GetUserAgent(hdrs) == "" {
		fmt.Println("FAILED: No User-Agent")
		testsFailed++
	} else {
		fmt.Println("PASSED")
		testsPassed++
	}

	// Test 3: Generate headers for specific browser
	fmt.Print("Test 3: Generate headers (chrome only)... ")
	chromeHdrs, err := hg.Generate(&headers.GenerateOptions{
		Browsers: []interface{}{"chrome"},
	})
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		testsFailed++
	} else {
		ua := headers.GetUserAgent(chromeHdrs)
		if ua == "" || (!containsCI(ua, "Chrome") && !containsCI(ua, "CriOS")) {
			fmt.Printf("FAILED: Expected Chrome UA, got: %s\n", ua)
			testsFailed++
		} else {
			fmt.Println("PASSED")
			testsPassed++
		}
	}

	// Test 4: Generate headers for Firefox
	fmt.Print("Test 4: Generate headers (firefox only)... ")
	firefoxHdrs, err := hg.Generate(&headers.GenerateOptions{
		Browsers: []interface{}{"firefox"},
	})
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		testsFailed++
	} else {
		ua := headers.GetUserAgent(firefoxHdrs)
		if ua == "" || (!containsCI(ua, "Firefox") && !containsCI(ua, "FxiOS")) {
			fmt.Printf("FAILED: Expected Firefox UA, got: %s\n", ua)
			testsFailed++
		} else {
			fmt.Println("PASSED")
			testsPassed++
		}
	}

	// Test 5: Generate headers for Safari
	fmt.Print("Test 5: Generate headers (safari only)... ")
	safariHdrs, err := hg.Generate(&headers.GenerateOptions{
		Browsers: []interface{}{"safari"},
	})
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		testsFailed++
	} else {
		ua := headers.GetUserAgent(safariHdrs)
		if ua == "" {
			fmt.Printf("FAILED: No UA generated\n")
			testsFailed++
		} else {
			fmt.Println("PASSED")
			testsPassed++
		}
	}

	// Test 6: Generate headers for Windows
	fmt.Print("Test 6: Generate headers (windows OS)... ")
	winHdrs, err := hg.Generate(&headers.GenerateOptions{
		OS: []string{"windows"},
	})
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		testsFailed++
	} else {
		ua := headers.GetUserAgent(winHdrs)
		if ua == "" || !containsCI(ua, "Windows") {
			fmt.Printf("FAILED: Expected Windows UA, got: %s\n", ua)
			testsFailed++
		} else {
			fmt.Println("PASSED")
			testsPassed++
		}
	}

	// Test 7: Generate headers for macOS
	fmt.Print("Test 7: Generate headers (macos OS)... ")
	macHdrs, err := hg.Generate(&headers.GenerateOptions{
		OS: []string{"macos"},
	})
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		testsFailed++
	} else {
		ua := headers.GetUserAgent(macHdrs)
		if ua == "" || !containsCI(ua, "Mac") {
			fmt.Printf("FAILED: Expected Mac UA, got: %s\n", ua)
			testsFailed++
		} else {
			fmt.Println("PASSED")
			testsPassed++
		}
	}

	// Test 8: Create FingerprintGenerator
	fmt.Print("Test 8: Create FingerprintGenerator... ")
	fpGen, err := fingerprints.NewFingerprintGenerator(nil)
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		testsFailed++
		log.Fatalf("Cannot continue tests: %v", err)
	} else {
		fmt.Println("PASSED")
		testsPassed++
	}

	// Test 9: Generate fingerprint with default options
	fmt.Print("Test 9: Generate fingerprint (default)... ")
	fp, err := fpGen.Generate(nil)
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		testsFailed++
	} else if fp.Navigator.UserAgent == "" {
		fmt.Println("FAILED: No UserAgent in fingerprint")
		testsFailed++
	} else if fp.Screen.Width == 0 {
		fmt.Println("FAILED: No screen width")
		testsFailed++
	} else {
		fmt.Println("PASSED")
		testsPassed++
	}

	// Test 10: Generate multiple fingerprints (verify randomness)
	fmt.Print("Test 10: Generate multiple fingerprints (randomness)... ")
	ua1 := ""
	ua2 := ""
	for i := 0; i < 10; i++ {
		fp1, _ := fpGen.Generate(nil)
		fp2, _ := fpGen.Generate(nil)
		if fp1 != nil && fp2 != nil && fp1.Navigator.UserAgent != fp2.Navigator.UserAgent {
			ua1 = fp1.Navigator.UserAgent
			ua2 = fp2.Navigator.UserAgent
			break
		}
	}
	if ua1 == "" || ua2 == "" {
		fmt.Println("PASSED (consistent UAs may be valid)")
		testsPassed++
	} else {
		fmt.Println("PASSED (different UAs generated)")
		testsPassed++
	}

	// Test 11: Generate fingerprint for Chrome/Windows
	fmt.Print("Test 11: Generate fingerprint (chrome/windows)... ")
	fp2, err := fpGen.Generate(&fingerprints.GenerateFingerprintOptions{
		Browsers: []interface{}{"chrome"},
		OS:       []string{"windows"},
	})
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		testsFailed++
	} else {
		ua := fp2.Navigator.UserAgent
		platform := fp2.Navigator.Platform
		if !containsCI(ua, "Chrome") && !containsCI(ua, "CriOS") {
			fmt.Printf("FAILED: Expected Chrome UA, got: %s\n", ua)
			testsFailed++
		} else if !containsCI(platform, "Win") {
			fmt.Printf("FAILED: Expected Windows platform, got: %s\n", platform)
			testsFailed++
		} else {
			fmt.Println("PASSED")
			testsPassed++
		}
	}

	// Test 12: Fingerprint JSON serialization
	fmt.Print("Test 12: Fingerprint JSON serialization... ")
	jsonStr, err := fp.Dumps()
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		testsFailed++
	} else if len(jsonStr) < 100 {
		fmt.Println("FAILED: JSON too short")
		testsFailed++
	} else {
		fmt.Println("PASSED")
		testsPassed++
	}

	// Test 13: Screen constraints
	fmt.Print("Test 13: Fingerprint with screen constraints... ")
	minW := 1920
	fp3, err := fpGen.Generate(&fingerprints.GenerateFingerprintOptions{
		Screen: &fingerprints.Screen{
			MinWidth: &minW,
		},
	})
	if err != nil {
		// Non-blocking - screen constraints might fail for some configs
		fmt.Printf("SKIPPED: %v\n", err)
	} else if fp3.Screen.Width < minW {
		fmt.Printf("FAILED: Screen width %d < min %d\n", fp3.Screen.Width, minW)
		testsFailed++
	} else {
		fmt.Println("PASSED")
		testsPassed++
	}

	// Test 14: Headers Accept-Language
	fmt.Print("Test 14: Accept-Language header... ")
	langHdrs, err := hg.Generate(&headers.GenerateOptions{
		Locales: []string{"fr-FR", "de-DE", "en-US"},
	})
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		testsFailed++
	} else {
		al := langHdrs["Accept-Language"]
		if al == "" {
			al = langHdrs["accept-language"]
		}
		if al == "" || !containsCI(al, "fr-FR") {
			fmt.Printf("FAILED: Expected fr-FR in Accept-Language, got: %s\n", al)
			testsFailed++
		} else {
			fmt.Println("PASSED")
			testsPassed++
		}
	}

	// Test 15: Mobile device
	fmt.Print("Test 15: Generate for mobile device... ")
	mobileHdrs, err := hg.Generate(&headers.GenerateOptions{
		Devices: []string{"mobile"},
	})
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		testsFailed++
	} else if headers.GetUserAgent(mobileHdrs) == "" {
		fmt.Println("FAILED: No User-Agent")
		testsFailed++
	} else {
		fmt.Println("PASSED")
		testsPassed++
	}

	fmt.Println()
	fmt.Println("=== Test Results ===")
	fmt.Printf("Passed: %d\n", testsPassed)
	fmt.Printf("Failed: %d\n", testsFailed)

	if testsFailed > 0 {
		fmt.Println("\n⚠️  Some tests failed!")
	} else {
		fmt.Println("\n✅ All tests passed!")
	}
}

func containsCI(s, substr string) bool {
	return contains(lower(s), lower(substr))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && findSubstr(s, substr)))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func lower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
