# Pagination Node

The `paginate` node type provides generic, automatic pagination support for any website.

## Overview

The pagination node automatically navigates through multiple pages, extracting items from each page. It supports common pagination patterns including Next buttons, page number links, and combined approaches.

## Basic Usage

```json
{
  "id": "paginate_products",
  "type": "paginate",
  "name": "Paginate Through Product Listings",
  "params": {
    "next_selector": "a.next-page",
    "max_pages": 10,
    "item_selector": "div.product a",
    "url_type": "product"
  }
}
```

## Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `next_selector` | string | * | - | CSS selector for "Next" button/link |
| `link_selector` | string | * | - | CSS selector for page number links (1, 2, 3...) |
| `item_selector` | string | Yes | - | CSS selector for items to extract from each page |
| `url_type` | string | Yes | - | Type to assign to discovered URLs |
| `max_pages` | int | No | 100 | Maximum pages to process |
| `type` | string | No | auto | Navigation type: `click`, `link`, or `auto` |
| `wait_after` | int | No | 2000 | Wait time after navigation (milliseconds) |

\* At least one of `next_selector` or `link_selector` is required

## Navigation Types

- **`auto`** (recommended): Automatically detects whether to click or follow href
- **`click`**: Clicks the element (for JavaScript-based pagination)
- **`link`**: Follows the href attribute (for URL-based pagination)

## Common Patterns

### Pattern 1: Next Button Only

```json
{
  "type": "paginate",
  "params": {
    "next_selector": "button.next",
    "max_pages": 20,
    "item_selector": "div.product a",
    "url_type": "product"
  }
}
```

### Pattern 2: Page Number Links

```json
{
  "type": "paginate",
  "params": {
    "link_selector": "div.pagination a.page",
    "max_pages": 15,
    "item_selector": "article a",
    "url_type": "article"
  }
}
```

### Pattern 3: Combined (Next + Page Numbers)

```json
{
  "type": "paginate",
  "params": {
    "next_selector": "a.btn-next",
    "link_selector": "ul.pagination li a",
    "max_pages": 10,
    "item_selector": "div.item a",
    "url_type": "item"
  }
}
```

## Real-World Examples

### E-commerce Site (Columbia Sports)

```json
{
  "id": "paginate_products",
  "type": "paginate",
  "params": {
    "next_selector": "div.direction_.next_ a.btn_",
    "link_selector": "div.pagination_ ul.number_ li a",
    "max_pages": 5,
    "type": "auto",
    "wait_after": 2000,
    "item_selector": "ul.block-thumbnail-t--items a",
    "url_type": "product"
  }
}
```

### Marketplace (Amazon-style)

```json
{
  "id": "paginate_listings",
  "type": "paginate",
  "params": {
    "next_selector": "a.s-pagination-next",
    "link_selector": "a.s-pagination-item",
    "max_pages": 20,
    "wait_after": 3000,
    "item_selector": "div.search-result h2 a",
    "url_type": "product"
  }
}
```

### Blog/News Site

```json
{
  "id": "paginate_articles",
  "type": "paginate",
  "params": {
    "next_selector": "a.older-posts",
    "max_pages": 50,
    "wait_after": 1500,
    "item_selector": "article.post a.post-link",
    "url_type": "article"
  }
}
```

## How It Works

1. **Extract**: Extracts items from current page using `item_selector`
2. **Navigate**: Attempts to navigate to next page:
   - First tries `next_selector` (Next button/link)
   - Falls back to `link_selector` (page number links)
3. **Repeat**: Continues steps 1-2 until:
   - `max_pages` limit reached
   - No more pages available (Next button disabled/hidden)
   - Navigation fails
4. **Enqueue**: All discovered links are enqueued with specified `url_type`

## Best Practices

### 1. Always Set max_pages
Prevents infinite loops on misconfigured sites:
```json
"max_pages": 20  // Stop after 20 pages
```

### 2. Use 'auto' Type
Let the system decide the best navigation method:
```json
"type": "auto"  // Recommended
```

### 3. Adjust wait_after for Site Speed
Slower sites need more wait time:
```json
"wait_after": 3000  // 3 seconds for slow sites
"wait_after": 1000  // 1 second for fast sites
```

### 4. Test with Small max_pages First
Test pagination with 2-3 pages before full crawl:
```json
"max_pages": 2  // Testing
"max_pages": 50  // Production
```

### 5. Combine Multiple Selectors
More robust - falls back if one fails:
```json
{
  "next_selector": "a.next",
  "link_selector": "div.pagination a"
}
```

### 6. Use Specific Selectors
Avoid clicking wrong elements:
```json
// ✅ Good - specific
"next_selector": "div.pagination a.next"

// ❌ Bad - too generic
"next_selector": "a"
```

## Workflow Integration

Pagination nodes work with URL discovery:

```json
{
  "url_discovery": [
    {
      "id": "discover_categories",
      "type": "extract_links",
      "params": {
        "selector": "nav a.category",
        "url_type": "category"
      }
    },
    {
      "id": "paginate_products",
      "type": "paginate",
      "params": {
        "next_selector": "a.next",
        "max_pages": 10,
        "item_selector": "div.product a",
        "url_type": "product"
      },
      "dependencies": ["discover_categories"]
    }
  ]
}
```

This will:
1. Discover category pages
2. For each category, paginate through product listings
3. Extract all product links from all pages

## Troubleshooting

### Pagination Not Working

**Problem**: Pagination stops after first page

**Solutions**:
1. Check selector is correct: `"next_selector": "a.next"`
2. Increase wait time: `"wait_after": 3000`
3. Try different type: `"type": "click"` or `"type": "link"`
4. Check if element is hidden/disabled

### Extracting Wrong Items

**Problem**: Items from wrong sections extracted

**Solutions**:
1. Use more specific selector: `"div.main-content div.product a"`
2. Test selector in browser console first
3. Check for dynamic content loading

### Hitting max_pages Too Soon

**Problem**: Only processes few pages when more exist

**Solutions**:
1. Increase max_pages: `"max_pages": 50`
2. Check if pagination selectors are correct
3. Verify navigation is succeeding

## Limitations

- Maximum 100 pages by default (configurable via `max_pages`)
- Does not support infinite scroll (use `scroll` interaction node instead)
- Does not support JavaScript-generated pagination without proper selectors
- Requires consistent pagination UI across all pages

## See Also

- [extract_links](./EXTRACT_LINKS.md) - Extract links without pagination
- [URL_DISCOVERY_LIMIT](./URL_DISCOVERY_LIMIT.md) - Limit links extracted per node
- [Examples](../examples/columbia_crawler.json) - Full crawler examples
