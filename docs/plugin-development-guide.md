# Plugin Development Guide

## Overview

Crawlify's plugin system allows you to create compiled Go plugins that extend the crawler's discovery and extraction capabilities. Plugins are compiled as shared libraries (`.so` files) and dynamically loaded at runtime.

## Quick Start

### 1. Create Plugin Project

Use the plugin template:

```bash
cp -r examples/plugin-template my-plugin
cd my-plugin
```

### 2. Implement Plugin Interface

Choose either `DiscoveryPlugin` or `ExtractionPlugin` interface:

**Discovery Plugin** (for finding URLs):
```go
type DiscoveryPlugin interface {
    Info() PluginInfo
    Discover(ctx context.Context, input *DiscoveryInput) (*DiscoveryOutput, error)
    Validate(config map[string]interface{}) error
    ConfigSchema() map[string]interface{}
}
```

**Extraction Plugin** (for extracting data):
```go
type ExtractionPlugin interface {
    Info() PluginInfo
    Extract(ctx context.Context, input *ExtractionInput) (*ExtractionOutput, error)
    Validate(config map[string]interface{}) error
    ConfigSchema() map[string]interface{}
}
```

### 3. Build Plugin

```bash
go build -buildmode=plugin -o myplugin.so
```

### 4. Test Plugin

Load and test your plugin:
```bash
# Add plugin to Crawlify plugins directory
cp myplugin.so /path/to/crawlify/plugins/

# Plugin will be auto-loaded on startup
```

## Plugin Structure

### Required Exports

Your plugin MUST export a constructor function:

**For Discovery Plugin**:
```go
func NewDiscoveryPlugin() plugins.DiscoveryPlugin {
    return &MyDiscoveryPlugin{}
}
```

**For Extraction Plugin**:
```go
func NewExtractionPlugin() plugins.ExtractionPlugin {
    return &MyExtractionPlugin{}
}
```

### Plugin Info

Provide metadata about your plugin:

```go
func (p *MyPlugin) Info() plugins.PluginInfo {
    return plugins.PluginInfo{
        ID:          "my-ecommerce-plugin",
        Name:        "My E-commerce Plugin",
        Version:     "1.0.0",
        Author:      "Your Name",
        AuthorEmail: "you@example.com",
        Description: "Discovers products on e-commerce sites",
        PhaseType:   models.PhaseTypeDiscovery,
        Repository:  "https://github.com/you/my-plugin",
        License:     "MIT",
    }
}
```

## Using the SDK

The Plugin SDK provides helper utilities to simplify development:

### Browser Helpers

```go
sdk := plugins.NewSDK(logger)
browserHelpers := sdk.NewBrowserHelpers(input.BrowserContext)

// Wait for element
browserHelpers.WaitForSelector(".product-list", 5000)

// Extract links
links, err := browserHelpers.ExtractLinks("a.product-link")

// Get text
title, err := browserHelpers.GetText("h1.product-title")

// Click element
browserHelpers.Click("button.load-more")

// Scroll to bottom
browserHelpers.ScrollToBottom(2000)
```

### URL Helpers

```go
urlHelpers := sdk.NewURLHelpers()

// Normalize URLs
normalizedURL, _ := urlHelpers.NormalizeURL(rawURL)

// Join URLs
absoluteURL, _ := urlHelpers.JoinURL(baseURL, relativePath)

// Check if absolute
if urlHelpers.IsAbsoluteURL(url) {
    // ...
}

// Pattern matching
matches, _ := urlHelpers.MatchesPattern(url, `product/(\d+)`)
```

### Data Helpers

```go
dataHelpers := sdk.NewDataHelpers()

// Clean text
cleanText := dataHelpers.CleanText(rawText)

// Extract numbers
prices := dataHelpers.ExtractNumbers("Price: $19.99")

// Parse HTML
doc, _ := dataHelpers.ParseHTML(htmlString)
values := dataHelpers.ExtractWithCSS(doc, ".price")
```

### Config Helpers

```go
configHelpers := sdk.NewConfigHelpers()

// Get config values safely
selector := configHelpers.GetString(config, "selector", ".default")
maxPages := configHelpers.GetInt(config, "max_pages", 10)
enabled := configHelpers.GetBool(config, "enabled", true)

// Require config values
apiKey, err := configHelpers.RequireString(config, "api_key")
```

### Logging

```go
logger := sdk.NewLogger("my-plugin")

logger.Info("Starting discovery", 
    zap.String("url", input.URL))
    
logger.Error("Failed to extract", err,
    zap.String("selector", selector))

logger.Debug("Found links", 
    zap.Int("count", len(links)))
```

## Example: Discovery Plugin

