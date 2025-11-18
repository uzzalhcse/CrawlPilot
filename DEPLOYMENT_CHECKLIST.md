# Deployment Checklist - Architecture Optimization

## ✅ Pre-Deployment Verification

### Database Migration
- [ ] Backup current database: `pg_dump -U postgres crawlify > backup_$(date +%Y%m%d).sql`
- [ ] Apply migration: `psql -U postgres -d crawlify -f migrations/001_complete_schema.up.sql`
- [ ] Verify tables: `psql -U postgres -d crawlify -c "\d url_queue"`
- [ ] Check new columns exist:
  - [ ] `url_queue.parent_url_id`
  - [ ] `url_queue.url_type`
  - [ ] `url_queue.discovered_by_node`
  - [ ] `node_executions.urls_discovered`
  - [ ] `node_executions.items_extracted`
  - [ ] `node_executions.duration_ms`
  - [ ] `extracted_items` table exists

### Application Build
- [ ] Build application: `go build -o crawler ./cmd/crawler`
- [ ] No compilation errors
- [ ] Binary created successfully

### Configuration
- [ ] Environment variables set:
  - [ ] `DATABASE_HOST`
  - [ ] `DATABASE_PORT`
  - [ ] `DATABASE_USER`
  - [ ] `DATABASE_PASSWORD`
  - [ ] `DATABASE_NAME`

## 🚀 Deployment Steps

### 1. Stop Existing Service
```bash
# Stop running crawler
pkill -f crawler

# Verify stopped
ps aux | grep crawler
```

### 2. Deploy New Version
```bash
# Copy new binary
cp crawler /usr/local/bin/crawler

# Set permissions
chmod +x /usr/local/bin/crawler

# Restart service
systemctl restart crawler
# OR
./crawler &
```

### 3. Verify Deployment
```bash
# Check service is running
curl http://localhost:8080/api/v1/workflows

# Check logs
tail -f /var/log/crawler.log
```

## 🧪 Post-Deployment Testing

### Test 1: Create Workflow
```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/amazon_bestsellers.json
```

**Expected:** 
- HTTP 201 Created
- Response contains `workflow_id`

### Test 2: Execute Workflow
```bash
# Use workflow_id from Test 1
curl -X POST http://localhost:8080/api/v1/workflows/{workflow_id}/execute
```

**Expected:**
- HTTP 200 OK
- Response contains `execution_id`

### Test 3: Verify New Analytics Endpoints
```bash
# Get execution ID from Test 2

# Timeline endpoint
curl http://localhost:8080/api/v1/executions/{execution_id}/timeline

# Hierarchy endpoint
curl http://localhost:8080/api/v1/executions/{execution_id}/hierarchy

# Performance endpoint
curl http://localhost:8080/api/v1/executions/{execution_id}/performance

# Items with hierarchy
curl http://localhost:8080/api/v1/executions/{execution_id}/items-with-hierarchy

# Bottlenecks
curl http://localhost:8080/api/v1/executions/{execution_id}/bottlenecks
```

**Expected:** All endpoints return JSON responses (may be empty initially)

### Test 4: Verify Database Schema
```bash
# Check URL hierarchy
psql -U postgres -d crawlify -c "
SELECT id, url, url_type, parent_url_id, discovered_by_node 
FROM url_queue 
LIMIT 5;
"

# Check node execution metrics
psql -U postgres -d crawlify -c "
SELECT node_id, urls_discovered, items_extracted, duration_ms 
FROM node_executions 
WHERE urls_discovered > 0 OR items_extracted > 0 
LIMIT 5;
"

# Check extracted items
psql -U postgres -d crawlify -c "
SELECT item_type, title, price, rating 
FROM extracted_items 
LIMIT 5;
"
```

### Test 5: Performance Query Test
```bash
# Test fast price query
psql -U postgres -d crawlify -c "
EXPLAIN ANALYZE
SELECT title, price, rating 
FROM extracted_items 
WHERE price < 50 AND rating > 4.0;
"
```

**Expected:** Query uses index on price/rating columns

## 📊 Monitoring

### Key Metrics to Watch

1. **Query Performance**
   - Monitor query execution time
   - Expected: 10x improvement on structured fields

