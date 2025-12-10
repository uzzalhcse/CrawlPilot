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

	// Default fallback version
	return &VersionInfo{Version: "135.0", Release: "0.1.3"}, nil
}

// UpdateUserAgentVersion updates the UserAgent to match the actual Firefox version
// This fixes the "Different browser version" detection
func UpdateUserAgentVersion(userAgent string, actualVersion string) string {
	if userAgent == "" || actualVersion == "" {
		return userAgent
	}

	// Pattern to match Firefox version in UserAgent
	// e.g., "Firefox/145.0" -> "Firefox/135.0"
	firefoxPattern := regexp.MustCompile(`Firefox/(\d+\.?\d*)`)
	rvPattern := regexp.MustCompile(`rv:(\d+\.?\d*)`)

	// Extract major version
	majorVersion := strings.Split(actualVersion, ".")[0]
	fullVersion := actualVersion
	if !strings.Contains(actualVersion, ".") {
		fullVersion = actualVersion + ".0"
	}

	// Replace Firefox version
	result := firefoxPattern.ReplaceAllString(userAgent, "Firefox/"+fullVersion)
	// Replace rv: version (should match major version)
	result = rvPattern.ReplaceAllString(result, "rv:"+majorVersion+".0")

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
