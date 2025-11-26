# Crawlify Backend Features

> **Platform Overview**: Crawlify is an advanced web scraping and data extraction platform built with Go (Golang), featuring a visual workflow designer, plugin marketplace, AI-powered auto-fix capabilities, and comprehensive monitoring system.

---

## Table of Contents
- [Core Architecture](#core-architecture)
- [Workflow Engine](#workflow-engine)
- [Plugin System](#plugin-system)
- [AI-Powered Features](#ai-powered-features)
- [Monitoring & Health Checks](#monitoring--health-checks)
- [Browser Automation](#browser-automation)
- [REST API](#rest-api)
- [Database Architecture](#database-architecture)
- [Configuration Management](#configuration-management)
- [Scheduling & Automation](#scheduling--automation)

---

## Core Architecture

### Technology Stack
- **Language**: Go 1.24.0
- **Web Framework**: Fiber v2 (high-performance HTTP framework)
- **Database**: PostgreSQL with pgx/v5 driver
- **Browser Automation**: Playwright-Go
- **Logging**: Uber Zap (structured logging)
- **Configuration**: Viper (YAML/JSON/ENV based)
- **HTML Parsing**: GoQuery (jQuery-like syntax)

### Project Structure
```
crawlify/
├── api/                    # HTTP handlers and API layer
├── cmd/crawler/            # Main application entry point
├── internal/               # Internal business logic
│   ├── workflow/          # Workflow execution engine
│   ├── browser/           # Browser pool management
│   ├── plugins/           # Plugin system
│   ├── ai/                # AI integration (Gemini, OpenRouter)
│   ├── monitoring/        # Health check and monitoring
│   ├── storage/           # Database repositories
│   ├── queue/             # URL queue management
│   └── extraction/        # Data extraction logic
├── pkg/models/            # Shared data models
├── migrations/            # Database migrations
└── frontend/              # Vue.js frontend (separate)
```

---

## Workflow Engine

### Visual Workflow Designer
- **DAG-Based Execution**: Directed Acyclic Graph for node dependencies
- **Phase-Based Processing**: Multi-phase workflows with URL routing
- **Real-Time Event Stream**: SSE (Server-Sent Events) for live execution updates
- **Versioning**: Full workflow version history with rollback support

### Workflow Node Types

#### 1. Discovery Nodes
| Node Type | Description | Key Features |
|-----------|-------------|--------------|
| **Navigate** | Navigate to URLs | • Phase routing<br>• URL pattern matching<br>• Depth control |
| **Extract Links** | Discover and extract links | • CSS/XPath selectors<br>• Link filtering<br>• URL normalization<br>• Deduplication |
| **Paginate** | Handle pagination | • Next button clicking<br>• URL-based pagination<br>• Page number substitution<br>• Infinite scroll support |

#### 2. Extraction Nodes
| Node Type | Description | Key Features |
|-----------|-------------|--------------|
| **Extract** | Extract structured data | • Multi-field extraction<br>• CSS/XPath selectors<br>• Data type conversion<br>• Required/optional fields |
| **Extract JSON** | Parse JSON APIs | • JSONPath queries<br>• Schema validation |

#### 3. Interaction Nodes
| Node Type | Description | Key Features |
|-----------|-------------|--------------|
| **Click** | Click elements | • Element selection<br>• Wait for navigation<br>• Retry logic |
| **Type** | Input text | • Form filling<br>• Keyboard simulation |
| **Scroll** | Scroll page | • Infinite scroll<br>• Lazy loading trigger |
| **Hover** | Hover over elements | • Tooltip activation<br>• Dropdown menus |
| **Wait** | Wait for duration | • Fixed delays<br>• Network idle wait |
| **Wait For** | Wait for element | • Element visibility<br>• DOM ready states |
| **Screenshot** | Capture screenshots | • Full page<br>• Specific element<br>• PNG format |
| **Conditional** | Conditional execution | • Element existence checks<br>• Branching logic |
| **Sequence** | Execute child nodes | • Sequential execution<br>• Error handling |

#### 4. Plugin Nodes
- **Dynamic Plugin Loading**: Load external Go plugins at runtime
- **Custom Node Types**: Extend workflow capabilities
- **Plugin Parameters**: Pass configuration to plugins
- **Plugin Marketplace Integration**: Install from marketplace

### Execution Features
- **URL Queue Management**: Priority-based queue with deduplication
- **Hierarchy Tracking**: Parent-child URL relationships
- **Retry Logic**: Configurable retry count and delay
- **Concurrent Processing**: Multi-worker execution (configurable)
- **Context Propagation**: Share data between nodes
- **Error Handling**: Graceful failures with detailed logging

---

## Plugin System

### Plugin Marketplace
- **Plugin Repository**: Centralized plugin registry
- **Version Management**: Semantic versioning support
- **Plugin Discovery**: Search, filter, and browse plugins
- **Categories**: Organize plugins by category
- **Ratings & Reviews**: User feedback system
- **Popular Plugins**: Trending and most-used plugins

### Plugin Lifecycle
1. **Creation**: Define plugin metadata (name, description, category)
2. **Versioning**: Publish new versions with changelogs
3. **Installation**: Install plugins to workspace
4. **Execution**: Load and execute during workflow
5. **Uninstallation**: Remove plugins from workspace

### Plugin Architecture
- **Go Plugin System**: Dynamic loading of `.so` files
- **Loader Interface**: `LoadPlugin(path string) (Plugin, error)`
- **Executor Interface**: `Execute(ctx context.Context, params map[string]interface{})` 
- **Input/Output Validation**: Type-safe parameter handling

### Plugin Features
| Feature | Description |
|---------|-------------|
| **Hot Reload** | Load plugins without restart |
| **Isolated Execution** | Sandboxed plugin execution |
| **Custom Parameters** | Flexible configuration |
| **Error Handling** | Plugin-level error isolation |
| **Logging Integration** | Structured logging for plugins |

---

## AI-Powered Features

### Auto-Fix Service
> AI-powered selector repair and workflow optimization using Google Gemini and OpenRouter

#### Capabilities
- **Broken Selector Detection**: Analyze failed extractions
- **Smart Selector Suggestions**: AI-generated CSS/XPath selectors
- **Multi-Provider Support**: Gemini (2.5-flash) and OpenRouter (Gemini 2.0-flash-exp)
- **Visual Analysis**: Screenshot + DOM analysis
- **Confidence Scoring**: 0-1 confidence for each suggestion
- **Alternative Selectors**: Multiple fallback options

#### AI Workflow
```
Snapshot Capture → AI Analysis → Suggestion Generation → Review → Apply → Verify → Revert (if needed)
```

#### Fix Suggestion Lifecycle
1. **Analyze**: AI examines DOM snapshot and screenshot
2. **Suggest**: Generate selector recommendations
3. **Review**: Human approval (pending/approved/rejected)
4. **Apply**: Auto-update workflow configuration
5. **Verify**: Test new selector against snapshot
6. **Revert**: Rollback if verification fails

#### AI Key Management
- **API Key Rotation**: Multiple keys for rate limit management
- **Usage Tracking**: Request/success/failure counts
- **Cooldown Management**: Automatic rate limit handling
- **Provider Selection**: Switch between Gemini/OpenRouter

### AI Integration Points
| Component | AI Function |
|-----------|-------------|
| **Selector Repair** | Fix broken CSS/XPath selectors |
| **Text Generation** | Generate descriptions, summaries |
| **Image Analysis** | Visual understanding of pages |
| **Error Diagnosis** | Suggest fixes for runtime errors |

---

## Monitoring & Health Checks

### Validation Engine
> Continuous monitoring and validation of workflow health

#### Features
- **Baseline Comparison**: Compare current vs. baseline execution
- **Phase-by-Phase Validation**: Track each workflow phase
- **Snapshot Capture**: DOM + screenshot for debugging
- **Element Validation**: Verify selector accuracy
- **Status Code Checks**: Monitor HTTP responses
- **Console Log Capture**: Record browser console errors

### Monitoring Reports
| Field | Description |
|-------|-------------|
| **Status** | `healthy`, `degraded`, `failed` |
| **Duration** | Execution time in milliseconds |
| **Results** | Per-phase validation results |
| **Summary** | Aggregate metrics and statistics |
| **Baseline Comparison** | Deviation from baseline |

### Scheduling
- **Cron-Based Scheduling**: `"@hourly"`, `"0 */6 * * *"`, etc.
- **Automatic Execution**: Background monitoring jobs
- **Notification Integration**: Alert on failures
- **Next Run Calculation**: Smart scheduling logic

### Snapshot System
- **Screenshot Storage**: Full-page PNG captures
- **DOM Snapshots**: Compressed HTML (gzip)
- **Metadata Tracking**: URL, title, status code
- **Selector Verification**: Elements found count
- **Error Messages**: Detailed failure information

---

## Browser Automation

### Browser Pool
- **Playwright Integration**: Chromium/Firefox/WebKit support
- **Connection Pooling**: Reusable browser contexts
- **Headless/Headed Modes**: Configurable display
- **Proxy Support**: Rotation and authentication
- **Context Isolation**: Separate contexts per execution

### Pool Configuration
```yaml
browser:
  pool_size: 5                    # Max concurrent browsers
  headless: false                 # Show browser UI
  timeout: 60000                  # Page timeout (ms)
  max_concurrency: 10             # Max concurrent pages
  context_lifetime: 300           # Context TTL (seconds)
  proxy:
    enabled: true
    server: "82.22.73.66:7272"
    username: "lnvmpyru"
    password: "5un1tb1azapa"
```

### Visual Selector Overlay
- **Interactive Selector Builder**: Visual element picking
- **Selector Preview**: Real-time selector validation
- **Multiple Selector Types**: CSS, XPath, Text, ID, Class
- **Element Highlighting**: Visual feedback
- **Selector Optimization**: Generate robust selectors

### Browser Features
| Feature | Description |
|---------|-------------|
| **Element Selection** | Smart element locators |
| **Field Conversion** | Auto-convert data types |
| **Interactions** | Click, type, scroll, hover |
| **Network Monitoring** | Track requests/responses |
| **Cookie Management** | Session persistence |
| **Local Storage** | State management |

---

## REST API

### API Endpoints

#### Workflow Management
```http
POST   /api/v1/workflows                 # Create workflow
GET    /api/v1/workflows                 # List workflows
GET    /api/v1/workflows/:id             # Get workflow
PUT    /api/v1/workflows/:id             # Update workflow
DELETE /api/v1/workflows/:id             # Delete workflow
```

#### Workflow Versions
```http
GET    /api/v1/workflows/:id/versions           # List versions
POST   /api/v1/workflows/:id/versions           # Create version
GET    /api/v1/workflows/:id/versions/:version  # Get version
POST   /api/v1/workflows/:id/versions/:version/restore  # Restore version
```

#### Execution Management
```http
POST   /api/v1/workflows/:id/execute     # Start execution
GET    /api/v1/executions                # List executions
GET    /api/v1/executions/:id            # Get execution
DELETE /api/v1/executions/:id            # Cancel execution
GET    /api/v1/executions/:id/stream     # SSE event stream
GET    /api/v1/executions/:id/items      # Get extracted items
```

#### Node Tree & Analytics
```http
GET    /api/v1/executions/:id/node-tree  # Execution tree visualization
GET    /api/v1/analytics/executions      # Execution analytics
GET    /api/v1/analytics/nodes           # Node performance stats
```

#### Monitoring & Health Checks
```http
POST   /api/v1/monitoring/:workflow_id/run        # Run health check
GET    /api/v1/monitoring/:workflow_id/reports    # List reports
GET    /api/v1/monitoring/reports/:id             # Get report
DELETE /api/v1/monitoring/reports/:id             # Delete report
POST   /api/v1/monitoring/:workflow_id/baseline   # Set baseline
```

#### Monitoring Schedules
```http
POST   /api/v1/schedules                 # Create schedule
GET    /api/v1/schedules/:workflow_id    # Get schedule
PUT    /api/v1/schedules/:id             # Update schedule
DELETE /api/v1/schedules/:id             # Delete schedule
```

#### AI Auto-Fix
```http
POST   /api/v1/snapshots/:id/analyze            # Analyze snapshot
GET    /api/v1/snapshots/:id/suggestions        # List suggestions
POST   /api/v1/suggestions/:id/approve          # Approve suggestion
POST   /api/v1/suggestions/:id/reject           # Reject suggestion
POST   /api/v1/suggestions/:id/apply            # Apply suggestion
POST   /api/v1/suggestions/:id/revert           # Revert suggestion
```

#### Plugin Marketplace
```http
# Plugin CRUD
POST   /api/v1/plugins                   # Create plugin
GET    /api/v1/plugins                   # List plugins
GET    /api/v1/plugins/:slug             # Get plugin
PUT    /api/v1/plugins/:id               # Update plugin
DELETE /api/v1/plugins/:id               # Delete plugin

# Plugin Versions
POST   /api/v1/plugins/:id/versions      # Publish version
GET    /api/v1/plugins/:id/versions      # List versions
GET    /api/v1/plugins/:id/versions/:v   # Get version

# Installation
POST   /api/v1/plugins/:id/install       # Install plugin
DELETE /api/v1/plugins/:id/uninstall     # Uninstall plugin
GET    /api/v1/plugins/installed         # List installed

# Reviews & Discovery
POST   /api/v1/plugins/:id/reviews       # Create review
GET    /api/v1/plugins/:id/reviews       # List reviews
GET    /api/v1/plugins/categories        # List categories
GET    /api/v1/plugins/search            # Search plugins
GET    /api/v1/plugins/popular           # Popular plugins
```

#### Visual Selector
```http
POST   /api/v1/selector/generate         # Generate selector from URL
```

### API Features
- **CORS Enabled**: Cross-origin support
- **Error Handling**: Standardized error responses
- **Pagination**: Offset/limit pagination
- **Filtering**: Query parameter filtering
- **Real-Time Updates**: SSE for live data
- **JSON Responses**: Consistent JSON format

---

## Database Architecture

### Core Tables

#### Workflows & Executions
| Table | Purpose | Key Fields |
|-------|---------|------------|
| `workflows` | Workflow definitions | id, name, config (JSONB), status |
| `workflow_versions` | Version history | workflow_id, version, config, change_reason |
| `workflow_executions` | Execution tracking | workflow_id, status, started_at, metadata |

#### URL Processing
| Table | Purpose | Key Fields |
|-------|---------|------------|
| `url_queue` | URL queue with hierarchy | execution_id, url, depth, parent_url_id, phase_id, marker |
| `node_executions` | Node execution logs | execution_id, node_id, status, duration_ms, error |
| `extracted_items` | Extracted data storage | execution_id, url_id, schema_name, data (JSONB) |

#### Monitoring
| Table | Purpose | Key Fields |
|-------|---------|------------|
| `monitoring_reports` | Health check reports | workflow_id, status, results (JSONB), baseline_report_id |
| `monitoring_schedules` | Scheduled checks | workflow_id, schedule (cron), enabled |
| `monitoring_snapshots` | DOM/screenshot snapshots | report_id, node_id, screenshot_path, dom_snapshot_path |

#### AI & Plugins
| Table | Purpose | Key Fields |
|-------|---------|------------|
| `ai_api_keys` | API key rotation | api_key, provider, total_requests, is_rate_limited |
| `fix_suggestions` | AI-generated fixes | snapshot_id, suggested_selector, confidence_score, status |
| `plugins` | Plugin marketplace | slug, name, category, downloads, rating |
| `plugin_versions` | Plugin versions | plugin_id, version, binary_path, changelog |
| `plugin_installations` | Installed plugins | plugin_id, version, installed_at |
| `plugin_reviews` | User reviews | plugin_id, rating, comment |

### Database Features
- **JSONB Storage**: Flexible schema for configs and data
- **Indexes**: Optimized for common queries
- **Foreign Keys**: Referential integrity
- **Cascading Deletes**: Automatic cleanup
- **Triggers**: Auto-update timestamps
- **Views**: `execution_stats` for analytics
- **Full-Text Search**: GIN indexes on JSONB

---

## Configuration Management

### Configuration File (config.yaml)
```yaml
server:                              # HTTP server settings
  port: 8080
  host: 0.0.0.0
  read_timeout: 30
  write_timeout: 30
  shutdown_timeout: 10

database:                            # PostgreSQL settings
  host: localhost
  port: 5432
  database: crawlify
  max_connections: 25
  max_idle_conns: 5
  conn_max_lifetime: 300

browser:                             # Browser pool config
  pool_size: 5
  headless: false
  timeout: 60000
  max_concurrency: 10
  proxy:
    enabled: true
    server: "proxy_host:port"
    username: "user"
    password: "pass"

crawler:                             # Crawler settings
  max_depth: 3
  user_agent: "Crawlify/1.0"
  respect_robots_txt: true
  max_retries: 3
  retry_delay: 1000
  concurrent_workers: 5
  queue_check_interval: 1000

ai:                                  # AI provider config
  gemini_model: "gemini-2.5-flash"
  openrouter_model: "google/gemini-2.0-flash-exp:free"
  provider: "gemini"                 # gemini | openrouter
  enabled: true
```

### Configuration Features
- **Environment Variables**: Override via env vars
- **Hot Reload**: Watch for config changes (via Viper)
- **Validation**: Type-safe configuration parsing
- **Multiple Formats**: YAML, JSON, TOML support
- **Defaults**: Sensible default values

---

## Scheduling & Automation

### Workflow Scheduling
- **Cron Expressions**: Standard cron syntax
- **Recurring Executions**: Hourly, daily, weekly, custom
- **Next Run Calculation**: Automatic scheduling
- **Enable/Disable**: Toggle schedules on/off
- **Last Run Tracking**: Execution history

### Monitoring Schedules
- **Automated Health Checks**: Periodic validation
- **Notification Configuration**: Alert routing
- **Baseline Comparison**: Automated regression detection
- **Schedule Management**: Per-workflow scheduling

---

## Additional Features

### Data Export
- **JSON Export**: Extracted items as JSON
- **JSONB Querying**: PostgreSQL JSONB queries
- **Bulk Export**: Export all execution data
- **Schema Filtering**: Filter by schema name

### Error Handling & Logging
- **Structured Logging**: Uber Zap with JSON format
- **Error Stack Traces**: Detailed error context
- **Request Logging**: HTTP request/response logs
- **Browser Console Logs**: Capture JS errors
- **Retry Logic**: Configurable retry strategies

### Performance Optimization
- **Connection Pooling**: Database and browser pools
- **Concurrent Processing**: Multi-worker execution
- **Efficient Querying**: Indexed database queries
- **Context Reuse**: Browser context caching
- **GZIP Compression**: Compressed DOM snapshots

### Security
- **CORS Configuration**: Cross-origin controls
- **Proxy Support**: Rotating proxy integration
- **API Key Management**: Secure key storage
- **Input Validation**: Request validation middleware
- **Error Sanitization**: Safe error messages

---

## Development Tools

### Makefile Commands
```bash
make run                # Start server
make build              # Build binary
make test               # Run tests
make migrate-up         # Apply migrations
make migrate-down       # Rollback migrations
```

### Testing Support
- **Unit Tests**: Internal package tests
- **Integration Tests**: End-to-end workflow tests
- **Mock Browser**: Test browser interactions
- **Test Fixtures**: Sample workflows and data

---

## Summary

Crawlify is a **production-ready, enterprise-grade web scraping platform** with:

✅ **19+ Built-in Workflow Nodes** (discovery, extraction, interaction)  
✅ **Plugin Marketplace** with dynamic loading and versioning  
✅ **AI-Powered Auto-Fix** for broken selectors  
✅ **Monitoring & Health Checks** with baseline comparison  
✅ **Browser Automation** with Playwright pooling  
✅ **Comprehensive REST API** (40+ endpoints)  
✅ **PostgreSQL Database** with JSONB flexibility  
✅ **Real-Time Event Streaming** (SSE)  
✅ **Workflow Versioning** with rollback support  
✅ **Scheduling & Automation** via cron  
✅ **Visual Selector Builder** for no-code workflows  
✅ **Multi-Provider AI** (Gemini, OpenRouter)  

**Tech Stack**: Go 1.24, Fiber v2, PostgreSQL, Playwright, Gemini AI
