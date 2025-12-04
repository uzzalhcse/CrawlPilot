# Crawlify: Scalable Web Scraping Architecture

Unified implementation plan for scaling Crawlify across **Local Development**, **GCP Cloud**, and **On-Premises** environments with consistent architecture.

---

## Executive Summary

**Goal**: Scale Crawlify to handle millions of URLs with a **unified architecture** that works identically across all environments.

**Deployment Options**:
- 🏠 **Local Development**: Single machine with Docker
- ☁️ **GCP Cloud**: Managed Instance Group with autoscaling
- 🏢 **On-Premises**: Multiple VMs/servers in your datacenter

**Key Principle**: Same codebase, same architecture, different infrastructure.

**Data Storage**: Extracted items stored directly in PostgreSQL database for simplicity and consistency.

## Architecture Design

### Unified Architecture Across All Environments

**Core Components** (same everywhere):
1. **API Server**: Accepts workflow execution requests
2. **Worker Pool**: Processes URLs through workflow phases  
3. **Queue System**: Distributes URLs to workers
4. **Database**: PostgreSQL for metadata
5. **Cache**: Redis for deduplication
6. **Storage**: File system or cloud storage for extracted data

**What Changes Per Environment**:
- **Queue Backend**: PostgreSQL (local) vs Cloud Tasks (GCP)
- **Storage**: Local disk (local/on-prem) vs Cloud Storage (GCP)
- **Scaling**: Manual (local) vs Autoscaling (GCP) vs Load Balancer (on-prem)
- **IP Strategy**: Single IP (local) vs Multiple IPs (cloud/on-prem)

### Environment Comparison

| Component | Local Dev | On-Premises | GCP Cloud |
|-----------|-----------|-------------|------------|
| **Worker** | Same binary | Same binary | Same binary |
| **Queue** | PostgreSQL table | Redis + PostgreSQL | Cloud Tasks |
| **Database** | PostgreSQL :5433 | PostgreSQL cluster | Cloud SQL |
| **Cache** | Redis :6380 | Redis cluster | Memorystore |
| **Storage** | Database (extracted_items) | Database (extracted_items) | Database (extracted_items) |
| **Scaling** | 1 worker | 2-50 workers | 2-1000 workers |
| **Load Balancer** | None | Nginx/HAProxy | GCP Load Balancer |
| **IP Per Worker** | 1 IP | 1 IP per VM | 1 IP per VM |

### Core Architecture

```mermaid
graph TB
    subgraph "Local Development"
        LocalAPI[API Server<br/>localhost:8080]
        LocalWorker[Worker Process<br/>Local machine]
        LocalDB[(PostgreSQL<br/>localhost:5432)]
        LocalRedis[(Redis<br/>localhost:6379)]
    end
    
    subgraph "GCP Production"
        API[API Server<br/>Cloud Run]
        Tasks[Cloud Tasks<br/>crawlify-urls]
        
        subgraph "Worker Pool - Managed Instance Group"
            W1[Worker VM 1<br/>n2-standard-4]
            W2[Worker VM 2<br/>n2-standard-4]
            W3[Worker VM N<br/>n2-standard-4]
        end
        
        SQL[(Cloud SQL<br/>PostgreSQL)]
        Redis[(Memorystore<br/>Redis)]
        GCS[(Cloud Storage<br/>Extracted Data)]
        
        LB[HTTP Load Balancer<br/>Task Receiver]
    end
    
    LocalAPI -.-> LocalWorker
    LocalWorker --> LocalDB
    LocalWorker --> LocalRedis
    
    API --> Tasks
    Tasks --> LB
    LB --> W1
    LB --> W2
    LB --> W3
    W1 --> SQL
    W2 --> SQL
    W3 --> SQL
    W1 --> Redis
    W2 --> Redis
    W3 --> Redis
    W1 --> GCS
    W2 --> GCS
    W3 --> GCS
    W1 --> Tasks
    W2 --> Tasks
    W3 --> Tasks
    
    style W1 fill:#34a853,stroke:#333,color:#fff
    style W2 fill:#34a853,stroke:#333,color:#fff
    style W3 fill:#34a853,stroke:#333,color:#fff
```

---

## Detailed Architecture Diagrams

### Complete System Architecture

```mermaid
graph TB
    subgraph "Development Environment - Local Machine"
        DevAPI[API Server<br/>:8080]
        DevWorker[Worker Process<br/>3-5 goroutines]
        DevPG[(PostgreSQL<br/>:5433)]
        DevRedis[(Redis<br/>:6380)]
        DevBrowser[Browser Pool<br/>3 contexts]
    end
    
    subgraph "Production Environment - GCP"
        subgraph "Frontend Layer"
            Users[Users/Frontend<br/>Next.js]
        end
        
        subgraph "API Layer - Cloud Run"
            ProdAPI[API Server<br/>Auto-scaling]
        end
        
        subgraph "Orchestration Layer"
            Tasks[Cloud Tasks Queue<br/>crawlify-urls<br/>Max 1000 concurrent]
        end
        
        subgraph "Load Balancing"
            ILB[Internal Load Balancer<br/>:8081]
            HealthCheck[Health Check<br/>/health endpoint]
        end
        
        subgraph "Compute Layer - Managed Instance Group"
            direction LR
            W1[Worker VM 1<br/>n2-standard-4<br/>IP: x.x.x.1]
            W2[Worker VM 2<br/>n2-standard-4<br/>IP: x.x.x.2]
            W3[Worker VM N<br/>n2-standard-4<br/>IP: x.x.x.N]
        end
        
        subgraph "Data Layer"
            CloudSQL[(Cloud SQL<br/>PostgreSQL 14<br/>db-n1-standard-2)]
            Memorystore[(Memorystore Redis<br/>10GB Standard)]
            GCS[(Cloud Storage<br/>crawlify-data bucket)]
        end
        
        subgraph "Monitoring"
            CloudMon[Cloud Monitoring<br/>Dashboards & Alerts]
            CloudLog[Cloud Logging<br/>Structured Logs]
        end
    end
    
    %% Local connections
    DevAPI --> DevPG
    DevAPI --> DevRedis
    DevWorker --> DevPG
    DevWorker --> DevRedis
    DevWorker --> DevBrowser
    DevAPI -."Enqueue URLs".-> DevPG
    DevWorker -."Poll Queue".-> DevPG
    
    %% Production connections
    Users --> ProdAPI
    ProdAPI --> Tasks
    ProdAPI --> CloudSQL
    Tasks --> ILB
    ILB --> HealthCheck
    HealthCheck -.-> W1
    HealthCheck -.-> W2
    HealthCheck -.-> W3
    ILB --> W1
    ILB --> W2
    ILB --> W3
    W1 --> CloudSQL
    W2 --> CloudSQL
    W3 --> CloudSQL
    W1 --> Memorystore
    W2 --> Memorystore
    W3 --> Memorystore
    W1 --> GCS
    W2 --> GCS
    W3 --> GCS
    W1 -."Enqueue discovered URLs".-> Tasks
    W2 -."Enqueue discovered URLs".-> Tasks
    W3 -."Enqueue discovered URLs".-> Tasks
    W1 --> CloudMon
    W2 --> CloudMon
    W3 --> CloudMon
    W1 --> CloudLog
    W2 --> CloudLog
    W3 --> CloudLog
    
    style W1 fill:#34a853,stroke:#333,color:#fff
    style W2 fill:#34a853,stroke:#333,color:#fff
    style W3 fill:#34a853,stroke:#333,color:#fff
    style DevWorker fill:#4285f4,stroke:#333,color:#fff
    style Tasks fill:#fbbc04,stroke:#333,color:#000
```

