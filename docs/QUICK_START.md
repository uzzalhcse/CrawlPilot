# Quick Start Guide

Get Crawlify up and running in 5 minutes!

## Option 1: Docker (Recommended)

The fastest way to get started:

```bash
# Clone the repository
git clone https://github.com/uzzalhcse/crawlify.git
cd crawlify

# Start all services
docker-compose up -d

# Wait for services to be ready (about 30 seconds)
docker-compose logs -f crawlify

# Check if it's running
curl http://localhost:8080/health
```

That's it! Your Crawlify instance is now running at `http://localhost:8080`

## Option 2: Local Development

### 1. Install Prerequisites

```bash
# macOS
brew install postgresql@14 go

# Ubuntu/Debian
sudo apt-get update
sudo apt-get install postgresql-14 golang-1.21

# Start PostgreSQL
brew services start postgresql@14  # macOS
sudo systemctl start postgresql    # Linux
```

### 2. Set Up Database

```bash
# Create database
createdb crawlify

# Run migrations
psql crawlify < migrations/001_initial_schema.up.sql
```

### 3. Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install Playwright browsers
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium
```

### 4. Run the Application

```bash
# Run directly
go run cmd/crawler/main.go

# Or build and run
go build -o crawlify cmd/crawler/main.go
./crawlify
```

## Create Your First Workflow

### 1. Create a Simple Workflow

Create a file `my_workflow.json`:

```json
{
  "name": "My First Crawler",
  "description": "Crawls a simple website",
  "config": {
    "start_urls": ["https://example.com"],
    "max_depth": 2,
    "rate_limit_delay": 1000,
    "url_discovery": [
      {
        "id": "extract_links",
        "type": "extract_links",
        "name": "Find all links",
        "params": {
          "selector": "a[href]"
        }
      }
    ],
    "data_extraction": [
      {
        "id": "get_title",
        "type": "extract",
        "name": "Get page title",
        "params": {
          "selector": "h1",
          "type": "text",
          "transform": [
            {"type": "trim"}
          ]
        },
        "output_key": "title"
      },
      {
        "id": "get_content",
        "type": "extract",
        "name": "Get main content",
        "params": {
          "selector": "p",
          "type": "text",
          "multiple": true
        },
        "output_key": "paragraphs"
      }
    ],
    "storage": {
      "type": "database",
      "table_name": "crawled_pages"
    }
  }
}
```

### 2. Create the Workflow via API

```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @my_workflow.json
```

You'll get a response with the workflow ID:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "My First Crawler",
  "status": "draft",
  ...
}
```

### 3. Activate the Workflow

```bash
curl -X PATCH http://localhost:8080/api/v1/workflows/550e8400-e29b-41d4-a716-446655440000/status \
  -H "Content-Type: application/json" \
  -d '{"status": "active"}'
```

### 4. Start Execution

```bash
curl -X POST http://localhost:8080/api/v1/workflows/550e8400-e29b-41d4-a716-446655440000/execute
```

Response:

```json
{
  "message": "Workflow execution started",
  "execution_id": "660e8400-e29b-41d4-a716-446655440000",
  "workflow_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### 5. Monitor Progress

```bash
# Check execution status
curl http://localhost:8080/api/v1/executions/660e8400-e29b-41d4-a716-446655440000

# Get detailed stats
curl http://localhost:8080/api/v1/executions/660e8400-e29b-41d4-a716-446655440000/stats
```

## Using Example Workflows

Crawlify comes with ready-to-use example workflows:

### Simple Crawler

```bash
# Convert YAML to JSON and create workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d "$(cat examples/simple_crawler.yaml)"
```

### E-commerce Scraper

```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d "$(cat examples/ecommerce_scraper.yaml)"
```

### Interactive Form

```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d "$(cat examples/interactive_form.yaml)"
```

## Common Tasks

### List All Workflows

```bash
curl http://localhost:8080/api/v1/workflows
```

### Get Specific Workflow

```bash
curl http://localhost:8080/api/v1/workflows/{workflow_id}
```

### Update Workflow

```bash
curl -X PUT http://localhost:8080/api/v1/workflows/{workflow_id} \
  -H "Content-Type: application/json" \
  -d @updated_workflow.json
