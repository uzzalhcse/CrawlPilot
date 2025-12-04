package browser

import (
	"fmt"
	"math/rand"
	"time"
)

// FingerprintGenerator generates realistic browser fingerprints
type FingerprintGenerator struct {
	rand *rand.Rand
}

func NewFingerprintGenerator() *FingerprintGenerator {
	return &FingerprintGenerator{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// BrowserFingerprint represents a complete browser fingerprint
type BrowserFingerprint struct {
	UserAgent           string
	Platform            string
	ScreenWidth         int
	ScreenHeight        int
	Timezone            string
	Locale              string
	Languages           []string
	WebGLVendor         string
	WebGLRenderer       string
	HardwareConcurrency int
	DeviceMemory        int
	Fonts               []string
}

// GenerateFingerprint generates a random but realistic browser fingerprint
func (fg *FingerprintGenerator) GenerateFingerprint(browserType string) (*BrowserFingerprint, error) {
	switch browserType {
	case "chromium":
		return fg.generateChromiumFingerprint(), nil
	case "firefox":
		return fg.generateFirefoxFingerprint(), nil
	case "webkit":
		return fg.generateWebKitFingerprint(), nil
	default:
		return nil, fmt.Errorf("unsupported browser type: %s", browserType)
	}
}

// generateChromiumFingerprint generates a realistic Chromium-based fingerprint
func (fg *FingerprintGenerator) generateChromiumFingerprint() *BrowserFingerprint {
	platforms := []string{"Win32", "MacIntel", "Linux x86_64"}
	platform := platforms[fg.rand.Intn(len(platforms))]

	var userAgent string
	var webglVendor, webglRenderer string

	switch platform {
	case "Win32":
		versions := []string{"117.0.0.0", "118.0.0.0", "119.0.0.0", "120.0.0.0"}
		version := versions[fg.rand.Intn(len(versions))]
		userAgent = fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", version)
		webglVendor = "Google Inc. (NVIDIA)"
		webglRenderer = "ANGLE (NVIDIA, NVIDIA GeForce GTX 1660 Direct3D11 vs_5_0 ps_5_0, D3D11)"
	case "MacIntel":
		versions := []string{"117.0.0.0", "118.0.0.0", "119.0.0.0", "120.0.0.0"}
		version := versions[fg.rand.Intn(len(versions))]
		userAgent = fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", version)
		webglVendor = "Intel Inc."
		webglRenderer = "Intel Iris OpenGL Engine"
	case "Linux x86_64":
		versions := []string{"117.0.0.0", "118.0.0.0", "119.0.0.0", "120.0.0.0"}
		version := versions[fg.rand.Intn(len(versions))]
		userAgent = fmt.Sprintf("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", version)
		webglVendor = "Mesa"
		webglRenderer = "Mesa Intel(R) UHD Graphics 620 (KBL GT2)"
	}

	return &BrowserFingerprint{
		UserAgent:           userAgent,
		Platform:            platform,
		ScreenWidth:         fg.randomScreenWidth(),
		ScreenHeight:        fg.randomScreenHeight(),
		Timezone:            fg.randomTimezone(),
		Locale:              "en-US",
		Languages:           []string{"en-US", "en"},
		WebGLVendor:         webglVendor,
		WebGLRenderer:       webglRenderer,
		HardwareConcurrency: fg.randomCPUCores(),
		DeviceMemory:        fg.randomMemory(),
		Fonts:               fg.commonFonts(),
	}
}

// generateFirefoxFingerprint generates a realistic Firefox fingerprint
func (fg *FingerprintGenerator) generateFirefoxFingerprint() *BrowserFingerprint {
	platforms := []string{"Win32", "MacIntel", "Linux x86_64"}
	platform := platforms[fg.rand.Intn(len(platforms))]

	var userAgent string
	var webglVendor, webglRenderer string

	switch platform {
	case "Win32":
		versions := []string{"115.0", "116.0", "117.0", "118.0"}
		version := versions[fg.rand.Intn(len(versions))]
		userAgent = fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:%s) Gecko/20100101 Firefox/%s", version, version)
		webglVendor = "NVIDIA Corporation"
		webglRenderer = "NVIDIA GeForce GTX 1660/PCIe/SSE2"
	case "MacIntel":
		versions := []string{"115.0", "116.0", "117.0", "118.0"}
		version := versions[fg.rand.Intn(len(versions))]
		userAgent = fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:%s) Gecko/20100101 Firefox/%s", version, version)
		webglVendor = "Intel Inc."
		webglRenderer = "Intel(R) Iris(TM) Graphics 6100"
	case "Linux x86_64":
		versions := []string{"115.0", "116.0", "117.0", "118.0"}
		version := versions[fg.rand.Intn(len(versions))]
		userAgent = fmt.Sprintf("Mozilla/5.0 (X11; Linux x86_64; rv:%s) Gecko/20100101 Firefox/%s", version, version)
		webglVendor = "Mesa/X.org"
		webglRenderer = "Mesa DRI Intel(R) UHD Graphics 620 (Kabylake GT2)"
	}

	return &BrowserFingerprint{
		UserAgent:           userAgent,
		Platform:            platform,
		ScreenWidth:         fg.randomScreenWidth(),
		ScreenHeight:        fg.randomScreenHeight(),
		Timezone:            fg.randomTimezone(),
		Locale:              "en-US",
		Languages:           []string{"en-US", "en"},
		WebGLVendor:         webglVendor,
		WebGLRenderer:       webglRenderer,
		HardwareConcurrency: fg.randomCPUCores(),
		DeviceMemory:        fg.randomMemory(),
		Fonts:               fg.commonFonts(),
	}
}