### URL Processing Flow

```mermaid
sequenceDiagram
    autonumber
    participant User
    participant API as API Server<br/>(Cloud Run)
    participant DB as Cloud SQL
    participant Tasks as Cloud Tasks<br/>Queue
    participant LB as Load Balancer
    participant W1 as Worker VM 1<br/>(IP: x.x.x.1)
    participant Redis as Memorystore<br/>Redis
    participant Target as Target Website
    participant GCS as Cloud Storage
    
    %% Workflow start
    User->>API: POST /executions<br/>{workflow_id}
    API->>DB: INSERT execution record<br/>status='running'
    API->>DB: SELECT workflow config<br/>with phases
    
    %% Enqueue start URLs
    loop For each start URL
        API->>Tasks: CreateTask<br/>{url, phase_config, execution_id}<br/>task_name=exec-url_hash
    end
    API-->>User: 202 Accepted<br/>{execution_id}
    
    %% Task processing
    Tasks->>LB: POST /process<br/>(dispatch task)
    LB->>W1: Route to healthy worker
    
    Note over W1: Worker receives task
    
    %% Deduplication check
    W1->>Redis: EXISTS url:exec-123:hash
    alt URL already processed
        Redis-->>W1: 1 (exists)
        W1-->>Tasks: 200 OK (skip)
    else URL not seen
        Redis-->>W1: 0 (not exists)
        W1->>Redis: SETEX url:exec-123:hash<br/>"processing" (24h TTL)
        
        %% Execute workflow phase
        Note over W1: Acquire browser context
        W1->>Target: Navigate to URL<br/>(from IP x.x.x.1)
        Target-->>W1: HTML response
        
        Note over W1: Execute phase nodes<br/>(extract_links or extract_items)
        
        alt Discovery Phase
            Note over W1: Extract links node
            W1->>W1: Parse HTML, extract URLs
            
            loop For each discovered URL
                W1->>Tasks: CreateTask (next phase)<br/>{url, extraction_phase}
            end
            
            W1->>DB: INSERT discovered URLs<br/>metadata
            W1->>Redis: SET url:exec-123:hash<br/>"completed" (7d TTL)
            
        else Extraction Phase
            Note over W1: Extract items node
            W1->>W1: Parse HTML, extract data
            W1->>GCS: Upload JSONL file<br/>exec-123/url_hash.jsonl
            W1->>DB: INSERT extraction metadata<br/>{url, items_count, gcs_path}
            W1->>Redis: SET url:exec-123:hash<br/>"completed" (7d TTL)
        end
        
        W1-->>Tasks: 200 OK
    end
    
    Note over Tasks: Task completed,<br/>remove from queue
```

### Worker Internal Architecture

```mermaid
graph TB
    subgraph "GCE Worker Instance (n2-standard-4)"
        subgraph "HTTP Server :8081"
            TaskReceiver[Task Receiver<br/>POST /process]
            HealthEP[Health Endpoint<br/>GET /health]
            MetricsEP[Metrics Endpoint<br/>GET /metrics]
        end
        
        subgraph "Task Processing Layer"
            TaskQueue[Internal Task Queue<br/>Buffered Channel<br/>Cap: 100]
            WorkerPool[Worker Pool<br/>5-10 Goroutines]
        end
        
        subgraph "Execution Layer"
            PhaseDetector[Phase Detector<br/>Determine which phase]
            NodeExecutor[Node Executor<br/>Run phase nodes]
            URLEnqueuer[URL Enqueuer<br/>Discover & enqueue]
        end
        
        subgraph "Browser Layer"
            BrowserPool[Browser Pool<br/>5 Contexts]
            Context1[Context 1<br/>Chromium]
            Context2[Context 2<br/>Chromium]
            Context3[Context N<br/>Chromium]
        end
        
        subgraph "Client Connections"
            SQLClient[Cloud SQL Client<br/>Connection Pool: 20]
            RedisClient[Redis Client<br/>Connection Pool: 10]
            TasksClient[Cloud Tasks Client<br/>HTTP/2]
            GCSClient[Cloud Storage Client<br/>Resumable Uploads]
        end
    end
    
    subgraph "External Services"
        CloudTasks[Cloud Tasks]
        CloudSQL[(Cloud SQL)]
        Redis[(Memorystore)]
        Storage[(Cloud Storage)]
        TargetSite[Target Websites]
    end
    
    %% Internal flow
    TaskReceiver --> TaskQueue
    TaskQueue --> WorkerPool
    WorkerPool --> PhaseDetector
    PhaseDetector --> NodeExecutor
    NodeExecutor --> BrowserPool
    BrowserPool --> Context1
    BrowserPool --> Context2
    BrowserPool --> Context3
    NodeExecutor --> URLEnqueuer
    
    %% Worker connections
    WorkerPool --> RedisClient
    WorkerPool --> SQLClient
    URLEnqueuer --> TasksClient
    NodeExecutor --> GCSClient
    
    %% External connections
    Context1 -."HTTPS Requests".-> TargetSite
    Context2 -."HTTPS Requests".-> TargetSite
    Context3 -."HTTPS Requests".-> TargetSite
    TasksClient --> CloudTasks
    SQLClient --> CloudSQL
    RedisClient --> Redis
    GCSClient --> Storage
    
    style BrowserPool fill:#e8f5e9,stroke:#2e7d32
    style WorkerPool fill:#e3f2fd,stroke:#1976d2
    style TaskQueue fill:#fff3e0,stroke:#f57c00
```

