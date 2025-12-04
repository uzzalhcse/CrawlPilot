.PHONY: help build run test clean docker-up docker-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development (uses config.yaml for docker-compose ports)
run-orchestrator: ## Run orchestrator with docker-compose infrastructure
	cd orchestrator && go run cmd/orchestrator/main.go

run-worker: ## Run worker with docker-compose infrastructure
	cd worker && go run cmd/worker/main.go

# Build
build-orchestrator: ## Build orchestrator binary
	cd orchestrator && go build -o bin/orchestrator cmd/orchestrator/main.go

build-worker: ## Build worker binary
	cd worker && go build -o bin/worker cmd/worker/main.go

build-all: build-orchestrator build-worker ## Build all services

# Docker
docker-build-orchestrator: ## Build orchestrator Docker image
	docker build -t crawlify-orchestrator:latest -f orchestrator/Dockerfile ..

docker-build-worker: ## Build worker Docker image
	docker build -t crawlify-worker:latest -f worker/Dockerfile ..

docker-build-all: docker-build-orchestrator docker-build-worker ## Build all Docker images

# Docker Compose
docker-up: ## Start all services with docker-compose
	cd infrastructure/docker-compose && docker compose up -d

docker-down: ## Stop all services
	cd infrastructure/docker-compose && docker compose down

docker-logs: ## View logs from all services
	cd infrastructure/docker-compose && docker compose logs -f

docker-logs-orchestrator: ## View orchestrator logs
	cd infrastructure/docker-compose && docker compose logs -f orchestrator

docker-logs-worker: ## View worker logs
	cd infrastructure/docker-compose && docker compose logs -f worker

docker-ps: ## Show running containers
	cd infrastructure/docker-compose && docker compose ps

# Testing
test-orchestrator: ## Run orchestrator tests
	cd orchestrator && go test ./...

test-worker: ## Run worker tests
	cd worker && go test ./...

test-shared: ## Run shared library tests
	cd shared && go test ./...

test-all: test-orchestrator test-worker test-shared ## Run all tests

# Dependencies
deps-orchestrator: ## Download orchestrator dependencies
	cd orchestrator && go mod download

deps-worker: ## Download worker dependencies
	cd worker && go mod download

deps-shared: ## Download shared dependencies
	cd shared && go mod download

deps-all: deps-orchestrator deps-worker deps-shared ## Download all dependencies

tidy-all: ## Tidy all go.mod files
	cd orchestrator && go mod tidy
	cd worker && go mod tidy
	cd shared && go mod tidy

# Clean
clean: ## Remove build artifacts
	rm -rf orchestrator/bin worker/bin
	cd infrastructure/docker-compose && docker compose down -v

# ============================================
# Database Migrations
# ============================================

.PHONY: migrate-up migrate-down migrate-fresh migrate-status migrate-up-standalone migrate-down-standalone

# Database connection strings
DB_HOST ?= localhost
DB_PORT ?= 5433
DB_USER ?= crawlify
DB_PASSWORD ?= dev_password
DB_NAME ?= crawlify
DB_URL = postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# For standalone (native PostgreSQL)
DB_STANDALONE_URL = postgresql://$(DB_USER):$(DB_PASSWORD)@localhost:5432/$(DB_NAME)?sslmode=disable

