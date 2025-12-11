// Package fingerprints provides browser fingerprint generation for browserforge-go.
package fingerprints

import (
	"encoding/json"
)

// Screen constrains the screen dimensions of the generated fingerprint
type Screen struct {
	MinWidth  *int
	MaxWidth  *int
	MinHeight *int
	MaxHeight *int
}

// IsSet returns true if any constraints are set
func (s *Screen) IsSet() bool {
	return s != nil && (s.MinWidth != nil || s.MaxWidth != nil || s.MinHeight != nil || s.MaxHeight != nil)
}

// ScreenFingerprint represents the screen properties of a fingerprint
type ScreenFingerprint struct {
	AvailHeight      int     `json:"availHeight"`
	AvailWidth       int     `json:"availWidth"`
	AvailTop         int     `json:"availTop"`
	AvailLeft        int     `json:"availLeft"`
	ColorDepth       int     `json:"colorDepth"`
	Height           int     `json:"height"`
	PixelDepth       int     `json:"pixelDepth"`
	Width            int     `json:"width"`
	DevicePixelRatio float64 `json:"devicePixelRatio"`
	PageXOffset      int     `json:"pageXOffset"`
	PageYOffset      int     `json:"pageYOffset"`
	InnerHeight      int     `json:"innerHeight"`
	OuterHeight      int     `json:"outerHeight"`
	OuterWidth       int     `json:"outerWidth"`
	InnerWidth       int     `json:"innerWidth"`
	ScreenX          int     `json:"screenX"`
	ClientWidth      int     `json:"clientWidth"`
	ClientHeight     int     `json:"clientHeight"`
	HasHDR           bool    `json:"hasHDR"`
}

// NavigatorFingerprint represents the navigator properties of a fingerprint
type NavigatorFingerprint struct {
	UserAgent           string                 `json:"userAgent"`
	UserAgentData       map[string]interface{} `json:"userAgentData"`
	DoNotTrack          *string                `json:"doNotTrack"`
	AppCodeName         string                 `json:"appCodeName"`
	AppName             string                 `json:"appName"`
	AppVersion          string                 `json:"appVersion"`
	Oscpu               string                 `json:"oscpu"`
	Webdriver           bool                   `json:"webdriver"`
	Language            string                 `json:"language"`
	Languages           []string               `json:"languages"`
	Platform            string                 `json:"platform"`
	DeviceMemory        *int                   `json:"deviceMemory"`
	HardwareConcurrency int                    `json:"hardwareConcurrency"`
	Product             string                 `json:"product"`
	ProductSub          string                 `json:"productSub"`
	Vendor              string                 `json:"vendor"`
	VendorSub           string                 `json:"vendorSub"`
	MaxTouchPoints      int                    `json:"maxTouchPoints"`
	ExtraProperties     map[string]interface{} `json:"extraProperties"`
}

// VideoCard represents the video card (WebGL) properties
type VideoCard struct {
	Renderer string `json:"renderer"`
	Vendor   string `json:"vendor"`
}

// Fingerprint represents a complete browser fingerprint
type Fingerprint struct {
	Screen            ScreenFingerprint      `json:"screen"`
	Navigator         NavigatorFingerprint   `json:"navigator"`
	Headers           map[string]string      `json:"headers"`
	VideoCodecs       map[string]string      `json:"videoCodecs"`
	AudioCodecs       map[string]string      `json:"audioCodecs"`
	PluginsData       map[string]interface{} `json:"pluginsData"`
	Battery           map[string]interface{} `json:"battery"`
	VideoCard         *VideoCard             `json:"videoCard"`
	MultimediaDevices map[string]interface{} `json:"multimediaDevices"`
	Fonts             []string               `json:"fonts"`
	MockWebRTC        bool                   `json:"mockWebRTC"`
	Slim              bool                   `json:"slim"`
}

// Dumps serializes the fingerprint to JSON string
func (f *Fingerprint) Dumps() (string, error) {
	data, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FingerprintGeneratorOptions contains options for fingerprint generation
type FingerprintGeneratorOptions struct {
	Screen     *Screen
	Strict     bool
	MockWebRTC bool
	Slim       bool

	// Header generation options (inherited from HeaderGenerator)
	Browsers    []interface{}
	OS          []string
	Devices     []string
	Locales     []string
	HTTPVersion string
}

// GenerateFingerprintOptions contains options for a single fingerprint generation call
type GenerateFingerprintOptions struct {
	Screen     *Screen
	Strict     *bool
	MockWebRTC *bool
	Slim       *bool

	// Header generation overrides
	Browsers    []interface{}
	OS          []string
	Devices     []string
	Locales     []string
	HTTPVersion *string
	UserAgents  []string
}
