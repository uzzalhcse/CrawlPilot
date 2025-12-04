# Columbia Product Discovery Plugin

A Crawlify discovery plugin for extracting product links from Columbia Sportswear category pages.

## Features

- Extracts product links from category pages
- Converts relative URLs to absolute URLs
- Respects configurable link limits
- Provides metadata about discovery results

## Building

```bash
cd examples/plugins/columbia-product-discovery
GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o columbia-product-discovery.so
cp columbia-product-discovery.so ../../../plugins/
```

Or use the Makefile from project root:
```bash
make build
```

## Usage in Workflow

```json
{
  "id": "discover_products",
  "type": "plugin",
  "dependencies": ["navigate_to_category"],
  "params": {
    "plugin_slug": "columbia-product-discovery",
    "config": {
      "limit": 10
    }
  }
}
```

## Configuration

- `limit` (optional): Maximum number of product links to extract per page

## Selectors Used

- Product links: `div.product-tile a.name-link`

## Output

Returns discovered product URLs that can be used in subsequent extraction phases.
