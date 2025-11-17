# API Documentation

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

Currently, the API does not require authentication. For production use, implement JWT or API key authentication.

## Workflows

### Create Workflow

Create a new workflow definition.

**Endpoint**: `POST /workflows`

**Request Body**:
```json
{
  "name": "My Crawler",
  "description": "Crawls example.com",
  "config": {
    "start_urls": ["https://example.com"],
    "max_depth": 2,
    "max_pages": 100,
    "rate_limit_delay": 1000,
    "url_discovery": [
      {
        "id": "extract_links",
        "type": "extract_links",
        "name": "Extract all links",
        "params": {
          "selector": "a[href]"
        },
        "output_key": "links"
      }
    ],
    "data_extraction": [
      {
        "id": "extract_title",
        "type": "extract",
        "name": "Extract page title",
        "params": {
          "selector": "h1",
          "type": "text",
          "transform": [
            {"type": "trim"}
          ]
        },
        "output_key": "title"
      }
    ],
    "storage": {
      "type": "database",
      "table_name": "crawled_pages"
    }
  }
}
```

**Response**: `201 Created`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "My Crawler",
  "description": "Crawls example.com",
  "config": {...},
  "status": "draft",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### List Workflows

List all workflows with optional filtering.

**Endpoint**: `GET /workflows`

**Query Parameters**:
- `status` (optional): Filter by status (draft, active, paused, archived)
- `limit` (optional): Maximum number of results (default: 50)
- `offset` (optional): Offset for pagination (default: 0)

**Example**:
```
GET /workflows?status=active&limit=10&offset=0
```

**Response**: `200 OK`
```json
{
  "workflows": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "My Crawler",
      "description": "Crawls example.com",
      "status": "active",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "count": 1
}
```

### Get Workflow

Retrieve a specific workflow by ID.

**Endpoint**: `GET /workflows/:id`

**Response**: `200 OK`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "My Crawler",
  "description": "Crawls example.com",
  "config": {...},
  "status": "active",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### Update Workflow

Update an existing workflow.

**Endpoint**: `PUT /workflows/:id`

**Request Body**:
```json
{
  "name": "Updated Crawler",
  "description": "Updated description",
  "config": {...},
  "status": "active"
}
```

**Response**: `200 OK`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Updated Crawler",
  "description": "Updated description",
  "config": {...},
  "status": "active",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T11:00:00Z"
}
```

### Delete Workflow

Delete a workflow.

**Endpoint**: `DELETE /workflows/:id`

**Response**: `204 No Content`

### Update Workflow Status

Update only the workflow status.

**Endpoint**: `PATCH /workflows/:id/status`

**Request Body**:
```json
{
  "status": "active"
}
```

**Valid Statuses**:
- `draft`: Workflow is in draft mode
- `active`: Workflow can be executed
- `paused`: Workflow is temporarily paused
- `archived`: Workflow is archived

**Response**: `200 OK`
```json
{
  "message": "Workflow status updated successfully",
  "status": "active"
}
```

## Executions

### Start Execution

Start executing a workflow.

**Endpoint**: `POST /workflows/:id/execute`

**Response**: `202 Accepted`
```json
{
  "message": "Workflow execution started",
  "execution_id": "660e8400-e29b-41d4-a716-446655440000",
  "workflow_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Get Execution Status

Get the current status of a workflow execution.

**Endpoint**: `GET /executions/:execution_id`

**Response**: `200 OK`
```json
{
  "execution_id": "660e8400-e29b-41d4-a716-446655440000",
  "running": true,
  "stats": {
    "pending": 45,
    "processing": 2,
    "completed": 53,
    "failed": 3
  }
}
```

### Stop Execution

Stop a running execution.

**Endpoint**: `DELETE /executions/:execution_id`

**Response**: `200 OK`
```json
{
  "message": "Execution stopped",
  "execution_id": "660e8400-e29b-41d4-a716-446655440000"
}
```

### Get Queue Statistics

Get detailed queue statistics for an execution.

**Endpoint**: `GET /executions/:execution_id/stats`

**Response**: `200 OK`
```json
{
  "execution_id": "660e8400-e29b-41d4-a716-446655440000",
  "stats": {
    "pending": 45,
    "processing": 2,
    "completed": 53,
    "failed": 3,
    "skipped": 1
  },
  "pending_count": 45
}
```

## Health Check

### Check API Health

Check if the API and database are healthy.

**Endpoint**: `GET /health`

**Response**: `200 OK`
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "time": "2024-01-15T10:30:00Z"
}
```

**Response**: `503 Service Unavailable` (if unhealthy)
```json
{
  "status": "unhealthy",
  "error": "database connection failed"
}
```

## Error Responses

All error responses follow this format:

```json
{
  "error": "Error message describing what went wrong"
}
```

### Common HTTP Status Codes

- `200 OK`: Request succeeded
- `201 Created`: Resource created successfully
- `204 No Content`: Request succeeded with no response body
- `400 Bad Request`: Invalid request data
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Service temporarily unavailable

## Rate Limiting

Currently, no rate limiting is implemented. For production use, implement rate limiting middleware.

## CORS

CORS is enabled for all origins. For production, configure specific allowed origins in the server configuration.
