package camoufox

import (
	"encoding/json"
	"math/rand"
	"os"
)

// BuildConfig creates the Camoufox configuration map from a fingerprint.
// This config is passed to Camoufox via environment variables.
func BuildConfig(fp *Fingerprint, opts *Options) map[string]interface{} {
	config := make(map[string]interface{})

	if fp == nil {
		return config
	}

	// Navigator properties
	if fp.Navigator.UserAgent != "" {
		config["navigator.userAgent"] = fp.Navigator.UserAgent
	}
	if fp.Navigator.Platform != "" {
		config["navigator.platform"] = fp.Navigator.Platform
	}
	if fp.Navigator.Language != "" {
		config["navigator.language"] = fp.Navigator.Language
	}
	if len(fp.Navigator.Languages) > 0 {
		config["navigator.languages"] = fp.Navigator.Languages
	}
	if fp.Navigator.OsCPU != "" {
		config["navigator.oscpu"] = fp.Navigator.OsCPU
	}
	if fp.Navigator.AppCodeName != "" {
		config["navigator.appCodeName"] = fp.Navigator.AppCodeName
	}
	if fp.Navigator.AppName != "" {
		config["navigator.appName"] = fp.Navigator.AppName
	}
	if fp.Navigator.AppVersion != "" {
		config["navigator.appVersion"] = fp.Navigator.AppVersion
	}
	if fp.Navigator.HardwareConcurrency > 0 {
		config["navigator.hardwareConcurrency"] = fp.Navigator.HardwareConcurrency
	}
	if fp.Navigator.MaxTouchPoints >= 0 {
		config["navigator.maxTouchPoints"] = fp.Navigator.MaxTouchPoints
	}
	if fp.Navigator.Product != "" {
		config["navigator.product"] = fp.Navigator.Product
	}
	if fp.Navigator.ProductSub != "" {
		config["navigator.productSub"] = fp.Navigator.ProductSub
	}

	// Screen properties
	if fp.Screen.Width > 0 {
		config["screen.width"] = fp.Screen.Width
		config["screen.availWidth"] = fp.Screen.Width
	}
	if fp.Screen.Height > 0 {
		config["screen.height"] = fp.Screen.Height
		config["screen.availHeight"] = fp.Screen.Height
	}
	if fp.Screen.ColorDepth > 0 {
		config["screen.colorDepth"] = fp.Screen.ColorDepth
		config["screen.pixelDepth"] = fp.Screen.ColorDepth
	}

	// Window properties
	if fp.Screen.OuterWidth > 0 {
		config["window.outerWidth"] = fp.Screen.OuterWidth
	}
	if fp.Screen.OuterHeight > 0 {
		config["window.outerHeight"] = fp.Screen.OuterHeight
	}
	if fp.Screen.InnerWidth > 0 {
		config["window.innerWidth"] = fp.Screen.InnerWidth
	}
	if fp.Screen.InnerHeight > 0 {
		config["window.innerHeight"] = fp.Screen.InnerHeight
	}

	// Random history length (1-5)
	config["window.history.length"] = rand.Intn(5) + 1

	// Apply options
	if opts != nil {
		// Block images
		if opts.BlockImages {
			config["camoufox.blockImages"] = true
		}

		// Block WebRTC
		if opts.BlockWebRTC {
			config["camoufox.blockWebRTC"] = true
		}

		// Block WebGL
		if opts.BlockWebGL {
			config["camoufox.blockWebGL"] = true
		}

		// Humanize mouse movement
		if opts.Humanize > 0 {
			config["camoufox.humanize"] = opts.Humanize
		}

		// Timezone
		if opts.Timezone != "" {
			config["timezone"] = opts.Timezone
		}

		// Locale
		if len(opts.Locale) > 0 {
			config["locale:language"] = opts.Locale[0]
			if len(opts.Locale) > 1 {
				config["locale:region"] = opts.Locale[1]
			}
		}

		// Custom fonts
		if len(opts.Fonts) > 0 {
			config["fonts"] = opts.Fonts
		}
	}

	return config
}

// GetEnvVars converts the config map to environment variables for Camoufox.
// It inherits the system environment (including DISPLAY for headed mode) and
// adds Camoufox-specific config variables (CAMOU_CONFIG_1, etc.)
func GetEnvVars(config map[string]interface{}, targetOS string) map[string]string {
	env := make(map[string]string)

	// Inherit important system environment variables
	// This is essential for headed mode to work (DISPLAY on Linux/X11)
	importantVars := []string{
		"DISPLAY",                  // X11 display (Linux)
		"WAYLAND_DISPLAY",          // Wayland display
		"XDG_RUNTIME_DIR",          // Wayland runtime
		"HOME",                     // Home directory
		"PATH",                     // System path
		"USER",                     // Username
		"LANG",                     // Locale
		"LC_ALL",                   // Locale
		"TZ",                       // Timezone
		"TMPDIR",                   // Temp directory
		"XDG_SESSION_TYPE",         // Session type (x11, wayland, tty)
		"DBUS_SESSION_BUS_ADDRESS", // D-Bus for clipboard, etc.
	}

	for _, key := range importantVars {
		if val := os.Getenv(key); val != "" {
			env[key] = val
		}
	}

	// Add Camoufox config as JSON chunks
	configJSON, err := json.Marshal(config)
	if err != nil {
		return env
	}

	configStr := string(configJSON)

	// Split into chunks (32KB max per env var on Linux, 2KB on Windows)
	chunkSize := 32767
	if targetOS == "windows" {
		chunkSize = 2047
	}

	for i := 0; i < len(configStr); i += chunkSize {
		end := i + chunkSize
		if end > len(configStr) {
			end = len(configStr)
		}
		chunkNum := (i / chunkSize) + 1
		envName := "CAMOU_CONFIG_" + string(rune('0'+chunkNum))
		if chunkNum > 9 {
			envName = "CAMOU_CONFIG_" + string(rune('0'+chunkNum/10)) + string(rune('0'+chunkNum%10))
		}
		env[envName] = configStr[i:end]
	}

	return env
}
