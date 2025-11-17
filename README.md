# Crawlify - Scalable Web Crawler API

A production-ready, workflow-based web crawler API built with Go, featuring DAG-based execution, PostgreSQL storage, and Playwright automation for complex browser interactions.

## Features

- **Workflow-Based Configuration**: Define crawling workflows using JSON/YAML with declarative syntax
- **DAG Execution Engine**: Execute workflows as Directed Acyclic Graphs for maximum flexibility
- **Browser Automation**: Full Playwright integration supporting complex interactions (click, scroll, type, etc.)
- **URL Queue with Deduplication**: PostgreSQL-backed queue with SHA-256 URL hashing
- **Browser Pool Management**: Efficient resource management with context sharing
- **Extraction Engine**: Powerful CSS/XPath selectors with data transformations
- **REST API**: Complete CRUD operations and execution control
- **Production-Ready**: Comprehensive logging, error handling, and graceful shutdown

## Architecture

```
┌─────────────────┐
│   REST API      │  (Fiber Framework)
└────────┬────────┘
         │
    ┌────┴─────┐
    │ Workflow │  (Parser + DAG Builder)
    │  Engine  │
    └────┬─────┘
         │
    ┌────┴──────────────────────┐
    │                           │
┌───▼──────┐           ┌────────▼──────┐
│ Browser  │           │  URL Queue    │
│  Pool    │           │ (PostgreSQL)  │
└───┬──────┘           └───────────────┘
    │
┌───▼──────────┐
│ Interaction  │
│   Engine     │
└──────────────┘
    │
┌───▼──────────┐
│ Extraction   │
│   Engine     │
└──────────────┘
```

## Tech Stack

- **Backend**: Go 1.21+
- **HTTP Framework**: Fiber v2
- **Browser Automation**: Playwright-go
- **Database**: PostgreSQL 14+
- **HTML Parsing**: goquery
- **Logging**: Zap
- **Configuration**: Viper

## Quick Start

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 14+
- Node.js (for Playwright browsers)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/uzzalhcse/crawlify.git
cd crawlify
```

2. Install dependencies:
```bash
go mod download
```

3. Set up PostgreSQL database:
```bash
createdb crawlify
psql crawlify < migrations/001_initial_schema.up.sql
```

4. Configure the application:
```bash
cp config.yaml config.local.yaml
# Edit config.local.yaml with your settings
```

5. Run the application:
```bash
go run cmd/crawler/main.go
```

The API server will start on `http://localhost:8080`

## Configuration

Edit `config.yaml` to customize:

```yaml
server:
  port: 8080
  host: 0.0.0.0

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  database: crawlify

browser:
  pool_size: 5
  headless: true
  timeout: 30000

crawler:
  max_depth: 3
  max_retries: 3
  concurrent_workers: 5
```

## Workflow Definition

Workflows are defined in YAML or JSON format with two main sections:

### Basic Structure

```yaml
start_urls:
  - "https://example.com"

max_depth: 2
max_pages: 100
rate_limit_delay: 1000

url_discovery:
  - id: "extract_links"
    type: "extract_links"
    params:
      selector: "a[href]"

data_extraction:
  - id: "extract_title"
    type: "extract"
    params:
      selector: "h1"
      type: "text"
```

### Node Types

#### URL Discovery Nodes

- `fetch`: Fetch a page
- `extract_links`: Extract links from page
- `filter_urls`: Filter URLs by patterns
- `navigate`: Navigate to a URL

#### Interaction Nodes

- `click`: Click an element
- `scroll`: Scroll the page
- `type`: Type text into an input
- `hover`: Hover over an element
- `wait`: Wait for a duration
- `wait_for`: Wait for an element
- `screenshot`: Take a screenshot

#### Extraction Nodes

- `extract`: Extract data using selectors
- `extract_text`: Extract text content
- `extract_attr`: Extract element attributes
- `extract_json`: Parse JSON from page

#### Transformation Nodes

- `transform`: Transform data
- `filter`: Filter data
- `map`: Map data
- `validate`: Validate data

## API Endpoints

### Workflows

