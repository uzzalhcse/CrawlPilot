# Postman Collection Guide

This guide will help you test the Crawlify API using the provided Postman collection.

## 📦 Import Files

Two files are available in the `/docs` folder:

1. **Crawlify_API.postman_collection.json** - Complete API collection
2. **Crawlify_API.postman_environment.json** - Environment variables

## 🚀 Quick Start

### 1. Import Collection

1. Open Postman
2. Click **Import** button
3. Select `docs/Crawlify_API.postman_collection.json`
4. Click **Import**

### 2. Import Environment

1. Click the **Environments** tab (left sidebar)
2. Click **Import**
3. Select `docs/Crawlify_API.postman_environment.json`
4. Click **Import**
5. Select **Crawlify Local** environment from the dropdown (top right)

### 3. Start the Server

Make sure Crawlify is running:

```bash
# Option 1: Docker
docker-compose up -d

# Option 2: Binary
./crawlify

# Option 3: Go run
go run cmd/crawler/main.go
```

### 4. Set Up Database

If you haven't already, run the migrations:

```bash
# Using psql
PGPASSWORD=your_password psql -h localhost -U postgres -d crawlify -f migrations/001_initial_schema.up.sql

# Or using make
make migrate-up
```

## 📝 Collection Structure

The collection includes:

### 1. Health Check
- **Health Check** - Verify API is running

### 2. Workflows
- **Create Workflow - Simple Crawler** - Creates a basic workflow with auto-save workflow_id
- **Create Workflow - E-commerce Scraper** - Creates an advanced workflow
- **List All Workflows** - Get all workflows
- **List Active Workflows** - Filter by status with pagination
- **Get Workflow by ID** - Get specific workflow
- **Update Workflow** - Modify workflow configuration
- **Update Workflow Status to Active** - Activate workflow
- **Update Workflow Status to Paused** - Pause workflow
- **Delete Workflow** - Remove workflow

### 3. Executions
- **Start Workflow Execution** - Begin crawling (auto-saves execution_id)
- **Get Execution Status** - Check if execution is running
- **Get Execution Queue Stats** - Detailed queue statistics
- **Stop Execution** - Halt a running execution

### 4. Complete Workflow Flow
A sequential flow demonstrating the complete lifecycle:
1. Health Check
2. Create Workflow
3. Activate Workflow
4. Start Execution
5. Check Execution Status
6. Get Queue Stats

## 🎯 Testing Workflow

### Complete Test Flow

Run these requests in order:

#### Step 1: Health Check
```
GET {{base_url}}/health
```
Expected: Status 200, `"status": "healthy"`

#### Step 2: Create Workflow
```
POST {{base_url}}/api/v1/workflows
```
- Automatically saves `workflow_id` to environment
- Status: 201 Created
- Response includes workflow ID

#### Step 3: Activate Workflow
```
PATCH {{base_url}}/api/v1/workflows/{{workflow_id}}/status
Body: {"status": "active"}
```
- Automatically uses saved `workflow_id`
- Status: 200 OK

#### Step 4: Start Execution
```
POST {{base_url}}/api/v1/workflows/{{workflow_id}}/execute
```
- Automatically saves `execution_id` to environment
- Status: 202 Accepted
- Workflow starts running in background

#### Step 5: Monitor Progress
```
GET {{base_url}}/api/v1/executions/{{execution_id}}
GET {{base_url}}/api/v1/executions/{{execution_id}}/stats
```
- Check if execution is running
- View queue statistics

## 🧪 Test Scenarios

### Scenario 1: Basic Workflow Lifecycle

1. Run "Health Check" ✓
2. Run "Create Workflow - Simple Crawler" ✓
3. Run "Update Workflow Status to Active" ✓
4. Run "Start Workflow Execution" ✓
5. Run "Get Execution Status" (multiple times) ✓
6. Run "Get Execution Queue Stats" ✓

### Scenario 2: E-commerce Scraper

1. Run "Create Workflow - E-commerce Scraper"
2. Update workflow status to "active"
3. Start execution
4. Monitor queue stats to see URLs being processed

### Scenario 3: Workflow Management

1. Create multiple workflows
2. List all workflows
3. Filter by status (active, draft, etc.)
4. Update workflow configurations
5. Delete unused workflows

## 📊 Expected Results

