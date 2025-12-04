# E-commerce Product Discovery Plugin

Discovers product URLs from e-commerce websites with automatic pagination support.

## Features

- ✅ CSS selector-based product link extraction
- ✅ Automatic pagination through multiple pages
- ✅ URL pattern filtering with regex support
- ✅ Configurable limits for products and pages
- ✅ Automatic URL normalization and deduplication
- ✅ Marker assignment for discovered products

## Configuration

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `product_selector` | string | ✅ Yes | - | CSS selector for product links |
| `next_page_selector` | string | No | `"a.next-page"` | CSS selector for next page button |
| `max_products` | integer | No | `100` | Maximum products to discover (1-10000) |
| `max_pages` | integer | No | `5` | Maximum pages to process (1-100) |
| `wait_after_click` | integer | No | `2000` | Wait time after clicking next page (ms) |
| `url_pattern` | string | No | `""` | Regex pattern to filter product URLs |

## Example Usage

### Basic Configuration

```json
{
  "id": "discover_products",
  "type": "plugin_ecommerce-product-discovery",
  "name": "Discover E-commerce Products",
  "params": {
    "product_selector": "a.product-link",
    "max_products": 50
  }
}
```

### Advanced Configuration with Pagination

```json
{
  "id": "discover_products",
  "type": "plugin_ecommerce-product-discovery",
  "name": "Discover E-commerce Products",
  "params": {
    "product_selector": ".product-grid a.product",
    "next_page_selector": "a.pagination-next",
    "max_products": 200,
    "max_pages": 10,
    "wait_after_click": 3000,
    "url_pattern": "/product/\\d+"
  }
}
```

### URL Pattern Filtering

```json
{
  "id": "discover_products",
  "type": "plugin_ecommerce-product-discovery",
  "name": "Discover Only Product Detail Pages",
  "params": {
    "product_selector": "a",
    "url_pattern": "/(product|item)/[a-z0-9-]+"
  }
}
```

## Building

```bash
# For Linux AMD64
GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o ecommerce-discovery-linux-amd64.so

# For macOS ARM64 (M1/M2)
GOOS=darwin GOARCH=arm64 go build -buildmode=plugin -o ecommerce-discovery-darwin-arm64.so
```

## How It Works

1. **Page Load**: Waits for product selector to appear on the page
2. **Link Extraction**: Extracts all links matching the product selector  
3. **URL Processing**:
   - Converts relative URLs to absolute
   - Normalizes URLs (removes fragments, etc.)
   - Applies pattern filter if specified
   - Deduplicates URLs
4. **Pagination**: Clicks next page button if available and limit not reached
5. **Repeat**: Continues until max products/pages reached or no more pages

## Workflow Integration

This plugin can be used in a discovery phase to find product URLs:

```json
{
  "phases": [
    {
      "id": "discover_products",
      "type": "discovery",
      "name": "Discover Product URLs",
      "nodes": [
        {
          "id": "navigate_home",
          "type": "navigate",
          "name": "Go to Homepage"
        },
        {
          "id": "find_products",
          "type": "plugin_ecommerce-product-discovery",
          "name": "Discover Products",
          "params": {
            "product_selector": "a.product",
            "max_products": 100
          }
        }
      ],
      "transition": {
        "condition": "all_nodes_complete",
        "next_phase": "extract_product_data"
      }
    }
  ]
}
```

## Common E-commerce Platforms

### Shopify

```json
{
  "product_selector": "a.product-item__title",
  "next_page_selector": "a.pagination__next"
}
```

### WooCommerce

```json
{
  "product_selector": ".products a.woocommerce-LoopProduct-link",
  "next_page_selector": "a.next.page-numbers"
}
```

### Magento

```json
{
  "product_selector": ".product-item-link",
  "next_page_selector": "a.action.next"
}
```

## Troubleshooting

### No products found

- Check if `product_selector` matches the actual HTML structure
- Try using browser DevTools to find the correct selector
- Ensure page has fully loaded before plugin executes

### Pagination not working

- Verify `next_page_selector` matches the next button
- Check `wait_after_click` is sufficient for page load
- Some sites use infinite scroll instead of pagination buttons

### URL pattern not filtering correctly

- Test regex pattern at https://regex101.com
- Remember to escape special characters with `\\`
- Pattern matches against full absolute URL

## License

MIT
