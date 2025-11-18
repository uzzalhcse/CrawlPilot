# Workflow Examples with URL Hierarchy

This directory contains example workflow configurations that demonstrate the new URL hierarchy tracking feature.

## Overview

With the optimized schema, workflows can now specify `url_type` for discovered URLs, enabling:
- **Full URL hierarchy tracking** - See parent-child relationships
- **Better categorization** - Distinguish between categories, products, pagination, etc.
- **Improved debugging** - Trace which nodes discovered which URLs
- **Enhanced analytics** - Query items by URL type

## URL Types

The following URL types are supported:

| Type | Description | Example Use Case |
|------|-------------|------------------|
| `seed` | Initial starting URLs | Homepage, category landing pages |
| `category` | Category/listing pages | Product categories, section pages |
| `subcategory` | Subcategory pages | Nested categories |
| `product` | Product detail pages | Individual product pages |
| `article` | Article/content pages | Blog posts, news articles |
| `pagination` | Pagination links | Next page, page numbers |
| `page` | Generic page (default) | Any other page type |

## Example Workflows

### 1. E-commerce with Hierarchy (`ecommerce_with_hierarchy.json`)

**Hierarchy Structure:**
```
seed (homepage)
  └── category (book categories)
       ├── product (individual books)
       └── pagination (next pages)
```

**Key Features:**
- Categorizes URLs by type
- Tracks discovery path from category to product
- Handles pagination separately

**Usage:**
```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/ecommerce_with_hierarchy.json
```

### 2. Amazon Bestsellers (`amazon_with_hierarchy.json`)

**Hierarchy Structure:**
```
seed (bestsellers homepage)
  └── category (main categories)
       ├── subcategory (sub-categories)
       │    ├── product (products)
       │    └── pagination
       └── product (featured products)
```

**Key Features:**
- Multi-level category hierarchy
- Distinguishes between main categories and subcategories
- Extracts comprehensive product data
- Handles pagination at category level

**Usage:**
```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/amazon_with_hierarchy.json
```

### 3. News Site Crawler (`news_site_with_hierarchy.json`)

**Hierarchy Structure:**
```
seed (homepage)
  ├── article (news articles)
  └── pagination (more pages)
```

**Key Features:**
- Simple two-level hierarchy
- Article-focused crawling
- Pagination handling

**Usage:**
```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/news_site_with_hierarchy.json
```

## How to Use URL Types in Your Workflows

### Basic Syntax

In any `extract_links` node, add the `url_type` parameter:

```json
{
  "id": "discover_products",
  "type": "extract_links",
  "params": {
    "selector": ".product-link",
    "url_type": "product",
    "filter": {
      "pattern": ".*products.*"
    }
  }
}
```

### Multiple URL Types from Same Page

You can have multiple `extract_links` nodes with different types:

```json
[
  {
    "id": "discover_categories",
    "type": "extract_links",
    "params": {
      "selector": ".category-link",
      "url_type": "category"
    }
  },
  {
    "id": "discover_products",
    "type": "extract_links",
    "params": {
      "selector": ".product-link",
      "url_type": "product"
    }
  },
  {
    "id": "discover_pagination",
    "type": "extract_links",
    "params": {
      "selector": ".next-page",
      "url_type": "pagination"
    }
  }
]
```

## Querying Hierarchical Data

After running a workflow with URL types, you can query the hierarchy:

### Get URL Hierarchy Tree
```bash
curl http://localhost:8080/api/v1/executions/{execution_id}/hierarchy
```

### Get Items with Parent URLs
```bash
curl http://localhost:8080/api/v1/executions/{execution_id}/items-with-hierarchy
```

### SQL Queries

**Find all products under a specific category:**
```sql
WITH RECURSIVE url_tree AS (
    -- Start with a category URL
    SELECT id, url, parent_url_id, url_type, 0 as level
    FROM url_queue 
    WHERE url = 'https://example.com/category/books'
    
    UNION ALL
    
    -- Get all children
    SELECT uq.id, uq.url, uq.parent_url_id, uq.url_type, ut.level + 1
    FROM url_queue uq
    INNER JOIN url_tree ut ON uq.parent_url_id = ut.id
)
SELECT ei.title, ei.price, ut.url
FROM url_tree ut
JOIN extracted_items ei ON ei.url_id = ut.id
WHERE ut.url_type = 'product';
```

**Count items by URL type:**
```sql
SELECT 
    uq.url_type,
    COUNT(ei.id) as item_count,
    AVG(ei.price) as avg_price
FROM url_queue uq
LEFT JOIN extracted_items ei ON ei.url_id = uq.id
WHERE uq.execution_id = '{execution_id}'
GROUP BY uq.url_type;
```

**Find which node discovered the most URLs:**
```sql
SELECT 
    discovered_by_node,
    url_type,
    COUNT(*) as urls_discovered
FROM url_queue
WHERE execution_id = '{execution_id}'
  AND discovered_by_node IS NOT NULL
GROUP BY discovered_by_node, url_type
ORDER BY urls_discovered DESC;
```

## Best Practices

### 1. Always Set URL Type for Discovery Nodes
```json
// ❌ Bad - No url_type
{
  "type": "extract_links",
  "params": {
    "selector": ".link"
  }
}

// ✅ Good - URL type specified
{
  "type": "extract_links",
  "params": {
    "selector": ".product-link",
    "url_type": "product"
  }
}
```

### 2. Use Meaningful URL Types
Choose types that reflect your site structure:
- E-commerce: `category`, `product`, `brand`
- News site: `section`, `article`, `author`
- Documentation: `section`, `page`, `api`

### 3. Handle Pagination Separately
Always tag pagination links with `url_type: "pagination"`:
```json
{
  "id": "discover_pagination",
  "type": "extract_links",
  "params": {
    "selector": ".next-page",
    "url_type": "pagination"
  }
}
```

### 4. Filter URLs Appropriately
Combine `url_type` with URL filters:
```json
{
  "params": {
    "selector": "a",
    "url_type": "product",
    "filter": {
      "pattern": ".*/products/.*",
      "exclude_pattern": ".*reviews.*"
    }
  }
}
```

## Debugging Tips

### View Execution Timeline
```bash
curl http://localhost:8080/api/v1/executions/{id}/timeline
```
Shows which nodes discovered URLs and when.

### Check URL Discovery Stats
```bash
curl http://localhost:8080/api/v1/executions/{id}/performance
```
Shows `urls_discovered` count per node.

### Visualize URL Tree
```bash
curl http://localhost:8080/api/v1/executions/{id}/hierarchy | jq
```
Shows the complete URL hierarchy as a tree.

## Migration from Old Workflows

If you have existing workflows without `url_type`, they will still work! 
The default type is `page`. To add hierarchy tracking:

1. Add `url_type` parameter to each `extract_links` node
2. Choose appropriate types based on your site structure
3. Re-run your workflow
4. Use the new analytics endpoints to view the hierarchy

## See Also

- [Architecture Improvements Documentation](../docs/ARCHITECTURE_IMPROVEMENTS.md)
- [API Analytics Endpoints](../docs/API_ANALYTICS_ENDPOINTS.md)
- [Workflow Guide](../docs/WORKFLOW_GUIDE.md)
