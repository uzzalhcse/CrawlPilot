package fingerprints

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/uzzalhcse/browserforge-go/bayesian"
	"github.com/uzzalhcse/browserforge-go/dataloader"
	"github.com/uzzalhcse/browserforge-go/headers"
)

// MissingValueToken represents a missing value in the dataset
const MissingValueToken = "*MISSING_VALUE*"

// StringifiedPrefix is the prefix for stringified JSON values
const StringifiedPrefix = "*STRINGIFIED*"

// FingerprintGenerator generates realistic browser fingerprints
type FingerprintGenerator struct {
	headerGenerator             *headers.HeaderGenerator
	fingerprintGeneratorNetwork *bayesian.BayesianNetwork

	// Default options
	Screen     *Screen
	Strict     bool
	MockWebRTC bool
	Slim       bool
}

// NewFingerprintGenerator creates a new FingerprintGenerator with the given options
func NewFingerprintGenerator(opts *FingerprintGeneratorOptions) (*FingerprintGenerator, error) {
	if opts == nil {
		opts = &FingerprintGeneratorOptions{}
	}

	// Create header generator with header-specific options
	headerOpts := &headers.HeaderGeneratorOptions{
		Browsers:    headers.DefaultHeaderGeneratorOptions().Browsers,
		OS:          headers.SupportedOperatingSystems,
		Devices:     headers.SupportedDevices,
		Locales:     []string{"en-US"},
		HTTPVersion: "2",
		Strict:      opts.Strict,
	}

	if len(opts.OS) > 0 {
		headerOpts.OS = opts.OS
	}
	if len(opts.Devices) > 0 {
		headerOpts.Devices = opts.Devices
	}
	if len(opts.Locales) > 0 {
		headerOpts.Locales = opts.Locales
	}
	if opts.HTTPVersion != "" {
		headerOpts.HTTPVersion = opts.HTTPVersion
	}

	headerGenerator, err := headers.NewHeaderGenerator(headerOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create header generator: %w", err)
	}

	// Load fingerprint network
	fingerprintNetworkDef, err := dataloader.GetFingerprintNetwork()
	if err != nil {
		return nil, fmt.Errorf("failed to load fingerprint network: %w", err)
	}
	fingerprintNetwork, err := bayesian.NewBayesianNetwork(fingerprintNetworkDef)
	if err != nil {
		return nil, fmt.Errorf("failed to create fingerprint network: %w", err)
	}

	return &FingerprintGenerator{
		headerGenerator:             headerGenerator,
		fingerprintGeneratorNetwork: fingerprintNetwork,
		Screen:                      opts.Screen,
		Strict:                      opts.Strict,
		MockWebRTC:                  opts.MockWebRTC,
		Slim:                        opts.Slim,
	}, nil
}

// Generate generates a fingerprint and matching headers
func (fg *FingerprintGenerator) Generate(opts *GenerateFingerprintOptions) (*Fingerprint, error) {
	if opts == nil {
		opts = &GenerateFingerprintOptions{}
	}

	// Merge options
	screen := fg.Screen
	if opts.Screen != nil {
		screen = opts.Screen
	}

	strict := fg.Strict
	if opts.Strict != nil {
		strict = *opts.Strict
	}

	mockWebRTC := fg.MockWebRTC
	if opts.MockWebRTC != nil {
		mockWebRTC = *opts.MockWebRTC
	}

	slim := fg.Slim
	if opts.Slim != nil {
		slim = *opts.Slim
	}

	filteredValues := make(map[string][]string)

	// Generate partial CSP if screen constraints are set
	var partialCSP map[string][]string
	if screen != nil && screen.IsSet() {
		partialCSP = fg.partialCsp(strict, screen, filteredValues)
	}

	// Build header generation options
	headerOpts := &headers.GenerateOptions{
		Browsers:    opts.Browsers,
		OS:          opts.OS,
		Devices:     opts.Devices,
		Locales:     opts.Locales,
		HTTPVersion: opts.HTTPVersion,
		UserAgents:  opts.UserAgents,
	}

	// If we have partial CSP with userAgent, use it
	if partialCSP != nil {
		if userAgents, ok := partialCSP["userAgent"]; ok && len(userAgents) > 0 {
			headerOpts.UserAgents = userAgents
		}
	}

	// Generate headers
	generatedHeaders, err := fg.headerGenerator.Generate(headerOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate headers: %w", err)
	}

	// Extract user agent from headers
	userAgent := headers.GetUserAgent(generatedHeaders)
	if userAgent == "" {
		return nil, fmt.Errorf("failed to find User-Agent in generated headers")
	}

	// Generate fingerprint consistent with user agent
	var fingerprint map[string]interface{}
	for {
		fingerprintConstraints := make(map[string][]string)
		for k, v := range filteredValues {
			fingerprintConstraints[k] = v
		}
		fingerprintConstraints["userAgent"] = []string{userAgent}

		fingerprint = fg.fingerprintGeneratorNetwork.GenerateConsistentSampleWhenPossible(fingerprintConstraints)
		if fingerprint != nil {
			break
		}

		if strict {
			return nil, fmt.Errorf("cannot generate fingerprint: User-Agent may be invalid or screen constraints too restrictive")
		}

		// Relax filtered values and try again
		filteredValues = make(map[string][]string)

		// Try once more without constraints
		fingerprint = fg.fingerprintGeneratorNetwork.GenerateSample(map[string]interface{}{
			"userAgent": userAgent,
		})
		if fingerprint != nil {
			break
		}

		return nil, fmt.Errorf("failed to generate fingerprint")
	}

	// Process fingerprint attributes
	for key, value := range fingerprint {
		// Handle missing values
		if strVal, ok := value.(string); ok {
			if strVal == MissingValueToken {
				fingerprint[key] = nil
			} else if strings.HasPrefix(strVal, StringifiedPrefix) {
				// Unpack stringified JSON
				jsonStr := strVal[len(StringifiedPrefix):]
				var unpacked interface{}
				if err := json.Unmarshal([]byte(jsonStr), &unpacked); err == nil {
					fingerprint[key] = unpacked
				}
			}
		}
	}

	// Add languages from Accept-Language header
	acceptLanguage := generatedHeaders["Accept-Language"]
	if acceptLanguage == "" {
		acceptLanguage = generatedHeaders["accept-language"]
	}
	languages := parseAcceptLanguage(acceptLanguage)
	fingerprint["languages"] = languages

	return fg.transformFingerprint(fingerprint, generatedHeaders, mockWebRTC, slim)
}

