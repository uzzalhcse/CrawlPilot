.PHONY: help build run test clean docker-build docker-up docker-down migrate-up migrate-down install-deps

# Variables
APP_NAME=crawlify
BINARY_NAME=crawlify
DOCKER_IMAGE=crawlify:latest
GO=go
GOFLAGS=-v

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install-deps: ## Install Go dependencies
	$(GO) mod download
	$(GO) mod verify

build: ## Build the application
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) ./cmd/crawler

run: ## Run the application
	$(GO) run ./cmd/crawler/main.go

test: ## Run tests
	$(GO) test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests and show coverage
	$(GO) tool cover -html=coverage.out

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME)
	rm -f coverage.out

docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE) .

docker-up: ## Start Docker containers
	docker-compose up -d

docker-down: ## Stop Docker containers
	docker-compose down

docker-logs: ## View Docker logs
	docker-compose logs -f crawlify

migrate-up: ## Run database migrations up
	psql -U postgres -d crawlify -f migrations/001_initial_schema.up.sql

migrate-down: ## Run database migrations down
	psql -U postgres -d crawlify -f migrations/001_initial_schema.down.sql

lint: ## Run linters
	golangci-lint run ./...

fmt: ## Format code
	$(GO) fmt ./...

vet: ## Run go vet
	$(GO) vet ./...

deps-update: ## Update dependencies
	$(GO) get -u ./...
	$(GO) mod tidy

dev: ## Run in development mode
	air

# Database operations
db-create: ## Create database
	createdb crawlify

db-drop: ## Drop database
	dropdb crawlify

db-reset: db-drop db-create migrate-up ## Reset database

# Docker operations
docker-shell: ## Open shell in running container
	docker exec -it crawlify-api /bin/sh

docker-clean: ## Clean Docker resources
	docker-compose down -v
	docker rmi $(DOCKER_IMAGE) || true

# Production
build-prod: ## Build for production
	CGO_ENABLED=0 GOOS=linux $(GO) build -a -installsuffix cgo -ldflags '-s -w' -o $(BINARY_NAME) ./cmd/crawler

# Quick start
quick-start: docker-up ## Quick start with Docker
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Crawlify is running at http://localhost:8080"
	@echo "PostgreSQL is running at localhost:5432"
	@echo "Check health: curl http://localhost:8080/health"
