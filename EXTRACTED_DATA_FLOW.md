# Extracted Data Flow Diagram

## Complete Data Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Workflow Execution Starts                       │
│                     (POST /api/v1/workflows/:id/execute)                 │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Executor.ExecuteWorkflow()                       │
│  • Enqueues start URLs                                                  │
│  • Processes URLs from queue                                            │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Executor.processURL()                            │
│  • Acquires browser context                                             │
│  • Navigates to URL                                                     │
│  • Creates ExecutionContext                                             │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    Execute URL Discovery Nodes (Optional)                │
│  • extract_links                                                        │
│  • filter_urls                                                          │
│  • Enqueue discovered URLs                                              │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     Execute Data Extraction Nodes                        │
│                                                                          │
│  Node 1: extract_title                                                  │
│    • selector: "h1"                                                     │
│    • output_key: "title"                                                │
│    • Result stored in: execCtx.Data["title"]                           │
│                                                                          │
│  Node 2: extract_price                                                  │
│    • selector: ".price"                                                 │
│    • output_key: "price"                                                │
│    • Result stored in: execCtx.Data["price"]                           │
│                                                                          │
│  Node 3: extract_description                                            │
│    • selector: "meta[name='description']"                               │
│    • output_key: "description"                                          │
│    • Result stored in: execCtx.Data["description"]                     │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    Executor.collectExtractedData()                       │
│                                                                          │
│  • Calls: execCtx.GetAll()                                              │
│  • Returns all data from ExecutionContext                               │
│  • Filters out internal fields:                                         │
│    ✗ "url"                                                              │
│    ✗ "depth"                                                            │
│  • Result: map[string]interface{}{                                      │
│      "title": "Product Title",                                          │
│      "price": 99.99,                                                    │
│      "description": "Product description"                               │
│    }                                                                    │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     Executor.saveExtractedData()                         │
│                                                                          │
│  • Creates ExtractedData struct:                                        │
│    {                                                                    │
│      ID: <generated UUID>                                               │
│      ExecutionID: "abc-123"                                             │
│      URL: "https://example.com/product/1"                               │
│      Data: JSONMap{...}  ← Converted from map[string]interface{}       │
│      Schema: ""                                                         │
│      ExtractedAt: <current timestamp>                                   │
│    }                                                                    │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              ExtractedDataRepository.Create()                            │
│                                                                          │
│  • JSONMap.Value() called - Marshals data to JSON                       │
│  • SQL Query:                                                           │
│    INSERT INTO extracted_data                                           │
│      (id, execution_id, url, data, schema, extracted_at)               │
│    VALUES ($1, $2, $3, $4, $5, $6)                                     │
│                                                                          │
│  • Data stored as JSONB in PostgreSQL                                   │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    Database (extracted_data table)                       │
│                                                                          │
│  ┌─────────────┬──────────────┬───────────┬──────────────┬──────────┐  │
│  │ id          │ execution_id │ url       │ data (JSONB) │ ext...   │  │
│  ├─────────────┼──────────────┼───────────┼──────────────┼──────────┤  │
│  │ uuid-1      │ abc-123      │ https://..│ {"title":... │ 2024-... │  │
│  │ uuid-2      │ abc-123      │ https://..│ {"title":... │ 2024-... │  │
│  └─────────────┴──────────────┴───────────┴──────────────┴──────────┘  │
└─────────────────────────────────────────────────────────────────────────┘

═══════════════════════════════════════════════════════════════════════════

                            DATA RETRIEVAL FLOW