### Autoscaling Flow

```mermaid
graph TB
    subgraph "Monitoring"
        API[API Server<br/>Publishes Metrics]
        QueueMetric["Queue Depth Metric<br/>custom.googleapis.com/<br/>crawlify/queue_depth"]
        CPUMetric["CPU Utilization<br/>compute.googleapis.com/<br/>instance/cpu/utilization"]
    end
    
    subgraph "Autoscaler"
        AS[Managed Instance Group<br/>Autoscaler]
        Policy[Autoscaling Policy<br/>Min: 2, Max: 50]
        Cooldown[Cooldown Period<br/>5 minutes]
    end
    
    subgraph "Decision Engine"
        QueueCheck{Queue Depth<br/>> 100 per worker?}
        CPUCheck{CPU > 70%?}
        ScaleUp[Scale Up<br/>Add instances]
        ScaleDown[Scale Down<br/>Remove instances]
        NoAction[No Action<br/>Current capacity OK]
    end
    
    subgraph "Instance Group"
        Current[Current: N instances]
        New[Target: N+X instances]
    end
    
    API -->|Every 60s| QueueMetric
    QueueMetric --> AS
    CPUMetric --> AS
    AS --> Policy
    Policy --> QueueCheck
    Policy --> CPUCheck
    
    QueueCheck -->|Yes| ScaleUp
    CPUCheck -->|Yes| ScaleUp
    QueueCheck -->|No| NoAction
    CPUCheck -->|No| NoAction
    
    ScaleUp --> Cooldown
    Cooldown --> New
    New -."Rolling Update".-> Current
    
    NoAction --> Current
    
    Current -->|Queue drops<br/>CPU < 40%| ScaleDown
    ScaleDown --> Cooldown
    
    style ScaleUp fill:#c8e6c9,stroke:#2e7d32
    style ScaleDown fill:#ffccbc,stroke:#d84315
    style NoAction fill:#e0e0e0,stroke:#616161
```

### IP Rotation Strategy

```mermaid
graph LR
    subgraph "Worker VMs with External IPs"
        W1[Worker 1<br/>IP: 35.1.1.1]
        W2[Worker 2<br/>IP: 35.1.1.2]
        W3[Worker 3<br/>IP: 35.1.1.3]
        WN[Worker N<br/>IP: 35.1.1.N]
    end
    
    subgraph "Target Websites"
        Site1[example.com]
        Site2[shop.com]
        Site3[product.io]
    end
    
    subgraph "Rate Limit Tracking - Redis"
        IPHealth["IP Health<br/>ip:35.1.1.1:status<br/>= 'healthy'"]
        DomainCount["Domain Counters<br/>domain:example.com<br/>:ip:35.1.1.1:requests<br/>= 45"]
        RateLimit["Rate Limits<br/>ip:35.1.1.2:rate_limited<br/>= 3 (count)"]
    end
    
    W1 -."Request 1".-> Site1
    W2 -."Request 2".-> Site1
    W3 -."Request 3".-> Site2
    WN -."Request N".-> Site3
    
    W1 --> IPHealth
    W2 --> DomainCount
    W3 --> RateLimit
    
    Site1 -."429 Rate Limited".-> W2
    W2 -->|Mark IP limited| RateLimit
    
    style W1 fill:#c8e6c9,stroke:#2e7d32
    style W2 fill:#ffccbc,stroke:#d84315,stroke-width:3px
    style W3 fill:#c8e6c9,stroke:#2e7d32
    style WN fill:#c8e6c9,stroke:#2e7d32
```

### Data Storage Architecture

```mermaid
graph TB
    subgraph "Worker Processing"
        Worker[Worker VM]
        ExtractedData[Extracted Data<br/>JSON objects]
    end
    
    subgraph "Cloud Storage - JSONL Files"
        Bucket[crawlify-data bucket]
        ExecFolder[/execution-123/]
        File1[url_hash_1.jsonl]
        File2[url_hash_2.jsonl]
        FileN[url_hash_N.jsonl]
    end
    
    subgraph "Cloud SQL - Metadata"
        ExecTable[(workflow_executions)]
        URLTable[(url_queue)]
        ExtractTable[(extracted_items)]
    end
    
    subgraph "BigQuery - Analytics (Optional)"
        Dataset[crawlify_analytics]
        URLsTable[urls_processed]
        ItemsTable[items_extracted]
    end
    
    Worker --> ExtractedData
    ExtractedData -->|Upload JSONL| Bucket
    Bucket --> ExecFolder
    ExecFolder --> File1
    ExecFolder --> File2
    ExecFolder --> FileN
    
    Worker -->|Save metadata| URLTable
    Worker -->|Save summary| ExtractTable
    ExtractTable -."Reference".-> File1
    
    File1 -."Daily export".-> Dataset
    File2 -."Daily export".-> Dataset
    Dataset --> URLsTable
    Dataset --> ItemsTable
    
    style Bucket fill:#fff3e0,stroke:#f57c00
    style Dataset fill:#e1f5fe,stroke:#0277bd
```

---

## Part 1: Local Development Setup

### Prerequisites