```

### Delete Workflow

```bash
curl -X DELETE http://localhost:8080/api/v1/workflows/{workflow_id}
```

### Stop Execution

```bash
curl -X DELETE http://localhost:8080/api/v1/executions/{execution_id}
```

## Viewing Extracted Data

Extracted data is stored in the PostgreSQL database:

```bash
# Connect to database
psql crawlify

# View extracted data
SELECT * FROM extracted_data ORDER BY extracted_at DESC LIMIT 10;

# View queue status
SELECT status, COUNT(*) FROM url_queue GROUP BY status;

# View workflow executions
SELECT * FROM workflow_executions ORDER BY started_at DESC;
```

## Troubleshooting

### Service Not Starting

```bash
# Check logs
docker-compose logs crawlify

# Restart services
docker-compose restart
```

### Database Connection Error

```bash
# Verify PostgreSQL is running
docker-compose ps postgres

# Check database
docker-compose exec postgres psql -U postgres -c "\l"
```

### Playwright Browser Issues

```bash
# Reinstall browsers in container
docker-compose exec crawlify npx playwright install chromium
```

## Next Steps

1. **Read the Documentation**
   - [API Reference](API.md)
   - [Workflow Guide](WORKFLOW_GUIDE.md)
   - [Deployment Guide](DEPLOYMENT.md)

2. **Customize Configuration**
   - Edit `config.yaml` for your needs
   - Adjust browser pool size
   - Configure rate limiting

3. **Create Advanced Workflows**
   - Use complex selectors
   - Add data transformations
   - Implement authentication
   - Handle dynamic content

4. **Monitor and Scale**
   - Set up health checks
   - Configure logging
   - Add monitoring
   - Scale horizontally

## Useful Commands

```bash
# Make commands (if using Makefile)
make help           # Show all available commands
make build          # Build the application
make run            # Run locally
make test           # Run tests
make docker-up      # Start with Docker
make docker-down    # Stop Docker services
make docker-logs    # View logs

# Docker commands
docker-compose up -d              # Start in background
docker-compose down               # Stop services
docker-compose logs -f crawlify   # Follow logs
docker-compose ps                 # List services
docker-compose restart crawlify   # Restart API

# Database commands
psql crawlify                     # Connect to database
psql crawlify -c "SELECT NOW()"   # Run query
pg_dump crawlify > backup.sql     # Backup database
```

## Example Use Cases

### 1. News Aggregator

Crawl multiple news sites and extract articles:

```yaml
start_urls:
  - "https://news-site-1.com"
  - "https://news-site-2.com"

data_extraction:
  - id: "extract_articles"
    type: "extract"
    params:
      selector: "article"
      multiple: true
      fields:
        title: {selector: "h1", type: "text"}
        author: {selector: ".author", type: "text"}
        date: {selector: "time", type: "attr", attribute: "datetime"}
        content: {selector: ".article-body", type: "text"}
```

### 2. Price Monitor

Monitor product prices across e-commerce sites:

```yaml
data_extraction:
  - id: "extract_price"
    type: "extract"
    params:
      selector: ".product-price"
      type: "text"
      transform:
        - type: "regex"
          params: {pattern: "[^0-9.]", replacement: ""}
        - type: "parse_float"
```

### 3. Job Listings

Scrape job postings:

```yaml
data_extraction:
  - id: "extract_jobs"
    type: "extract"
    params:
      selector: ".job-listing"
      multiple: true
      fields:
        title: {selector: ".job-title", type: "text"}
        company: {selector: ".company-name", type: "text"}
        location: {selector: ".job-location", type: "text"}
        salary: {selector: ".salary", type: "text"}
        link: {selector: "a", type: "href"}
```

## Getting Help

- **Documentation**: Check the `/docs` folder
- **Issues**: Report bugs on GitHub
- **Examples**: See `/examples` folder for more workflows

Happy crawling! 🕷️