┌─────────────────────────────────────────────────────────────────────────┐
│                      Client API Request                                  │
│         GET /api/v1/executions/{execution_id}/data?limit=100&offset=0   │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              ExecutionHandler.GetExtractedData()                         │
│                                                                          │
│  • Parse query parameters (limit, offset)                               │
│  • Validate parameters                                                  │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│          ExtractedDataRepository.GetByExecutionID()                      │
│                                                                          │
│  • SQL Query:                                                           │
│    SELECT id, execution_id, url, data, schema, extracted_at            │
│    FROM extracted_data                                                  │
│    WHERE execution_id = $1                                              │
│    ORDER BY extracted_at DESC                                           │
│    LIMIT $2 OFFSET $3                                                   │
│                                                                          │
│  • For each row: JSONMap.Scan() unmarshals JSONB to map                │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│            ExtractedDataRepository.Count()                               │
│                                                                          │
│  • SQL Query:                                                           │
│    SELECT COUNT(*) FROM extracted_data WHERE execution_id = $1         │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         JSON Response                                    │
│                                                                          │
│  {                                                                      │
│    "execution_id": "abc-123",                                           │
│    "data": [                                                            │
│      {                                                                  │
│        "id": "uuid-1",                                                  │
│        "execution_id": "abc-123",                                       │
│        "url": "https://example.com/product/1",                          │
│        "data": {                                                        │
│          "title": "Product Title",                                      │
│          "price": 99.99,                                                │
│          "description": "Product description"                           │
│        },                                                               │
│        "schema": "",                                                    │
│        "extracted_at": "2024-01-15T10:30:00Z"                          │
│      }                                                                  │
│    ],                                                                   │
│    "total": 1,                                                          │
│    "limit": 100,                                                        │
│    "offset": 0                                                          │
│  }                                                                      │
└─────────────────────────────────────────────────────────────────────────┘
```

## Key Components

### 1. JSONMap Type
```go
type JSONMap map[string]interface{}

// Implements sql.Scanner - reads from database
func (jm *JSONMap) Scan(value interface{}) error

// Implements driver.Valuer - writes to database
func (jm JSONMap) Value() (driver.Value, error)
```

### 2. ExecutionContext
```go
type ExecutionContext struct {
    Data      map[string]interface{}  // Stores extracted data
    Variables map[string]string
    Metadata  map[string]interface{}
}

// New method added
func (ec *ExecutionContext) GetAll() map[string]interface{}
```

### 3. ExtractedData Model
```go
type ExtractedData struct {
    ID          string
    ExecutionID string
    URL         string
    Data        JSONMap  // Changed from map[string]interface{}
    Schema      string
    ExtractedAt time.Time
}
```

## Data Transformation

```
Extraction Node Output
        ↓
ExecutionContext.Data["output_key"] = value
        ↓
map[string]interface{} (in memory)
        ↓
JSONMap (type conversion)
        ↓
json.Marshal() via JSONMap.Value()
        ↓
[]byte (JSON string)
        ↓
PostgreSQL JSONB column
        ↓
json.Unmarshal() via JSONMap.Scan()
        ↓
map[string]interface{} (API response)
```

## Error Handling

```
┌──────────────────────────┐
│ Extraction Node Fails    │
└────────────┬─────────────┘
             │
             ▼
      ┌─────────────┐      Yes    ┌──────────────────┐
      │ Optional?   ├─────────────▶│ Log warning,     │
      │             │              │ continue         │
      └──────┬──────┘              └──────────────────┘
             │ No
             ▼
      ┌──────────────────┐
      │ Fail workflow    │
      │ execution        │
      └──────────────────┘

┌──────────────────────────┐
│ Save Data Fails          │
└────────────┬─────────────┘
             │
             ▼
      ┌──────────────────┐
      │ Log error,       │
      │ continue         │
      │ (non-blocking)   │
      └──────────────────┘
```

## Performance Considerations

1. **Batch Processing**: Each URL is processed independently
2. **JSONB Indexing**: PostgreSQL JSONB supports GIN indexes
3. **Pagination**: Limits memory usage for large datasets
4. **Connection Pooling**: Database connections are pooled
5. **Asynchronous Execution**: Workflow runs in background goroutine

## Example Queries

### Get all extracted data
```sql
SELECT * FROM extracted_data 
WHERE execution_id = 'abc-123'
ORDER BY extracted_at DESC;
```

### Query specific field in JSONB
```sql
SELECT url, data->>'title' as title, data->>'price' as price
FROM extracted_data
WHERE execution_id = 'abc-123'
  AND data->>'price' IS NOT NULL;
```

### Filter by JSONB field value
```sql
SELECT * FROM extracted_data
WHERE execution_id = 'abc-123'
  AND (data->>'price')::numeric > 50;
```
