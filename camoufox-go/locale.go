package camoufox

import (
	_ "embed"
	"encoding/xml"
	"math/rand"
	"strings"
)

//go:embed internal/territoryInfo.xml
var territoryInfoXML []byte

// TerritoryInfo holds territory data for statistical locale selection
type TerritoryInfo struct {
	Territories []Territory `xml:"territory"`
}

// Territory represents a country/region
type Territory struct {
	Type                string               `xml:"type,attr"`
	LiteracyPercent     float64              `xml:"literacyPercent,attr"`
	Population          int64                `xml:"population,attr"`
	LanguagePopulations []LanguagePopulation `xml:"languagePopulation"`
}

// LanguagePopulation represents language usage within a territory
type LanguagePopulation struct {
	Type              string  `xml:"type,attr"`
	PopulationPercent float64 `xml:"populationPercent,attr"`
	WritingPercent    float64 `xml:"writingPercent,attr"`
	OfficialStatus    string  `xml:"officialStatus,attr"`
}

// Locale represents a language-region combination
type Locale struct {
	Language string
	Region   string
	Script   string
}

// AsString returns the locale as a string (e.g., "en-US")
func (l *Locale) AsString() string {
	if l.Region != "" {
		return l.Language + "-" + l.Region
	}
	return l.Language
}

var territories map[string]*Territory

func init() {
	// Parse territory info on init
	var info struct {
		Territories []Territory `xml:"territory"`
	}
	if err := xml.Unmarshal(territoryInfoXML, &info); err == nil {
		territories = make(map[string]*Territory)
		for i := range info.Territories {
			territories[info.Territories[i].Type] = &info.Territories[i]
		}
	} else {
		territories = make(map[string]*Territory)
	}
}

// GetLocaleForRegion returns a statistically-selected locale for a region (ISO code)
// Based on the probability that a person speaks the language in the territory
func GetLocaleForRegion(regionCode string) *Locale {
	territory, ok := territories[strings.ToUpper(regionCode)]
	if !ok || len(territory.LanguagePopulations) == 0 {
		return getDefaultLocale(regionCode)
	}

	// Build probability distribution
	var totalPercent float64
	for _, lp := range territory.LanguagePopulations {
		totalPercent += lp.PopulationPercent
	}

	if totalPercent == 0 {
		return getDefaultLocale(regionCode)
	}

	// Select randomly based on population percentage
	roll := rand.Float64() * totalPercent
	var cumulative float64
	for _, lp := range territory.LanguagePopulations {
		cumulative += lp.PopulationPercent
		if roll <= cumulative {
			// Convert language code format (en_US -> en)
			lang := strings.Split(lp.Type, "_")[0]
			return &Locale{
				Language: lang,
				Region:   regionCode,
			}
		}
	}

	// Fallback to first language
	lang := strings.Split(territory.LanguagePopulations[0].Type, "_")[0]
	return &Locale{
		Language: lang,
		Region:   regionCode,
	}
}

// GetLocaleForLanguage returns a statistically-selected region for a language
func GetLocaleForLanguage(language string) *Locale {
	type RegionWeight struct {
		Region string
		Weight float64
	}

	var regions []RegionWeight
	var totalWeight float64

	for regionCode, territory := range territories {
		for _, lp := range territory.LanguagePopulations {
			langCode := strings.Split(lp.Type, "_")[0]
			if langCode == language {
				// Weight by population percent * literacy * total population
				weight := lp.PopulationPercent * territory.LiteracyPercent * float64(territory.Population) / 100000000
				regions = append(regions, RegionWeight{
					Region: regionCode,
					Weight: weight,
				})
				totalWeight += weight
				break
			}
		}
	}

	if len(regions) == 0 || totalWeight == 0 {
		// Fallback to common regions
		return getDefaultLocaleForLanguage(language)
	}

	// Select randomly based on weight
	roll := rand.Float64() * totalWeight
	var cumulative float64
	for _, r := range regions {
		cumulative += r.Weight
		if roll <= cumulative {
			return &Locale{
				Language: language,
				Region:   r.Region,
			}
		}
	}

	return &Locale{
		Language: language,
		Region:   regions[0].Region,
	}
}

// getDefaultLocale returns a default locale for common regions
func getDefaultLocale(region string) *Locale {
	defaults := map[string]string{
		"US": "en", "GB": "en", "CA": "en", "AU": "en", "NZ": "en",
		"DE": "de", "AT": "de", "CH": "de",
		"FR": "fr", "BE": "fr",
		"ES": "es", "MX": "es", "AR": "es",
		"IT": "it",
		"JP": "ja",
		"CN": "zh", "TW": "zh", "HK": "zh",
		"KR": "ko",
		"RU": "ru",
		"BR": "pt", "PT": "pt",
		"NL": "nl",
		"PL": "pl",
		"SE": "sv",
		"NO": "nb",
		"DK": "da",
		"FI": "fi",
		"IN": "hi",
		"ID": "id",
		"TH": "th",
		"VN": "vi",
		"TR": "tr",
	}

	if lang, ok := defaults[strings.ToUpper(region)]; ok {
		return &Locale{Language: lang, Region: region}
	}
	return &Locale{Language: "en", Region: region}
}

// getDefaultLocaleForLanguage returns default region for a language
func getDefaultLocaleForLanguage(language string) *Locale {
	defaults := map[string]string{
		"en": "US", "de": "DE", "fr": "FR", "es": "ES", "it": "IT",
		"ja": "JP", "zh": "CN", "ko": "KR", "ru": "RU", "pt": "BR",
		"nl": "NL", "pl": "PL", "sv": "SE", "nb": "NO", "da": "DK",
		"fi": "FI", "hi": "IN", "id": "ID", "th": "TH", "vi": "VN",
		"tr": "TR", "ar": "SA", "he": "IL",
	}

	if region, ok := defaults[strings.ToLower(language)]; ok {
		return &Locale{Language: language, Region: region}
	}
	return &Locale{Language: language, Region: "US"}
}

// ApplyLocaleToConfig applies locale settings to the config
func ApplyLocaleToConfig(locale *Locale, config map[string]interface{}) {
	if locale == nil {
		return
	}

	config["locale:language"] = locale.Language
	if locale.Region != "" {
		config["locale:region"] = locale.Region
	}
	if locale.Script != "" {
		config["locale:script"] = locale.Script
	}

	// Also set navigator.language and Accept-Language header
	config["navigator.language"] = locale.AsString()
	config["navigator.languages"] = []string{locale.AsString(), locale.Language}
	config["headers.Accept-Language"] = locale.AsString() + "," + locale.Language + ";q=0.9"
}
