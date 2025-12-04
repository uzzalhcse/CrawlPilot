# Crawlify Microservices

Crawlify is a distributed web scraping platform with orchestrator and worker services.

## Quick Start (Docker Compose)

```bash
# 1. Start infrastructure (PostgreSQL, Redis, Pub/Sub emulator)
make docker-up

# 2. Setup Pub/Sub topics
make setup-pubsub-local

# 3. Run orchestrator (in new terminal)
make run-orchestrator

# 4. Run worker (in new terminal)
make run-worker
```

## Configuration

All configuration is managed through `config.yaml` files:
- `orchestrator/config.yaml` - Orchestrator configuration
- `worker/config.yaml` - Worker configuration

**Docker Compose Ports:**
- PostgreSQL: `5433` (mapped from container's 5432)
- Redis: `6380` (mapped from container's 6379)
- Pub/Sub Emulator: `8095` (mapped from container's 8085)
- PgBouncer: `6432`

## Development Modes

### Option 1: Docker Compose (Recommended)
Uses `config.yaml` with docker-compose infrastructure.

```bash
make run-orchestrator  # Uses docker-compose ports
make run-worker
```

### Option 2: Standalone (Native Services)
For running with locally installed PostgreSQL, Redis, and Pub/Sub emulator.

```bash
make pubsub-local                     # Terminal 1
make setup-pubsub-topics-standalone   # Terminal 2
make run-orchestrator-standalone      # Terminal 3
make run-worker-standalone            # Terminal 4
```

## Available Commands

```bash
make help              # Show all available commands
make docker-up         # Start docker-compose infrastructure
make docker-down       # Stop docker-compose infrastructure
make docker-logs       # View all logs
make build-all         # Build all services
make test-all          # Run all tests
make local-quickstart  # Show detailed quick start guide
```

### Database Migrations

```bash
# Docker Compose (port 5433)
make migrate-up        # Run database migrations
make migrate-down      # Drop all tables (destructive!)
make migrate-fresh     # Drop and recreate all tables
make migrate-status    # Check current database schema

# Standalone PostgreSQL (port 5432)
make migrate-up-standalone
make migrate-down-standalone
```

**Note:** The `dev` command automatically runs migrations.

## Architecture

```
┌─────────────┐      ┌──────────────┐      ┌────────┐
│ Orchestrator│─────▶│   Pub/Sub    │─────▶│ Worker │
│   (8080)    │      │  Emulator    │      │ (8081) │
└─────────────┘      └──────────────┘      └────────┘
       │                                         │
       ├─────────────────┬───────────────────────┤
       │                 │                       │
  ┌─────────┐      ┌──────────┐           ┌─────────┐
  │PostgreSQL│      │  Redis   │           │ Browser │
  │  (5433) │      │  (6380)  │           │  Pool   │
  └─────────┘      └──────────┘           └─────────┘
```

## Services

- **Orchestrator**: Manages workflows and distributes tasks
- **Worker**: Executes scraping tasks using browser automation
- **PostgreSQL**: Stores workflows and execution data
- **Redis**: Caching layer
- **Pub/Sub Emulator**: Message queue for task distribution

## Environment Variables

**All configuration is now managed transparently through `config.yaml` files.**

The application prioritizes configuration in this order:
1. Environment variables (highest priority - overrides config.yaml)
2. `config.yaml` (default source)

**No environment variables are required** - everything has sensible defaults in config.yaml.

Optional overrides for environment-specific adjustments:
- `DATABASE_PORT` - Override database port
- `REDIS_PORT` - Override Redis port
- `GCP_PUBSUB_EMULATOR_HOST` - Override Pub/Sub emulator host
- `SERVER_PORT` - Override server port (worker only)

## Configuration Files

### `orchestrator/config.yaml`
- Server configuration (host, port, timeouts)
- Database connection (PostgreSQL via port 5433)
- Redis cache (port 6380)
- Pub/Sub emulator (port 8095)
- Browser pool settings

### `worker/config.yaml`
- Server configuration (port 8081)
- Database connection (PostgreSQL via port 5433)
- Redis cache (port 6380)
- Pub/Sub emulator (port 8095)
- **Orchestrator URL** (`http://localhost:8080`)
- Browser automation settings
