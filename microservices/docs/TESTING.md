# Testing the Microservices

## Quick Start

### 1. Start All Services

```bash
cd microservices
./scripts/setup-local.sh
```

This will:
- Start PostgreSQL, Redis, Pub/Sub emulator
- Run database migrations
- Start orchestrator and worker services

### 2. Verify Services

```bash
# Check orchestrator health
curl http://localhost:8080/health

# Check worker health
curl http://localhost:8081/health
```

## Testing Workflow Execution

### 1. Create a Workflow

```bash
# Generate a workflow ID
WORKFLOW_ID=$(uuidgen)

# Create workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/amazon-scraper-workflow.json

# Response:
# {
#   "id": "uuid",
#   "name": "Amazon Product Scraper",
#   "status": "active",
#   ...
# }
```

### 2. Start an Execution

```bash
# Start execution
curl -X POST http://localhost:8080/api/v1/workflows/{workflow-id}/execute

# Response:
# {
#   "execution_id": "execution-uuid",
#   "workflow_id": "workflow-uuid",
#   "status": "running",
#   "started_at": "2024-..."
# }
```

### 3. Monitor Execution

```bash
# Get execution status
curl http://localhost:8080/api/v1/executions/{execution-id}

# Response:
# {
#   "id": "execution-uuid",
#   "workflow_id": "workflow-uuid",
#   "status": "running",
#   "urls_processed": 15,
#   "urls_discovered": 45,
#   "items_extracted": 12,
#   "errors": 0
# }
```

### 4. View Logs

```bash
# Orchestrator logs
docker-compose -f infrastructure/docker-compose/docker-compose.yml logs -f orchestrator

# Worker logs
docker-compose -f infrastructure/docker-compose/docker-compose.yml logs -f worker

# All logs
make docker-logs
```

## Testing Individual Components

### Database

```bash
# Connect to PostgreSQL
docker-compose -f infrastructure/docker-compose/docker-compose.yml exec postgres psql -U crawlify

# Check tables
\dt

# View workflows
SELECT id, name, status FROM workflows;

# View executions
SELECT id, workflow_id, status, urls_processed FROM workflow_executions;
```

### Redis Cache

```bash
# Connect to Redis
docker-compose -f infrastructure/docker-compose/docker-compose.yml exec redis redis-cli

# Check keys
KEYS *

# View workflow cache
GET workflow:your-workflow-id

# View deduplication
KEYS dedup:exec:*
```

### Pub/Sub

```bash
# View Pub/Sub status
curl http://localhost:8085/v1/projects/crawlify-local/topics

# Pull messages (for debugging)
curl -X POST http://localhost:8085/v1/projects/crawlify-local/subscriptions/crawlify-tasks-sub:pull \
  -H "Content-Type: application/json" \
  -d '{"maxMessages": 10}'
```

## Sample Workflows

### Simple Page Scraper

```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Simple Scraper",
    "config": {
      "start_urls": ["https://example.com"],
      "phases": [{
        "id": "phase-1",
        "name": "Main",
        "nodes": ["navigate", "extract"]
      }],
      "nodes": [
        {
          "id": "navigate",
          "type": "navigate",
          "config": {"timeout": 30000}
        },
        {
          "id": "extract",
          "type": "extract",
          "config": {
            "schema": "page",
            "selectors": {
              "title": {"selector": "h1", "type": "text"},
              "content": {"selector": "p", "type": "text", "multiple": true}
            }
          }
        }
      ]
    }
  }'
```

## Troubleshooting

### Services Not Starting

```bash
# Check Docker status
docker ps

# Restart all services
make docker-restart

# View logs
make docker-logs
```

### Database Connection Issues

```bash
# Check PostgreSQL is running
docker-compose -f infrastructure/docker-compose/docker-compose.yml ps postgres

# Check database exists
docker-compose -f infrastructure/docker-compose/docker-compose.yml exec postgres psql -U crawlify -c "\l"

# Re-run migrations
./scripts/setup-local.sh
```

### Worker Not Processing Tasks

```bash
# Check Pub/Sub emulator
curl http://localhost:8085/v1/projects/crawlify-local/topics/crawlify-tasks

# Check worker logs
docker-compose -f infrastructure/docker-compose/docker-compose.yml logs worker

# Restart worker
docker-compose -f infrastructure/docker-compose/docker-compose.yml restart worker
```

### No Items Extracted

Check:
1. Browser pool initialized (check worker logs)
2. Selectors are correct for target site
3. Pages are loading (check for navigation errors)
4. GCS client initialized (optional)

## Performance Testing

### Load Test

```bash
# Create multiple executions
for i in {1..10}; do
  curl -X POST http://localhost:8080/api/v1/workflows/{workflow-id}/execute
  sleep 1
done
```

### Monitor Stats

```bash
# Watch execution stats
watch -n 2 'curl -s http://localhost:8080/api/v1/executions/{execution-id} | jq'
```

## Cleanup

```bash
# Stop all services
make docker-down

# Remove volumes (deletes all data)
docker-compose -f infrastructure/docker-compose/docker-compose.yml down -v
```