// partialCsp generates partial content security policy based on screen constraints
func (fg *FingerprintGenerator) partialCsp(strict bool, screen *Screen, filteredValues map[string][]string) map[string][]string {
	if screen == nil || !screen.IsSet() {
		return nil
	}

	// Get the screen node from the network
	screenNode, exists := fg.fingerprintGeneratorNetwork.NodesByName["screen"]
	if !exists {
		return nil
	}

	// Filter possible screen values
	validScreens := make([]string, 0)
	for _, screenString := range screenNode.PossibleValues() {
		if fg.isScreenWithinConstraints(screenString, screen) {
			validScreens = append(validScreens, screenString)
		}
	}

	if len(validScreens) == 0 {
		return nil
	}

	filteredValues["screen"] = validScreens

	result, err := bayesian.GetPossibleValues(fg.fingerprintGeneratorNetwork, filteredValues)
	if err != nil {
		if strict {
			return nil
		}
		delete(filteredValues, "screen")
		return nil
	}

	return result
}

// isScreenWithinConstraints checks if a screen value is within constraints
func (fg *FingerprintGenerator) isScreenWithinConstraints(screenString string, screen *Screen) bool {
	if !strings.HasPrefix(screenString, StringifiedPrefix) {
		return false
	}

	jsonStr := screenString[len(StringifiedPrefix):]
	var screenData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &screenData); err != nil {
		return false
	}

	width := getIntFromMap(screenData, "width", -1)
	height := getIntFromMap(screenData, "height", -1)

	if screen.MinWidth != nil && width < *screen.MinWidth {
		return false
	}
	if screen.MaxWidth != nil && width > *screen.MaxWidth {
		return false
	}
	if screen.MinHeight != nil && height < *screen.MinHeight {
		return false
	}
	if screen.MaxHeight != nil && height > *screen.MaxHeight {
		return false
	}

	return true
}

