# Port Mapping Reference

The docker-compose setup uses **non-standard ports** to avoid conflicts with local services:

| Service | Internal Port | External Port | Purpose |
|---------|--------------|---------------|---------|
| PostgreSQL | 5432 | **5433** | Database |
| Redis | 6379 | **6380** | Cache |
| Pub/Sub Emulator | 8085 | **8095** | Message Queue |
| Orchestrator | 8080 | **8090** | API Gateway |
| Worker | 8080 | **8091** | Task Processor |

## Accessing Services

### From Host Machine:
```bash
# PostgreSQL
psql -h localhost -p 5433 -U crawlify -d crawlify

# Redis
redis-cli -h localhost -p 6380

# Orchestrator API
curl http://localhost:8090/health

# Worker Health
curl http://localhost:8091/health

# Pub/Sub Emulator
curl http://localhost:8095/v1/projects/crawlify-local/topics
```

### From Docker Containers:
Containers communicate using **internal ports** and service names:
```yaml
DATABASE_HOST: postgres
DATABASE_PORT: 5432  # Internal port

REDIS_HOST: redis  
REDIS_PORT: 6379  # Internal port

ORCHESTRATOR_URL: http://orchestrator:8080  # Internal port
```

## Configuration Files to Match

If running services **outside Docker**, update config files to use **external ports**:

### orchestrator/config.yaml:
```yaml
database:
  port: 5433  # External port

redis:
  port: 6380  # External port
```

### worker/config.yaml:
```yaml
database:
  port: 5433  # External port

redis:
  port: 6380  # External port
```

## Testing

```bash
# Create workflow (note port 8090)
curl -X POST http://localhost:8090/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/aqua_site_crawler_phase.json

# Start execution
curl -X POST http://localhost:8090/api/v1/workflows/{id}/execute
```
