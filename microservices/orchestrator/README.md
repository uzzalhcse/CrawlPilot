# Crawlify Orchestrator Service

The orchestrator is the central management service for Crawlify. It handles workflow management, execution orchestration, and task distribution.

## Responsibilities

- ✅ Workflow CRUD operations
- ✅ Execution initiation and tracking
- ✅ Task publishing to Pub/Sub
- ✅ Monitoring and analytics
- ✅ User authentication (future)
- ✅ Plugin marketplace (future)

## Architecture

```
orchestrator/
├── cmd/orchestrator/      # Main entry point
├── internal/
│   ├── api/              # HTTP handlers
│   ├── service/          # Business logic
│   ├── repository/       # Database access
│   └── cloudtasks/       # Task distribution (future)
├── config.yaml           # Configuration
├── Dockerfile            # Container image
└── go.mod                # Dependencies
```

## Configuration

See `config.yaml` for configuration options. Key settings:

- **Server**: HTTP server configuration
- **Database**: PostgreSQL connection (10 connections)
- **Redis**: Cache configuration
- **GCP**: Pub/Sub and Cloud Storage settings

## Environment Variables

Override config values via environment:

```bash
export DATABASE_HOST=localhost
export REDIS_HOST=localhost
export GCP_PROJECT_ID=my-project
```

## Running Locally

### With Docker Compose

```bash
cd ../infrastructure/docker-compose
docker-compose up orchestrator
```

### Standalone

```bash
# Install dependencies
go mod download

# Run
go run cmd/orchestrator/main.go
```

## API Endpoints

### Health Check

```bash
GET /health
```

### Workflows (TODO)

```bash
GET    /api/v1/workflows
POST   /api/v1/workflows
GET    /api/v1/workflows/:id
PUT    /api/v1/workflows/:id
DELETE /api/v1/workflows/:id
POST   /api/v1/workflows/:id/execute
```

## Development

### Adding New Endpoints

1. Create handler in `internal/api/handlers/`
2. Register route in `cmd/orchestrator/main.go`
3. Implement service logic in `internal/service/`
4. Add repository methods in `internal/repository/`

### Testing

```bash
go test ./...
```

## Deployment

### Build Docker Image

```bash
docker build -t crawlify-orchestrator:latest .
```

### Deploy to Cloud Run

```bash
gcloud run deploy crawlify-orchestrator \
  --image gcr.io/PROJECT_ID/crawlify-orchestrator:latest \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated
```

## Monitoring

- Structured logging with Zap
- Health check endpoint at `/health`
- Request logging middleware active