**Required Software**:
- Docker Desktop (for PostgreSQL and Redis)
- Go 1.21+
- Make
- Google Cloud SDK (gcloud CLI)

### 1.1 Local Database Setup

**PostgreSQL with Docker** (using port 5433 to avoid conflicts):
```bash
# Start PostgreSQL container on port 5433
docker run -d \
  --name crawlify-postgres-dev \
  -e POSTGRES_USER=crawlify \
  -e POSTGRES_PASSWORD=dev_password \
  -e POSTGRES_DB=crawlify \
  -p 5433:5432 \
  postgres:14-alpine

# Run existing migrations (update connection string)
DATABASE_URL="postgresql://crawlify:dev_password@localhost:5433/crawlify" make migrate-up
```

**Redis with Docker** (using port 6380 to avoid conflicts):
```bash
# Start Redis container on port 6380
docker run -d \
  --name crawlify-redis-dev \
  -p 6380:6379 \
  redis:7-alpine

# Test connection
redis-cli -p 6380 ping
```

### 1.2 Local Worker Configuration

**Environment Configuration** (`config.local.yaml`):
```yaml
server:
  port: 8080
  host: localhost

database:
  host: localhost
  port: 5433  # Use 5433 to avoid conflict with existing PostgreSQL
  user: crawlify
  password: dev_password
  database: crawlify

redis:
  enabled: true
  host: localhost
  port: 6380  # Use 6380 to avoid conflict with existing Redis
  db: 0

browser:
  pool_size: 3
  headless: true
  timeout: 60000

worker:
  mode: local  # local | gcp
  concurrency: 3  # Number of concurrent URL processors
  poll_interval: 1000  # ms between queue polls
  
  # For local development, use direct database queue
  queue_type: postgres  # postgres | cloud_tasks
  
  # For GCP, use Cloud Tasks
  # queue_type: cloud_tasks
  # gcp_project: "your-project-id"
  # gcp_location: "us-central1"
  # gcp_queue: "crawlify-urls"
```

### 1.3 Local Worker Architecture

**Worker Process Structure**:
1. **Main Process**: Manages worker pool
2. **Worker Goroutines**: Process URLs concurrently (3-5 workers)
3. **Browser Pool**: Shared browser contexts (3 contexts)
4. **Queue Poller**: Polls PostgreSQL `url_queue` table every second

**Local Queue Mechanism** (PostgreSQL-based):
- API Server enqueues URLs to `url_queue` table
- Worker polls with `SELECT FOR UPDATE SKIP LOCKED`
- No Cloud Tasks needed locally
- Same workflow execution logic

### 1.4 Local Development Workflow

**Step 1: Start Services**
```bash
# Terminal 1: Start PostgreSQL on port 5433
docker run -d --name crawlify-postgres-dev \
  -e POSTGRES_USER=crawlify \
  -e POSTGRES_PASSWORD=dev_password \
  -e POSTGRES_DB=crawlify \
  -p 5433:5432 \
  postgres:14-alpine

# Start Redis on port 6380
docker run -d --name crawlify-redis-dev \
  -p 6380:6379 \
  redis:7-alpine

# Verify connections
psql postgresql://crawlify:dev_password@localhost:5433/crawlify -c "SELECT 1;"
redis-cli -p 6380 ping

# Terminal 2: Start API server
export DATABASE_PORT=5433 REDIS_PORT=6380
make run-api

# Terminal 3: Start worker
export DATABASE_PORT=5433 REDIS_PORT=6380
make run-worker
```

**Step 2: Create and Run Workflow**
```bash
# Use frontend or API to create workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -d '{"name": "test-workflow", "config": {...}}'

# Start execution
curl -X POST http://localhost:8080/api/v1/executions \
  -d '{"workflow_id": "xxx"}'

# Watch worker logs
tail -f logs/worker.log
```

**Step 3: Monitor Progress**
- Check Redis for deduplication: `redis-cli GET url:exec-123:hash`
- Query database: `SELECT * FROM url_queue WHERE execution_id = 'xxx'`
- View logs in frontend dashboard

---

## Part 2: GCP Cloud Production Setup

### 2.1 Infrastructure Architecture

**Production Components**:
```
GCP Project: crawlify-prod
├── Network
│   └── VPC: crawlify-vpc (auto-mode)
│       └── Subnet: us-central1 (10.128.0.0/20)
│
├── Compute
│   ├── API Server: Cloud Run
│   └── Workers: Managed Instance Group (2-50 VMs)
│       └── Instance Template: n2-standard-4
│
├── Data Stores
│   ├── Cloud SQL: PostgreSQL 14 (db-n1-standard-2)
│   ├── Memorystore: Redis Standard 10GB
│   └── Cloud Storage: crawlify-data bucket
│
├── Orchestration
│   └── Cloud Tasks: crawlify-urls queue
│
└── Networking
    ├── Cloud Load Balancer (for worker HTTP endpoint)
    └── Cloud NAT (for worker outbound traffic)
```

### 2.2 GCE Worker Instance Configuration

**Machine Type**: n2-standard-4
- 4 vCPU
- 16 GB RAM
- Handles 5-10 concurrent URLs
- Cost: ~$150/month per instance

**Disk**:
- Boot disk: 20 GB SSD (enough for OS + app)
- No persistent disk needed (stateless workers)

**Startup Script**:
The VM automatically starts the worker process on boot using a startup script that:
1. Pulls latest worker binary from Cloud Storage
2. Fetches secrets from Secret Manager
3. Starts worker process
4. Registers with health check endpoint

**Network Configuration**:
- VPC: crawlify-vpc
- Subnet: us-central1
- Internal IP: Assigned by DHCP
- **External IP: Ephemeral (unique per instance)**
- Service Account: crawlify-worker@project.iam.gserviceaccount.com

**Why External IPs per Instance?**
- ✅ Each worker has unique IP address
- ✅ 50 workers = 50 different IPs
- ✅ Better rate limit distribution
- ✅ Target websites see requests from different IPs
- ✅ Cost: $3/month per IP (minimal vs compute costs)

### 2.3 Cloud Tasks Integration

