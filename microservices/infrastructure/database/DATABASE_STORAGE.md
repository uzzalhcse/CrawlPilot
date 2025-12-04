# Database Storage for Extracted Data

## Overview

Added database storage fallback for extracted items when GCS (Google Cloud Storage) is disabled. This ensures extracted data is never lost during local development.

## Changes Made

### 1. Database Migration
- **002_add_extracted_items_table_up.sql**: Creates `extracted_items` table
- **002_add_extracted_items_table_down.sql**: Rollback migration

### 2. Database Storage Client
- **worker/internal/storage/db_storage.go**: New module to save/retrieve extracted items from PostgreSQL

### 3. Configuration
- Added `storage_enabled: false/true` flag to `GCPConfig`
- Default: `false` for local development

### 4. Task Executor Updates
- Checks `storage_enabled` flag
- Uses GCS when enabled (production)
- Falls back to database when disabled (local dev)

## Database Schema

```sql
CREATE TABLE extracted_items (
    id UUID PRIMARY KEY,
    execution_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    task_id VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    data JSONB NOT NULL,  -- The actual extracted item
    extracted_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## Usage

### Local Development
```yaml
# worker/config.yaml
gcp:
  storage_enabled: false  # Use database storage
```

### Production
```yaml
# worker/config.yaml
gcp:
  storage_enabled: true   # Use GCS
  storage_bucket: "your-bucket-name"
```

## Querying Extracted Data

```sql
-- Get all extracted items for an execution
SELECT data FROM extracted_items 
WHERE execution_id = 'your-execution-id'
ORDER BY extracted_at ASC;

-- Count items by workflow
SELECT workflow_id, COUNT(*) 
FROM extracted_items 
GROUP BY workflow_id;

-- Get recent extractions
SELECT execution_id, url, data, extracted_at 
FROM extracted_items 
ORDER BY extracted_at DESC 
LIMIT 100;
```

## Benefits

✅ **No data loss** - Extracted data is always saved, even in local development
✅ **Easy inspection** - Query database directly to see extracted data
✅ **Cost-effective** - No GCS costs for local development
✅ **Transparent** - Same extraction logic, different storage backend
