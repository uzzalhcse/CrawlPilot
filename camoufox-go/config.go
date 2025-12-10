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

	// Get actual Camoufox version to fix UserAgent mismatch
	versionInfo, _ := GetVersionInfo()
	actualVersion := "142.0"
	if versionInfo != nil && versionInfo.Version != "" {
		actualVersion = versionInfo.Version
	}

	// Navigator properties (matching browserforge.yml)
	// Fix UserAgent version to match actual Camoufox Firefox version
	if fp.Navigator.UserAgent != "" {
		fixedUA := UpdateUserAgentVersion(fp.Navigator.UserAgent, actualVersion)
		config["navigator.userAgent"] = fixedUA
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
		// Also fix version in oscpu (may contain "rv:1XX.0")
		fixedOsCPU := UpdateUserAgentVersion(fp.Navigator.OsCPU, actualVersion)
		config["navigator.oscpu"] = fixedOsCPU
	}
	if fp.Navigator.AppCodeName != "" {
		config["navigator.appCodeName"] = fp.Navigator.AppCodeName
	}
	if fp.Navigator.AppName != "" {
		config["navigator.appName"] = fp.Navigator.AppName
	}
	if fp.Navigator.AppVersion != "" {
		// Also fix version in appVersion
		fixedAppVersion := UpdateUserAgentVersion(fp.Navigator.AppVersion, actualVersion)
		config["navigator.appVersion"] = fixedAppVersion
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
	// doNotTrack (can be "1", "0", or nil)
	if fp.Navigator.DoNotTrack != nil {
		config["navigator.doNotTrack"] = *fp.Navigator.DoNotTrack
	}
	// globalPrivacyControl from navigator.extraProperties
	if fp.Navigator.GlobalPrivacyControl != nil {
		config["navigator.globalPrivacyControl"] = *fp.Navigator.GlobalPrivacyControl
	}

	// Screen properties (matching browserforge.yml)
	if fp.Screen.Width > 0 {
		config["screen.width"] = fp.Screen.Width
	}
	if fp.Screen.Height > 0 {
		config["screen.height"] = fp.Screen.Height
	}
	if fp.Screen.AvailWidth > 0 {
		config["screen.availWidth"] = fp.Screen.AvailWidth
	}
	if fp.Screen.AvailHeight > 0 {
		config["screen.availHeight"] = fp.Screen.AvailHeight
	}
	// availLeft and availTop (can be 0)
	config["screen.availLeft"] = fp.Screen.AvailLeft
	config["screen.availTop"] = fp.Screen.AvailTop
	if fp.Screen.ColorDepth > 0 {
		config["screen.colorDepth"] = fp.Screen.ColorDepth
	}
	if fp.Screen.PixelDepth > 0 {
		config["screen.pixelDepth"] = fp.Screen.PixelDepth
	}
	// pageXOffset and pageYOffset
	config["screen.pageXOffset"] = fp.Screen.PageXOffset
	config["screen.pageYOffset"] = fp.Screen.PageYOffset

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
	// screenX and screenY (can be 0)
	config["window.screenX"] = fp.Screen.ScreenX
	config["window.screenY"] = fp.Screen.ScreenY

	// Headers (matching browserforge.yml)
	if fp.Headers.AcceptEncoding != "" {
		config["headers.Accept-Encoding"] = fp.Headers.AcceptEncoding
	}

	// Battery properties
	if fp.Battery.Level > 0 {
		config["battery:charging"] = fp.Battery.Charging
		config["battery:chargingTime"] = fp.Battery.ChargingTime
		if fp.Battery.DischargingTime != nil {
			config["battery:dischargingTime"] = *fp.Battery.DischargingTime
		}
		config["battery:level"] = fp.Battery.Level
	}

	// Random history length (1-5)
	config["window.history.length"] = rand.Intn(5) + 1

	// Canvas anti-fingerprinting
	config["canvas:aaOffset"] = GetRandomCanvasOffset()
	config["canvas:aaCapOffset"] = true

	// Font spacing seed
	config["fonts:spacing_seed"] = GetRandomFontSpacing()

	// Apply options
	if opts != nil {
		// Get target OS for fonts/WebGL
		targetOS := opts.OS
		if targetOS == "" {
			targetOS = "linux"
		}

		// Apply OS-specific fonts
		if !opts.CustomFontsOnly {
			osFonts := GetFontsForOS(targetOS)
			if len(opts.Fonts) > 0 {
				// Merge custom fonts with OS fonts
				config["fonts"] = MergeFonts(opts.Fonts, osFonts)
			} else {
				config["fonts"] = osFonts
			}
		} else if len(opts.Fonts) > 0 {
			config["fonts"] = opts.Fonts
		}

		// Apply WebGL config (if not blocking WebGL)
		if !opts.BlockWebGL {
			var webglCfg *WebGLConfig
			if opts.WebGLConfig != nil {
				webglCfg = opts.WebGLConfig
			} else {
				// Sample random WebGL config for OS
				webglCfg, _ = SampleWebGL(targetOS)
			}
			if webglCfg != nil {
				config["webGl:vendor"] = webglCfg.Vendor
				config["webGl:renderer"] = webglCfg.Renderer
			}
		}

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
			config["humanize"] = true
			config["humanize:maxTime"] = opts.Humanize
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

		// Main world eval
		if opts.MainWorldEval {
			config["allowMainWorld"] = true
		}

		// Force scope access for closed shadow DOM (required for CAPTCHA solving)
		if opts.ForceScopeAccess {
			config["forceScopeAccess"] = true
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