// transformFingerprint converts the raw fingerprint map to a Fingerprint struct
func (fg *FingerprintGenerator) transformFingerprint(
	fingerprint map[string]interface{},
	generatedHeaders map[string]string,
	mockWebRTC bool,
	slim bool,
) (*Fingerprint, error) {
	// Extract screen fingerprint
	screenData, _ := fingerprint["screen"].(map[string]interface{})
	screenFP := ScreenFingerprint{
		AvailHeight:      getIntFromMap(screenData, "availHeight", 0),
		AvailWidth:       getIntFromMap(screenData, "availWidth", 0),
		AvailTop:         getIntFromMap(screenData, "availTop", 0),
		AvailLeft:        getIntFromMap(screenData, "availLeft", 0),
		ColorDepth:       getIntFromMap(screenData, "colorDepth", 24),
		Height:           getIntFromMap(screenData, "height", 0),
		PixelDepth:       getIntFromMap(screenData, "pixelDepth", 24),
		Width:            getIntFromMap(screenData, "width", 0),
		DevicePixelRatio: getFloatFromMap(screenData, "devicePixelRatio", 1),
		PageXOffset:      getIntFromMap(screenData, "pageXOffset", 0),
		PageYOffset:      getIntFromMap(screenData, "pageYOffset", 0),
		InnerHeight:      getIntFromMap(screenData, "innerHeight", 0),
		OuterHeight:      getIntFromMap(screenData, "outerHeight", 0),
		OuterWidth:       getIntFromMap(screenData, "outerWidth", 0),
		InnerWidth:       getIntFromMap(screenData, "innerWidth", 0),
		ScreenX:          getIntFromMap(screenData, "screenX", 0),
		ClientWidth:      getIntFromMap(screenData, "clientWidth", 0),
		ClientHeight:     getIntFromMap(screenData, "clientHeight", 0),
		HasHDR:           getBoolFromMap(screenData, "hasHDR", false),
	}

	// Extract languages
	languages := getStringSliceFromMap(fingerprint, "languages")
	language := ""
	if len(languages) > 0 {
		language = languages[0]
	}

	// Extract navigator fingerprint
	navFP := NavigatorFingerprint{
		UserAgent:           getStringFromMap(fingerprint, "userAgent", ""),
		UserAgentData:       getMapFromMap(fingerprint, "userAgentData"),
		DoNotTrack:          getStringPtrFromMap(fingerprint, "doNotTrack"),
		AppCodeName:         getStringFromMap(fingerprint, "appCodeName", "Mozilla"),
		AppName:             getStringFromMap(fingerprint, "appName", "Netscape"),
		AppVersion:          getStringFromMap(fingerprint, "appVersion", ""),
		Oscpu:               getStringFromMap(fingerprint, "oscpu", ""),
		Webdriver:           getBoolFromMap(fingerprint, "webdriver", false),
		Language:            language,
		Languages:           languages,
		Platform:            getStringFromMap(fingerprint, "platform", ""),
		DeviceMemory:        getIntPtrFromMap(fingerprint, "deviceMemory"),
		HardwareConcurrency: getIntFromMap(fingerprint, "hardwareConcurrency", 4),
		Product:             getStringFromMap(fingerprint, "product", "Gecko"),
		ProductSub:          getStringFromMap(fingerprint, "productSub", ""),
		Vendor:              getStringFromMap(fingerprint, "vendor", ""),
		VendorSub:           getStringFromMap(fingerprint, "vendorSub", ""),
		MaxTouchPoints:      getIntFromMap(fingerprint, "maxTouchPoints", 0),
		ExtraProperties:     getMapFromMap(fingerprint, "extraProperties"),
	}

	// Extract video card
	var videoCard *VideoCard
	if vcData, ok := fingerprint["videoCard"].(map[string]interface{}); ok && vcData != nil {
		videoCard = &VideoCard{
			Renderer: getStringFromMap(vcData, "renderer", ""),
			Vendor:   getStringFromMap(vcData, "vendor", ""),
		}
	}

	return &Fingerprint{
		Screen:            screenFP,
		Navigator:         navFP,
		Headers:           generatedHeaders,
		VideoCodecs:       getStringMapFromMap(fingerprint, "videoCodecs"),
		AudioCodecs:       getStringMapFromMap(fingerprint, "audioCodecs"),
		PluginsData:       getMapFromMap(fingerprint, "pluginsData"),
		Battery:           getMapFromMap(fingerprint, "battery"),
		VideoCard:         videoCard,
		MultimediaDevices: getMapFromMap(fingerprint, "multimediaDevices"),
		Fonts:             getStringSliceFromMap(fingerprint, "fonts"),
		MockWebRTC:        mockWebRTC,
		Slim:              slim,
	}, nil
}

// parseAcceptLanguage parses Accept-Language header to extract locales
func parseAcceptLanguage(header string) []string {
	if header == "" {
		return []string{"en-US"}
	}

	parts := strings.Split(header, ",")
	locales := make([]string, 0, len(parts))
	for _, part := range parts {
		locale := strings.TrimSpace(strings.Split(part, ";")[0])
		if locale != "" {
			locales = append(locales, locale)
		}
	}

	if len(locales) == 0 {
		return []string{"en-US"}
	}
	return locales
}

// Helper functions for extracting values from maps

func getIntFromMap(m map[string]interface{}, key string, defaultVal int) int {
	if m == nil {
		return defaultVal
	}
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		}
	}
	return defaultVal
}

func getFloatFromMap(m map[string]interface{}, key string, defaultVal float64) float64 {
	if m == nil {
		return defaultVal
	}
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		}
	}
	return defaultVal
}

func getBoolFromMap(m map[string]interface{}, key string, defaultVal bool) bool {
	if m == nil {
		return defaultVal
	}
	if v, ok := m[key].(bool); ok {
		return v
	}
	return defaultVal
}

func getStringFromMap(m map[string]interface{}, key string, defaultVal string) string {
	if m == nil {
		return defaultVal
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultVal
}

func getStringPtrFromMap(m map[string]interface{}, key string) *string {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(string); ok {
		return &v
	}
	return nil
}

func getIntPtrFromMap(m map[string]interface{}, key string) *int {
	if m == nil {
		return nil
	}
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			i := int(val)
			return &i
		case int:
			return &val
		}
	}
	return nil
}

func getStringSliceFromMap(m map[string]interface{}, key string) []string {
	if m == nil {
		return nil
	}
	if v, ok := m[key].([]interface{}); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	if v, ok := m[key].([]string); ok {
		return v
	}
	return nil
}

func getMapFromMap(m map[string]interface{}, key string) map[string]interface{} {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

func getStringMapFromMap(m map[string]interface{}, key string) map[string]string {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(map[string]interface{}); ok {
		result := make(map[string]string, len(v))
		for k, val := range v {
			if s, ok := val.(string); ok {
				result[k] = s
			}
		}
		return result
	}
	return nil
}
