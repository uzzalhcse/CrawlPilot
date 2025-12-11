// Package headers provides HTTP header generation for browserforge-go.
package headers

// Supported browsers for header generation
var SupportedBrowsers = []string{"chrome", "firefox", "safari", "edge"}

// Supported operating systems for header generation
var SupportedOperatingSystems = []string{"windows", "macos", "linux", "android", "ios"}

// Supported devices for header generation
var SupportedDevices = []string{"desktop", "mobile"}

// Supported HTTP versions
var SupportedHTTPVersions = []string{"1", "2"}

// MissingValueDatasetToken represents a missing value in the dataset
const MissingValueDatasetToken = "*MISSING_VALUE*"

// HTTP1SecFetchAttributes are the Sec-Fetch headers for HTTP/1
var HTTP1SecFetchAttributes = map[string]string{
	"Sec-Fetch-Mode": "same-site",
	"Sec-Fetch-Dest": "navigate",
	"Sec-Fetch-Site": "?1",
	"Sec-Fetch-User": "document",
}

// HTTP2SecFetchAttributes are the Sec-Fetch headers for HTTP/2
var HTTP2SecFetchAttributes = map[string]string{
	"sec-fetch-mode": "same-site",
	"sec-fetch-dest": "navigate",
	"sec-fetch-site": "?1",
	"sec-fetch-user": "document",
}

// Browser represents a browser specification with name, min/max version, and HTTP version
type Browser struct {
	Name        string
	MinVersion  *int
	MaxVersion  *int
	HTTPVersion string
}

// NewBrowser creates a new Browser with the given name and HTTP version
func NewBrowser(name string, httpVersion string) Browser {
	return Browser{
		Name:        name,
		HTTPVersion: httpVersion,
	}
}

// NewBrowserWithVersion creates a new Browser with version constraints
func NewBrowserWithVersion(name string, minVersion, maxVersion *int, httpVersion string) Browser {
	return Browser{
		Name:        name,
		MinVersion:  minVersion,
		MaxVersion:  maxVersion,
		HTTPVersion: httpVersion,
	}
}

// HttpBrowserObject represents a parsed HTTP browser object
type HttpBrowserObject struct {
	Name           *string
	Version        []int
	CompleteString string
	HTTPVersion    string
}

// IsHTTP2 returns true if this browser uses HTTP/2
func (h *HttpBrowserObject) IsHTTP2() bool {
	return h.HTTPVersion == "2"
}

// HeaderGeneratorOptions contains options for header generation
type HeaderGeneratorOptions struct {
	Browsers    []Browser
	OS          []string
	Devices     []string
	Locales     []string
	HTTPVersion string
	Strict      bool
}

// GenerateOptions contains options for a single header generation call
type GenerateOptions struct {
	Browsers                []interface{} // Can be string or Browser
	OS                      []string
	Devices                 []string
	Locales                 []string
	HTTPVersion             *string
	UserAgents              []string
	Strict                  *bool
	RequestDependentHeaders map[string]string
}

// DefaultHeaderGeneratorOptions returns default options for header generation
func DefaultHeaderGeneratorOptions() HeaderGeneratorOptions {
	return HeaderGeneratorOptions{
		Browsers:    browsersFromStrings(SupportedBrowsers, "2"),
		OS:          SupportedOperatingSystems,
		Devices:     SupportedDevices,
		Locales:     []string{"en-US"},
		HTTPVersion: "2",
		Strict:      false,
	}
}

// browsersFromStrings converts a list of browser name strings to Browser objects
func browsersFromStrings(names []string, httpVersion string) []Browser {
	browsers := make([]Browser, len(names))
	for i, name := range names {
		browsers[i] = NewBrowser(name, httpVersion)
	}
	return browsers
}
