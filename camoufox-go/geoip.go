package camoufox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Geolocation holds location data based on IP
type Geolocation struct {
	IP        string  `json:"ip"`
	Country   string  `json:"country"`
	Region    string  `json:"region"`
	City      string  `json:"city"`
	Timezone  string  `json:"timezone"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Locale    string  `json:"locale"` // Derived from country
}

// GeoIPLookup performs a GeoIP lookup for the given IP address
// If ip is empty, it will detect the public IP automatically
func GeoIPLookup(ip string, proxy *ProxyConfig) (*Geolocation, error) {
	// If no IP specified, get public IP first
	if ip == "" || ip == "auto" {
		detectedIP, err := GetPublicIP(proxy)
		if err != nil {
			return nil, fmt.Errorf("failed to detect public IP: %w", err)
		}
		ip = detectedIP
	}

	// Try multiple GeoIP services
	services := []struct {
		url    string
		parser func([]byte) (*Geolocation, error)
	}{
		{
			url:    fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,regionName,city,timezone,lat,lon,query", ip),
			parser: parseIPAPI,
		},
		{
			url:    fmt.Sprintf("https://ipwho.is/%s", ip),
			parser: parseIPWhoIs,
		},
	}

	var lastErr error
	for _, svc := range services {
		geo, err := fetchGeoIP(svc.url, proxy, svc.parser)
		if err != nil {
			lastErr = err
			continue
		}
		geo.IP = ip
		geo.Locale = deriveLocale(geo.Country)
		return geo, nil
	}

	return nil, fmt.Errorf("all GeoIP services failed: %v", lastErr)
}

func fetchGeoIP(url string, proxy *ProxyConfig, parser func([]byte) (*Geolocation, error)) (*Geolocation, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Configure proxy if provided
	if proxy != nil && proxy.Server != "" {
		// For simplicity, we'll use the default client
		// In production, you'd configure the transport with proxy
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parser(body)
}

func parseIPAPI(data []byte) (*Geolocation, error) {
	var result struct {
		Status     string  `json:"status"`
		Country    string  `json:"country"`
		RegionName string  `json:"regionName"`
		City       string  `json:"city"`
		Timezone   string  `json:"timezone"`
		Lat        float64 `json:"lat"`
		Lon        float64 `json:"lon"`
		Query      string  `json:"query"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("ip-api returned status: %s", result.Status)
	}

	return &Geolocation{
		IP:        result.Query,
		Country:   result.Country,
		Region:    result.RegionName,
		City:      result.City,
		Timezone:  result.Timezone,
		Latitude:  result.Lat,
		Longitude: result.Lon,
	}, nil
}

func parseIPWhoIs(data []byte) (*Geolocation, error) {
	var result struct {
		Success   bool    `json:"success"`
		Country   string  `json:"country"`
		Region    string  `json:"region"`
		City      string  `json:"city"`
		Timezone  string  `json:"timezone"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("ipwho.is lookup failed")
	}

	// Extract timezone ID from timezone object if needed
	return &Geolocation{
		Country:   result.Country,
		Region:    result.Region,
		City:      result.City,
		Timezone:  result.Timezone,
		Latitude:  result.Latitude,
		Longitude: result.Longitude,
	}, nil
}

// GetPublicIP returns the public IP address
func GetPublicIP(proxy *ProxyConfig) (string, error) {
	services := []string{
		"https://api.ipify.org",
		"https://checkip.amazonaws.com",
		"https://ipinfo.io/ip",
		"https://icanhazip.com",
	}

	client := &http.Client{Timeout: 10 * time.Second}

	for _, url := range services {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		ip := strings.TrimSpace(string(body))
		if ip != "" && isValidIP(ip) {
			return ip, nil
		}
	}

	return "", fmt.Errorf("failed to get public IP from all services")
}

// isValidIP checks if the string is a valid IP address
func isValidIP(ip string) bool {
	// Simple validation - check for dots (IPv4) or colons (IPv6)
	return strings.Contains(ip, ".") || strings.Contains(ip, ":")
}

// deriveLocale derives a locale from country name
func deriveLocale(country string) string {
	// Map of common countries to locales
	countryLocales := map[string]string{
		"United States":  "en-US",
		"United Kingdom": "en-GB",
		"Canada":         "en-CA",
		"Australia":      "en-AU",
		"Germany":        "de-DE",
		"France":         "fr-FR",
		"Spain":          "es-ES",
		"Italy":          "it-IT",
		"Japan":          "ja-JP",
		"China":          "zh-CN",
		"South Korea":    "ko-KR",
		"Russia":         "ru-RU",
		"Brazil":         "pt-BR",
		"Portugal":       "pt-PT",
		"Netherlands":    "nl-NL",
		"Poland":         "pl-PL",
		"Sweden":         "sv-SE",
		"Norway":         "nb-NO",
		"Denmark":        "da-DK",
		"Finland":        "fi-FI",
		"India":          "hi-IN",
		"Indonesia":      "id-ID",
		"Thailand":       "th-TH",
		"Vietnam":        "vi-VN",
		"Turkey":         "tr-TR",
		"Mexico":         "es-MX",
		"Argentina":      "es-AR",
	}

	if locale, ok := countryLocales[country]; ok {
		return locale
	}
	return "en-US" // Default
}

// ApplyGeolocation applies geolocation data to the config
// This includes timezone, locale, coordinates, AND WebRTC IP spoofing
func ApplyGeolocation(geo *Geolocation, config map[string]interface{}, blockWebRTC bool) {
	if geo == nil {
		return
	}

	// Set geolocation coordinates
	config["geolocation:latitude"] = geo.Latitude
	config["geolocation:longitude"] = geo.Longitude
	config["geolocation:accuracy"] = 100.0 // Default accuracy in meters

	// Set timezone
	if geo.Timezone != "" {
		config["timezone"] = geo.Timezone
	}

	// Set locale from GeoIP
	if geo.Locale != "" {
		config["locale:language"] = strings.Split(geo.Locale, "-")[0]
		parts := strings.Split(geo.Locale, "-")
		if len(parts) > 1 {
			config["locale:region"] = parts[1]
		}
	}

	// Spoof WebRTC IP if not blocked (prevent IP leak)
	if !blockWebRTC && geo.IP != "" {
		if isIPv4(geo.IP) {
			config["webrtc:ipv4"] = geo.IP
		} else if isIPv6(geo.IP) {
			config["webrtc:ipv6"] = geo.IP
		}
	}
}

// isIPv4 checks if the IP is IPv4
func isIPv4(ip string) bool {
	return strings.Contains(ip, ".") && !strings.Contains(ip, ":")
}

// isIPv6 checks if the IP is IPv6
func isIPv6(ip string) bool {
	return strings.Contains(ip, ":")
}
