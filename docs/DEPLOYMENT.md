# Deployment Guide

This guide covers deploying Crawlify in various environments.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Local Development](#local-development)
- [Docker Deployment](#docker-deployment)
- [Production Deployment](#production-deployment)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements

- **CPU**: 2+ cores recommended
- **RAM**: 4GB minimum, 8GB+ recommended
- **Storage**: 20GB+ for database and logs
- **OS**: Linux (Ubuntu 20.04+, CentOS 7+), macOS, Windows with WSL2

### Software Requirements

- Go 1.21+ (for local development)
- PostgreSQL 14+
- Node.js 16+ (for Playwright browsers)
- Docker & Docker Compose (for containerized deployment)

## Local Development

### 1. Install Dependencies

```bash
# Install Go dependencies
make install-deps

# Or manually
go mod download
```

### 2. Set Up Database

```bash
# Create database
createdb crawlify

# Run migrations
make migrate-up

# Or manually
psql -U postgres -d crawlify -f migrations/001_initial_schema.up.sql
```

### 3. Configure Application

Create `config.yaml`:

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
  ssl_mode: disable

browser:
  pool_size: 3
  headless: true
  timeout: 30000
```

### 4. Install Playwright Browsers

```bash
# Install Playwright and browsers
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium
```

### 5. Run Application

```bash
# Using make
make run

# Or directly
go run cmd/crawler/main.go
```

The API will be available at `http://localhost:8080`

## Docker Deployment

### Quick Start

```bash
# Start all services
make docker-up

# Or manually
docker-compose up -d
```

This starts:
- Crawlify API on port 8080
- PostgreSQL on port 5432
- Redis on port 6379

### Custom Configuration

1. Create `config.yaml` with your settings
2. Update `docker-compose.yml` environment variables
3. Restart services:

```bash
docker-compose down
docker-compose up -d
```

### View Logs

```bash
# All services
docker-compose logs -f

# Crawlify only
docker-compose logs -f crawlify
```

### Stop Services

```bash
docker-compose down

# Remove volumes
docker-compose down -v
```

## Production Deployment

### 1. Environment Variables

Create `.env` file:

```bash
# Database
DB_HOST=your-db-host
DB_PORT=5432
DB_USER=crawlify_user
DB_PASSWORD=secure_password
DB_NAME=crawlify
DB_SSL_MODE=require

# Redis (optional)
REDIS_ENABLED=true
REDIS_HOST=your-redis-host
REDIS_PORT=6379
REDIS_PASSWORD=redis_password

# Server
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# Browser
BROWSER_POOL_SIZE=10
BROWSER_HEADLESS=true
```

### 2. Build Production Binary

```bash
make build-prod
```

This creates an optimized binary with reduced size.

### 3. Database Setup

```bash
# Create production database
CREATE DATABASE crawlify;
CREATE USER crawlify_user WITH ENCRYPTED PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE crawlify TO crawlify_user;

# Run migrations
psql -h your-db-host -U crawlify_user -d crawlify -f migrations/001_initial_schema.up.sql
```

### 4. Systemd Service

Create `/etc/systemd/system/crawlify.service`:

```ini
[Unit]
Description=Crawlify Web Crawler API
After=network.target postgresql.service

[Service]
Type=simple
User=crawlify
Group=crawlify
WorkingDirectory=/opt/crawlify
ExecStart=/opt/crawlify/crawlify
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=crawlify

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/crawlify/data

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable crawlify
sudo systemctl start crawlify
sudo systemctl status crawlify
```

### 5. Nginx Reverse Proxy

Create `/etc/nginx/sites-available/crawlify`:

```nginx
upstream crawlify {
    server 127.0.0.1:8080;
}

server {
    listen 80;
    server_name crawlify.example.com;

    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name crawlify.example.com;

    # SSL configuration
    ssl_certificate /etc/letsencrypt/live/crawlify.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/crawlify.example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Logging
    access_log /var/log/nginx/crawlify-access.log;
    error_log /var/log/nginx/crawlify-error.log;

    # Proxy settings
    location / {
        proxy_pass http://crawlify;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Health check
    location /health {
        proxy_pass http://crawlify;
        access_log off;
    }
}
```

Enable and restart Nginx:

```bash
sudo ln -s /etc/nginx/sites-available/crawlify /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

## Kubernetes Deployment

### 1. Create Namespace

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: crawlify
```

### 2. ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: crawlify-config
  namespace: crawlify
data:
  config.yaml: |
    server:
      port: 8080
      host: 0.0.0.0
    database:
      host: postgres-service
      port: 5432
      user: crawlify
      database: crawlify
    browser:
      pool_size: 5
      headless: true
```

### 3. Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: crawlify-secrets
  namespace: crawlify
type: Opaque
stringData:
  db-password: "your-secure-password"
```

### 4. Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: crawlify
  namespace: crawlify
spec:
  replicas: 3
  selector:
    matchLabels:
      app: crawlify
  template:
    metadata:
      labels:
        app: crawlify
    spec:
      containers:
      - name: crawlify
        image: crawlify:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: crawlify-secrets
              key: db-password
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
      volumes:
      - name: config
        configMap:
          name: crawlify-config
```

### 5. Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: crawlify-service
  namespace: crawlify
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
  selector:
    app: crawlify
```

Apply configurations:

```bash
kubectl apply -f k8s/
```

## Monitoring

### Health Checks

```bash
# Basic health check
curl http://localhost:8080/health

# Expected response
{
  "status": "healthy",
  "version": "1.0.0",
  "time": "2024-01-15T10:30:00Z"
}
```

### Logs

```bash
# Docker
docker-compose logs -f crawlify

# Systemd
journalctl -u crawlify -f

# Kubernetes
kubectl logs -f deployment/crawlify -n crawlify
```

### Metrics

Monitor key metrics:

- API response times
- Database connection pool usage
- Browser pool availability
- Queue depth and processing rate
- Memory and CPU usage

### Prometheus Integration

Add metrics endpoint to your application and configure Prometheus scraping.

## Troubleshooting

### Database Connection Issues

```bash
# Test connection
psql -h localhost -U postgres -d crawlify -c "SELECT 1"

# Check migrations
psql -h localhost -U postgres -d crawlify -c "\dt"
```

### Playwright Browser Issues

```bash
# Reinstall browsers
npx playwright install chromium --force

# Check browser installation
npx playwright install --help
```

### Memory Issues

- Reduce `browser.pool_size` in config
- Lower `crawler.concurrent_workers`
- Increase container/VM memory allocation

### Port Already in Use

```bash
# Find process using port 8080
lsof -i :8080

# Kill process
kill -9 <PID>
```

### Docker Issues

```bash
# Clean up
docker-compose down -v
docker system prune -a

# Rebuild
docker-compose build --no-cache
docker-compose up -d
```

## Performance Tuning

### Database

```sql
-- Increase shared buffers
ALTER SYSTEM SET shared_buffers = '256MB';

-- Increase work mem
ALTER SYSTEM SET work_mem = '16MB';

-- Reload configuration
SELECT pg_reload_conf();
```

### Application

- Adjust `browser.pool_size` based on available memory
- Tune `crawler.concurrent_workers` for CPU cores
- Enable Redis caching for frequently accessed data
- Use connection pooling (`database.max_connections`)

### Operating System

```bash
# Increase file descriptors
ulimit -n 65536

# Add to /etc/security/limits.conf
* soft nofile 65536
* hard nofile 65536
```

## Backup and Recovery

### Database Backup

```bash
# Backup
pg_dump -U postgres crawlify > backup.sql

# Restore
psql -U postgres -d crawlify < backup.sql
```

### Automated Backups

```bash
#!/bin/bash
# /opt/crawlify/backup.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups"
pg_dump -U crawlify crawlify | gzip > $BACKUP_DIR/crawlify_$DATE.sql.gz

# Keep only last 7 days
find $BACKUP_DIR -name "crawlify_*.sql.gz" -mtime +7 -delete
```

Add to crontab:

```bash
0 2 * * * /opt/crawlify/backup.sh
```

## Security Considerations

1. **Use HTTPS**: Always use TLS/SSL in production
2. **Environment Variables**: Store secrets in environment variables, not config files
3. **Database Encryption**: Enable SSL for database connections
4. **Network Isolation**: Use private networks for database access
5. **Rate Limiting**: Implement rate limiting to prevent abuse
6. **Authentication**: Add API authentication (JWT, API keys)
7. **Input Validation**: Validate all workflow configurations
8. **Regular Updates**: Keep dependencies and OS packages updated

## Scaling

### Horizontal Scaling

- Run multiple API instances behind a load balancer
- Share PostgreSQL and Redis across instances
- Use distributed browser pools

### Vertical Scaling

- Increase CPU and memory allocations
- Optimize database queries and indexes
- Tune browser pool and worker settings
