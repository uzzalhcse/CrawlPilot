package headers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/uzzalhcse/browserforge-go/bayesian"
	"github.com/uzzalhcse/browserforge-go/dataloader"
)

// HeaderGenerator generates HTTP headers based on a set of constraints
type HeaderGenerator struct {
	Options        HeaderGeneratorOptions
	UniqueBrowsers []*HttpBrowserObject
	HeadersOrder   map[string][]string

	inputGeneratorNetwork  *bayesian.BayesianNetwork
	headerGeneratorNetwork *bayesian.BayesianNetwork
}

// RelaxationOrder defines the order in which to relax constraints
var RelaxationOrder = []string{"locales", "devices", "operatingSystems", "browsers"}

// NewHeaderGenerator creates a new HeaderGenerator with the given options
func NewHeaderGenerator(opts *HeaderGeneratorOptions) (*HeaderGenerator, error) {
	if opts == nil {
		defaultOpts := DefaultHeaderGeneratorOptions()
		opts = &defaultOpts
	}

	// Load networks
	inputNetworkDef, err := dataloader.GetInputNetwork()
	if err != nil {
		return nil, fmt.Errorf("failed to load input network: %w", err)
	}
	inputNetwork, err := bayesian.NewBayesianNetwork(inputNetworkDef)
	if err != nil {
		return nil, fmt.Errorf("failed to create input network: %w", err)
	}

	headerNetworkDef, err := dataloader.GetHeaderNetwork()
	if err != nil {
		return nil, fmt.Errorf("failed to load header network: %w", err)
	}
	headerNetwork, err := bayesian.NewBayesianNetwork(headerNetworkDef)
	if err != nil {
		return nil, fmt.Errorf("failed to create header network: %w", err)
	}

	// Load header order
	headersOrder, err := dataloader.GetHeadersOrder()
	if err != nil {
		return nil, fmt.Errorf("failed to load headers order: %w", err)
	}

	// Load unique browsers
	browserStrings, err := dataloader.GetBrowserHelperFile()
	if err != nil {
		return nil, fmt.Errorf("failed to load browser helper file: %w", err)
	}

	uniqueBrowsers := make([]*HttpBrowserObject, 0)
	for _, browserStr := range browserStrings {
		if browserStr != MissingValueDatasetToken {
			browser := prepareHttpBrowserObject(browserStr)
			if browser != nil {
				uniqueBrowsers = append(uniqueBrowsers, browser)
			}
		}
	}

	return &HeaderGenerator{
		Options:                *opts,
		UniqueBrowsers:         uniqueBrowsers,
		HeadersOrder:           headersOrder,
		inputGeneratorNetwork:  inputNetwork,
		headerGeneratorNetwork: headerNetwork,
	}, nil
}

// Generate generates headers using the default options and possible overrides
func (hg *HeaderGenerator) Generate(opts *GenerateOptions) (map[string]string, error) {
	if opts == nil {
		opts = &GenerateOptions{}
	}

	// Merge options
	httpVersion := hg.Options.HTTPVersion
	if opts.HTTPVersion != nil {
		httpVersion = *opts.HTTPVersion
	}

	// Prepare browsers config if specified in opts
	browsers := hg.Options.Browsers
	if len(opts.Browsers) > 0 {
		browsers = hg.prepareBrowsersConfig(opts.Browsers, httpVersion)
	}

	os := hg.Options.OS
	if len(opts.OS) > 0 {
		os = opts.OS
	}

	devices := hg.Options.Devices
	if len(opts.Devices) > 0 {
		devices = opts.Devices
	}

	locales := hg.Options.Locales
	if len(opts.Locales) > 0 {
		locales = opts.Locales
	}

	strict := hg.Options.Strict
	if opts.Strict != nil {
		strict = *opts.Strict
	}

	headers, err := hg.getHeaders(
		browsers,
		os,
		devices,
		locales,
		httpVersion,
		strict,
		opts.UserAgents,
		opts.RequestDependentHeaders,
	)
	if err != nil {
		return nil, err
	}

	if httpVersion == "2" {
		return PascalizeHeaders(headers), nil
	}
	return headers, nil
}

