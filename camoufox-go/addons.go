package camoufox

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DefaultAddon represents a built-in addon
type DefaultAddon string

const (
	// AddonUBlockOrigin is the uBlock Origin ad blocker
	AddonUBlockOrigin DefaultAddon = "https://addons.mozilla.org/firefox/downloads/latest/ublock-origin/latest.xpi"
)

// AddonManager handles addon downloading and installation
type AddonManager struct {
	addonsDir string
	mu        sync.Mutex
}

// NewAddonManager creates a new addon manager
func NewAddonManager() (*AddonManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	addonsDir := filepath.Join(homeDir, ".cache", "camoufox-go", "addons")
	if err := os.MkdirAll(addonsDir, 0755); err != nil {
		return nil, err
	}

	return &AddonManager{addonsDir: addonsDir}, nil
}

// GetAddonPath returns the path to an addon, downloading if necessary
func (am *AddonManager) GetAddonPath(addon DefaultAddon) (string, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	addonName := addonNameFromURL(string(addon))
	addonPath := filepath.Join(am.addonsDir, addonName)

	// Check if already downloaded
	manifestPath := filepath.Join(addonPath, "manifest.json")
	if _, err := os.Stat(manifestPath); err == nil {
		return addonPath, nil
	}

	// Download and extract
	if err := am.downloadAddon(string(addon), addonPath, addonName); err != nil {
		return "", err
	}

	return addonPath, nil
}

// downloadAddon downloads and extracts an addon
func (am *AddonManager) downloadAddon(url, destPath, name string) error {
	fmt.Printf("Downloading addon: %s...\n", name)

	// Download XPI file
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download addon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("addon download failed with status: %d", resp.StatusCode)
	}

	// Read into memory
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read addon data: %w", err)
	}

	// Create destination directory
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create addon directory: %w", err)
	}

	// Extract XPI (it's a ZIP file)
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to read addon as zip: %w", err)
	}

	for _, file := range zipReader.File {
		filePath := filepath.Join(destPath, file.Name)

		// Security check
		if !isPathSafe(destPath, filePath) {
			continue
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, file.Mode())
			continue
		}

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			continue
		}

		// Extract file
		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			continue
		}

		io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
	}

	fmt.Printf("Addon installed: %s\n", name)
	return nil
}

// GetDefaultAddons returns paths to all default addons
func (am *AddonManager) GetDefaultAddons(exclude ...DefaultAddon) ([]string, error) {
	excludeSet := make(map[DefaultAddon]bool)
	for _, e := range exclude {
		excludeSet[e] = true
	}

	defaultAddons := []DefaultAddon{
		AddonUBlockOrigin,
	}

	var paths []string
	for _, addon := range defaultAddons {
		if excludeSet[addon] {
			continue
		}

		path, err := am.GetAddonPath(addon)
		if err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to get addon: %v\n", err)
			continue
		}
		paths = append(paths, path)
	}

	return paths, nil
}

// ValidateAddonPath checks if an addon path is valid
func ValidateAddonPath(path string) error {
	manifestPath := filepath.Join(path, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("addon path missing manifest.json: %s", path)
	}
	return nil
}

// ValidateAddonPaths validates multiple addon paths
func ValidateAddonPaths(paths []string) error {
	for _, path := range paths {
		if err := ValidateAddonPath(path); err != nil {
			return err
		}
	}
	return nil
}

// addonNameFromURL extracts addon name from URL
func addonNameFromURL(url string) string {
	// Extract from URL like ".../ublock-origin/..."
	parts := []string{"ublock-origin"}
	for _, part := range parts {
		if contains(url, part) {
			return part
		}
	}
	return "addon"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// isPathSafe checks for zip slip vulnerability
func isPathSafe(baseDir, filePath string) bool {
	absBase, _ := filepath.Abs(baseDir)
	absPath, _ := filepath.Abs(filePath)
	return len(absPath) >= len(absBase) && absPath[:len(absBase)] == absBase
}
