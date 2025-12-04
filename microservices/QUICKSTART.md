# Crawlify Microservices - Quick Start Guide

## Prerequisites

- Docker and Docker Compose
- Go 1.24+ (for local development without Docker)
- Make (optional, for using Makefile commands)

## Quick Start (Docker)

### 1. Start All Services

```bash
cd microservices
make dev
```

This will:
- Start PostgreSQL
- Start Redis
- Start Pub/Sub emulator
- Start Orchestrator
- Start Worker (2 replicas)
- Setup Pub/Sub topics and subscriptions

### 2. Verify Services

Check all services are running:

```bash
make docker-ps
```

Check orchestrator health:

```bash
make health-orchestrator
# or
curl http://localhost:8080/health
```

### 3. View Logs

```bash
# All services
make docker-logs

# Orchestrator only
make docker-logs-orchestrator

# Worker only
make docker-logs-worker
```

### 4. Stop Services

```bash
make docker-down
```

## Development Without Docker

### 1. Start Infrastructure

You still need PostgreSQL, Redis, and Pub/Sub emulator:

```bash
# Start just infrastructure services
cd infrastructure/docker-compose
docker-compose up postgres redis pubsub-emulator -d
```

### 2. Setup Pub/Sub

```bash
make setup-pubsub-local
```

### 3. Run Orchestrator

```bash
cd orchestrator
go mod download
go run cmd/orchestrator/main.go
```

### 4. Run Worker

In another terminal:

```bash
cd worker
go mod download
go run cmd/worker/main.go
```

## Testing the Setup

### 1. Health Check

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "service": "orchestrator",
  "version": "1.0.0",
  "time": "2024-12-04T14:12:03Z"
}
```

### 2. Test Workflow Endpoint (Coming Soon)

```bash
curl http://localhost:8080/api/v1/workflows
```

## Project Structure

```
microservices/
├── orchestrator/          # API and orchestration service
├── worker/               # Task execution service
├── shared/               # Shared libraries
├── infrastructure/       # Docker and Terraform configs
├── Makefile             # Development commands
└── README.md            # This file
```

## Common Commands

### Build

```bash
make build-all              # Build all binaries
make docker-build-all       # Build all Docker images
```

### Run

```bash
make run-orchestrator       # Run orchestrator locally
make run-worker            # Run worker locally
make dev                   # Start full Docker environment
```

### Test

```bash
make test-all              # Run all tests
make test-orchestrator     # Test orchestrator only
make test-worker           # Test worker only
```

### Clean

```bash
make clean                 # Remove build artifacts and volumes
```

## Configuration

### Environment Variables

Create a `.env` file in `infrastructure/docker-compose/`:

```env
DATABASE_PASSWORD=your_secure_password
REDIS_PASSWORD=your_redis_password
GCP_PROJECT_ID=your-project-id
```

### Config Files

- `orchestrator/config.yaml` - Orchestrator configuration
- `worker/config.yaml` - Worker configuration

## Troubleshooting

### Services won't start

```bash
# Check logs
make docker-logs

# Restart services
make docker-down
make docker-up
```

### Database connection issues

```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Check PostgreSQL logs
docker logs crawlify-postgres
```

### Pub/Sub issues

```bash
# Check emulator is running
docker ps | grep pubsub

# Re-setup topics
make setup-pubsub-local
```

### Port already in use

If port 8080 or others are in use, you can change them in `docker-compose.yml`.

## Next Steps

1. **Phase 2**: Implement basic workflow execution
2. **Database Migration**: Copy schema from monolith
3. **API Implementation**: Workflow CRUD endpoints
4. **Worker Logic**: Task processing implementation

See [MIGRATION.md](../../MIGRATION.md) for full migration plan.

## Getting Help

- Check service logs: `make docker-logs`
- Check health endpoints: `make health-all`
- Read service READMEs:
  - [Orchestrator README](./orchestrator/README.md)
  - [Worker README](./worker/README.md)

## Resources

- [Migration Guide](../../MIGRATION.md)
- [Implementation Plan](../../.gemini/antigravity/brain/*/implementation_plan.md)
- [Scalability Analysis](../../.gemini/antigravity/brain/*/scalability_analysis.md)
