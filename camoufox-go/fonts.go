package camoufox

import (
	_ "embed"
	"encoding/json"
	"math/rand"
)

//go:embed internal/fonts/fonts.json
var fontsJSON []byte

// FontsByOS maps operating system to list of font families
var fontsByOS map[string][]string

func init() {
	if err := json.Unmarshal(fontsJSON, &fontsByOS); err != nil {
		// Fallback to empty map if parsing fails
		fontsByOS = make(map[string][]string)
	}
}

// GetFontsForOS returns the list of font families for the given OS
// targetOS should be "win", "mac", or "lin"
func GetFontsForOS(targetOS string) []string {
	// Convert common OS names to internal format
	osKey := targetOS
	switch targetOS {
	case "windows":
		osKey = "win"
	case "macos", "darwin":
		osKey = "mac"
	case "linux":
		osKey = "lin"
	}

	fonts, ok := fontsByOS[osKey]
	if !ok {
		return []string{}
	}
	return fonts
}

// GetRandomFonts returns a random subset of fonts for the given OS
func GetRandomFonts(targetOS string, count int) []string {
	allFonts := GetFontsForOS(targetOS)
	if len(allFonts) == 0 || count <= 0 {
		return []string{}
	}

	if count >= len(allFonts) {
		return allFonts
	}

	// Shuffle and take first N fonts
	shuffled := make([]string, len(allFonts))
	copy(shuffled, allFonts)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:count]
}

// MergeFonts merges custom fonts with OS fonts, removing duplicates
func MergeFonts(customFonts []string, osFonts []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(customFonts)+len(osFonts))

	// Add custom fonts first (priority)
	for _, font := range customFonts {
		if !seen[font] {
			seen[font] = true
			result = append(result, font)
		}
	}

	// Add OS fonts
	for _, font := range osFonts {
		if !seen[font] {
			seen[font] = true
			result = append(result, font)
		}
	}

	return result
}
