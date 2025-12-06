package driver

import (
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"time"
)

// UserAgentGenerator generates realistic user agents matching JA3 fingerprints
type UserAgentGenerator struct {
	rng *rand.Rand
}

// NewUserAgentGenerator creates a new generator
func NewUserAgentGenerator() *UserAgentGenerator {
	return &UserAgentGenerator{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Generate returns a user agent matching the given browser name
// This should match the JA3 fingerprint browser for consistency
func (g *UserAgentGenerator) Generate(browserName string) string {
	browserName = strings.ToLower(browserName)

	switch browserName {
	case "chrome":
		return g.generateChrome()
	case "firefox":
		return g.generateFirefox()
	case "safari":
		return g.generateSafari()
	case "edge":
		return g.generateEdge()
	case "ios":
		return g.generateIOS()
	case "android":
		return g.generateAndroid()
	default:
		return g.generateChrome() // Default to Chrome
	}
}

// generateChrome generates a Chrome user agent
func (g *UserAgentGenerator) generateChrome() string {
	// Recent Chrome versions
	versions := []string{"120.0.0.0", "121.0.0.0", "122.0.0.0", "123.0.0.0", "124.0.0.0", "125.0.0.0"}
	version := versions[g.rng.Intn(len(versions))]

	platform := g.getWindowsPlatform()
	return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", platform, version)
}

// generateFirefox generates a Firefox user agent
func (g *UserAgentGenerator) generateFirefox() string {
	// Recent Firefox versions
	versions := []string{"120.0", "121.0", "122.0", "123.0", "124.0", "125.0"}
	version := versions[g.rng.Intn(len(versions))]

	platform := g.getWindowsPlatform()
	return fmt.Sprintf("Mozilla/5.0 (%s; rv:%s) Gecko/20100101 Firefox/%s", platform, version, version)
}

// generateSafari generates a Safari user agent
func (g *UserAgentGenerator) generateSafari() string {
	// Safari on macOS
	macVersions := []string{"10_15_7", "11_0", "12_0", "13_0", "14_0"}
	macVersion := macVersions[g.rng.Intn(len(macVersions))]

	safariVersions := []string{"605.1.15", "604.1", "603.1.30"}
	safariVersion := safariVersions[g.rng.Intn(len(safariVersions))]

	return fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X %s) AppleWebKit/%s (KHTML, like Gecko) Version/17.0 Safari/%s",
		macVersion, safariVersion, safariVersion)
}

// generateEdge generates an Edge user agent
func (g *UserAgentGenerator) generateEdge() string {
	// Edge is Chromium-based
	chromeVersions := []string{"120.0.0.0", "121.0.0.0", "122.0.0.0", "123.0.0.0"}
	edgeVersions := []string{"120.0.2210.91", "121.0.2277.83", "122.0.2365.52"}

	chromeVersion := chromeVersions[g.rng.Intn(len(chromeVersions))]
	edgeVersion := edgeVersions[g.rng.Intn(len(edgeVersions))]

	platform := g.getWindowsPlatform()
	return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36 Edg/%s",
		platform, chromeVersion, edgeVersion)
}

// generateIOS generates an iOS Safari user agent
func (g *UserAgentGenerator) generateIOS() string {
	iosVersions := []string{"15_0", "16_0", "17_0", "17_1", "17_2"}
	iosVersion := iosVersions[g.rng.Intn(len(iosVersions))]

	devices := []string{"iPhone", "iPad"}
	device := devices[g.rng.Intn(len(devices))]

	return fmt.Sprintf("Mozilla/5.0 (%s; CPU iPhone OS %s like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		device, iosVersion)
}

// generateAndroid generates an Android Chrome user agent
func (g *UserAgentGenerator) generateAndroid() string {
	androidVersions := []string{"11", "12", "13", "14"}
	chromeVersions := []string{"120.0.6099.43", "121.0.6167.101", "122.0.6261.64"}

	androidVersion := androidVersions[g.rng.Intn(len(androidVersions))]
	chromeVersion := chromeVersions[g.rng.Intn(len(chromeVersions))]

	return fmt.Sprintf("Mozilla/5.0 (Linux; Android %s; SM-G991B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Mobile Safari/537.36",
		androidVersion, chromeVersion)
}

// getWindowsPlatform returns a Windows platform string
func (g *UserAgentGenerator) getWindowsPlatform() string {
	if runtime.GOOS == "darwin" {
		return "Macintosh; Intel Mac OS X 10_15_7"
	}
	// Randomize Windows versions
	winVersions := []string{
		"Windows NT 10.0; Win64; x64",
		"Windows NT 10.0; WOW64",
		"Windows NT 11.0; Win64; x64",
	}
	return winVersions[g.rng.Intn(len(winVersions))]
}

// Default generator instance
var defaultUAGenerator = NewUserAgentGenerator()

// GenerateUserAgent generates a user agent for the given browser name
func GenerateUserAgent(browserName string) string {
	return defaultUAGenerator.Generate(browserName)
}
