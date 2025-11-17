# Extracted Data Implementation

## Overview
This document describes the implementation of the extracted data storage functionality in the Crawlify web crawler system.

## What Was Implemented

### 1. Data Model Enhancement (`pkg/models/queue.go`)
- **Added `JSONMap` type**: A custom type that implements `sql.Scanner` and `driver.Valuer` interfaces for proper JSON serialization/deserialization with PostgreSQL JSONB fields.
- **Updated `ExtractedData` model**: Changed the `Data` field from `map[string]interface{}` to `JSONMap` to support database operations.

### 2. Execution Context Enhancement (`pkg/models/execution.go`)
- **Added `GetAll()` method**: Returns all data stored in the execution context, enabling the collection of extracted data from workflow nodes.

### 3. Workflow Executor (`internal/workflow/executor.go`)
- **Integrated extracted data repository**: The executor now has access to the `ExtractedDataRepository` for saving extracted data.
- **`collectExtractedData()` method**: Collects all extracted data from the execution context, filtering out internal fields like `url` and `depth`.
- **`saveExtractedData()` method**: Saves extracted data to the database using the repository, converting the data to `JSONMap` format.
- **Automatic data persistence**: After executing data extraction nodes, the workflow executor automatically saves any extracted data to the database.

### 4. API Handler Enhancement (`api/handlers/execution.go`)
- **Added `GetExtractedData()` handler**: New API endpoint to retrieve extracted data for a specific execution.
- **Features**:
  - Pagination support (limit and offset query parameters)
  - Total count of extracted records
  - Input validation for pagination parameters
  - Error handling and logging

### 5. Main Application Setup (`cmd/crawler/main.go`)
- **Initialized `ExtractedDataRepository`**: Added repository initialization in the main application.
- **Updated handler initialization**: Passed the extracted data repository to the execution handler.
- **Added API route**: Registered the new `/api/v1/executions/:execution_id/data` endpoint.

### 6. Repository Implementation (`internal/storage/extracted_data_repository.go`)
The repository was already implemented with the following methods:
- `Create()`: Saves a new extracted data record
- `GetByExecutionID()`: Retrieves extracted data with pagination
- `Count()`: Gets the total count of extracted records for an execution

## Database Schema

The `extracted_data` table structure:
```sql
CREATE TABLE IF NOT EXISTS extracted_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    data JSONB NOT NULL,
    schema VARCHAR(255),
    extracted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

## API Endpoint

### GET `/api/v1/executions/:execution_id/data`
Retrieve extracted data for a specific workflow execution.

**Query Parameters:**
- `limit` (optional, default: 100, max: 1000): Number of records to return
- `offset` (optional, default: 0): Number of records to skip

**Response:**
```json
{
  "execution_id": "uuid",
  "data": [
    {
      "id": "uuid",
      "execution_id": "uuid",
      "url": "https://example.com/page",
      "data": {
        "title": "Page Title",
        "description": "Page description",
        "content": "Page content"
      },
      "schema": "product",
      "extracted_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 100,
  "limit": 100,
  "offset": 0
}
```

## How It Works

### Data Extraction Flow

1. **Workflow Execution**: A workflow is started with configured data extraction nodes.

2. **Node Execution**: Each extraction node executes and stores results in the execution context:
   ```yaml
   data_extraction:
     - id: "extract_title"
       type: "extract"
       params:
         selector: "h1"
         type: "text"
       output_key: "title"  # Stored in context
   ```

3. **Data Collection**: After all data extraction nodes complete, the executor calls `collectExtractedData()` which:
   - Retrieves all data from the execution context
   - Filters out internal fields (url, depth)
   - Returns a map of extracted field values

4. **Data Persistence**: The executor calls `saveExtractedData()` which:
   - Creates an `ExtractedData` record with the collected data
   - Converts the data map to `JSONMap` type
   - Saves it to the database via the repository

5. **Data Retrieval**: Clients can fetch the extracted data via the API endpoint with pagination support.

## Example Usage

### Workflow Configuration
```yaml
data_extraction:
  - id: "extract_title"
    type: "extract"
    name: "Extract product title"
    params:
      selector: ".product-title"
      type: "text"
    output_key: "title"

  - id: "extract_price"
    type: "extract"
    name: "Extract product price"
    params:
      selector: ".price"
      type: "text"
      transform:
        - type: "trim"
        - type: "parse_float"
    output_key: "price"
```

### Retrieving Extracted Data
```bash
# Get first 100 records
curl http://localhost:8080/api/v1/executions/{execution_id}/data

# Get records with pagination
curl http://localhost:8080/api/v1/executions/{execution_id}/data?limit=50&offset=100
```

## Key Features

1. **Type Safety**: `JSONMap` type ensures proper serialization/deserialization
2. **Automatic Persistence**: No manual intervention needed - data is automatically saved
3. **Flexible Schema**: JSONB storage allows any data structure
4. **Pagination**: Efficient retrieval of large datasets
5. **URL Tracking**: Each extracted data record is linked to the source URL
6. **Execution Linking**: All data is tied to a specific workflow execution
7. **Timestamp Tracking**: Extraction time is automatically recorded

## Technical Details

### JSONMap Implementation
The `JSONMap` type implements two critical interfaces:
- `sql.Scanner`: Handles reading JSONB data from PostgreSQL
- `driver.Valuer`: Handles writing JSONB data to PostgreSQL

This ensures seamless integration with the database layer and proper JSON handling.

### Error Handling
- Repository errors are logged but don't fail the workflow execution
- Extraction continues even if data saving fails
- API endpoints return appropriate error codes and messages

## Benefits

1. **Decoupled Storage**: Extraction logic is separate from storage logic
2. **Scalable**: JSONB indexing and pagination support large datasets
3. **Flexible**: Schema-less storage accommodates varying data structures
4. **Traceable**: Complete audit trail with URLs and timestamps
5. **Queryable**: PostgreSQL JSONB supports complex queries on extracted data

## Future Enhancements

Potential improvements:
- Add filtering by URL pattern in the API
- Support exporting data in different formats (CSV, JSON, XML)
- Add webhook notifications when data is extracted
- Implement data transformation pipelines
- Add schema validation for extracted data
- Support for custom storage backends (S3, files, etc.)