// getHeaders generates HTTP headers based on the given constraints
func (hg *HeaderGenerator) getHeaders(
	browsers []Browser,
	os []string,
	devices []string,
	locales []string,
	httpVersion string,
	strict bool,
	userAgents []string,
	requestDependentHeaders map[string]string,
) (map[string]string, error) {
	if requestDependentHeaders == nil {
		requestDependentHeaders = make(map[string]string)
	}

	// Get browser HTTP options
	browserHTTPOptions := hg.getBrowserHTTPOptions(browsers)

	// Build possible attribute values
	possibleAttributeValues := map[string][]string{
		"*BROWSER_HTTP":     browserHTTPOptions,
		"*OPERATING_SYSTEM": os,
	}
	if len(devices) > 0 {
		possibleAttributeValues["*DEVICE"] = devices
	}

	// Get possible values from user agents if specified
	var http1Values, http2Values map[string][]string
	if len(userAgents) > 0 {
		http1Values, _ = bayesian.GetPossibleValues(hg.headerGeneratorNetwork, map[string][]string{"User-Agent": userAgents})
		http2Values, _ = bayesian.GetPossibleValues(hg.headerGeneratorNetwork, map[string][]string{"user-agent": userAgents})
	}

	// Prepare constraints
	constraints := hg.prepareConstraints(possibleAttributeValues, http1Values, http2Values)

	// Generate input sample
	inputSample := hg.inputGeneratorNetwork.GenerateConsistentSampleWhenPossible(constraints)

	if inputSample == nil {
		// Try HTTP/2 if HTTP/1 failed
		if httpVersion == "1" {
			return hg.getHeaders(browsers, os, devices, locales, "2", strict, userAgents, requestDependentHeaders)
		}

		if strict {
			return nil, fmt.Errorf("no headers based on this input can be generated")
		}

		// Relax constraints - just return a basic sample
		inputSample = hg.inputGeneratorNetwork.GenerateSample(nil)
		if inputSample == nil {
			return nil, fmt.Errorf("failed to generate input sample")
		}
	}

	// Generate headers from sample
	generatedSample := hg.headerGeneratorNetwork.GenerateSample(inputSample)

	// Parse browser HTTP object
	browserHTTPStr := ""
	if val, ok := generatedSample["*BROWSER_HTTP"].(string); ok {
		browserHTTPStr = val
	}
	generatedBrowser := prepareHttpBrowserObject(browserHTTPStr)

	// Add Accept-Language header
	acceptLanguageField := "Accept-Language"
	if generatedBrowser != nil && generatedBrowser.IsHTTP2() {
		acceptLanguageField = "accept-language"
	}
	generatedSample[acceptLanguageField] = hg.getAcceptLanguageHeader(locales)

	// Add Sec-Fetch headers if appropriate
	if generatedBrowser != nil && hg.shouldAddSecFetch(generatedBrowser) {
		if generatedBrowser.IsHTTP2() {
			for k, v := range HTTP2SecFetchAttributes {
				generatedSample[k] = v
			}
		} else {
			for k, v := range HTTP1SecFetchAttributes {
				generatedSample[k] = v
			}
		}
	}

	// Build final headers, omitting internal keys and missing values
	result := make(map[string]string)
	for k, v := range generatedSample {
		vStr, ok := v.(string)
		if !ok {
			continue
		}
		// Skip internal keys and missing values
		if strings.HasPrefix(k, "*") {
			continue
		}
		if vStr == MissingValueDatasetToken {
			continue
		}
		// Skip connection: close
		if strings.ToLower(k) == "connection" && vStr == "close" {
			continue
		}
		result[k] = vStr
	}

	// Add request dependent headers
	for k, v := range requestDependentHeaders {
		result[k] = v
	}

	// Order headers
	return hg.orderHeaders(result), nil
}

// prepareBrowsersConfig converts browser specifications to Browser objects
func (hg *HeaderGenerator) prepareBrowsersConfig(browsers []interface{}, httpVersion string) []Browser {
	result := make([]Browser, 0, len(browsers))
	for _, b := range browsers {
		switch v := b.(type) {
		case string:
			result = append(result, NewBrowser(v, httpVersion))
		case Browser:
			result = append(result, v)
		}
	}
	return result
}

// getBrowserHTTPOptions retrieves browser HTTP options based on browser specifications
func (hg *HeaderGenerator) getBrowserHTTPOptions(browsers []Browser) []string {
	options := make([]string, 0)
	for _, browser := range browsers {
		for _, uniqueBrowser := range hg.UniqueBrowsers {
			if uniqueBrowser.Name == nil || *uniqueBrowser.Name != browser.Name {
				continue
			}
			if browser.MinVersion != nil && len(uniqueBrowser.Version) > 0 && uniqueBrowser.Version[0] < *browser.MinVersion {
				continue
			}
			if browser.MaxVersion != nil && len(uniqueBrowser.Version) > 0 && uniqueBrowser.Version[0] > *browser.MaxVersion {
				continue
			}
			if browser.HTTPVersion != "" && uniqueBrowser.HTTPVersion != browser.HTTPVersion {
				continue
			}
			options = append(options, uniqueBrowser.CompleteString)
		}
	}
	return options
}

