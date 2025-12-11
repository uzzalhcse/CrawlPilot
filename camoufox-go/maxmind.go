package camoufox

import (
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/oschwald/geoip2-golang"
)

const (
	// GeoLite2 database download URL (community-maintained mirror, updated twice/month)
	geoLite2DownloadURL = "https://github.com/wp-statistics/GeoLite2-City/raw/master/GeoLite2-City.mmdb.gz"
	geoLite2FileName    = "GeoLite2-City.mmdb"
)

// Geolocation holds location data based on IP
type Geolocation struct {
	IP          string  `json:"ip"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"` // ISO 3166-1 alpha-2 code for locale selection
	Region      string  `json:"region"`
	City        string  `json:"city"`
	Timezone    string  `json:"timezone"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Locale      string  `json:"locale"`
}

// MaxMindReader provides GeoIP lookup using MaxMind GeoLite2 database
type MaxMindReader struct {
	db   *geoip2.Reader
	path string
	mu   sync.RWMutex
}

var (
	maxmindReader     *MaxMindReader
	maxmindReaderOnce sync.Once
	maxmindInitErr    error
)

// GetMaxMindReader returns the singleton MaxMind database reader
// Downloads the database if it doesn't exist
func GetMaxMindReader() (*MaxMindReader, error) {
	maxmindReaderOnce.Do(func() {
		maxmindReader, maxmindInitErr = NewMaxMindReader("")
	})
	return maxmindReader, maxmindInitErr
}

// NewMaxMindReader creates a new MaxMind GeoLite2 database reader
// Downloads the database if it doesn't exist
func NewMaxMindReader(dbPath string) (*MaxMindReader, error) {
	if dbPath == "" {
		dbPath = findMaxMindDB()
	}

	// Download if not found
	if dbPath == "" {
		var err error
		dbPath, err = downloadGeoLiteDB()
		if err != nil {
			return nil, fmt.Errorf("failed to download GeoLite2 database: %w", err)
		}
	}

	db, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open GeoLite2 database: %w", err)
	}

	return &MaxMindReader{
		db:   db,
		path: dbPath,
	}, nil
}

// findMaxMindDB searches for GeoLite2-City.mmdb in standard locations
func findMaxMindDB() string {
	homeDir, _ := os.UserHomeDir()

	searchPaths := []string{
		filepath.Join(homeDir, ".cache", "camoufox-go", geoLite2FileName),
		filepath.Join(homeDir, ".cache", "camoufox", geoLite2FileName),
		"/usr/share/GeoIP/" + geoLite2FileName,
		"/var/lib/GeoIP/" + geoLite2FileName,
		geoLite2FileName,
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// downloadGeoLiteDB downloads the GeoLite2 database
func downloadGeoLiteDB() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(homeDir, ".cache", "camoufox-go")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}

	destPath := filepath.Join(cacheDir, geoLite2FileName)

	fmt.Println("Downloading GeoLite2-City database...")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(geoLite2DownloadURL)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Decompress gzip
	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Write to file
	outFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, gzReader); err != nil {
		os.Remove(destPath)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("GeoLite2 database installed: %s\n", destPath)
	return destPath, nil
}

// Lookup performs a GeoIP lookup for the given IP address
func (r *MaxMindReader) Lookup(ip string) (*Geolocation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	record, err := r.db.City(parsedIP)
	if err != nil {
		return nil, fmt.Errorf("lookup failed: %w", err)
	}

	cityName := ""
	if name, ok := record.City.Names["en"]; ok {
		cityName = name
	}

	countryName := ""
	if name, ok := record.Country.Names["en"]; ok {
		countryName = name
	}

	regionName := ""
	if len(record.Subdivisions) > 0 {
		if name, ok := record.Subdivisions[0].Names["en"]; ok {
			regionName = name
		}
	}

	return &Geolocation{
		IP:          ip,
		Country:     countryName,
		CountryCode: record.Country.IsoCode, // ISO 3166-1 alpha-2 code
		Region:      regionName,
		City:        cityName,
		Timezone:    record.Location.TimeZone,
		Latitude:    record.Location.Latitude,
		Longitude:   record.Location.Longitude,
		Locale:      deriveLocale(countryName),
	}, nil
}

// Close closes the MaxMind database reader
func (r *MaxMindReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// ExtractIPFromProxy extracts the IP address from a proxy server URL
// Supports formats: "82.22.69.28:7235", "http://236.22.02.144:6952", "socks5://user:pass@1.2.3.4:1080"
func ExtractIPFromProxy(proxyServer string) string {
	if proxyServer == "" {
		return ""
	}

	// Parse as URL to handle various formats
	server := proxyServer

	// Remove scheme if present
	if idx := strings.Index(server, "://"); idx != -1 {
		server = server[idx+3:]
	}

	// Remove userinfo (user:pass@) if present
	if idx := strings.Index(server, "@"); idx != -1 {
		server = server[idx+1:]
	}

	// Extract host (remove port)
	host := server
	if idx := strings.LastIndex(server, ":"); idx != -1 {
		host = server[:idx]
	}

	// Validate it's an IP address (not a hostname)
	if ip := net.ParseIP(host); ip != nil {
		return host
	}

	return ""
}

// GeoIPLookup performs GeoIP lookup using MaxMind database
func GeoIPLookup(ip string, proxy *ProxyConfig) (*Geolocation, error) {
	reader, err := GetMaxMindReader()
	if err != nil {
		return nil, err
	}

	// If ip is "auto", try to extract from proxy first (no API call needed)
	if ip == "" || ip == "auto" {
		if proxy != nil && proxy.Server != "" {
			// Try to extract IP from proxy URL (e.g., "82.22.69.28:7235")
			extractedIP := ExtractIPFromProxy(proxy.Server)
			if extractedIP != "" {
				ip = extractedIP
			}
		}

		// Fallback to public IP detection if extraction failed
		if ip == "" || ip == "auto" {
			detectedIP, err := GetPublicIP(proxy)
			if err != nil {
				return nil, fmt.Errorf("failed to detect public IP: %w", err)
			}
			ip = detectedIP
		}
	}

	return reader.Lookup(ip)
}

// ApplyGeolocation applies geolocation data to the config
// This follows the Python implementation by using statistical locale selection
// and applying full locale config (locale:language, locale:region, etc.)
func ApplyGeolocation(geo *Geolocation, config map[string]interface{}, blockWebRTC bool) {
	ApplyGeolocationWithPrefs(geo, config, nil, blockWebRTC)
}

// ApplyGeolocationWithPrefs applies geolocation data to the config and Firefox preferences
// When using IPv4, it also disables IPv6 DNS to match the Python implementation
func ApplyGeolocationWithPrefs(geo *Geolocation, config map[string]interface{}, firefoxPrefs map[string]interface{}, blockWebRTC bool) {
	if geo == nil {
		return
	}

	// Set geolocation data
	config["geolocation:latitude"] = geo.Latitude
	config["geolocation:longitude"] = geo.Longitude
	config["geolocation:accuracy"] = 100.0
	config["timezone"] = geo.Timezone

	// Apply statistically-selected locale based on country code
	// This matches the Python implementation which uses get_geolocation() -> Geolocation.as_config()
	if geo.CountryCode != "" {
		locale := GetLocaleForRegion(geo.CountryCode)
		ApplyLocaleToConfig(locale, config)
	}

	// Spoof WebRTC if not blocked
	if !blockWebRTC && geo.IP != "" {
		if isIPv4(geo.IP) {
			config["webrtc:ipv4"] = geo.IP
			// Disable IPv6 DNS when using IPv4 (matches Python implementation)
			if firefoxPrefs != nil {
				firefoxPrefs["network.dns.disableIPv6"] = true
			}
		} else if isIPv6(geo.IP) {
			config["webrtc:ipv6"] = geo.IP
		}
	}
}

// GetPublicIP returns the public IP address (through proxy if configured)
func GetPublicIP(proxy *ProxyConfig) (string, error) {
	services := []string{
		"https://api.ipify.org",
		"https://checkip.amazonaws.com",
		"https://ipinfo.io/ip",
		"https://icanhazip.com",
	}

	client := createHTTPClient(proxy)

	for _, svcURL := range services {
		resp, err := client.Get(svcURL)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		ip := trimSpace(string(body))
		if ip != "" && isValidIP(ip) {
			return ip, nil
		}
	}

	return "", fmt.Errorf("failed to get public IP")
}

// Helper functions

func deriveLocale(country string) string {
	countryToLocale := map[string]string{
		"United States": "en-US", "United Kingdom": "en-GB", "Canada": "en-CA",
		"Australia": "en-AU", "Germany": "de-DE", "France": "fr-FR",
		"Spain": "es-ES", "Italy": "it-IT", "Japan": "ja-JP",
		"China": "zh-CN", "Russia": "ru-RU", "Brazil": "pt-BR",
		"India": "en-IN", "Netherlands": "nl-NL", "South Korea": "ko-KR",
	}
	if locale, ok := countryToLocale[country]; ok {
		return locale
	}
	return "en-US"
}

func isIPv4(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil && parsedIP.To4() != nil
}

func isIPv6(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil && parsedIP.To4() == nil
}

func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func trimSpace(s string) string {
	result := ""
	for _, c := range s {
		if c != ' ' && c != '\n' && c != '\r' && c != '\t' {
			result += string(c)
		}
	}
	return result
}

func createHTTPClient(proxy *ProxyConfig) *http.Client {
	transport := &http.Transport{}

	if proxy != nil && proxy.Server != "" {
		proxyURL, err := parseProxyURL(proxy)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}
}

// parseProxyURL converts ProxyConfig to *url.URL
func parseProxyURL(proxy *ProxyConfig) (*url.URL, error) {
	if proxy == nil || proxy.Server == "" {
		return nil, fmt.Errorf("no proxy configured")
	}

	proxyURL, err := url.Parse(proxy.Server)
	if err != nil {
		return nil, err
	}

	// Add authentication if provided
	if proxy.Username != "" {
		proxyURL.User = url.UserPassword(proxy.Username, proxy.Password)
	}

	return proxyURL, nil
}
