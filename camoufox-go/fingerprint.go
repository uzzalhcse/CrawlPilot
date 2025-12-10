package camoufox

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

// Fingerprint represents a BrowserForge-generated browser fingerprint
type Fingerprint struct {
	Navigator NavigatorFingerprint `json:"navigator"`
	Screen    ScreenFingerprint    `json:"screen"`
	Headers   HeadersFingerprint   `json:"headers,omitempty"`
	Battery   BatteryFingerprint   `json:"battery,omitempty"`
}

// NavigatorFingerprint contains navigator property values
type NavigatorFingerprint struct {
	UserAgent            string   `json:"userAgent"`
	AppCodeName          string   `json:"appCodeName,omitempty"`
	AppName              string   `json:"appName,omitempty"`
	AppVersion           string   `json:"appVersion,omitempty"`
	Platform             string   `json:"platform,omitempty"`
	OsCPU                string   `json:"oscpu,omitempty"`
	Language             string   `json:"language,omitempty"`
	Languages            []string `json:"languages,omitempty"`
	HardwareConcurrency  int      `json:"hardwareConcurrency,omitempty"`
	MaxTouchPoints       int      `json:"maxTouchPoints,omitempty"`
	Product              string   `json:"product,omitempty"`
	DoNotTrack           *string  `json:"doNotTrack,omitempty"`
	GlobalPrivacyControl *bool    `json:"globalPrivacyControl,omitempty"`
}

// ScreenFingerprint contains screen/window property values
type ScreenFingerprint struct {
	Width       int `json:"width,omitempty"`
	Height      int `json:"height,omitempty"`
	AvailWidth  int `json:"availWidth,omitempty"`
	AvailHeight int `json:"availHeight,omitempty"`
	AvailLeft   int `json:"availLeft,omitempty"`
	AvailTop    int `json:"availTop,omitempty"`
	ColorDepth  int `json:"colorDepth,omitempty"`
	PixelDepth  int `json:"pixelDepth,omitempty"`
	OuterWidth  int `json:"outerWidth,omitempty"`
	OuterHeight int `json:"outerHeight,omitempty"`
	InnerWidth  int `json:"innerWidth,omitempty"`
	InnerHeight int `json:"innerHeight,omitempty"`
	ScreenX     int `json:"screenX,omitempty"`
	ScreenY     int `json:"screenY,omitempty"`
	PageXOffset int `json:"pageXOffset,omitempty"`
	PageYOffset int `json:"pageYOffset,omitempty"`
}

// HeadersFingerprint contains HTTP header values
type HeadersFingerprint struct {
	AcceptEncoding string `json:"Accept-Encoding,omitempty"`
}

// BatteryFingerprint contains battery status
type BatteryFingerprint struct {
	Charging        bool     `json:"charging,omitempty"`
	ChargingTime    float64  `json:"chargingTime,omitempty"`
	DischargingTime *float64 `json:"dischargingTime,omitempty"`
	Level           float64  `json:"level,omitempty"`
}

// GenerateFingerprint generates a browser fingerprint using BrowserForge via Python script.
// It calls the bundled Python script which uses browserforge to generate realistic fingerprints.
// If browserforge is not installed, it will automatically install it.
func GenerateFingerprint(os string, screen *Screen) (*Fingerprint, error) {
	// Ensure browserforge is installed
	if err := ensureBrowserForgeInstalled(); err != nil {
		return nil, fmt.Errorf("failed to setup browserforge: %w", err)
	}

	scriptPath, err := getScriptPath()
	if err != nil {
		return nil, fmt.Errorf("fingerprint script not found: %w", err)
	}

	// Build arguments
	args := []string{scriptPath}
	if os != "" {
		args = append(args, os)
	} else {
		args = append(args, "linux")
	}

	if screen != nil && screen.MaxWidth > 0 {
		args = append(args, strconv.Itoa(screen.MaxWidth))
		if screen.MaxHeight > 0 {
			args = append(args, strconv.Itoa(screen.MaxHeight))
		}
	}

	// Execute Python script
	cmd := exec.Command("python3", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("fingerprint generation failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to execute fingerprint script: %w", err)
	}

	// Parse JSON output
	var fp Fingerprint
	if err := json.Unmarshal(output, &fp); err != nil {
		return nil, fmt.Errorf("failed to parse fingerprint JSON: %w", err)
	}

	return &fp, nil
}

// ensureBrowserForgeInstalled checks if browserforge is installed and installs it if not
func ensureBrowserForgeInstalled() error {
	// Check if browserforge is already installed
	checkCmd := exec.Command("python3", "-c", "import browserforge")
	if err := checkCmd.Run(); err != nil {
		fmt.Println("📦 Installing browserforge (required for fingerprint generation)...")

		// Try pip first, then pip3
		pipCmd := "pip"
		if _, err := exec.LookPath("pip"); err != nil {
			pipCmd = "pip3"
		}

		// Try without --break-system-packages first
		installCmd := exec.Command(pipCmd, "install", "browserforge")
		if err := installCmd.Run(); err != nil {
			// Try with --break-system-packages for PEP 668 systems
			installCmd = exec.Command(pipCmd, "install", "--break-system-packages", "browserforge")
			if err := installCmd.Run(); err != nil {
				// Try with --user flag
				installCmd = exec.Command(pipCmd, "install", "--user", "browserforge")
				if err := installCmd.Run(); err != nil {
					return fmt.Errorf("failed to install browserforge. Please run manually: pip install browserforge")
				}
			}
		}
		fmt.Println("✅ browserforge installed successfully")
	}
	return nil
}

// getScriptPath returns the path to the fingerprint generation script
func getScriptPath() (string, error) {
	// Get the directory of this Go file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to get current file path")
	}

	scriptPath := filepath.Join(filepath.Dir(filename), "scripts", "generate_fingerprint.py")
	return scriptPath, nil
}

// GenerateFingerprintFromJSON creates a Fingerprint from raw JSON
func GenerateFingerprintFromJSON(data []byte) (*Fingerprint, error) {
	var fp Fingerprint
	if err := json.Unmarshal(data, &fp); err != nil {
		return nil, err
	}
	return &fp, nil
}
