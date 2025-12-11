// Package dataloader provides embedded data files for browserforge-go.
// It handles loading and extracting the Bayesian network definition files
// and browser configuration data from embedded zip/json files.
package dataloader

import (
	"archive/zip"
	"bytes"
	"embed"
	"encoding/json"
	"io"
	"path/filepath"
	"sync"
)

//go:embed data/*
var dataFS embed.FS

// Cached network definitions
var (
	inputNetworkOnce sync.Once
	inputNetworkDef  map[string]interface{}
	inputNetworkErr  error

	headerNetworkOnce sync.Once
	headerNetworkDef  map[string]interface{}
	headerNetworkErr  error

	fingerprintNetworkOnce sync.Once
	fingerprintNetworkDef  map[string]interface{}
	fingerprintNetworkErr  error

	headersOrderOnce sync.Once
	headersOrderData map[string][]string
	headersOrderErr  error

	browserHelperOnce sync.Once
	browserHelperData []string
	browserHelperErr  error
)

// ExtractJSON extracts and parses JSON from a file path.
// If the file is a zip, it extracts the first JSON file from the archive.
func ExtractJSON(path string) (map[string]interface{}, error) {
	data, err := dataFS.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Check if it's a zip file
	if filepath.Ext(path) == ".zip" {
		return extractJSONFromZip(data)
	}

	// Direct JSON file
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// extractJSONFromZip extracts the first JSON file from a zip archive
func extractJSONFromZip(data []byte) (map[string]interface{}, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	for _, file := range reader.File {
		if filepath.Ext(file.Name) == ".json" {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(content, &result); err != nil {
				return nil, err
			}
			return result, nil
		}
	}

	return nil, nil
}

// GetInputNetwork returns the input network definition for the Bayesian network.
func GetInputNetwork() (map[string]interface{}, error) {
	inputNetworkOnce.Do(func() {
		inputNetworkDef, inputNetworkErr = ExtractJSON("data/input-network-definition.zip")
	})
	return inputNetworkDef, inputNetworkErr
}

// GetHeaderNetwork returns the header network definition for the Bayesian network.
func GetHeaderNetwork() (map[string]interface{}, error) {
	headerNetworkOnce.Do(func() {
		headerNetworkDef, headerNetworkErr = ExtractJSON("data/header-network-definition.zip")
	})
	return headerNetworkDef, headerNetworkErr
}

// GetFingerprintNetwork returns the fingerprint network definition for the Bayesian network.
func GetFingerprintNetwork() (map[string]interface{}, error) {
	fingerprintNetworkOnce.Do(func() {
		fingerprintNetworkDef, fingerprintNetworkErr = ExtractJSON("data/fingerprint-network-definition.zip")
	})
	return fingerprintNetworkDef, fingerprintNetworkErr
}

// GetHeadersOrder returns the browser-specific header ordering.
func GetHeadersOrder() (map[string][]string, error) {
	headersOrderOnce.Do(func() {
		data, err := dataFS.ReadFile("data/headers-order.json")
		if err != nil {
			headersOrderErr = err
			return
		}
		headersOrderErr = json.Unmarshal(data, &headersOrderData)
	})
	return headersOrderData, headersOrderErr
}

// GetBrowserHelperFile returns the list of browser helper strings.
func GetBrowserHelperFile() ([]string, error) {
	browserHelperOnce.Do(func() {
		data, err := dataFS.ReadFile("data/browser-helper-file.json")
		if err != nil {
			browserHelperErr = err
			return
		}
		browserHelperErr = json.Unmarshal(data, &browserHelperData)
	})
	return browserHelperData, browserHelperErr
}
