# Crawlify Migration Guide

## Overview

This document outlines the strategy for migrating from the monolithic Crawlify application to the new microservices architecture.

## Migration Strategy

### Approach: Greenfield + Gradual Migration

We're building the new architecture from scratch while keeping the old monolith running. Features will be migrated incrementally.

### Project Structure

```
Crawlify/
├── [OLD] Monolithic Application
│   ├── api/
│   ├── cmd/crawler/
│   ├── internal/
│   ├── frontend/
│   └── ...
│
└── microservices/          [NEW] Microservices Architecture
    ├── orchestrator/
    ├── worker/
    ├── shared/
    └── infrastructure/
```

## Migration Phases

### Phase 1: Core Infrastructure ✅ COMPLETED

**Goal**: Set up basic microservices infrastructure

- [x] Create microservices folder structure
- [x] Implement shared libraries (models, config, logger, database, cache, queue)
- [x] Create orchestrator service skeleton
- [x] Create worker service skeleton
- [x] Setup Docker containers
- [x] Create docker-compose for local development

### Phase 2: Basic Workflow Execution (NEXT)

**Goal**: Implement core workflow execution capability

**Tasks**:
1. **Database Schema** (reuse from monolith)
   - Copy relevant migration files
   - Create orchestrator repository layer
   - Create worker repository layer

2. **Orchestrator Features**:
   - Workflow CRUD endpoints
   - Execution initiation
   - Task publishing to Pub/Sub
   - Execution status tracking

3. **Worker Features**:
   - Task consumption from Pub/Sub
   - Browser pool integration
   - Basic navigation and extraction
   - Result storage

**What to Migrate**:
- `internal/workflow/parser.go` → `worker/internal/executor/`
- `internal/browser/pool.go` → `worker/internal/browser/`
- `internal/storage/workflow_repository.go` → `orchestrator/internal/repository/`

### Phase 3: Advanced Features

**Goal**: Add caching, batching, and optimization

**Tasks**:
1. Implement Redis caching for workflows
2. Implement URL deduplication in Redis
3. Add async batch writer for extracted data
4. Integrate Cloud Storage for large payloads
5. Add PgBouncer if needed

**What to Migrate**:
- URL deduplication logic → Redis-based
- Extracted items storage → Cloud Storage + metadata DB

### Phase 4: Browser Automation

**Goal**: Full browser automation support

**Tasks**:
1. Integrate Playwright in workers
2. Implement all node types (click, type, scroll, etc.)
3. Browser profile support
4. Proxy integration

**What to Migrate**:
- `internal/browser/` → `worker/internal/browser/`
- `internal/extraction/` → `worker/internal/extraction/`

### Phase 5: Plugin System

**Goal**: Plugin marketplace and execution

**Tasks**:
1. Plugin management APIs in orchestrator
2. Plugin loading and execution in workers
3. Plugin marketplace UI

**What to Migrate**:
- `internal/plugin/` → Split between orchestrator and worker
- Plugin database schema
- Plugin execution logic

### Phase 6: Monitoring & AI Features

**Goal**: Health checks and auto-fix

**Tasks**:
1. Monitoring service in orchestrator
2. Snapshot capture in workers
3. AI auto-fix integration
4. Analytics and dashboards

**What to Migrate**:
- `internal/monitoring/` → `orchestrator/internal/monitoring/`
- `internal/ai/` → `orchestrator/internal/ai/`

### Phase 7: Frontend Integration

**Goal**: Connect frontend to new backend

**Tasks**:
1. Update API client to point to orchestrator
2. Test all workflows
3. Update UI for any new features
4. Performance testing

### Phase 8: Production Deployment

**Goal**: Deploy to GCP Cloud Run

**Tasks**:
1. Setup GCP infrastructure (Cloud SQL, Memorystore, Pub/Sub)
2. Deploy orchestrator to Cloud Run
3. Deploy workers to Cloud Run
4. Setup monitoring and alerts
5. Gradual traffic migration (blue-green deployment)
6. Decommission old monolith

## Code Migration Guidelines

### 1. Shared Code

Place reusable code in `microservices/shared/`:
- Data models
- Configuration
- Database utilities
- Cache utilities
- Queue utilities
- Logger

### 2. Service-Specific Code

**Orchestrator** (API, management, orchestration):
- API handlers
- Workflow management
- Execution orchestration
- Task distribution
- Monitoring aggregation
- User management

**Worker** (execution only):
- Task execution
- Browser automation
- Data extraction
- URL discovery
- Result storage

### 3. Database Access Patterns

**Orchestrator**:
- CRUD operations for workflows
- Execution status updates
- Monitoring data aggregation
- Analytics queries

**Worker**:
- Read workflow config (cached)
- Write extracted data (batched)
- Update execution progress (batched)
- Minimal direct DB access

### 4. Caching Strategy

**What to Cache (Redis)**:
- Workflow configurations (1 hour TTL)
- Browser profiles (1 hour TTL)
- Plugin metadata (1 hour TTL)
- URL deduplication (24 hour TTL)
- Execution metadata (temporary)

**What NOT to Cache**:
- Extracted data (use Cloud Storage)
- Large payloads (> 1MB)
- Frequently changing data

## Testing Strategy

### Local Development

```bash
# Start all services
cd microservices/infrastructure/docker-compose
docker-compose up -d

# Check services
docker-compose ps

# View logs
docker-compose logs -f orchestrator
docker-compose logs -f worker

# Test orchestrator
curl http://localhost:8080/health

# Stop services
docker-compose down
```

### Integration Testing

1. Create test workflow via orchestrator API
2. Start execution
3. Verify tasks appear in Pub/Sub
4. Verify workers process tasks
5. Check extracted data in storage
6. Verify execution status updates

### Load Testing

1. Create workflow with 1000 URLs
2. Monitor worker scaling
3. Monitor database connections
4. Monitor Pub/Sub queue depth
5. Measure end-to-end latency

## Rollback Strategy

If issues arise during migration:

1. **Immediate Rollback**: Route traffic back to old monolith
2. **Database Rollback**: Both systems use same database
3. **Gradual Rollback**: Reduce traffic to new system incrementally

## Success Criteria

Each phase is complete when:

- ✅ All features work as expected
- ✅ Unit tests pass
- ✅ Integration tests pass
- ✅ Performance meets targets
- ✅ No increase in error rates
- ✅ Documentation updated

## Timeline Estimate

- Phase 1: ✅ Complete
- Phase 2: 2 weeks
- Phase 3: 1 week
- Phase 4: 2 weeks
- Phase 5: 1 week
- Phase 6: 2 weeks
- Phase 7: 1 week
- Phase 8: 1 week

**Total**: ~10 weeks (2.5 months)

## Next Steps

1. Review this migration plan
2. Start Phase 2: Basic Workflow Execution
3. Setup Pub/Sub emulator for local testing
4. Implement orchestrator workflow APIs
5. Implement worker task processor
