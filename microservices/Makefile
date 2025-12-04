.PHONY: help build run test clean docker-up docker-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development
run-orchestrator: ## Run orchestrator locally
	cd orchestrator && go run cmd/orchestrator/main.go

run-worker: ## Run worker locally
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

# Setup
setup-pubsub-local: ## Setup Pub/Sub emulator topics
	@echo "Setting up Pub/Sub emulator..."
	@curl -X PUT http://localhost:8095/v1/projects/crawlify-local/topics/crawlify-tasks
	@curl -X PUT http://localhost:8095/v1/projects/crawlify-local/subscriptions/crawlify-tasks-sub \
		-H "Content-Type: application/json" \
		-d '{"topic":"projects/crawlify-local/topics/crawlify-tasks"}'
	@echo "Pub/Sub setup complete!"

# Development workflow
dev: docker-up setup-pubsub-local ## Start full development environment
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
# Local Development (without Docker)
# ============================================

.PHONY: pubsub-local run-orchestrator-local run-worker-local test-workflow local-setup setup-pubsub-topics local-quickstart

# Start Pub/Sub emulator locally
pubsub-local: ## Start Pub/Sub emulator (run in separate terminal)
	@echo "Starting Pub/Sub emulator on port 8085..."
	@gcloud beta emulators pubsub start --project=crawlify-local --host-port=0.0.0.0:8085

# Run orchestrator locally
run-orchestrator-local: ## Run orchestrator locally
	@echo "Starting Orchestrator..."
	@export SERVER_HOST=0.0.0.0 && \
	export SERVER_PORT=8080 && \
	export DATABASE_HOST=localhost && \
	export DATABASE_PORT=5432 && \
	export DATABASE_USER=crawlify && \
	export DATABASE_PASSWORD=dev_password && \
	export DATABASE_NAME=crawlify && \
	export REDIS_HOST=localhost && \
	export REDIS_PORT=6379 && \
	export GCP_PROJECT_ID=crawlify-local && \
	export GCP_PUBSUB_TOPIC=crawlify-tasks && \
	export PUBSUB_EMULATOR_HOST=localhost:8085 && \
	cd orchestrator && go run cmd/orchestrator/main.go

# Run worker locally
run-worker-local: ## Run worker locally
	@echo "Starting Worker..."
	@export PORT=8081 && \
	export DATABASE_HOST=localhost && \
	export DATABASE_PORT=5432 && \
	export DATABASE_USER=crawlify && \
	export DATABASE_PASSWORD=dev_password && \
	export DATABASE_NAME=crawlify && \
	export REDIS_HOST=localhost && \
	export REDIS_PORT=6379 && \
	export GCP_PROJECT_ID=crawlify-local && \
	export GCP_PUBSUB_SUBSCRIPTION=crawlify-tasks-sub && \
	export PUBSUB_EMULATOR_HOST=localhost:8085 && \
	export ORCHESTRATOR_URL=http://localhost:8080 && \
	export BROWSER_HEADLESS=true && \
	export BROWSER_POOL_SIZE=3 && \
	cd worker && go run cmd/worker/main.go

# Setup Pub/Sub topics (run after starting emulator)
setup-pubsub-topics: ## Setup Pub/Sub topics and subscriptions
	@sleep 2
	@echo "Creating Pub/Sub topic..."
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
	@echo "🚀 Crawlify Local Development Quick Start"
	@echo ""
	@echo "Terminal 1 - Pub/Sub Emulator:"
	@echo "  make pubsub-local"
	@echo ""
	@echo "Terminal 2 - Setup topics (after emulator starts):"
	@echo "  make setup-pubsub-topics"
	@echo ""
	@echo "Terminal 3 - Orchestrator:"
	@echo "  make run-orchestrator-local"
	@echo ""
	@echo "Terminal 4 - Worker:"
	@echo "  make run-worker-local"
	@echo ""
	@echo "Terminal 5 - Testing:"
	@echo "  make test-workflow"
	@echo ""
