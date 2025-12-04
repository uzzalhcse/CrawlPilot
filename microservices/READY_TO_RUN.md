# 🚀 Crawlify Microservices - Ready to Run!

## ✅ Setup Complete

All dependencies installed:
- ✅ PostgreSQL database (crawlify)
- ✅ Redis running
- ✅ Go dependencies downloaded
- ✅ Playwright browsers (installing...)

## 🎯 Next Steps - Run Services

### Step 1: Start Pub/Sub Emulator (Terminal 1)
```bash
cd microservices
make pubsub-local
```
Wait for: `Server started, listening on 8085`

### Step 2: Setup Pub/Sub Topics (Terminal 2)
```bash
cd microservices
sleep 5  # Wait for emulator to fully start
make setup-pubsub-topics
```

### Step 3: Start Orchestrator (Terminal 3)
```bash
cd microservices
make run-orchestrator-local
```
Wait for: `Orchestrator starting`

### Step 4: Start Worker (Terminal 4)
```bash
cd microservices
make run-worker-local
```
Wait for: `Worker service started`

### Step 5: Test! (Terminal 5)
```bash
cd microservices

# Create workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/aqua_site_crawler_phase.json | jq

# Copy the "id" from response, then:
WORKFLOW_ID="<paste-id-here>"

# Start execution
curl -X POST http://localhost:8080/api/v1/workflows/$WORKFLOW_ID/execute | jq

# Monitor (copy execution id)
EXECUTION_ID="<paste-execution-id-here>"
curl http://localhost:8080/api/v1/executions/$EXECUTION_ID | jq
```

## 📊 Service URLs

| Service | URL |
|---------|-----|
| Orchestrator API | http://localhost:8080 |
| Worker | http://localhost:8081 |
| Pub/Sub Emulator | http://localhost:8085 |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |

## 🎉 You're Ready to Crawl!

Run `make local-quickstart` to see this guide again.
