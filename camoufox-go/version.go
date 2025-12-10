package camoufox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// VersionInfo contains Camoufox version information
type VersionInfo struct {
	Version string `json:"version"` // Firefox version (e.g., "135.0")
	Release string `json:"release"` // Camoufox release (e.g., "0.1.3")
}

// GetVersionInfo reads the version info from the installed Camoufox
func GetVersionInfo() (*VersionInfo, error) {
	// Try to find version.json in Camoufox install directory
	searchPaths := []string{
		filepath.Join(os.Getenv("HOME"), ".cache", "camoufox", "version.json"),
		filepath.Join(os.Getenv("HOME"), ".cache", "camoufox-go", "version.json"),
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var info VersionInfo
			if err := json.Unmarshal(data, &info); err != nil {
				continue
			}
			return &info, nil
		}
	}

	// Fallback: try to get version from executable path
	execPath, err := GetExecutablePath()
	if err == nil {
		// Check for version.json next to executable
		dir := filepath.Dir(execPath)
		versionPath := filepath.Join(dir, "..", "version.json")
		if data, err := os.ReadFile(versionPath); err == nil {
			var info VersionInfo
			if json.Unmarshal(data, &info) == nil {
				return &info, nil
			}
		}
	}

	// Default fallback version (Firefox 142 via coryking's fork)
	return &VersionInfo{Version: "142.0", Release: "0.4.12"}, nil
}

// UpdateUserAgentVersion updates the UserAgent to match the actual Firefox version
// This fixes the "Different browser version" detection
// Python uses: re.sub(r'(?<!\d)(1[0-9]{2})(\.0)(?!\d)', rf'{ff_version}\g<2>', data)
func UpdateUserAgentVersion(userAgent string, actualVersion string) string {
	if userAgent == "" || actualVersion == "" {
		return userAgent
	}

	// Extract major version number (e.g., "135" from "135.0.1")
	majorVersion := strings.Split(actualVersion, ".")[0]

	// Python pattern: (?<!\d)(1[0-9]{2})(\.0)(?!\d)
	// This matches 100-199 followed by .0 (like "145.0" in BrowserForge fingerprints)
	// Go doesn't support lookbehind, so we use a different approach

	// Replace Firefox/1XX.0 pattern (e.g., Firefox/145.0 -> Firefox/135.0)
	firefoxPattern := regexp.MustCompile(`Firefox/1\d{2}\.0`)
	result := firefoxPattern.ReplaceAllString(userAgent, "Firefox/"+majorVersion+".0")

	// Replace rv:1XX.0 pattern (e.g., rv:145.0 -> rv:135.0)
	rvPattern := regexp.MustCompile(`rv:1\d{2}\.0`)
	result = rvPattern.ReplaceAllString(result, "rv:"+majorVersion+".0")

	// Also fix Gecko/20100101 - this should stay the same (it's a date, not version)
	// No change needed

	return result
}

// UpdateAppVersionVersion updates appVersion to match Firefox version
func UpdateAppVersionVersion(appVersion string, actualVersion string) string {
	if appVersion == "" || actualVersion == "" {
		return appVersion
	}

	// Replace the version in appVersion (e.g., "5.0 (X11)" stays same, version is in UA)
	return appVersion
}
