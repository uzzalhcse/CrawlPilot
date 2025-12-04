package plugin

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// SourceManager handles reading and writing plugin source code
type SourceManager struct {
	BasePath string
}

// NewSourceManager creates a new SourceManager
func NewSourceManager(basePath string) *SourceManager {
	return &SourceManager{BasePath: basePath}
}

// GetSource returns a map of filenames to their content for a given plugin
func (m *SourceManager) GetSource(pluginSlug string) (map[string]string, error) {
	pluginDir := filepath.Join(m.BasePath, pluginSlug)

	// Check if directory exists
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin source directory not found: %s", pluginSlug)
	}

	files := make(map[string]string)

	// Walk through the directory
	err := filepath.Walk(pluginDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only include .go, .mod, .sum, .md files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".go" && ext != ".mod" && ext != ".sum" && ext != ".md" {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(pluginDir, path)
		if err != nil {
			return err
		}

		// Read file content
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		files[relPath] = string(content)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read plugin files: %w", err)
	}

	return files, nil
}

// UpdateSource updates the source code for a plugin
func (m *SourceManager) UpdateSource(pluginSlug string, files map[string]string) error {
	pluginDir := filepath.Join(m.BasePath, pluginSlug)

	// Check if directory exists
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return fmt.Errorf("plugin source directory not found: %s", pluginSlug)
	}

	for filename, content := range files {
		// Validate filename to prevent directory traversal
		if strings.Contains(filename, "..") || strings.HasPrefix(filename, "/") {
			return fmt.Errorf("invalid filename: %s", filename)
		}

		fullPath := filepath.Join(pluginDir, filename)

		// Ensure the file belongs to the plugin directory
		if !strings.HasPrefix(fullPath, pluginDir) {
			return fmt.Errorf("invalid file path: %s", fullPath)
		}

		// Create parent directories if needed
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", filename, err)
		}

		// Write file content
		if err := ioutil.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filename, err)
		}
	}

	return nil
}
