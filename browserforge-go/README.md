# BrowserForge-Go

🎭 Intelligent browser header & fingerprint generator for Go

A Go port of the Python [browserforge](https://github.com/daijro/browserforge) library, which is a reimplementation of [Apify's fingerprint-suite](https://github.com/apify/fingerprint-suite).

## Features

- Uses a Bayesian generative network to mimic actual web traffic
- Extremely fast runtime
- Extensive customization options for browsers, operating systems, devices, locales, and HTTP version
- Generates realistic browser fingerprints including screen, navigator, video card, codecs, and fonts

## Installation

```bash
go get github.com/uzzalhcse/browserforge-go
```

## Usage

### Generating Headers

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/uzzalhcse/browserforge-go/headers"
)

func main() {
    // Create a header generator with default options
    headerGen, err := headers.NewHeaderGenerator(nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // Generate headers
    hdrs, err := headerGen.Generate(nil)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("User-Agent: %s\n", headers.GetUserAgent(hdrs))
}
```

### Constraining Headers

```go
// Generate headers for Chrome on Windows
hdrs, err := headerGen.Generate(&headers.GenerateOptions{
    Browsers: []interface{}{"chrome"},
    OS:       []string{"windows"},
    Devices:  []string{"desktop"},
    Locales:  []string{"en-US", "en"},
})
```

### Generating Fingerprints

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/uzzalhcse/browserforge-go/fingerprints"
)

func main() {
    // Create a fingerprint generator
    fpGen, err := fingerprints.NewFingerprintGenerator(nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // Generate a fingerprint
    fp, err := fpGen.Generate(nil)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("User-Agent: %s\n", fp.Navigator.UserAgent)
    fmt.Printf("Platform: %s\n", fp.Navigator.Platform)
    fmt.Printf("Screen: %dx%d\n", fp.Screen.Width, fp.Screen.Height)
}
```

### Constraining Fingerprints

```go
// Set screen constraints
minWidth := 1920
minHeight := 1080

fp, err := fpGen.Generate(&fingerprints.GenerateFingerprintOptions{
    Screen: &fingerprints.Screen{
        MinWidth:  &minWidth,
        MinHeight: &minHeight,
    },
})
```

### Fingerprint Output Example

```go
Fingerprint{
    Screen: ScreenFingerprint{
        Width: 1920, Height: 1080,
        ColorDepth: 24, PixelDepth: 24,
        DevicePixelRatio: 1,
    },
    Navigator: NavigatorFingerprint{
        UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) ...",
        Platform: "Win32",
        Languages: ["en-US"],
        HardwareConcurrency: 12,
        DeviceMemory: 8,
    },
    VideoCard: &VideoCard{
        Renderer: "ANGLE (AMD, AMD Radeon ...)",
        Vendor: "Google Inc. (AMD)",
    },
    Headers: map[string]string{...},
    VideoCodecs: map[string]string{...},
    AudioCodecs: map[string]string{...},
    Fonts: []string{...},
}
```

## API Reference

### HeaderGenerator

```go
// Create with default options
gen, err := headers.NewHeaderGenerator(nil)

// Create with custom options
gen, err := headers.NewHeaderGenerator(&headers.HeaderGeneratorOptions{
    Browsers:    []Browser{{Name: "chrome"}, {Name: "firefox"}},
    OS:          []string{"windows", "macos"},
    Devices:     []string{"desktop"},
    Locales:     []string{"en-US"},
    HTTPVersion: "2",
    Strict:      false,
})

// Generate headers
hdrs, err := gen.Generate(nil)
```

### FingerprintGenerator

```go
// Create with default options
gen, err := fingerprints.NewFingerprintGenerator(nil)

// Create with custom options
gen, err := fingerprints.NewFingerprintGenerator(&fingerprints.FingerprintGeneratorOptions{
    Screen:     &fingerprints.Screen{MinWidth: intPtr(1024)},
    Strict:     false,
    MockWebRTC: false,
    Slim:       false,
})

// Generate fingerprint
fp, err := gen.Generate(nil)
```

## License

Apache-2.0 (same as original browserforge)