### Health Check
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "time": "2025-11-17T10:00:00Z"
}
```

### Create Workflow
```json
{
  "id": "uuid-here",
  "name": "Test Simple Crawler",
  "status": "draft",
  "created_at": "2025-11-17T10:00:00Z"
}
```

### Start Execution
```json
{
  "message": "Workflow execution started",
  "execution_id": "uuid-here",
  "workflow_id": "uuid-here"
}
```

### Execution Stats
```json
{
  "execution_id": "uuid-here",
  "stats": {
    "pending": 5,
    "processing": 1,
    "completed": 10,
    "failed": 0
  },
  "pending_count": 5
}
```

## 🔧 Environment Variables

The collection uses these variables (auto-managed):

- `base_url` - API base URL (default: http://localhost:8080)
- `workflow_id` - Auto-saved from create workflow
- `execution_id` - Auto-saved from start execution

### Manual Environment Setup

If needed, manually set:

1. Click **Environments** → **Crawlify Local**
2. Edit values:
   - `base_url`: Your API URL
   - `workflow_id`: Specific workflow ID
   - `execution_id`: Specific execution ID

## 📝 Test Scripts

The collection includes automated test scripts:

### Create Workflow Request
```javascript
if (pm.response.code === 201) {
    var jsonData = pm.response.json();
    pm.environment.set("workflow_id", jsonData.id);
    pm.test("Workflow created successfully", function () {
        pm.expect(jsonData.id).to.not.be.empty;
        pm.expect(jsonData.status).to.eql("draft");
    });
}
```

### Start Execution Request
```javascript
if (pm.response.code === 202) {
    var jsonData = pm.response.json();
    pm.environment.set("execution_id", jsonData.execution_id);
    pm.test("Execution started", function () {
        pm.expect(jsonData.execution_id).to.not.be.empty;
    });
}
```

## 🐛 Troubleshooting

### Connection Refused
**Problem**: Cannot connect to API

**Solution**:
- Ensure server is running: `curl http://localhost:8080/health`
- Check correct port in environment (default: 8080)
- Verify no firewall blocking

### Database Error
**Problem**: "relation does not exist" error

**Solution**:
```bash
# Run migrations
psql -h localhost -U postgres -d crawlify -f migrations/001_initial_schema.up.sql
```

### Workflow Must Be Active
**Problem**: "Workflow must be in active status to execute"

**Solution**:
1. Run "Update Workflow Status to Active" request first
2. Then run "Start Workflow Execution"

### Invalid Workflow Configuration
**Problem**: Workflow validation failed

**Solution**:
- Check workflow JSON syntax
- Ensure all required fields are present
- Validate node types and parameters
- Review examples in `/examples` folder

## 📚 Advanced Usage

### Custom Workflows

Create your own workflow by modifying the request body:

```json
{
  "name": "My Custom Crawler",
  "description": "Custom description",
  "config": {
    "start_urls": ["https://your-site.com"],
    "max_depth": 2,
    "url_discovery": [...],
    "data_extraction": [...]
  }
}
```

### Pagination

List workflows with pagination:

```
GET {{base_url}}/api/v1/workflows?limit=10&offset=20
```

### Status Filtering

Filter workflows by status:

```
GET {{base_url}}/api/v1/workflows?status=active
```

Valid statuses: `draft`, `active`, `paused`, `archived`

## 🔍 Monitoring Executions

### Real-time Monitoring

1. Start an execution
2. Repeatedly call "Get Execution Stats" to monitor:
   - URLs being processed
   - Completion rate
   - Failure rate

### Example Monitoring Script

Create a Postman test to repeatedly check status:

```javascript
// In Tests tab
setTimeout(function() {
    pm.sendRequest(pm.request, function(err, res) {
        // Check again after 2 seconds
    });
}, 2000);
```

## 📖 Additional Resources

- **API Documentation**: `/docs/API.md`
- **Workflow Guide**: `/docs/WORKFLOW_GUIDE.md`
- **Example Workflows**: `/examples` directory
- **Quick Start**: `/docs/QUICK_START.md`

## ✅ Verification Checklist

After importing and setting up:

- [ ] Collection imported successfully
- [ ] Environment imported and selected
- [ ] Server is running (health check passes)
- [ ] Database migrations completed
- [ ] Can create workflows
- [ ] Can activate workflows
- [ ] Can start executions
- [ ] Can monitor execution progress

## 🎉 Success!

You should now be able to:
- ✅ Create and manage workflows via Postman
- ✅ Start and monitor executions
- ✅ Test all API endpoints
- ✅ Understand the complete workflow lifecycle

Happy crawling! 🕷️