migrate-up: ## Run database migrations (docker-compose)
	@echo "Running database migrations on docker-compose PostgreSQL (port 5433)..."
	@for file in infrastructure/database/migrations/*_up.sql; do \
		echo "Applying migration: $$(basename $$file)"; \
		PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f $$file; \
	done
	@echo "✅ Migrations completed!"

migrate-down: ## Rollback database migrations (docker-compose)
	@echo "⚠️  WARNING: This will ROLLBACK ALL MIGRATIONS!"
	@echo "Press Ctrl+C to cancel, or Enter to continue..."
	@read confirm
	@echo "Rolling back migrations..."
	@for file in $$(ls -r infrastructure/database/migrations/*_down.sql); do \
		echo "Rolling back: $$(basename $$file)"; \
		PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f $$file; \
	done
	@echo "✅ All migrations rolled back!"

migrate-fresh: migrate-down migrate-up ## Rollback and re-run all migrations (docker-compose)
	@echo "✅ Database refreshed!"

migrate-status: ## Check database schema status (docker-compose)
	@echo "Database: $(DB_NAME) on $(DB_HOST):$(DB_PORT)"
	@echo ""
	@echo "Tables:"
	@PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -c \
		"SELECT table_name FROM information_schema.tables WHERE table_schema='public' ORDER BY table_name;"

migrate-up-standalone: ## Run database migrations (standalone PostgreSQL port 5432)
	@echo "Running database migrations on standalone PostgreSQL (port 5432)..."
	@for file in infrastructure/database/migrations/*_up.sql; do \
		echo "Applying migration: $$(basename $$file)"; \
		PGPASSWORD=$(DB_PASSWORD) psql -h localhost -p 5432 -U $(DB_USER) -d $(DB_NAME) -f $$file; \
	done
	@echo "✅ Migrations completed!"

migrate-down-standalone: ## Rollback database migrations (standalone PostgreSQL)
	@echo "⚠️  WARNING: This will ROLLBACK ALL MIGRATIONS!"
	@echo "Press Ctrl+C to cancel, or Enter to continue..."
	@read confirm
	@echo "Rolling back migrations..."
	@for file in $$(ls -r infrastructure/database/migrations/*_down.sql); do \
		echo "Rolling back: $$(basename $$file)"; \
		PGPASSWORD=$(DB_PASSWORD) psql -h localhost -p 5432 -U $(DB_USER) -d $(DB_NAME) -f $$file; \
	done
	@echo "✅ All migrations rolled back!"

# Setup
setup-pubsub-local: ## Setup Pub/Sub emulator topics
	@echo "Setting up Pub/Sub emulator..."
	@curl -X PUT http://localhost:8095/v1/projects/crawlify-local/topics/crawlify-tasks
	@curl -X PUT http://localhost:8095/v1/projects/crawlify-local/subscriptions/crawlify-tasks-sub \
		-H "Content-Type: application/json" \
		-d '{"topic":"projects/crawlify-local/topics/crawlify-tasks"}'
	@echo "Pub/Sub setup complete!"

# Development workflow
dev: docker-up migrate-up setup-pubsub-local ## Start full development environment
	@echo "Development environment ready!"
	@echo "Orchestrator: http://localhost:8090"
	@echo "Worker: http://localhost:8091"
	@echo "Pub/Sub Emulator: http://localhost:8095"
	@echo "PostgreSQL: localhost:5433"
	@echo "Redis: localhost:6380"

# Health checks
health-orchestrator: ## Check orchestrator health
	@curl http://localhost:8090/health | jq .

health-worker: ## Check worker health
	@curl http://localhost:8091/health | jq .

health-all: ## Check all services health
	@echo "=== Orchestrator ==="
	@curl -s http://localhost:8090/health | jq .
	@echo "\n=== Worker ==="
	@curl -s http://localhost:8091/health | jq .

# ============================================
# Standalone Local Development (Native Services)
# For running with locally installed PostgreSQL, Redis, and Pub/Sub emulator
# NOT for use with docker-compose
# ============================================

.PHONY: pubsub-local run-orchestrator-standalone run-worker-standalone test-workflow local-setup setup-pubsub-topics-standalone local-quickstart

# Start Pub/Sub emulator locally on native port
pubsub-local: ## Start Pub/Sub emulator on port 8085 (run in separate terminal)
	@echo "Starting Pub/Sub emulator on port 8085..."
	@gcloud beta emulators pubsub start --project=crawlify-local --host-port=0.0.0.0:8085

# Run orchestrator with native services (not docker-compose)
run-orchestrator-standalone: ## Run orchestrator with native PostgreSQL/Redis/PubSub
	@echo "Starting Orchestrator (standalone mode)..."
	@echo "NOTE: Requires native PostgreSQL:5432, Redis:6379, PubSub:8085"
	@export DATABASE_PORT=5432 && \
	export REDIS_PORT=6379 && \
	export GCP_PUBSUB_EMULATOR_HOST=localhost:8085 && \
	cd orchestrator && go run cmd/orchestrator/main.go

# Run worker with native services (not docker-compose)
run-worker-standalone: ## Run worker with native PostgreSQL/Redis/PubSub
	@echo "Starting Worker (standalone mode)..."
	@echo "NOTE: Requires native PostgreSQL:5432, Redis:6379, PubSub:8085"
	@export SERVER_PORT=8081 && \
	export DATABASE_PORT=5432 && \
	export REDIS_PORT=6379 && \
	export GCP_PUBSUB_EMULATOR_HOST=localhost:8085 && \
	cd worker && go run cmd/worker/main.go

# Setup Pub/Sub topics for standalone emulator on port 8085
setup-pubsub-topics-standalone: ## Setup topics for standalone Pub/Sub emulator (port 8085)
	@sleep 2
	@echo "Creating Pub/Sub topic on standalone emulator (port 8085)..."
	@curl -X PUT http://localhost:8085/v1/projects/crawlify-local/topics/crawlify-tasks
	@echo ""
	@echo "Creating Pub/Sub subscription..."
	@curl -X PUT http://localhost:8085/v1/projects/crawlify-local/subscriptions/crawlify-tasks-sub \
		-H "Content-Type: application/json" \
		-d '{"topic":"projects/crawlify-local/topics/crawlify-tasks"}'
	@echo ""
	@echo "✅ Pub/Sub setup complete!"

# Test workflow
test-workflow: ## Create and test a workflow
	@echo "Creating workflow..."
	@curl -X POST http://localhost:8080/api/v1/workflows \
		-H "Content-Type: application/json" \
		-d @examples/aqua_site_crawler_phase.json | jq .

# Complete local setup
local-setup: ## Complete local development setup
	@chmod +x scripts/setup-local-dev.sh
	@./scripts/setup-local-dev.sh

# Quick start guide
local-quickstart: ## Show quick start guide
	@echo "🚀 Crawlify Development Quick Start"
	@echo ""
	@echo "=== Option 1: Docker Compose (RECOMMENDED) ==="
	@echo "Terminal 1 - Start infrastructure:"
	@echo "  make docker-up"
	@echo ""
	@echo "Terminal 2 - Setup Pub/Sub topics:"
	@echo "  make setup-pubsub-local"
	@echo ""
	@echo "Terminal 3 - Run orchestrator:"
	@echo "  make run-orchestrator"
	@echo ""
	@echo "Terminal 4 - Run worker:"
	@echo "  make run-worker"
	@echo ""
	@echo "=== Option 2: Standalone (Native Services) ==="
	@echo "Terminal 1 - Pub/Sub Emulator:"
	@echo "  make pubsub-local"
	@echo ""
	@echo "Terminal 2 - Setup topics (after emulator starts):"
	@echo "  make setup-pubsub-topics-standalone"
	@echo ""
	@echo "Terminal 3 - Orchestrator:"
	@echo "  make run-orchestrator-standalone"
	@echo ""
	@echo "Terminal 4 - Worker:"
	@echo "  make run-worker-standalone"
	@echo ""