**Queue Configuration**:
```
Queue Name: crawlify-urls
Region: us-central1
Dispatch Rate: 100 tasks/second
Max Concurrent: 500 tasks
Task Target Type: HTTP
Target URL: https://workers.crawlify.internal/process
```

**Task Payload** (same as before):
```json
{
  "task_id": "uuid",
  "execution_id": "exec-123",
  "url": "https://example.com/page",
  "phase_config": {...},
  "workflow_config": {...}
}
```

**Worker HTTP Endpoint**:
- Each GCE worker runs an HTTP server on port 8081
- Load Balancer routes Cloud Tasks to available workers
- Workers respond with 200 OK when task completes
- Cloud Tasks handles retries on 5xx errors

### 2.4 Load Balancer Setup

**Purpose**: Distribute Cloud Tasks to worker VMs

**Configuration**:
- Type: Internal HTTP(S) Load Balancer
- Backend: Managed Instance Group (workers)
- Health Check: HTTP /health endpoint on port 8081
- Session Affinity: None (stateless tasks)
- Timeout: 600 seconds (10 minutes)

**URL Map**:
- `/process` → Worker task handler
- `/health` → Health check endpoint
- `/metrics` → Prometheus metrics

### 2.5 Autoscaling Strategy

**Managed Instance Group Autoscaler**:

**Scale UP when**:
- Cloud Tasks queue depth > 100 tasks per current instance
- Average CPU utilization > 70%
- Custom metric: `crawlify/active_workers` < queue_depth / 10

**Scale DOWN when**:
- Queue depth < 50 tasks per instance
- CPU utilization < 40%
- Wait 5 minutes before scaling down (cooldown period)

**Autoscaling Policy**:
```yaml
minReplicas: 2      # Always keep 2 instances running
maxReplicas: 50     # Cap at 50 instances (~$7,500/month max)
coolDownPeriod: 300 # 5 minutes

metrics:
  - type: custom.googleapis.com/crawlify/queue_depth_per_worker
    target: 100
    
  - type: compute.googleapis.com/instance/cpu/utilization
    target: 0.7
    
scalingSchedules:
  - name: peak-hours
    schedule: "0 8 * * *"  # 8 AM UTC
    minReplicas: 10
    
  - name: off-hours
    schedule: "0 20 * * *"  # 8 PM UTC
    minReplicas: 2
```

**Custom Queue Depth Metric**:
- API server publishes queue depth every 60 seconds
- Metric name: `crawlify/queue_depth`
- Labels: `{queue="crawlify-urls"}`
- Autoscaler uses this for intelligent scaling

---

## Part 3: Worker Implementation Details

### 3.1 Worker Process Architecture

**Main Components**:

**1. Task Receiver (HTTP Server)**
- Port: 8081
- Endpoints: `/process`, `/health`, `/metrics`
- Receives tasks from Cloud Tasks via Load Balancer
- Queues tasks to internal channel

**2. Worker Pool**
- 5-10 worker goroutines per VM
- Each goroutine processes one URL at a time
- Shared browser pool (5 contexts)
- Shared database connection pool

**3. Browser Pool Manager**
- Maintains pool of browser contexts
- Reuses contexts for efficiency
- Recreates contexts every 50 URLs (prevent memory leaks)
- Headless Chrome via Playwright

**4. Redis Client**
- Connection pool to Memorystore
- Used for URL deduplication
- Circuit breaker for Redis failures

**5. Cloud SQL Client**
- Connection pool (max 20 connections per worker)
- Saves execution metadata
- **Saves extracted items** to `extracted_items` table
- Updates URL status

### 3.2 Worker Lifecycle

**Startup Sequence**:
1. Load configuration from environment variables
2. Connect to Cloud SQL (retry with exponential backoff)
3. Connect to Redis
4. Initialize Cloud Tasks client
5. Initialize browser pool
6. Start HTTP server
7. Register with Load Balancer health check
8. Start processing tasks

**Task Processing Flow**:
1. Receive POST /process request from Cloud Tasks
2. Parse URL task payload
3. Check Redis deduplication
4. Acquire browser context from pool
5. Navigate to URL
6. Execute phase nodes (extract_links or extract_items)
7. **Save extracted items directly to database** (`extracted_items` table)
8. Enqueue discovered URLs to Cloud Tasks
9. Update execution metadata in database
10. Release browser context
11. Return 200 OK to Cloud Tasks

**Shutdown Sequence** (SIGTERM):
1. Stop accepting new tasks
2. Wait for in-flight tasks (max 5 minutes)
3. Close browser contexts
4. Close database connections
5. Deregister from health check
6. Exit gracefully

### 3.3 Deduplication Strategy

**Three-Level Deduplication** (same as Cloud Run):

**Level 1: Cloud Tasks**
- Task name includes URL hash
- Prevents duplicate enqueuing

**Level 2: Redis (Before Processing)**
- Key: `url:{execution_id}:{url_hash}`
- Check before processing
- Skip if already completed

