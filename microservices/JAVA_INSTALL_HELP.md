# Java Installation Failed - Alternative Solutions

The Pub/Sub emulator requires Java, but installation failed due to network issues.

## Option 1: Install Java Manually (Recommended)

```bash
# Try with a different mirror or later
sudo apt-get update
sudo apt-get install -y default-jre

# Or install OpenJDK directly
sudo apt-get install -y openjdk-21-jre

# Verify installation
java -version
```

After Java is installed, try again:
```bash
make pubsub-local
```

## Option 2: Run Without Pub/Sub (Local Testing)

You can test the orchestrator API without Pub/Sub initially:

### Start Orchestrator Only
```bash
cd microservices
make run-orchestrator-local
```

### Test API Endpoints
```bash
# Create workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/aqua_site_crawler_phase.json | jq

# List workflows  
curl http://localhost:8080/api/v1/workflows | jq
```

**Note**: Without Pub/Sub emulator:
- ✅ Workflow CRUD works
- ✅ Execution creation works
- ❌ Worker won't receive tasks (no message queue)
- ❌ Actual crawling won't happen

## Option 3: Use Docker for Pub/Sub Only

If you have Docker but don't want full Docker Compose:

```bash
# Run only Pub/Sub emulator in Docker
docker run -d --name pubsub-emulator \
  -p 8085:8085 \
 google/cloud-sdk:emulators \
  gcloud beta emulators pubsub start --host-port=0.0.0.0:8085

# Setup topics
sleep 5
make setup-pubsub-topics

# Stop when done
docker stop pubsub-emulator
docker rm pubsub-emulator
```

## Option 4: Test End-to-End with Full Docker

Go back to Docker Compose (fix the lint errors first):

```bash
cd microservices
# Fix the lint errors in worker code
# Then:
make docker-up
```

## Recommended Next Steps

1. **Install Java** (simplest solution):
   ```bash
   sudo apt-get install -y default-jre
   java -version
   make pubsub-local
   ```

2. **OR skip Pub/Sub for now** and test orchestrator API only

3. **OR use Docker** for the full stack

Choose the option that works best for your setup!