// orderHeaders orders headers based on browser-specific header order
func (hg *HeaderGenerator) orderHeaders(headers map[string]string) map[string]string {
	userAgent := GetUserAgent(headers)
	if userAgent == "" {
		return headers
	}

	browserName := GetBrowser(userAgent)
	if browserName == "" {
		return headers
	}

	headerOrder, exists := hg.HeadersOrder[browserName]
	if !exists || len(headerOrder) == 0 {
		return headers
	}

	// Order headers according to browser's header order
	orderedHeaders := make(map[string]string)
	for _, key := range headerOrder {
		if val, ok := headers[key]; ok {
			orderedHeaders[key] = val
		}
	}

	// Add any remaining headers not in the order
	for k, v := range headers {
		if _, exists := orderedHeaders[k]; !exists {
			orderedHeaders[k] = v
		}
	}

	return orderedHeaders
}

// shouldAddSecFetch determines whether Sec-Fetch headers should be added
func (hg *HeaderGenerator) shouldAddSecFetch(browser *HttpBrowserObject) bool {
	if browser == nil || browser.Name == nil || len(browser.Version) == 0 {
		return false
	}

	switch *browser.Name {
	case "chrome":
		return browser.Version[0] >= 76
	case "firefox":
		return browser.Version[0] >= 90
	case "edge":
		return browser.Version[0] >= 79
	}
	return false
}

// getAcceptLanguageHeader generates the Accept-Language header from locales
func (hg *HeaderGenerator) getAcceptLanguageHeader(locales []string) string {
	if len(locales) == 0 {
		return "en-US;q=1.0"
	}

	parts := make([]string, len(locales))
	for i, locale := range locales {
		q := 1.0 - float64(i)*0.1
		parts[i] = fmt.Sprintf("%s;q=%.1f", locale, q)
	}
	return strings.Join(parts, ", ")
}

// prepareConstraints prepares constraints for generating consistent samples
func (hg *HeaderGenerator) prepareConstraints(
	possibleAttributeValues map[string][]string,
	http1Values, http2Values map[string][]string,
) map[string][]string {
	constraints := make(map[string][]string)

	for key, values := range possibleAttributeValues {
		filtered := make([]string, 0)
		for _, value := range values {
			if key == "*BROWSER_HTTP" {
				if hg.filterBrowserHTTP(value, http1Values, http2Values) {
					filtered = append(filtered, value)
				}
			} else {
				if hg.filterOtherValues(value, http1Values, http2Values, key) {
					filtered = append(filtered, value)
				}
			}
		}
		if len(filtered) > 0 {
			constraints[key] = filtered
		}
	}

	return constraints
}

// filterBrowserHTTP filters browser HTTP values based on HTTP/1 and HTTP/2 values
func (hg *HeaderGenerator) filterBrowserHTTP(value string, http1Values, http2Values map[string][]string) bool {
	parts := strings.Split(value, "|")
	if len(parts) != 2 {
		return false
	}
	browserName := parts[0]
	httpVersion := parts[1]

	if httpVersion == "1" {
		if len(http1Values) == 0 {
			return true
		}
		return ContainsString(http1Values["*BROWSER"], browserName)
	}
	if len(http2Values) == 0 {
		return true
	}
	return ContainsString(http2Values["*BROWSER"], browserName)
}

// filterOtherValues filters other attribute values based on HTTP/1 and HTTP/2 values
func (hg *HeaderGenerator) filterOtherValues(value string, http1Values, http2Values map[string][]string, key string) bool {
	if len(http1Values) == 0 && len(http2Values) == 0 {
		return true
	}
	return ContainsString(http1Values[key], value) || ContainsString(http2Values[key], value)
}

// prepareHttpBrowserObject extracts structured browser info from a string
func prepareHttpBrowserObject(httpBrowserString string) *HttpBrowserObject {
	if httpBrowserString == "" || httpBrowserString == MissingValueDatasetToken {
		return nil
	}

	parts := strings.Split(httpBrowserString, "|")
	if len(parts) != 2 {
		return nil
	}

	browserString := parts[0]
	httpVersion := parts[1]

	browserParts := strings.Split(browserString, "/")
	if len(browserParts) != 2 {
		return nil
	}

	browserName := browserParts[0]
	versionString := browserParts[1]

	versionParts := strings.Split(versionString, ".")
	version := make([]int, 0, len(versionParts))
	for _, part := range versionParts {
		v, err := strconv.Atoi(part)
		if err != nil {
			v = 0
		}
		version = append(version, v)
	}

	return &HttpBrowserObject{
		Name:           &browserName,
		Version:        version,
		CompleteString: httpBrowserString,
		HTTPVersion:    httpVersion,
	}
}