```go
package main

import (
    "context"
    "github.com/uzzalhcse/crawlify/pkg/models"
    "github.com/uzzalhcse/crawlify/pkg/plugins"
)

type EcommerceDiscovery struct {
    sdk *plugins.SDK
}

func NewDiscoveryPlugin() plugins.DiscoveryPlugin {
    return &EcommerceDiscovery{
        sdk: plugins.NewSDK(nil),
    }
}

func (p *EcommerceDiscovery) Info() plugins.PluginInfo {
    return plugins.PluginInfo{
        ID:          "ecommerce-discovery",
        Name:        "E-commerce Product Discovery",
        Version:     "1.0.0",
        Author:      "Crawlify Team",
        Description: "Discovers product URLs from e-commerce sites",
        PhaseType:   models.PhaseTypeDiscovery,
    }
}

func (p *EcommerceDiscovery) Discover(ctx context.Context, input *plugins.DiscoveryInput) (*plugins.DiscoveryOutput, error) {
    // Initialize helpers
    browserHelpers := p.sdk.NewBrowserHelpers(input.BrowserContext)
    configHelpers := p.sdk.NewConfigHelpers()
    logger := p.sdk.NewLogger("ecommerce-discovery")
    
    // Get configuration
    productSelector := configHelpers.GetString(input.Config, "product_selector", "a.product")
    maxProducts := configHelpers.GetInt(input.Config, "max_products", 100)
    
    logger.Info("Discovering products", 
        zap.String("url", input.URL),
        zap.String("selector", productSelector))
    
    // Extract product links
    links, err := browserHelpers.ExtractLinks(productSelector)
    if err != nil {
        return nil, err
    }
    
    // Limit results
    if len(links) > maxProducts {
        links = links[:maxProducts]
    }
    
    logger.Info("Discovered products", zap.Int("count", len(links)))
    
    return &plugins.DiscoveryOutput{
        DiscoveredURLs: links,
        Metadata: map[string]interface{}{
            "product_count": len(links),
        },
    }, nil
}

func (p *EcommerceDiscovery) Validate(config map[string]interface{}) error {
    // Validate configuration
    return nil
}

func (p *EcommerceDiscovery) ConfigSchema() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "product_selector": map[string]interface{}{
                "type": "string",
                "description": "CSS selector for product links",
                "default": "a.product",
            },
            "max_products": map[string]interface{}{
                "type": "integer",
                "description": "Maximum number of products to discover",
                "default": 100,
            },
        },
    }
}
```

## Example: Extraction Plugin

```go
package main

import (
    "context"
    "github.com/uzzalhcse/crawlify/pkg/models"
    "github.com/uzzalhcse/crawlify/pkg/plugins"
)

type ProductExtractor struct {
    sdk *plugins.SDK
}

func NewExtractionPlugin() plugins.ExtractionPlugin {
    return &ProductExtractor{
        sdk: plugins.NewSDK(nil),
    }
}

func (p *ProductExtractor) Info() plugins.PluginInfo {
    return plugins.PluginInfo{
        ID:          "product-extractor",
        Name:        "Product Data Extractor",
        Version:     "1.0.0",
        Author:      "Crawlify Team",
        Description: "Extracts product data from e-commerce pages",
        PhaseType:   models.PhaseTypeExtraction,
    }
}

func (p *ProductExtractor) Extract(ctx context.Context, input *plugins.ExtractionInput) (*plugins.ExtractionOutput, error) {
    browserHelpers := p.sdk.NewBrowserHelpers(input.BrowserContext)
    dataHelpers := p.sdk.NewDataHelpers()
    logger := p.sdk.NewLogger("product-extractor")
    
    logger.Info("Extracting product data", zap.String("url", input.URL))
    
    // Extract product data
    name, _ := browserHelpers.GetText("h1.product-name")
    priceText, _ := browserHelpers.GetText(".price")
    description, _ := browserHelpers.GetText(".description")
    
    // Clean data
    name = dataHelpers.CleanText(name)
    description = dataHelpers.CleanText(description)
    
    // Extract price number
    prices := dataHelpers.ExtractNumbers(priceText)
    var price string
    if len(prices) > 0 {
        price = prices[0]
    }
    
    data := map[string]interface{}{
        "url":         input.URL,
        "name":        name,
        "price":       price,
        "description": description,
    }
    
    return &plugins.ExtractionOutput{
        Data:       data,
        SchemaName: "product",
        Metadata: map[string]interface{}{
            "extracted_fields": 4,
        },
    }, nil
}

func (p *ProductExtractor) Validate(config map[string]interface{}) error {
    return nil
}

func (p *ProductExtractor) ConfigSchema() map[string]interface{}{
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{},
    }
}
```

## Building for Multiple Platforms

Build for different platforms:

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o myplugin-linux-amd64.so

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -buildmode=plugin -o myplugin-linux-arm64.so

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -buildmode=plugin -o myplugin-darwin-amd64.so

# macOS ARM64 (M1/M2)
GOOS=darwin GOARCH=arm64 go build -buildmode=plugin -o myplugin-darwin-arm64.so
```

## Best Practices

1. **Error Handling**: Always handle errors gracefully
2. **Logging**: Use structured logging for debugging
3. **Configuration**: Provide sensible defaults
4. **Validation**: Validate all user inputs
5. **Timeouts**: Respect context timeouts
6. **Resource Cleanup**: Clean up resources properly
7. **Testing**: Write unit tests for your plugin
8. **Documentation**: Document configuration options

## Publishing to Marketplace

1. Build for all platforms
2. Generate SHA-256 hashes
3. Create changelog
4. Upload via API or UI
5. Add documentation and examples
6. Request verification (optional)

## Troubleshooting

### Plugin not loading

- Check Go version compatibility
- Verify plugin exports correct symbol
- Check for import conflicts

### Panic in plugin

- Use defer/recover in critical sections
- Plugin executor already handles panics

### Performance issues

- Use timeouts
- Limit resource usage
- Profile your plugin

## Support

- Documentation: https://crawlify.io/docs/plugins
- Examples: https://github.com/crawlify/plugins
- Community: https://discord.gg/crawlify
