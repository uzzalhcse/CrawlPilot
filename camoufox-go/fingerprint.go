package camoufox

import (
	"encoding/json"
	"fmt"

	bffingerprints "github.com/uzzalhcse/browserforge-go/fingerprints"
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

// fingerprintGenerator is a cached fingerprint generator instance
var fingerprintGenerator *bffingerprints.FingerprintGenerator

// getFingerprintGenerator returns a cached fingerprint generator
func getFingerprintGenerator() (*bffingerprints.FingerprintGenerator, error) {
	if fingerprintGenerator != nil {
		return fingerprintGenerator, nil
	}

	gen, err := bffingerprints.NewFingerprintGenerator(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create fingerprint generator: %w", err)
	}
	fingerprintGenerator = gen
	return gen, nil
}

// GenerateFingerprint generates a browser fingerprint using browserforge-go.
// This is a pure Go implementation - no Python dependency required.
func GenerateFingerprint(os string, screen *Screen) (*Fingerprint, error) {
	gen, err := getFingerprintGenerator()
	if err != nil {
		return nil, err
	}

	// Build generation options
	opts := &bffingerprints.GenerateFingerprintOptions{}

	// Force Firefox browser since Camoufox is Firefox-based
	// This ensures the user-agent matches the actual browser engine
	opts.Browsers = []interface{}{"firefox"}

	// Map OS string to browserforge-go format
	if os != "" {
		switch os {
		case "windows", "win", "win32", "win64":
			opts.OS = []string{"windows"}
		case "macos", "mac", "darwin", "osx":
			opts.OS = []string{"macos"}
		case "linux":
			opts.OS = []string{"linux"}
		case "android":
			opts.OS = []string{"android"}
		case "ios":
			opts.OS = []string{"ios"}
		default:
			opts.OS = []string{os}
		}
	}

	// Apply screen constraints
	if screen != nil {
		bfScreen := &bffingerprints.Screen{}
		if screen.MinWidth > 0 {
			bfScreen.MinWidth = &screen.MinWidth
		}
		if screen.MaxWidth > 0 {
			bfScreen.MaxWidth = &screen.MaxWidth
		}
		if screen.MinHeight > 0 {
			bfScreen.MinHeight = &screen.MinHeight
		}
		if screen.MaxHeight > 0 {
			bfScreen.MaxHeight = &screen.MaxHeight
		}
		if bfScreen.MinWidth != nil || bfScreen.MaxWidth != nil ||
			bfScreen.MinHeight != nil || bfScreen.MaxHeight != nil {
			opts.Screen = bfScreen
		}
	}

	// Generate fingerprint using browserforge-go
	bfFp, err := gen.Generate(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate fingerprint: %w", err)
	}

	// Convert to camoufox Fingerprint type
	fp := &Fingerprint{
		Navigator: NavigatorFingerprint{
			UserAgent:           bfFp.Navigator.UserAgent,
			AppCodeName:         bfFp.Navigator.AppCodeName,
			AppName:             bfFp.Navigator.AppName,
			AppVersion:          bfFp.Navigator.AppVersion,
			Platform:            bfFp.Navigator.Platform,
			OsCPU:               bfFp.Navigator.Oscpu,
			Language:            bfFp.Navigator.Language,
			Languages:           bfFp.Navigator.Languages,
			HardwareConcurrency: bfFp.Navigator.HardwareConcurrency,
			MaxTouchPoints:      bfFp.Navigator.MaxTouchPoints,
			Product:             bfFp.Navigator.Product,
			DoNotTrack:          bfFp.Navigator.DoNotTrack,
		},
		Screen: ScreenFingerprint{
			Width:       bfFp.Screen.Width,
			Height:      bfFp.Screen.Height,
			AvailWidth:  bfFp.Screen.AvailWidth,
			AvailHeight: bfFp.Screen.AvailHeight,
			AvailLeft:   bfFp.Screen.AvailLeft,
			AvailTop:    bfFp.Screen.AvailTop,
			ColorDepth:  bfFp.Screen.ColorDepth,
			PixelDepth:  bfFp.Screen.PixelDepth,
			OuterWidth:  bfFp.Screen.OuterWidth,
			OuterHeight: bfFp.Screen.OuterHeight,
			InnerWidth:  bfFp.Screen.InnerWidth,
			InnerHeight: bfFp.Screen.InnerHeight,
			ScreenX:     bfFp.Screen.ScreenX,
			PageXOffset: bfFp.Screen.PageXOffset,
			PageYOffset: bfFp.Screen.PageYOffset,
		},
	}

	// Extract Accept-Encoding from headers if available
	if acceptEncoding, ok := bfFp.Headers["Accept-Encoding"]; ok {
		fp.Headers = HeadersFingerprint{
			AcceptEncoding: acceptEncoding,
		}
	}

	// Extract battery if available
	if bfFp.Battery != nil {
		if charging, ok := bfFp.Battery["charging"].(bool); ok {
			fp.Battery.Charging = charging
		}
		if level, ok := bfFp.Battery["level"].(float64); ok {
			fp.Battery.Level = level
		}
		if chargingTime, ok := bfFp.Battery["chargingTime"].(float64); ok {
			fp.Battery.ChargingTime = chargingTime
		}
	}

	return fp, nil
}

// GenerateFingerprintFromJSON creates a Fingerprint from raw JSON
// (kept for backwards compatibility)
func GenerateFingerprintFromJSON(data []byte) (*Fingerprint, error) {
	var fp Fingerprint
	if err := json.Unmarshal(data, &fp); err != nil {
		return nil, err
	}
	return &fp, nil
}
