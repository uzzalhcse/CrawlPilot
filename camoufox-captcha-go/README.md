# Camoufox Captcha Go

A Go library for automatically solving CAPTCHA challenges with [playwright-go](https://github.com/playwright-community/playwright-go). Currently supports Cloudflare challenges (interstitial and turnstile).

This is a Go port of the Python [camoufox-captcha](https://github.com/techinz/camoufox-captcha) library.

## Features

- **Closed Shadow DOM Traversal**: Navigates complex closed Shadow DOM structures to find and interact with challenge elements
- **Advanced Retry Logic**: Implements retry mechanisms with configurable attempts and delays
- **Verification Steps**: Confirms successful solving through multiple validation methods
- **Standalone Package**: Works with any playwright-go browser (optimized for Camoufox)

## Requirements

- Go 1.22+
- [playwright-go](https://github.com/playwright-community/playwright-go) v0.5200.1+
- [camoufox-go](https://github.com/uzzalhcse/camoufox-go) (recommended for best compatibility)

## Installation

```bash
go get github.com/uzzalhcse/camoufox-captcha-go
```

## Usage

### ⚠️ Important: Browser Configuration

When using Camoufox, you **must** enable both `DisableCOOP` and `ForceScopeAccess` for proper Shadow DOM access:

```go
browser, err := camoufox.NewBrowser(camoufox.Options{
    DisableCOOP:      true,  // Required: Disables Cross-Origin-Opener-Policy
    ForceScopeAccess: true,  // Required: Enables access to closed shadow roots
})
```

> **Note**: Without `ForceScopeAccess: true`, the solver cannot access closed shadow roots where Cloudflare iframes are located.

### Cloudflare Interstitial Example

```go
package main

import (
    "log"
    "time"

    captcha "github.com/uzzalhcse/camoufox-captcha-go"
    "github.com/uzzalhcse/camoufox-go"
)

func main() {
    browser, err := camoufox.NewBrowser(camoufox.Options{
        DisableCOOP:      true,
        ForceScopeAccess: true,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer browser.Close()

    page, _ := browser.NewPage()
    page.Goto("https://nopecha.com/demo/cloudflare")
    
    // Wait for page to load
    time.Sleep(5 * time.Second)

    // Solve the CAPTCHA
    opts := captcha.DefaultOptions()
    opts.ChallengeType = captcha.ChallengeInterstitial
    opts.Debug = true

    success, err := captcha.SolveCaptchaPage(page, opts)
    if err != nil {
        log.Fatal(err)
    }

    if success {
        log.Println("✅ Successfully solved CAPTCHA!")
    }
}
```

### Cloudflare Turnstile Example

```go
package main

import (
    "log"
    "time"

    captcha "github.com/uzzalhcse/camoufox-captcha-go"
    "github.com/uzzalhcse/camoufox-go"
)

func main() {
    browser, err := camoufox.NewBrowser(camoufox.Options{
        DisableCOOP:      true,
        ForceScopeAccess: true,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer browser.Close()

    page, _ := browser.NewPage()
    page.Goto("https://nopecha.com/demo/turnstile")
    time.Sleep(5 * time.Second)

    // Find the Turnstile container
    container, err := page.WaitForSelector(".cf-turnstile")
    if err != nil {
        log.Fatal("Turnstile container not found")
    }

    // Solve the Turnstile CAPTCHA
    opts := captcha.DefaultOptions()
    opts.ChallengeType = captcha.ChallengeTurnstile
    opts.Debug = true

    success, err := captcha.SolveCaptchaElement(container, opts)
    if err != nil {
        log.Fatal(err)
    }

    if success {
        log.Println("✅ Successfully solved Turnstile!")
    }
}
```

### With Content Verification

```go
opts := captcha.DefaultOptions()
opts.ChallengeType = captcha.ChallengeInterstitial
opts.ExpectedContentSelector = "#main-content" // Verify this appears after solving

success, err := captcha.SolveCaptchaPage(page, opts)
```

## Configuration Options

```go
captcha.SolveOptions{
    CaptchaType:             captcha.CaptchaCloudflare,     // CAPTCHA provider
    ChallengeType:           captcha.ChallengeInterstitial, // or ChallengeTurnstile
    ExpectedContentSelector: "",              // CSS selector to verify success
    SolveAttempts:           3,               // Max attempts to solve
    SolveClickDelay:         6*time.Second,   // Wait after clicking checkbox
    WaitCheckboxAttempts:    10,              // Max attempts to find checkbox
    WaitCheckboxDelay:       6*time.Second,   // Delay between checkbox checks
    CheckboxClickAttempts:   3,               // Max click attempts
    AttemptDelay:            5*time.Second,   // Delay between solve attempts
    Debug:                   false,           // Enable debug logging
}
```

## API Reference

### Functions

| Function | Description |
|----------|-------------|
| `SolveCaptchaPage(page, opts)` | Solve CAPTCHA on a playwright.Page |
| `SolveCaptchaFrame(frame, opts)` | Solve CAPTCHA on a playwright.Frame |
| `SolveCaptchaElement(element, opts)` | Solve CAPTCHA within a playwright.ElementHandle (for Turnstile) |
| `DefaultOptions()` | Returns default SolveOptions |

### Types

| Type | Values |
|------|--------|
| `CaptchaType` | `CaptchaCloudflare` |
| `ChallengeType` | `ChallengeInterstitial`, `ChallengeTurnstile` |

## How It Works

### Cloudflare Interstitial

1. Detects the Cloudflare challenge page through specific DOM elements
2. Finds all iframes in the page's Shadow DOM tree (using `shadowRootUnl` for closed roots)
3. Searches for the checkbox inside security frames
4. Simulates a user click on the verification checkbox
5. Waits for the page to reload or challenge to disappear
6. Verifies success by checking for absence of challenge

### Cloudflare Turnstile

1. Targets the Turnstile widget container element
2. Finds iframes within the element's Shadow DOM
3. Searches for the checkbox inside security frames
4. Simulates a user click on the verification checkbox
5. Monitors for completion by watching for success state elements

## Troubleshooting

### "Cloudflare iframes not found"

Make sure you have enabled **both** options:
```go
camoufox.Options{
    DisableCOOP:      true,
    ForceScopeAccess: true,  // This enables shadowRootUnl
}
```

### Checkbox not clicked

Increase the wait times:
```go
opts := captcha.DefaultOptions()
opts.WaitCheckboxAttempts = 15
opts.WaitCheckboxDelay = 8 * time.Second
```

## Legal Disclaimer

**THIS TOOL IS PROVIDED FOR EDUCATIONAL AND RESEARCH PURPOSES ONLY**

Using this tool against websites without authorization may violate terms of service and applicable laws. Users are solely responsible for ensuring their use complies with all applicable laws and regulations.

## License

MIT License