**Level 3: Database (Fallback)**
- Unique constraint on [(execution_id, url_hash)](file:///home/uzzalh/Workplace/github/uzzalhcse/Crawlify/internal/workflow/executor.go#68-87)
- Catches any duplicates that slip through
```

---

## Part 4: Deployment Strategy

### 4.1 Local to Cloud Migration Path

**Phase 1: Local Development (Week 1)**
- Develop worker logic locally
- Use PostgreSQL queue (no Cloud Tasks)
- Test with small workflows (100 URLs)
- Validate extraction logic

**Phase 2: Local with Cloud Services (Week 2)**
- Switch to Cloud SQL and Memorystore (development instances)
- Integrate Cloud Tasks emulator locally
- Test full flow with GCP services
- Validate cost estimates

**Phase 3: Staging Deployment (Week 3)**
- Deploy to GCP staging project
- 2-5 worker VMs
- Test autoscaling with 10k URLs
- Monitor costs and performance

**Phase 4: Production Rollout (Week 4)**
- Deploy to production project
- Start with 5 workers, max 20
- Gradually migrate workflows
- Monitor and optimize

### 4.2 GCP Infrastructure Provisioning

**Terraform Configuration** (recommended for reproducibility):

**Step 1: Enable APIs**
```bash
gcloud services enable compute.googleapis.com
gcloud services enable sqladmin.googleapis.com
gcloud services enable redis.googleapis.com
gcloud services enable cloudtasks.googleapis.com
gcloud services enable storage.googleapis.com
```

**Step 2: Create VPC and Networking**
- VPC: Auto-mode (easier) or custom subnets
- **NO Cloud NAT**: Each instance gets external IP
- Firewall rules: 
  - Allow health checks from Load Balancer (internal)
  - Allow outbound HTTPS (for web scraping)
  - Deny all inbound except health checks

**Step 3: Provision Data Stores**
- Cloud SQL: db-n1-standard-2, 100GB SSD
- Memorystore Redis: Standard tier, 10GB
- Database will store all extracted data in `extracted_items` table

**Step 4: Create Instance Template**
- Machine type: n2-standard-4
- Boot image: Ubuntu 22.04 LTS
- Startup script: Pull worker binary and start
- Service account: crawlify-worker
- Scopes: Cloud SQL, Cloud Storage, Cloud Tasks

**Step 5: Create Managed Instance Group**
- Instance template: crawlify-worker-template
- Region: us-central1
- Autoscaling: 2-50 instances
- Health check: HTTP /health

**Step 6: Create Load Balancer**
- Type: Internal HTTP(S)
- Backend: Managed Instance Group
- Frontend: Internal IP
- Health check: /health every 10s

**Step 7: Create Cloud Tasks Queue**
- Queue name: crawlify-urls
- Target: Load Balancer internal IP
- Authentication: Service account

### 4.3 Worker Deployment Process

**Build and Deploy Pipeline**:

**Step 1: Build Worker Binary**
```bash
# On local machine or CI/CD
make build-worker
# Produces binary: bin/worker
```

**Step 2: Upload to Cloud Storage**
```bash
# Upload binary to versioned bucket
gsutil cp bin/worker gs://crawlify-artifacts/worker/v1.2.3/worker
```

**Step 3: Update Instance Template**
```bash
# Create new instance template with updated startup script
gcloud compute instance-templates create crawlify-worker-v2 \
  --source-instance-template=crawlify-worker-v1 \
  --metadata=worker-version=v1.2.3
```

**Step 4: Rolling Update**
```bash
# Update instance group with new template (rolling update)
gcloud compute instance-groups managed rolling-action start-update \
  crawlify-workers \
  --version=template=crawlify-worker-v2 \
  --max-surge=3 \
  --max-unavailable=0
```

**Step 5: Monitor Rollout**
- Watch instance group status
- Check worker logs in Cloud Logging
- Verify new instances are healthy
- Rollback if errors detected

---

## Part 4B: On-Premises Deployment

### Infrastructure Requirements

**Minimum Setup**:
- 3-5 Linux servers (Ubuntu 22.04 or similar)
- 4 CPU, 16GB RAM each
- PostgreSQL database server (or cluster)
- Redis server (or cluster)
- Load balancer (Nginx/HAProxy)
- Shared storage (NFS or similar)

### Architecture

```mermaid
graph TB
    subgraph "On-Premises Datacenter"
        LB[Load Balancer<br/>Nginx/HAProxy]
        
        subgraph "Worker Servers"
            W1[Server 1<br/>Worker Process<br/>IP: 10.0.1.1]
            W2[Server 2<br/>Worker Process<br/>IP: 10.0.1.2]
            W3[Server N<br/>Worker Process<br/>IP: 10.0.1.N]
        end
        
        DB[(PostgreSQL<br/>Primary + Replicas)]
        Redis[(Redis<br/>Master + Replicas)]
        NFS[/Shared Storage<br/>NFS/Ceph/]
    end
    
    LB --> W1
    LB --> W2
    LB --> W3
    W1 --> DB
    W2 --> DB
    W3 --> DB
    W1 --> Redis
    W2 --> Redis
    W3 --> Redis
    W1 --> NFS
    W2 --> NFS
    W3 --> NFS
```

### Worker Deployment

**1. Build Worker Binary**:
```bash
# On development machine
make build-worker
scp bin/worker deploy@server1:/opt/crawlify/
```

**2. Systemd Service** (`/etc/systemd/system/crawlify-worker.service`):
```ini
[Unit]
Description=Crawlify Worker
After=network.target

[Service]
Type=simple
User=crawlify
WorkingDirectory=/opt/crawlify
Environment="DATABASE_HOST=db.internal"
Environment="DATABASE_PORT=5432"
Environment="REDIS_HOST=redis.internal"
Environment="REDIS_PORT=6379"
Environment="WORKER_MODE=onpremises"
ExecStart=/opt/crawlify/worker
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

**3. Start Workers**:
```bash
# On each server
sudo systemctl enable crawlify-worker
sudo systemctl start crawlify-worker
sudo systemctl status crawlify-worker
```

### Queue System (Redis-based)

**Use Redis for on-premises** (simpler than Cloud Tasks):

**Configuration** (`config.yaml`):
```yaml
worker:
  mode: onpremises
  queue_type: redis  # Instead of cloud_tasks
  redis_queue:
    host: redis.internal
    port: 6379
    db: 1  # Separate DB for queue
    key_prefix: "crawlify:queue:"
```

**Queue Implementation**:
- API Server pushes URL tasks to Redis list
- Workers pop from Redis list (blocking)
- Same task format as Cloud Tasks
- Redis handles distribution automatically

### Load Balancer Configuration

**Nginx Config** (`/etc/nginx/sites-available/crawlify`):
```nginx
upstream crawlify_workers {
    least_conn;  # Route to least busy worker
    server 10.0.1.1:8081 max_fails=3 fail_timeout=30s;
    server 10.0.1.2:8081 max_fails=3 fail_timeout=30s;
    server 10.0.1.3:8081 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    server_name workers.internal;
    
    location /process {
        proxy_pass http://crawlify_workers;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_timeout 600s;  # 10 minutes for long URLs
    }
    
    location /health {
        proxy_pass http://crawlify_workers;
        proxy_timeout 5s;
    }
}
```

### Storage Configuration

**Database Storage** (recommended):
```yaml
storage:
  type: database
  # All extracted items stored in extracted_items table
  # No file system storage needed
```

**Optional: Export to Files** (for backup/analytics):
```yaml
export:
  enabled: true
  format: jsonl  # or csv
  destination: "/mnt/crawlify-exports"
  schedule: "0 2 * * *"  # Daily at 2 AM
```

### Monitoring (Prometheus + Grafana)

**Worker Metrics Endpoint**:
- Each worker exposes `/metrics` endpoint
- Prometheus scrapes all workers
- Grafana visualizes metrics

**Prometheus Config** (`prometheus.yml`):
```yaml
scrape_configs:
  - job_name: 'crawlify-workers'
    static_configs:
      - targets:
        - '10.0.1.1:8081'
        - '10.0.1.2:8081'
        - '10.0.1.3:8081'
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### Scaling On-Premises

**Manual Scaling**:
1. Provision new server
2. Deploy worker binary
3. Start systemd service
4. Add to load balancer upstream

**Semi-Automated** (Ansible/Terraform):
```bash
# Scale to 10 workers
ansible-playbook -i inventory scale-workers.yml -e "worker_count=10"
```

### On-Premises vs Cloud Comparison

| Aspect | On-Premises | GCP Cloud |
|--------|-------------|------------|
| **Setup Time** | 1-2 days | 1-2 hours |
| **Scaling** | Manual | Automatic |
| **Cost (20 workers)** | Hardware + power | $3,000/month |
| **Maintenance** | You manage | Google manages |
| **Network** | Your IPs | GCP IPs |
| **Reliability** | Your SLA | 99.95% SLA |
| **Best For** | Existing infrastructure | Cloud-first |

---

## Part 5: Monitoring & Operations

### 5.1 Monitoring Dashboard

**Cloud Monitoring Workspace**: `crawlify-prod`

**Dashboard 1: Worker Health**
- Active worker instances (count)
- CPU utilization per instance
- Memory usage per instance
- Unhealthy instances count

**Dashboard 2: Queue Metrics**
- Queue depth over time
- Tasks dispatched per minute
- Task success rate
- Average task duration

**Dashboard 3: Processing Metrics**
- URLs processed per minute
- Active browser contexts
- Extraction success rate
- Data uploaded to Cloud Storage (GB/hour)

**Dashboard 4: Cost Tracking**
- Compute Engine costs (daily)
- Cloud SQL CPU/Memory  
- Redis memory usage
- Cloud Storage operations
- Estimated monthly burn rate

### 5.2 Alerting Configuration

**Critical Alerts** (PagerDuty/Oncall):
- All workers unhealthy for > 5 minutes
- Cloud SQL CPU > 95% for > 5 minutes
- Redis memory > 90%
- Queue depth > 1 million tasks

**Warning Alerts** (Email/Slack):
- Worker instance count at max (50)
- Task error rate > 10%
- Average task duration > 2 minutes
- Cloud Storage upload failures

### 5.3 Log Management

**Cloud Logging Setup**:

**Worker Logs**:
- Severity levels: DEBUG, INFO, WARN, ERROR
- Structured JSON format
- Log retention: 30 days
- Export to BigQuery for analysis

**Important Log Queries**:
```
# Failed URL processing
resource.type="gce_instance"
severity="ERROR"
jsonPayload.event="processing_failed"

# Slow URLs (>60s)
resource.type="gce_instance"
jsonPayload.duration_ms > 60000

# Redis connection errors
resource.type="gce_instance"
jsonPayload.error=~"redis"
```

### 5.4 Performance Optimization

**Browser Context Reuse**:
- Reuse contexts for same domain
- Pool size: 5 contexts per worker
- Recreate after 50 uses

**Connection Pooling**:
- Cloud SQL: 20 connections per worker
- Redis: 10 connections per worker
- HTTP client: Keep-alive enabled

**Batch Operations**:
- Enqueue URLs in batches of 100
- Upload extracted data in 10MB chunks
- Update database in transactions

---

## Part 6: Cost Analysis

### 6.1 Monthly Cost Breakdown (1M URLs/day)

**Compute Engine (Workers)**:
- Average: 20 instances × n2-standard-4
- Cost per instance: $150/month
- **Total**: 20 × $150 = **$3,000/month**

**Cloud SQL**:
- Instance: db-n1-standard-2
- **Cost**: **$170/month**

**Memorystore Redis**:
- Standard 10GB
- **Cost**: **$200/month**

**Cloud Tasks**:
- 30M tasks/month × $0.40/1M
- **Cost**: **$12/month**

**External IPs**:
- 20 ephemeral IPs: **Free** (no charge for ephemeral)
- OR 100 static IPs: $3/month each = $300/month (optional)
- **Cost**: **$0** (using ephemeral)

**Load Balancer**:
- Internal LB (cheaper than external)
- **Cost**: **$18/month**

**Network Egress**:
- Web scraping traffic: ~500GB/month
- Standard tier (recommended): 500GB × $0.085/GB
- **Cost**: **$43/month**

**Optional: Proxy Service** (if needed for strict rate limits):
- Rotating proxies: ~1M requests/month
- **Cost**: **$500-2,000/month** (only if needed)

**TOTAL (Baseline)**: **~$3,443/month** for 1M URLs/day
**With Proxies**: **~$3,961-5,461/month** (if needed for rate limits)

**Comparison**: 
- Cloud Run: $56,000/month
- GCE Hybrid (baseline): $3,443/month  
- **Savings: 94% cheaper** 🎉

### 6.2 Cost Optimization Strategies

**1. Committed Use Discounts**
- Commit to 1-year or 3-year reservations
- Save 25-40% on compute costs
- Example: 10 instances × 1-year = 25% discount

**2. Preemptible VMs for Non-Critical Work**
- Use for low-priority workflows
- 80% discount on compute
- Handle interruptions gracefully

**3. Scheduled Scaling**
- Scale down during off-hours (nights, weekends)
- Keep 2 instances minimum
- Scale up during peak hours

**4. Instance Right-Sizing**
- Monitor CPU/Memory usage
- Switch to n2-standard-2 if underutilized
- Or n2-standard-8 for heavy workflows

**5. Storage Lifecycle Policies**
- Move old data to Nearline after 30 days
- Move to Coldline after 90 days
- 80% storage cost reduction

---

## Part 7: Migration Roadmap

### Week 1: Local Development

**Day 1-2**:
- [ ] Set up local PostgreSQL and Redis with Docker
- [ ] Configure worker for local mode (PostgreSQL queue)
- [ ] Test basic workflow execution locally

**Day 3-4**:
- [ ] Implement HTTP task receiver endpoint
- [ ] Add Cloud Tasks client (for GCP mode)
- [ ] Test phase transitions locally

**Day 5**:
- [ ] Optimize browser pool management
- [ ] Add comprehensive logging
- [ ] Local performance testing

### Week 2: GCP Infrastructure Setup

**Day 1-2**:
- [ ] Create GCP project and enable APIs
- [ ] Provision Cloud SQL and run migrations
- [ ] Create Memorystore Redis instance

**Day 3-4**:
- [ ] Create VPC, subnets, Cloud NAT
- [ ] Set up Cloud Storage bucket
- [ ] Create Cloud Tasks queue

**Day 5**:
- [ ] Create instance template for workers
- [ ] Create initial Managed Instance Group (2 instances)
- [ ] Set up Internal Load Balancer

### Week 3: Testing & Validation

**Day 1-2**:
- [ ] Deploy worker to staging GCE instances
- [ ] Switch worker config to Cloud Tasks mode
- [ ] Test single URL processing end-to-end

**Day 3-4**:
- [ ] Load test with 10k URLs
- [ ] Monitor autoscaling behavior
- [ ] Test failure scenarios and retries

**Day 5**:
- [ ] Set up monitoring dashboards
- [ ] Configure alerting rules
- [ ] Document runbooks for common issues

### Week 4: Production Rollout

**Day 1**:
- [ ] Deploy to production with conservative limits (max 10 instances)
- [ ] Migrate 10% of workflows

**Day 2-3**:
- [ ] Monitor costs, performance, errors
- [ ] Gradually increase to 50% of workflows
- [ ] Tune autoscaling parameters

**Day 4-5**:
- [ ] Complete migration to 100%
- [ ] Decommission old VM-based setup
- [ ] Optimize and celebrate! 🎉

---

## Part 8: Development Workflow

### 8.1 Developer Setup (First Time)

```bash
# 1. Clone repository
git clone https://github.com/your-org/crawlify
cd crawlify

# 2. Start local services
docker-compose up -d

# 3. Run migrations
make migrate-up

# 4. Start API server
make run-api

# 5. Start worker (in separate terminal)
make run-worker

# 6. Access frontend
open http://localhost:3000
```

### 8.2 Daily Development

**Making Changes to Worker Logic**:
1. Edit worker code in `cmd/worker/`
2. Run tests: `make test-worker`
3. Start worker: `make run-worker`
4. Submit test workflow in frontend
5. Check logs for verification

**Testing with Real Websites**:
1. Create workflow targeting test site
2. Start execution from frontend
3. Monitor worker logs
4. Check extracted data in database

### 8.3 Deploying to GCP

**Quick Deploy** (after initial setup):
```bash
# 1. Build worker binary
make build-worker

# 2. Upload to Cloud Storage
./scripts/upload-worker-binary.sh v1.2.3

# 3. Rolling update
gcloud compute instance-groups managed rolling-action start-update \
  crawlify-workers --version=template=crawlify-worker-v1.2.3
```

---

## Part 9: Troubleshooting Guide

### Common Issues

**Issue 1: Workers Not Processing Tasks**

**Symptoms**:
- Queue depth increasing
- No logs from workers
- Health check failing

**Diagnosis**:
```bash
# Check instance group health
gcloud compute instance-groups managed list-instances crawlify-workers

# Check worker logs
gcloud logging read "resource.type=gce_instance" --limit 50

# Check Load Balancer health
gcloud compute backend-services get-health crawlify-worker-backend
```

**Solutions**:
- Restart unhealthy instances
- Check service account permissions
- Verify network connectivity to Cloud SQL/Redis

**Issue 2: High Error Rate**

**Symptoms**:
- Task retries increasing
- 5xx errors from workers
- Extraction failures

**Diagnosis**:
```bash
# Check error rate by type
gcloud logging read "severity=ERROR" \
  --format="table(jsonPayload.error_type)" \
  --limit 100

# Check failed URLs
SELECT url, error, retry_count 
FROM url_queue 
WHERE status='failed' 
ORDER BY created_at DESC 
LIMIT 50;
```

**Solutions**:
- Investigate specific error types
- Check target website changes
- Adjust timeouts/retries

**Issue 3: Autoscaling Not Working**

**Symptoms**:
- Worker count not changing with queue depth
- Always at min or max instances

**Diagnosis**:
```bash
# Check autoscaler status
gcloud compute instance-groups managed describe crawlify-workers

# Check custom metrics
gcloud monitoring time-series list \
  --filter='metric.type="custom.googleapis.com/crawlify/queue_depth"'
```

**Solutions**:
- Verify custom metric is being published
- Check autoscaling policy configuration
- Review cooldown periods

---

## Next Steps

1. **Start with Local Setup** (Week 1)
   - Set up Docker for PostgreSQL and Redis
   - Run worker locally with existing workflows
   - Validate functionality before cloud deployment

2. **Provision GCP Infrastructure** (Week 2)
   - Create GCP project with billing
   - Set up Cloud SQL, Redis, Storage
   - Create initial 2-instance worker group

3. **Deploy and Test** (Week 3)
   - Deploy worker to GCP
   - Run test workflows
   - Monitor costs and performance

4. **Scale and Optimize** (Week 4)
   - Gradually increase traffic
   - Tune autoscaling
   - Optimize for cost and performance

**Questions or need help with specific sections?** Let me know!
