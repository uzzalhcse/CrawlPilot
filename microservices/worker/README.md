# Crawlify Worker Service

The worker is a stateless execution service that processes scraping tasks from the Pub/Sub queue.

## Responsibilities

- ✅ Pull tasks from Pub/Sub
- ✅ Execute workflow nodes
- ✅ Browser automation (future)
- ✅ Data extraction (future)
- ✅ URL discovery and re-queuing
- ✅ Result storage

## Architecture

```
worker/
├── cmd/worker/           # Main entry point
├── internal/
│   ├── executor/        # Workflow execution
│   ├── browser/         # Browser automation (future)
│   ├── handler/         # Task processing
│   └── storage/         # Data persistence
├── config.yaml          # Configuration
├── Dockerfile           # Container with Playwright
└── go.mod               # Dependencies
```

## Configuration

See `config.yaml` for configuration options. Key settings:

- **Database**: PostgreSQL connection (5 connections - limited)
- **Redis**: Required for caching and deduplication
- **GCP**: Pub/Sub subscription config
- **Browser**: Playwright settings

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
docker-compose up worker
```

### Standalone

```bash
# Install dependencies
go mod download

# Install Playwright
npx playwright install chromium

# Run
go run cmd/worker/main.go
```

## Task Processing

Workers pull tasks from Pub/Sub and process them:

1. Pull task from subscription
2. Deserialize task payload
3. Execute workflow nodes
4. Extract data
5. Discover new URLs
6. Re-queue discovered URLs
7. Acknowledge task

## Scaling

Workers auto-scale based on Pub/Sub queue depth:

- **Min instances**: 0 (scales to zero when idle)
- **Max instances**: 1000 per service
- **Concurrency**: 1 task per container instance

For higher throughput, deploy multiple worker services.

## Browser Automation (Future)

Workers include Playwright for browser automation:

- Chromium browser
- Headless mode
- 2 browser contexts per worker
- Automatic cleanup

## Development

### Adding Node Types

1. Create executor in `internal/executor/`
2. Register node type in execution engine
3. Test with sample workflow

### Testing

```bash
go test ./...
```

## Deployment

### Build Docker Image

```bash
docker build -t crawlify-worker:latest .
```

### Deploy to Cloud Run

```bash
gcloud run deploy crawlify-worker \
  --image gcr.io/PROJECT_ID/crawlify-worker:latest \
  --platform managed \
  --region us-central1 \
  --min-instances 0 \
  --max-instances 1000 \
  --concurrency 1 \
  --timeout 900 \
  --cpu 1 \
  --memory 2Gi \
  --no-allow-unauthenticated
```

## Monitoring

- Structured logging with Zap
- Health check endpoint at `/health`
- Task processing metrics
- Worker ID from `K_REVISION` env var