2. **Execution Success Rate**
   - Check `node_executions` status
   - Monitor failure rates

3. **URL Discovery**
   - Verify `urls_discovered` is being populated
   - Check hierarchy relationships

4. **Data Extraction**
   - Verify `items_extracted` count
   - Check `extracted_items` table growth

### Health Checks
```bash
# Application health
curl http://localhost:8080/health

# Database connections
psql -U postgres -d crawlify -c "
SELECT count(*) FROM pg_stat_activity 
WHERE datname = 'crawlify';
"

# Recent executions
psql -U postgres -d crawlify -c "
SELECT id, status, started_at 
FROM workflow_executions 
ORDER BY started_at DESC 
LIMIT 10;
"
```

## 🐛 Troubleshooting

### Issue: Migration Fails
```bash
# Check current schema version
psql -U postgres -d crawlify -c "\d url_queue"

# If column already exists, skip or rollback
psql -U postgres -d crawlify -f migrations/001_complete_schema.down.sql
```

### Issue: "column stats does not exist"
**Status:** ✅ FIXED in `internal/storage/execution_repository.go`

Solution applied: Stats stored in metadata JSONB column

### Issue: Workflow Creation Fails
Check logs for:
- Duplicate workflow name
- Invalid JSON structure
- Database connection issues

Solution:
```bash
# Delete duplicate workflows
psql -U postgres -d crawlify -c "
DELETE FROM workflows 
WHERE name = 'Amazon Best Sellers Scraper - Optimized';
"
```

### Issue: No Data in Analytics Endpoints
**Cause:** Workflow needs to run and process URLs

**Solution:** Wait for workflow execution to progress

## 🔄 Rollback Plan

If issues occur, rollback with:

```bash
# 1. Stop application
pkill -f crawler

# 2. Rollback database
psql -U postgres -d crawlify -f migrations/001_complete_schema.down.sql

# 3. Restore from backup
psql -U postgres -d crawlify < backup_YYYYMMDD.sql

# 4. Deploy old binary
cp crawler.old /usr/local/bin/crawler
./crawler &
```

## ✅ Sign-Off Checklist

- [ ] All tests passed
- [ ] Monitoring dashboards updated
- [ ] Documentation reviewed
- [ ] Team notified of new features
- [ ] Backup verified
- [ ] Rollback plan tested

## 📚 New Features Documentation

### For Developers

**New Workflow Configuration:**
All `extract_links` nodes should include `url_type`:

```json
{
  "id": "discover_products",
  "type": "extract_links",
  "params": {
    "selector": ".product-link",
    "url_type": "product"
  }
}
```

**Available URL Types:**
- `seed` - Starting URLs
- `category` - Category pages
- `subcategory` - Subcategory pages
- `product` - Product detail pages
- `article` - Article/content pages
- `pagination` - Pagination links
- `page` - Generic page (default)

### For Data Analysts

**New Query Capabilities:**

```sql
-- Get URL hierarchy tree
WITH RECURSIVE url_tree AS (
    SELECT id, url, parent_url_id, url_type, 0 as level
    FROM url_queue WHERE parent_url_id IS NULL
    UNION ALL
    SELECT uq.id, uq.url, uq.parent_url_id, uq.url_type, ut.level + 1
    FROM url_queue uq
    JOIN url_tree ut ON uq.parent_url_id = ut.id
)
SELECT * FROM url_tree ORDER BY level, url;

-- Fast product queries
SELECT title, price, rating 
FROM extracted_items 
WHERE price < 50 AND rating > 4.0
ORDER BY review_count DESC;

-- Performance analysis
SELECT node_id, 
       AVG(duration_ms) as avg_ms,
       SUM(urls_discovered) as total_urls,
       SUM(items_extracted) as total_items
FROM node_executions 
GROUP BY node_id;
```

## 📞 Support

For issues or questions:
- Check logs: `/var/log/crawler.log`
- Review documentation: `docs/ARCHITECTURE_IMPROVEMENTS.md`
- API reference: `docs/API_ANALYTICS_ENDPOINTS.md`

---

**Deployment Date:** _____________

**Deployed By:** _____________

**Verified By:** _____________
