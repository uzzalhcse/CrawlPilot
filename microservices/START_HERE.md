# ✅ Local Development Setup - COMPLETE

## What's Ready:

1. ✅ PostgreSQL database `crawlify` created
2. ✅ Database schema applied (4 tables with indexes)
3. ✅ Redis running
4. ✅ Go dependencies downloaded (orchestrator & worker)
5. ✅ Playwright Chromium installed

## 🚀 Start Services (5 Terminals)

### Terminal 1: Pub/Sub Emulator
```bash
cd microservices
make pubsub-local
```
**Wait for**: `Server started, listening on 8085`

### Terminal 2: Setup Pub/Sub (do this once)
```bash
cd microservices
sleep 5
make setup-pubsub-topics
```
**Output**: `✅ Pub/Sub setup complete!`

### Terminal 3: Orchestrator
```bash
cd microservices
make run-orchestrator-local
```
**Wait for**: `Orchestrator starting`

### Terminal 4: Worker
```bash
cd microservices
make run-worker-local
```
**Wait for**: `Worker service started` or `Subscribing to Pub/Sub`

### Terminal 5: Test
```bash
cd microservices

# 1. Create workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/aqua_site_crawler_phase.json | jq

# Copy the "id" from response

# 2. Start execution
WORKFLOW_ID="paste-id-here"
curl -X POST http://localhost:8080/api/v1/workflows/$WORKFLOW_ID/execute | jq

# Copy execution "id"

# 3. Monitor progress
EXECUTION_ID="paste-execution-id-here"
watch -n 2 "curl -s http://localhost:8080/api/v1/executions/$EXECUTION_ID | jq"
```

## 📍 Service URLs

- Orchestrator: http://localhost:8080
- Pub/Sub Emulator: http://localhost:8085
- PostgreSQL: localhost:5432 (user: crawlify, pass: dev_password)
- Redis: localhost:6379

## 🎉 System is Ready!

All components installed and configured. Just start the 4 services and test!