// generateWebKitFingerprint generates a realistic WebKit/Safari fingerprint
func (fg *FingerprintGenerator) generateWebKitFingerprint() *BrowserFingerprint {
	// WebKit is primarily Safari on macOS/iOS
	versions := []string{"16.6", "17.0", "17.1", "17.2"}
	safariVersion := versions[fg.rand.Intn(len(versions))]

	userAgent := fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/%s Safari/605.1.15", safariVersion)

	return &BrowserFingerprint{
		UserAgent:           userAgent,
		Platform:            "MacIntel",
		ScreenWidth:         fg.randomScreenWidth(),
		ScreenHeight:        fg.randomScreenHeight(),
		Timezone:            fg.randomTimezone(),
		Locale:              "en-US",
		Languages:           []string{"en-US", "en"},
		WebGLVendor:         "Apple Inc.",
		WebGLRenderer:       "Apple GPU",
		HardwareConcurrency: fg.randomCPUCores(),
		DeviceMemory:        fg.randomMemory(),
		Fonts:               fg.commonFonts(),
	}
}

// randomScreenWidth returns a common screen width
func (fg *FingerprintGenerator) randomScreenWidth() int {
	widths := []int{1920, 1366, 1440, 1536, 2560, 3840}
	return widths[fg.rand.Intn(len(widths))]
}

// randomScreenHeight returns a common screen height
func (fg *FingerprintGenerator) randomScreenHeight() int {
	heights := []int{1080, 768, 900, 864, 1440, 2160}
	return heights[fg.rand.Intn(len(heights))]
}

// randomTimezone returns a common timezone
func (fg *FingerprintGenerator) randomTimezone() string {
	timezones := []string{
		"America/New_York",
		"America/Chicago",
		"America/Los_Angeles",
		"Europe/London",
		"Europe/Paris",
		"Europe/Berlin",
		"Asia/Tokyo",
		"Asia/Shanghai",
		"Australia/Sydney",
	}
	return timezones[fg.rand.Intn(len(timezones))]
}

// randomCPUCores returns a realistic CPU core count
func (fg *FingerprintGenerator) randomCPUCores() int {
	cores := []int{2, 4, 6, 8, 12, 16}
	return cores[fg.rand.Intn(len(cores))]
}

// randomMemory returns a realistic memory amount in GB
func (fg *FingerprintGenerator) randomMemory() int {
	memory := []int{4, 8, 16, 32}
	return memory[fg.rand.Intn(len(memory))]
}

// commonFonts returns a list of common system fonts
func (fg *FingerprintGenerator) commonFonts() []string {
	return []string{
		"Arial",
		"Courier New",
		"Georgia",
		"Times New Roman",
		"Trebuchet MS",
		"Verdana",
		"Helvetica",
		"Comic Sans MS",
		"Impact",
		"Tahoma",
	}
}

// GenerateRandomUserAgent generates a random user agent for the specified browser type
func (fg *FingerprintGenerator) GenerateRandomUserAgent(browserType string) (string, error) {
	fingerprint, err := fg.GenerateFingerprint(browserType)
	if err != nil {
		return "", err
	}
	return fingerprint.UserAgent, nil
}