- `POST /api/v1/workflows` - Create a workflow
- `GET /api/v1/workflows` - List all workflows
- `GET /api/v1/workflows/:id` - Get workflow by ID
- `PUT /api/v1/workflows/:id` - Update workflow
- `DELETE /api/v1/workflows/:id` - Delete workflow
- `PATCH /api/v1/workflows/:id/status` - Update workflow status

### Executions

- `POST /api/v1/workflows/:id/execute` - Start workflow execution
- `GET /api/v1/executions/:execution_id` - Get execution status
- `DELETE /api/v1/executions/:execution_id` - Stop execution
- `GET /api/v1/executions/:execution_id/stats` - Get queue statistics

### Health

- `GET /health` - Health check endpoint

## Testing with Postman

A complete Postman collection is available for easy API testing:

📦 **Import Files:**
- `docs/Crawlify_API.postman_collection.json` - Full API collection with 20+ requests
- `docs/Crawlify_API.postman_environment.json` - Environment variables

**Features:**
- ✅ Auto-save workflow_id and execution_id
- ✅ Pre-configured test scripts
- ✅ Complete workflow lifecycle examples
- ✅ Multiple workflow templates

See [POSTMAN_GUIDE.md](docs/POSTMAN_GUIDE.md) for detailed instructions.

## Usage Examples

### Create a Workflow

```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/simple_crawler.yaml
```

### Start Execution

```bash
curl -X POST http://localhost:8080/api/v1/workflows/{workflow_id}/execute
```

### Check Execution Status

```bash
curl http://localhost:8080/api/v1/executions/{execution_id}
```

## Example Workflows

See the `examples/` directory for complete workflow examples:

- **simple_crawler.yaml**: Basic web crawling
- **ecommerce_scraper.yaml**: E-commerce product scraping with complex interactions
- **interactive_form.yaml**: Form filling and submission

## Data Transformations

Support for various data transformations:

- `trim`: Remove whitespace
- `lowercase`/`uppercase`: Case conversion
- `regex`: Regular expression replacement
- `replace`: String replacement
- `split`/`join`: Array operations
- `parse_int`/`parse_float`: Type conversion

## Deployment

### Docker

```bash
docker build -t crawlify .
docker run -p 8080:8080 crawlify
```

### Docker Compose

```bash
docker-compose up -d
```

## Development

### Project Structure

```
crawlify/
├── cmd/
│   └── crawler/          # Application entry point
├── api/
│   ├── handlers/         # HTTP handlers
│   └── middleware/       # HTTP middleware
├── internal/
│   ├── browser/          # Browser pool & interactions
│   ├── config/           # Configuration management
│   ├── extraction/       # Data extraction engine
│   ├── logger/           # Logging utilities
│   ├── queue/            # URL queue management
│   ├── storage/          # Database layer
│   └── workflow/         # Workflow engine
├── pkg/
│   └── models/           # Data models
├── migrations/           # Database migrations
├── examples/             # Example workflows
└── docs/                 # Documentation
```

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o crawlify cmd/crawler/main.go
```

## Performance Considerations

- **Browser Pool**: Adjust `pool_size` based on available resources
- **Concurrent Workers**: Set `concurrent_workers` to control parallelism
- **Rate Limiting**: Use `rate_limit_delay` to avoid overwhelming targets
- **Database Connections**: Configure `max_connections` for optimal throughput
- **Queue Optimization**: Indexes ensure efficient deduplication

## Security

- Always respect `robots.txt`
- Use rate limiting to avoid overwhelming servers
- Implement proper authentication for production deployments
- Sanitize extracted data before storage
- Use environment variables for sensitive configuration

## Limitations

- JavaScript execution requires Playwright browsers
- Memory usage scales with browser pool size
- PostgreSQL required (no SQLite support)

## Contributing

Contributions are welcome! Please submit pull requests or open issues.

## License

MIT License - see LICENSE file for details

## Acknowledgments

- [Playwright](https://playwright.dev/) for browser automation
- [Fiber](https://gofiber.io/) for HTTP framework
- [goquery](https://github.com/PuerkitoBio/goquery) for HTML parsing
