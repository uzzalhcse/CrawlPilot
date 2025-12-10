package camoufox

import (
	"database/sql"
	_ "embed"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"

	_ "github.com/mattn/go-sqlite3"
)

// WebGLConfig holds WebGL vendor/renderer pair
type WebGLConfig struct {
	Vendor   string `json:"webGl:vendor"`
	Renderer string `json:"webGl:renderer"`
	WebGL2   bool   `json:"webGl2Enabled"`
}

//go:embed internal/webgl_data.db
var webglDBData []byte

var webglDBPath string

func init() {
	// Write embedded database to temp file for sqlite3 to read
	tmpDir := os.TempDir()
	webglDBPath = filepath.Join(tmpDir, "camoufox_webgl.db")
	os.WriteFile(webglDBPath, webglDBData, 0644)
}

// SampleWebGL returns a random realistic WebGL vendor/renderer for the target OS
func SampleWebGL(targetOS string) (*WebGLConfig, error) {
	return sampleWebGLInternal(targetOS, "", "")
}

// SampleWebGLWithConfig returns a WebGL config matching the specified vendor/renderer
func SampleWebGLWithConfig(targetOS, vendor, renderer string) (*WebGLConfig, error) {
	return sampleWebGLInternal(targetOS, vendor, renderer)
}

func sampleWebGLInternal(targetOS, vendor, renderer string) (*WebGLConfig, error) {
	// Convert OS name to database format
	osKey := targetOS
	switch targetOS {
	case "windows", "win":
		osKey = "win"
	case "macos", "darwin", "mac":
		osKey = "mac"
	case "linux", "lin":
		osKey = "lin"
	}

	db, err := sql.Open("sqlite3", webglDBPath)
	if err != nil {
		return getDefaultWebGL(osKey), nil
	}
	defer db.Close()

	var query string
	var args []interface{}

	if vendor != "" && renderer != "" {
		// Specific vendor/renderer requested
		query = `SELECT vendor, renderer, webgl2 FROM webgl 
				 WHERE os = ? AND vendor = ? AND renderer = ?
				 LIMIT 1`
		args = []interface{}{osKey, vendor, renderer}
	} else {
		// Random selection for OS
		query = `SELECT vendor, renderer, webgl2 FROM webgl 
				 WHERE os = ? 
				 ORDER BY RANDOM() 
				 LIMIT 1`
		args = []interface{}{osKey}
	}

	row := db.QueryRow(query, args...)

	var cfg WebGLConfig
	var webgl2 int
	if err := row.Scan(&cfg.Vendor, &cfg.Renderer, &webgl2); err != nil {
		return getDefaultWebGL(osKey), nil
	}

	cfg.WebGL2 = webgl2 == 1
	return &cfg, nil
}

// getDefaultWebGL returns a default WebGL config for the OS
func getDefaultWebGL(osKey string) *WebGLConfig {
	defaults := map[string]*WebGLConfig{
		"win": {
			Vendor:   "Google Inc. (NVIDIA)",
			Renderer: "ANGLE (NVIDIA, NVIDIA GeForce GTX 1060 Direct3D11 vs_5_0 ps_5_0, D3D11)",
			WebGL2:   true,
		},
		"mac": {
			Vendor:   "Apple Inc.",
			Renderer: "Apple M1",
			WebGL2:   true,
		},
		"lin": {
			Vendor:   "Intel",
			Renderer: "Mesa Intel(R) UHD Graphics 620 (KBL GT2)",
			WebGL2:   true,
		},
	}

	if cfg, ok := defaults[osKey]; ok {
		return cfg
	}
	return defaults["lin"]
}

// GetWebGLVendors returns all available vendors for the target OS
func GetWebGLVendors(targetOS string) ([]string, error) {
	osKey := targetOS
	switch targetOS {
	case "windows", "win":
		osKey = "win"
	case "macos", "darwin", "mac":
		osKey = "mac"
	case "linux", "lin":
		osKey = "lin"
	}

	db, err := sql.Open("sqlite3", webglDBPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT DISTINCT vendor FROM webgl WHERE os = ?`, osKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vendors []string
	for rows.Next() {
		var vendor string
		if err := rows.Scan(&vendor); err != nil {
			continue
		}
		vendors = append(vendors, vendor)
	}

	return vendors, nil
}

// GetRandomCanvasOffset returns random canvas anti-fingerprinting offset
func GetRandomCanvasOffset() int {
	return rand.Intn(101) - 50 // -50 to 50
}

// GetRandomFontSpacing returns random font spacing seed
func GetRandomFontSpacing() int {
	return rand.Intn(1073741824) // 0 to 2^30 - 1
}

// ensureWebGLDB ensures the database file exists
func ensureWebGLDB() error {
	if _, err := os.Stat(webglDBPath); os.IsNotExist(err) {
		return fmt.Errorf("webgl database not found at %s", webglDBPath)
	}
	return nil
}

// init ensures we have runtime context
func getOSKey() string {
	switch runtime.GOOS {
	case "windows":
		return "win"
	case "darwin":
		return "mac"
	default:
		return "lin"
	}
}
