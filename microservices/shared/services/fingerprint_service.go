package services

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
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
	scriptPath string
	cache      map[string]*cachedFingerprint
	mu         sync.RWMutex
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
		scriptPath, scriptErr := getFingerprintScriptPath()
		if scriptErr != nil {
			err = scriptErr
			return
		}
		instance = &FingerprintService{
			scriptPath: scriptPath,
			cache:      make(map[string]*cachedFingerprint),
		}
	})
	return instance, err
}

// GenerateFingerprint generates a browser fingerprint using BrowserForge
// Results are cached for 5 minutes to avoid repeated Python calls
func (s *FingerprintService) GenerateFingerprint(os string, maxWidth, maxHeight int) (*Fingerprint, error) {
	cacheKey := fmt.Sprintf("%s-%d-%d", os, maxWidth, maxHeight)

	// Check cache
	s.mu.RLock()
	if cached, ok := s.cache[cacheKey]; ok {
		if time.Since(cached.createdAt) < 5*time.Minute {
			s.mu.RUnlock()
			return cached.fingerprint, nil
		}
	}
	s.mu.RUnlock()

	// Generate new fingerprint
	args := []string{s.scriptPath}
	if os != "" {
		args = append(args, os)
	} else {
		args = append(args, "linux")
	}

	if maxWidth > 0 {
		args = append(args, strconv.Itoa(maxWidth))
		if maxHeight > 0 {
			args = append(args, strconv.Itoa(maxHeight))
		}
	}

	cmd := exec.Command("python3", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("fingerprint generation failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to execute fingerprint script: %w", err)
	}

	var fp Fingerprint
	if err := json.Unmarshal(output, &fp); err != nil {
		return nil, fmt.Errorf("failed to parse fingerprint JSON: %w", err)
	}

	// Cache the result
	s.mu.Lock()
	s.cache[cacheKey] = &cachedFingerprint{
		fingerprint: &fp,
		createdAt:   time.Now(),
	}
	s.mu.Unlock()

	return &fp, nil
}

// getFingerprintScriptPath returns the path to the fingerprint generation script
func getFingerprintScriptPath() (string, error) {
	// Get the directory of this Go file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to get current file path")
	}

	// Navigate to the shared location: microservices/shared/scripts/generate_fingerprint.py
	scriptPath := filepath.Join(filepath.Dir(filename), "..", "scripts", "generate_fingerprint.py")
	return scriptPath, nil
}
