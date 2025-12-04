# Local Development Setup Guide

## Prerequisites

Make sure you have the following installed:
- PostgreSQL (running locally)
- Redis (running locally)
- Go 1.21+
- Google Cloud SDK (for Pub/Sub emulator)
- jq (for JSON formatting)

## Quick Start

### Option 1: Automated Setup

```bash
cd microservices
make local-setup
```

This will:
- ✅ Check dependencies
- ✅ Start PostgreSQL and Redis
- ✅ Create database and user
- ✅ Apply database schema
- ✅ Install Go dependencies
- ✅ Install Playwright browsers

### Option 2: Manual Setup

Follow these steps in separate terminals:

#### Terminal 1: Start Pub/Sub Emulator
```bash
cd microservices
make pubsub-local
```

Wait for: `Server started, listening on 8085`

#### Terminal 2: Setup Pub/Sub Topics
```bash
cd microservices
make setup-pubsub-topics
```

#### Terminal 3: Start Orchestrator
```bash
cd microservices
make run-orchestrator-local
```

Wait for: `Orchestrator starting`

#### Terminal 4: Start Worker
```bash
cd microservices
make run-worker-local
```

Wait for: `Worker service started`

#### Terminal 5: Test
```bash
cd microservices
make test-workflow
```

## Configuration

Services use these local endpoints:

| Service | Port | URL |
|---------|------|-----|
| PostgreSQL | 5432 | localhost:5432 |
| Redis | 6379 | localhost:6379 |
| Pub/Sub Emulator | 8085 | localhost:8085 |
| Orchestrator | 8080 | http://localhost:8080 |
| Worker | 8081 | http://localhost:8081 |

## Testing Your Setup

### 1. Check Services
```bash
# Orchestrator health
curl http://localhost:8080/health | jq

# Worker health (if exposed)
curl http://localhost:8081/health | jq
```

### 2. Create Workflow
```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/aqua_site_crawler_phase.json | jq
```

Copy the `id` from the response.

### 3. Start Execution
```bash
WORKFLOW_ID="<paste-id-here>"
curl -X POST http://localhost:8080/api/v1/workflows/$WORKFLOW_ID/execute | jq
```

Copy the `id` (execution ID) from the response.

### 4. Monitor Execution
```bash
EXECUTION_ID="<paste-execution-id-here>"
curl http://localhost:8080/api/v1/executions/$EXECUTION_ID | jq
```

## Troubleshooting

### PostgreSQL Connection Failed
```bash
# Check if PostgreSQL is running
sudo systemctl status postgresql

# Start PostgreSQL
sudo systemctl start postgresql

# Create database manually
sudo -u postgres createdb crawlify
sudo -u postgres psql -c "CREATE USER crawlify WITH PASSWORD 'dev_password';"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE crawlify TO crawlify;"
```

### Redis Connection Failed
```bash
# Check if Redis is running
sudo systemctl status redis-server

# Start Redis
sudo systemctl start redis-server
```

### Pub/Sub Emulator Not Found
```bash
# Install gcloud components
gcloud components install pubsub-emulator

# Or install full Google Cloud SDK
# https://cloud.google.com/sdk/docs/install
```

### Worker Can't Find Playwright
```bash
cd worker
npx playwright install chromium
```

### Database Schema Not Applied
```bash
cd microservices
PGPASSWORD=dev_password psql -h localhost -U crawlify -d crawlify < infrastructure/database/schema.sql
```

## Development Workflow

1. **Make Code Changes**
   - Edit files in `orchestrator/`, `worker/`, or `shared/`

2. **Restart Service**
   - Stop the process (Ctrl+C)
   - Run `make run-orchestrator-local` or `make run-worker-local` again

3. **Test Changes**
   - Use `curl` commands or create workflows via API

## Logs

All services log to stdout. Check the terminal where you started each service.

## Stopping Services

Press `Ctrl+C` in each terminal to stop the services.

## Clean Up

```bash
# Drop database
sudo -u postgres dropdb crawlify

# Clear Redis
redis-cli FLUSHALL
```

## Next Steps

- Read [TESTING.md](docs/TESTING.md) for comprehensive testing guide
- Read [PRODUCTION_READINESS.md](PRODUCTION_READINESS.md) for deployment info
- See [DEPLOYMENT.md](DEPLOYMENT.md) for cloud deployment
