# Analytics & Visualization API Endpoints

Quick reference for the new analytics endpoints added to improve debugging and visualization.

## Base URL
```
http://localhost:8080/api/v1
```

## Endpoints Overview

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/executions/:id/timeline` | GET | Complete execution timeline |
| `/executions/:id/hierarchy` | GET | URL tree structure |
| `/executions/:id/performance` | GET | Performance metrics |
| `/executions/:id/items-with-hierarchy` | GET | Items with URL context |
| `/executions/:id/bottlenecks` | GET | Slow operations |

---

## 1. Execution Timeline

**Endpoint:** `GET /executions/:executionId/timeline`

**Purpose:** View complete execution lifecycle with all node executions in chronological order.

**Example Request:**
```bash
curl http://localhost:8080/api/v1/executions/abc-123/timeline
```

**Example Response:**
```json
{
  "execution_id": "abc-123",
  "timeline": [
    {
      "timestamp": "2024-01-15T10:30:00Z",
      "node_name": "navigate_home",
      "node_type": "navigate",
      "status": "completed",
      "duration_ms": 450,
      "url": "https://example.com",
      "url_type": "seed",
      "urls_discovered": 0,
      "items_extracted": 0
    },
    {
      "timestamp": "2024-01-15T10:30:01Z",
      "node_name": "discover_categories",
      "node_type": "discover_urls",
      "status": "completed",
      "duration_ms": 320,
      "url": "https://example.com",
      "url_type": "seed",
      "urls_discovered": 15,
      "items_extracted": 0
    },
    {
      "timestamp": "2024-01-15T10:30:05Z",
      "node_name": "extract_products",
      "node_type": "extract_items",
      "status": "completed",
      "duration_ms": 1250,
      "url": "https://example.com/product/123",
      "url_type": "product",
      "urls_discovered": 0,
      "items_extracted": 1
    }
  ],
  "summary": {
    "total_nodes": 150,
    "completed_nodes": 145,
    "failed_nodes": 5,
    "total_urls_discovered": 500,
    "total_items_extracted": 1200,
    "average_duration_ms": 856.3
  }
}
```

**Use Cases:**
- Debug workflow execution flow
- See what each node accomplished
- Identify when failures occurred
- Understand execution sequence

---

## 2. URL Hierarchy

**Endpoint:** `GET /executions/:executionId/hierarchy`

**Purpose:** View URL tree structure showing parent-child relationships.

**Example Request:**
```bash
curl http://localhost:8080/api/v1/executions/abc-123/hierarchy
```

**Example Response:**
```json
{
  "execution_id": "abc-123",
  "tree": [
    {
      "id": "url-001",
      "url": "https://example.com",
      "url_type": "seed",
      "depth": 0,
      "status": "completed",
      "items_extracted": 0,
      "children": [
        {
          "id": "url-002",
          "url": "https://example.com/category/books",
          "url_type": "category",
          "depth": 1,
          "status": "completed",
          "discovered_by_node": "discover_categories",
          "items_extracted": 0,
          "children": [
            {
              "id": "url-003",
              "url": "https://example.com/product/123",
              "url_type": "product",
              "depth": 2,
              "status": "completed",
              "discovered_by_node": "discover_products",
              "items_extracted": 1
            }
          ]
        }
      ]
    }
  ],
  "stats": {
    "total_urls": 500,
    "max_depth": 3,
    "urls_by_type": {
      "seed": 1,
      "category": 15,
      "product": 484
    },
    "urls_by_status": {
      "completed": 480,
      "pending": 15,
      "failed": 5
    }
  }
}
```

**Use Cases:**
- Visualize crawl tree
- Debug URL discovery flow
- See which nodes discovered which URLs
- Understand crawl breadth and depth
- Track failed URLs in context

---

## 3. Performance Metrics

**Endpoint:** `GET /executions/:executionId/performance`

**Purpose:** Analyze performance by node type and identify optimization opportunities.

**Example Request:**
```bash
curl http://localhost:8080/api/v1/executions/abc-123/performance
```

**Example Response:**
```json
{
  "execution_id": "abc-123",
  "node_metrics": [
    {
      "node_name": "extract_products",
      "node_type": "extract_items",
      "executions": 484,
      "avg_duration_ms": 1250.5,
      "total_urls_discovered": 0,
      "total_items_extracted": 1200,
      "failures": 5,
      "success_rate": 98.97
    },
    {
      "node_name": "discover_pagination",
      "node_type": "discover_urls",
      "executions": 50,
      "avg_duration_ms": 450.2,
      "total_urls_discovered": 450,
      "total_items_extracted": 0,
      "failures": 0,
      "success_rate": 100.0
    },
    {
      "node_name": "navigate_home",
      "node_type": "navigate",
      "executions": 1,
      "avg_duration_ms": 450.0,
      "total_urls_discovered": 0,
      "total_items_extracted": 0,
      "failures": 0,
      "success_rate": 100.0
    }
  ],
  "total_duration_ms": 612500,
  "url_processing_rate": 0.81
}
```

**Use Cases:**
- Identify slow nodes
- Compare node performance
- Track success rates
- Optimize workflow configuration
- Calculate total execution time

---

## 4. Items with Hierarchy

**Endpoint:** `GET /executions/:executionId/items-with-hierarchy`

**Purpose:** Get extracted items with their URL hierarchy context.

**Example Request:**
```bash
curl http://localhost:8080/api/v1/executions/abc-123/items-with-hierarchy
```

**Example Response:**
```json
{
  "execution_id": "abc-123",
  "items": [
    {
      "id": "item-001",
      "execution_id": "abc-123",
      "url_id": "url-003",
      "item_type": "product",
      "schema_name": "product_schema",
      "title": "Python Programming Book",
      "price": 29.99,
      "currency": "USD",
      "availability": "In Stock",
      "rating": 4.5,
      "review_count": 256,
      "attributes": {
        "author": "John Doe",
        "isbn": "978-1234567890",
        "pages": 450
      },
      "extracted_at": "2024-01-15T10:30:05Z",
      "url": "https://example.com/product/123",
      "url_type": "product",
      "url_level": 2,
      "parent_url": "https://example.com/category/books",
      "parent_url_type": "category"
    }
  ],
  "count": 1200
}
```

**Use Cases:**
- Trace items back to their source
- Understand item discovery path
- Group items by category URL
- Quality assurance
- Data export with context

---

## 5. Bottlenecks

**Endpoint:** `GET /executions/:executionId/bottlenecks`

**Purpose:** Identify slow operations exceeding 5 seconds threshold.

**Example Request:**
```bash
curl http://localhost:8080/api/v1/executions/abc-123/bottlenecks
```

**Example Response:**
```json
{
  "execution_id": "abc-123",
  "bottlenecks": [
    {
      "node_execution_id": "ne-001",
      "node_name": "extract_products",
      "node_type": "extract_items",
      "url": "https://example.com/slow-page",
      "duration_ms": 15000,
      "status": "completed",
      "error_message": null
    },
    {
      "node_execution_id": "ne-045",
      "node_name": "navigate_category",
      "node_type": "navigate",
      "url": "https://example.com/heavy-page",
      "duration_ms": 8500,
      "status": "completed",
      "error_message": null
    },
    {
      "node_execution_id": "ne-089",
      "node_name": "extract_products",
      "node_type": "extract_items",
      "url": "https://example.com/timeout-page",
      "duration_ms": 30000,
      "status": "failed",
      "error_message": "Timeout exceeded"
    }
  ],
  "count": 12
}
```

**Use Cases:**
- Find slow pages
- Identify timeout issues
- Optimize extraction selectors
- Set appropriate timeouts
- Performance tuning

---

## Testing with curl

### Get Timeline
```bash
curl -X GET http://localhost:8080/api/v1/executions/YOUR_EXECUTION_ID/timeline | jq
```

### Get Hierarchy
```bash
curl -X GET http://localhost:8080/api/v1/executions/YOUR_EXECUTION_ID/hierarchy | jq
```

### Get Performance Metrics
```bash
curl -X GET http://localhost:8080/api/v1/executions/YOUR_EXECUTION_ID/performance | jq
```

### Get Items with Hierarchy
```bash
curl -X GET http://localhost:8080/api/v1/executions/YOUR_EXECUTION_ID/items-with-hierarchy | jq
```

### Get Bottlenecks
```bash
curl -X GET http://localhost:8080/api/v1/executions/YOUR_EXECUTION_ID/bottlenecks | jq
```

---

## Error Responses

All endpoints return standard error responses:

```json
{
  "error": "Failed to get node executions: <error details>"
}
```

**Status Codes:**
- `200 OK` - Success
- `500 Internal Server Error` - Database or processing error

---

## Integration Example

### JavaScript/Frontend
```javascript
async function getExecutionTimeline(executionId) {
  const response = await fetch(
    `http://localhost:8080/api/v1/executions/${executionId}/timeline`
  );
  const data = await response.json();
  
  // Display timeline
  data.timeline.forEach(entry => {
    console.log(`${entry.timestamp}: ${entry.node_name} - ${entry.status}`);
  });
  
  // Show summary
  console.log(`Total nodes: ${data.summary.total_nodes}`);
  console.log(`Completed: ${data.summary.completed_nodes}`);
  console.log(`Failed: ${data.summary.failed_nodes}`);
}
```

### Python
```python
import requests

def get_performance_metrics(execution_id):
    url = f"http://localhost:8080/api/v1/executions/{execution_id}/performance"
    response = requests.get(url)
    data = response.json()
    
    # Analyze metrics
    for metric in data['node_metrics']:
        print(f"Node: {metric['node_name']}")
        print(f"  Avg Duration: {metric['avg_duration_ms']}ms")
        print(f"  Success Rate: {metric['success_rate']}%")
        print()
```

---

## Notes

- All timestamps are in ISO 8601 format (UTC)
- Duration is always in milliseconds
- Replace `:executionId` with actual execution ID from workflow execution
- Use `jq` for pretty JSON formatting in terminal
- Endpoints return empty arrays/objects if no data found

---

## See Also

- [Architecture Improvements Documentation](./ARCHITECTURE_IMPROVEMENTS.md)
- [Main API Documentation](./API.md)
- [Workflow Guide](./WORKFLOW_GUIDE.md)
