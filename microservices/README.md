# Crawlify Microservices Architecture

This directory contains the new cloud-native microservices implementation of Crawlify.

## Architecture Overview

The system is split into two main services:

1. **Orchestrator**: API server, workflow management, monitoring, and task distribution
2. **Worker**: Stateless workflow execution workers

## Directory Structure

```
microservices/
├── orchestrator/          # Orchestrator service
│   ├── cmd/              # Application entry point
│   ├── internal/         # Internal packages
│   ├── api/              # HTTP handlers
│   ├── Dockerfile        # Container image
│   └── go.mod            # Dependencies
│
├── worker/               # Worker service
│   ├── cmd/              # Application entry point
│   ├── internal/         # Internal packages
│   ├── Dockerfile        # Container image
│   └── go.mod            # Dependencies
│
├── shared/               # Shared libraries
│   ├── models/           # Data models
│   ├── config/           # Configuration
│   ├── cloudtasks/       # Cloud Tasks client
│   ├── pubsub/           # Pub/Sub client
│   └── storage/          # Database interfaces
│
├── infrastructure/       # Infrastructure as Code
│   ├── terraform/        # Terraform configurations
│   ├── kubernetes/       # K8s manifests (if needed)
│   └── docker-compose/   # Local development
│
└── docs/                 # Documentation
    ├── architecture.md
    ├── deployment.md
    └── migration.md
```

## Development Principles

1. **Clean Architecture**: Clear separation of concerns
2. **Dependency Injection**: Testable and maintainable code
3. **Interface-Driven**: Programming to interfaces, not implementations
4. **Cloud-Native**: Designed for GCP Cloud Run and serverless
5. **12-Factor App**: Follow 12-factor app principles
6. **Observability**: Structured logging, metrics, and tracing

## Tech Stack

- **Language**: Go 1.24
- **Framework**: Fiber v2
- **Database**: PostgreSQL via pgx/v5
- **Cache**: Redis (Memorystore)
- **Queue**: Google Cloud Pub/Sub
- **Storage**: Google Cloud Storage
- **Monitoring**: Cloud Monitoring & Logging

## Getting Started

See individual service READMEs:
- [Orchestrator README](./orchestrator/README.md)
- [Worker README](./worker/README.md)

## Migration Strategy

We're migrating features from the monolithic application incrementally:

1. Core workflow execution (✅ Phase 1)
2. Browser automation and extraction
3. Plugin system
4. Monitoring and health checks
5. AI auto-fix features
6. Analytics and reporting

See [MIGRATION.md](../MIGRATION.md) for detailed migration plan.
