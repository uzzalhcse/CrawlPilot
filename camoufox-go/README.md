# 🦊 camoufox-go

[![Go Reference](https://pkg.go.dev/badge/github.com/uzzalhcse/camoufox-go.svg)](https://pkg.go.dev/github.com/uzzalhcse/camoufox-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Go library for using [Camoufox](https://github.com/daijro/camoufox) anti-detect browser natively with [playwright-go](https://github.com/playwright-community/playwright-go).

Camoufox is a stealthy, custom build of Firefox designed for web scraping and automation with robust anti-bot evasion capabilities.

## Features

- 🎭 **Anti-detect browser** - Invisible to all major anti-bot systems
- 🔄 **Fingerprint rotation** - Realistic fingerprints via [BrowserForge](https://github.com/daijro/browserforge)
- 🖱️ **Human-like behavior** - Optional humanized mouse movement
- 🔌 **Playwright compatible** - Works seamlessly with playwright-go
- 🌐 **Proxy support** - Built-in proxy configuration
- ⚡ **Simple API** - Just swap your browser initialization

## Installation

```bash
go get github.com/uzzalhcse/camoufox-go
```

### Prerequisites

**Python 3.8+** is required. Then install all dependencies with one call:

```go
import "github.com/uzzalhcse/camoufox-go"

// Install all dependencies (browserforge + camoufox + browser binary)
if err := camoufox.Install(); err != nil {
    log.Fatal(err)
}
```

Or install manually via pip:
```bash
pip install browserforge camoufox
python -c "import camoufox; camoufox.install()"
```

## Quick Start

```go
package main

import (
    "log"
    "github.com/uzzalhcse/camoufox-go"
)

func main() {
    // Launch Camoufox browser
    browser, err := camoufox.NewBrowser(camoufox.Options{
        OS: "linux",      // Fingerprint OS: windows, macos, linux
        Headless: false,  // Run in headful mode
    })
    if err != nil {
        log.Fatal(err)
    }
    defer browser.Close()

    // Create a page and navigate
    page, _ := browser.NewPage()
    page.Goto("https://bot.sannysoft.com/")
    
    // Your automation code here...
}
```

## Options

```go
camoufox.Options{
    // Target OS for fingerprint generation
    OS: "linux",  // "windows", "macos", "linux"
    
    // Run in headless mode
    Headless: false,
    
    // Enable human-like mouse movement (max duration in seconds)
    Humanize: 1.5,
    
    // Proxy configuration
    Proxy: &camoufox.ProxyConfig{
        Server:   "http://proxy.example.com:8080",
        Username: "user",
        Password: "pass",
    },
    
    // Screen constraints for fingerprint
    Screen: &camoufox.Screen{
        MaxWidth:  1920,
        MaxHeight: 1080,
    },
    
    // Custom executable path (optional)
    ExecutablePath: "/path/to/camoufox",
    
    // Block resources
    BlockImages: true,
    BlockWebRTC: true,
}
```

## Examples

### With Proxy

```go
browser, _ := camoufox.NewBrowser(camoufox.Options{
    OS: "windows",
    Proxy: &camoufox.ProxyConfig{
        Server:   "socks5://proxy.example.com:1080",
        Username: "user",
        Password: "pass",
    },
})
```

### Check & Install

```go
// Check if Camoufox is installed, install if not
if !camoufox.IsInstalled() {
    fmt.Println("Installing Camoufox and dependencies...")
    if err := camoufox.Install(); err != nil {
        log.Fatal(err)
    }
}

// Now ready to use
browser, _ := camoufox.NewBrowser(camoufox.Options{})
```

## How It Works

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│   Go App    │────▶│ camoufox-go  │────▶│ Camoufox Binary │
└─────────────┘     └──────────────┘     └─────────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ BrowserForge │
                    │   (Python)   │
                    └──────────────┘
```

1. **camoufox-go** calls a Python script to generate realistic fingerprints using BrowserForge
2. Fingerprint config is passed to **Camoufox** via environment variables
3. You get a **playwright-go** browser instance with anti-detect capabilities

## FAQ

**Q: Why is Python required?**  
A: BrowserForge uses a Bayesian neural network for realistic fingerprint generation. The ~0.1ms Python call happens once per browser session.

**Q: Does this work in Docker?**  
A: Yes! Ensure Python 3.8+ and browserforge are installed in your container.

**Q: What about Chromium?**  
A: Camoufox is Firefox-based. Firefox is preferred for anti-detection due to Juggler protocol (vs CDP) and better fingerprinting research.

## Credits

- [Camoufox](https://github.com/daijro/camoufox) - The anti-detect Firefox browser
- [BrowserForge](https://github.com/daijro/browserforge) - Fingerprint generation
- [playwright-go](https://github.com/playwright-community/playwright-go) - Go bindings for Playwright

## License

MIT License - see [LICENSE](LICENSE) for details.
