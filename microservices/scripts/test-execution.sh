#!/bin/bash
# Test workflow execution

echo "🧪 Testing Crawlify Microservices"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check services
echo "1️⃣ Checking services..."
if curl -s http://localhost:8080/health > /dev/null; then
    echo -e "${GREEN}✓${NC} Orchestrator (8080)"
else
    echo -e "${RED}✗${NC} Orchestrator not responding"
    exit 1
fi

if curl -s http://localhost:8081/health > /dev/null; then
    echo -e "${GREEN}✓${NC} Worker (8081)"
else
    echo -e "${RED}✗${NC} Worker not responding"
    exit 1
fi

echo ""
echo "2️⃣ Creating workflow..."
WORKFLOW_ID=$(curl -s -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/aqua_site_crawler_phase.json | jq -r '.id')

if [ -z "$WORKFLOW_ID" ] || [ "$WORKFLOW_ID" == "null" ]; then
    echo -e "${RED}✗${NC} Failed to create workflow"
    exit 1
fi

echo -e "${GREEN}✓${NC} Workflow created: $WORKFLOW_ID"

echo ""
echo "3️⃣ Starting execution..."
EXECUTION=$(curl -s -X POST http://localhost:8080/api/v1/workflows/$WORKFLOW_ID/execute)
EXECUTION_ID=$(echo "$EXECUTION" | jq -r '.execution_id')

if [ -z "$EXECUTION_ID" ] || [ "$EXECUTION_ID" == "null" ]; then
    echo -e "${RED}✗${NC} Failed to start execution"
    echo "Response: $EXECUTION"
    exit 1
fi

echo -e "${GREEN}✓${NC} Execution started: $EXECUTION_ID"
echo "$EXECUTION" | jq

echo ""
echo "4️⃣ Monitoring execution (15 seconds)..."
for i in {1..5}; do
    sleep 3
    STATUS=$(curl -s http://localhost:8080/api/v1/executions/$EXECUTION_ID | jq)
    echo "[$i/5] Status:"
    echo "$STATUS" | jq
    
    # Check if completed
    STATE=$(echo "$STATUS" | jq -r '.status')
    if [ "$STATE" == "completed" ] || [ "$STATE" == "failed" ]; then
        echo ""
        echo -e "${GREEN}✓${NC} Execution finished with status: $STATE"
        break
    fi
done

echo ""
echo "5️⃣ Final status:"
curl -s http://localhost:8080/api/v1/executions/$EXECUTION_ID | jq

echo ""
echo "🔍 Check worker terminal for task processing logs"
echo "🔍 Check orchestrator terminal for API request logs"
