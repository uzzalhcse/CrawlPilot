# My Discovery Plugin

A template plugin for Crawlify's discovery phase.

## Description

This plugin demonstrates how to create a discovery phase plugin that finds and extracts links from web pages.

## Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `link_selector` | string | `"a"` | CSS selector for links to discover |
| `max_links` | integer | `100` | Maximum number of links to discover |

## Building

```bash
# Build for Linux AMD64
GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o my-discovery-plugin-linux-amd64.so

# Build for Linux ARM64
GOOS=linux GOARCH=arm64 go build -buildmode=plugin -o my-discovery-plugin-linux-arm64.so

# Build for macOS AMD64
GOOS=darwin GOARCH=amd64 go build -buildmode=plugin -o my-discovery-plugin-darwin-amd64.so

# Build for macOS ARM64 (M1/M2)
GOOS=darwin GOARCH=arm64 go build -buildmode=plugin -o my-discovery-plugin-darwin-arm64.so
```

## Testing

```bash
go test -v
```

## Usage Example

```json
{
  "id": "discover_links",
  "type": "plugin_my-discovery-plugin",
  "name": "Discover Product Links",
  "params": {
    "link_selector": "a.product-link",
    "max_links": 50
  }
}
```

## License

MIT
