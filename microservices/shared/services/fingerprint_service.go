package services

import (
	"fmt"
	"sync"
	"time"

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

// FingerprintService manages BrowserForge fingerprint generation
type FingerprintService struct {
	generator *bffingerprints.FingerprintGenerator
	cache     map[string]*cachedFingerprint
	mu        sync.RWMutex
}

type cachedFingerprint struct {
	fingerprint *Fingerprint
	createdAt   time.Time
}

var (
	instance *FingerprintService
	once     sync.Once
)

// GetFingerprintService returns the singleton fingerprint service
func GetFingerprintService() (*FingerprintService, error) {
	var err error
	once.Do(func() {
		generator, genErr := bffingerprints.NewFingerprintGenerator(nil)
		if genErr != nil {
			err = fmt.Errorf("failed to create fingerprint generator: %w", genErr)
			return
		}
		instance = &FingerprintService{
			generator: generator,
			cache:     make(map[string]*cachedFingerprint),
		}
	})
	return instance, err
}

// GenerateFingerprint generates a browser fingerprint using browserforge-go
// Results are cached for 5 minutes to avoid repeated generation overhead
func (s *FingerprintService) GenerateFingerprint(os string, maxWidth, maxHeight int) (*Fingerprint, error) {
	return s.GenerateFingerprintWithBrowser(os, "", maxWidth, maxHeight)
}

// GenerateFingerprintWithBrowser generates a browser fingerprint for a specific browser type
func (s *FingerprintService) GenerateFingerprintWithBrowser(os, browserType string, maxWidth, maxHeight int) (*Fingerprint, error) {
	cacheKey := fmt.Sprintf("%s-%s-%d-%d", os, browserType, maxWidth, maxHeight)

	// Check cache
	s.mu.RLock()
	if cached, ok := s.cache[cacheKey]; ok {
		if time.Since(cached.createdAt) < 5*time.Minute {
			s.mu.RUnlock()
			return cached.fingerprint, nil
		}
	}
	s.mu.RUnlock()

	// Build generation options
	opts := &bffingerprints.GenerateFingerprintOptions{}

	// Set browser type constraint to match actual browser
	if browserType != "" {
		switch browserType {
		case "chromium", "chrome":
			opts.Browsers = []interface{}{"chrome"}
		case "firefox":
			opts.Browsers = []interface{}{"firefox"}
		case "webkit", "safari":
			opts.Browsers = []interface{}{"safari"}
		case "edge":
			opts.Browsers = []interface{}{"edge"}
		}
	}

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
	if maxWidth > 0 || maxHeight > 0 {
		screen := &bffingerprints.Screen{}
		if maxWidth > 0 {
			screen.MaxWidth = &maxWidth
		}
		if maxHeight > 0 {
			screen.MaxHeight = &maxHeight
		}
		opts.Screen = screen
	}

	// Generate fingerprint
	bfFp, err := s.generator.Generate(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate fingerprint: %w", err)
	}

	// Convert to service fingerprint type
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

	// Cache the result
	s.mu.Lock()
	s.cache[cacheKey] = &cachedFingerprint{
		fingerprint: fp,
		createdAt:   time.Now(),
	}
	s.mu.Unlock()

	return fp, nil
}
