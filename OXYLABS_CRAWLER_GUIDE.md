# Oxylabs Sandbox Crawler - Complete Guide

## 🎯 Objective

Crawl the complete Oxylabs sandbox e-commerce site: https://sandbox.oxylabs.io/products

## 📦 Workflow Created

**Workflow ID**: `0eb04510-aaf9-44d6-b5ec-01e109e76d7e`
**Execution ID**: `3f5fcbe4-49fb-4f36-bb85-7fceac65d532`
**Status**: Active and Running

## 🛠️ Workflow Configuration

### Start URL
```
https://sandbox.oxylabs.io/products
```

### Settings
- **Max Depth**: 3 levels
- **Max Pages**: 500 pages
- **Rate Limit**: 2000ms (2 seconds between requests)
- **Custom Headers**: Browser-like User-Agent

### What It Crawls

#### 1. URL Discovery (Finding Pages)
- ✅ Product listing pages
- ✅ Individual product pages
- ✅ Pagination links (page=1, page=2, etc.)
- ✅ Category pages
- ✅ Dynamic content (with scrolling)

#### 2. Data Extraction (What's Collected)
For each product page:
- **Title** - Product name
- **Price** - Product price (cleaned and normalized)
- **Description** - Product description
- **Image** - Main product image URL
- **Category** - Product category/breadcrumb
- **Rating** - Product rating (if available)
- **SKU** - Product code
- **Brand** - Brand/manufacturer
- **Availability** - In stock/out of stock
- **Reviews** - Number of reviews
- **All Text** - Complete page content

## 📊 Monitoring the Crawl

### Check Status
```bash
# Using curl
EXECUTION_ID="3f5fcbe4-49fb-4f36-bb85-7fceac65d532"

# Get execution status
curl http://localhost:8080/api/v1/executions/$EXECUTION_ID | jq .

# Get detailed queue statistics
curl http://localhost:8080/api/v1/executions/$EXECUTION_ID/stats | jq .
```

### Using Postman

1. Import the Postman collection
2. Set `execution_id` to: `3f5fcbe4-49fb-4f36-bb85-7fceac65d532`
3. Run "Get Execution Status" repeatedly to monitor
4. Run "Get Execution Queue Stats" to see detailed stats

### Expected Queue Statistics

```json
{
  "execution_id": "3f5fcbe4-49fb-4f36-bb85-7fceac65d532",
  "stats": {
    "pending": 45,      // URLs waiting to be processed
    "processing": 2,    // URLs currently being crawled
    "completed": 103,   // URLs successfully crawled
    "failed": 3         // URLs that failed
  },
  "pending_count": 45
}
```

## 🗄️ Viewing Extracted Data

### Query the Database

```sql
-- Connect to database
psql -h localhost -U postgres -d crawlify

-- View all extracted products
SELECT
    id,
    url,
    data->>'title' as title,
    data->>'price' as price,
    data->>'category' as category,
    extracted_at
FROM extracted_data
WHERE execution_id = '3f5fcbe4-49fb-4f36-bb85-7fceac65d532'
ORDER BY extracted_at DESC;

-- Count total products extracted
SELECT COUNT(*) as total_products
FROM extracted_data
WHERE execution_id = '3f5fcbe4-49fb-4f36-bb85-7fceac65d532';

-- View products by category
SELECT
    data->>'category' as category,
    COUNT(*) as product_count
FROM extracted_data
WHERE execution_id = '3f5fcbe4-49fb-4f36-bb85-7fceac65d532'
GROUP BY data->>'category';

-- View price range
SELECT
    MIN((data->>'price')::numeric) as min_price,
    MAX((data->>'price')::numeric) as max_price,
    AVG((data->>'price')::numeric) as avg_price
FROM extracted_data
WHERE execution_id = '3f5fcbe4-49fb-4f36-bb85-7fceac65d532'
AND data->>'price' != '';

-- Export to CSV
\copy (SELECT url, data->>'title', data->>'price', data->>'category', extracted_at FROM extracted_data WHERE execution_id = '3f5fcbe4-49fb-4f36-bb85-7fceac65d532') TO '/tmp/oxylabs_products.csv' CSV HEADER;
```

### Queue Status

```sql
-- Check queue progress
SELECT
    status,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (), 2) as percentage
FROM url_queue
WHERE execution_id = '3f5fcbe4-49fb-4f36-bb85-7fceac65d532'
GROUP BY status;

-- View URLs being processed
SELECT
    url,
    depth,
    retry_count,
    status,
    created_at
FROM url_queue
WHERE execution_id = '3f5fcbe4-49fb-4f36-bb85-7fceac65d532'
AND status = 'processing';

-- View failed URLs
SELECT
    url,
    error,
    retry_count
FROM url_queue
WHERE execution_id = '3f5fcbe4-49fb-4f36-bb85-7fceac65d532'
AND status = 'failed';
```

## 🎛️ Control the Crawl

### Stop Execution

```bash
# Using curl
curl -X DELETE http://localhost:8080/api/v1/executions/3f5fcbe4-49fb-4f36-bb85-7fceac65d532

# Using Postman
# Run "Stop Execution" request
```

### Restart with Different Settings

```bash
# 1. Update workflow configuration
curl -X PUT http://localhost:8080/api/v1/workflows/0eb04510-aaf9-44d6-b5ec-01e109e76d7e \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Oxylabs Full Site Crawler",
    "config": {
      "max_depth": 2,
      "max_pages": 100,
      "rate_limit_delay": 1000,
      ...
    }
  }'

# 2. Start new execution
curl -X POST http://localhost:8080/api/v1/workflows/0eb04510-aaf9-44d6-b5ec-01e109e76d7e/execute
```

## 📈 Expected Results

### Site Structure

Based on typical e-commerce sites:
- **Product Pages**: ~50-200 individual products
- **Category Pages**: ~5-15 categories
- **Pagination**: 2-10 pages per category
- **Total URLs**: ~100-500 URLs

### Processing Time

- **Small crawl** (100 URLs): ~5-10 minutes
- **Medium crawl** (250 URLs): ~15-25 minutes
- **Large crawl** (500 URLs): ~30-45 minutes

*Time depends on rate limiting (2 seconds per page)*

### Data Volume

- Average ~10-20 fields per product
- JSON data ~2-5KB per product
- Total database size: ~1-10MB

## 🔧 Troubleshooting

### Crawl Not Starting

```bash
# Check if workflow is active
curl http://localhost:8080/api/v1/workflows/0eb04510-aaf9-44d6-b5ec-01e109e76d7e | jq '.status'

# Should return: "active"

# If not, activate it:
curl -X PATCH http://localhost:8080/api/v1/workflows/0eb04510-aaf9-44d6-b5ec-01e109e76d7e/status \
  -H "Content-Type: application/json" \
  -d '{"status": "active"}'
```

### No Data Extracted

Check if pages are being processed:
```sql
SELECT COUNT(*) FROM url_queue WHERE execution_id = '3f5fcbe4-49fb-4f36-bb85-7fceac65d532';
```

If 0, the start URL might not have been enqueued. Check logs:
```bash
docker logs crawlify-api | grep -i "oxylabs\|execution\|error"
```

### Browser Issues

If Playwright fails:
```bash
# Reinstall browsers
docker exec crawlify npx playwright install chromium

# Or locally
npx playwright install chromium
```

### Rate Limiting

If getting blocked:
```json
{
  "config": {
    "rate_limit_delay": 5000,  // Increase to 5 seconds
    "headers": {
      "User-Agent": "Mozilla/5.0 ...",
      "Accept-Language": "en-US,en;q=0.9"
    }
  }
}
```

## 📊 Real-Time Dashboard Queries

### Create Monitoring Views

```sql
-- Create a monitoring view
CREATE VIEW crawl_progress AS
SELECT
    we.id as execution_id,
    w.name as workflow_name,
    we.status,
    we.started_at,
    (SELECT COUNT(*) FROM url_queue WHERE execution_id = we.id AND status = 'completed') as urls_completed,
    (SELECT COUNT(*) FROM url_queue WHERE execution_id = we.id AND status = 'pending') as urls_pending,
    (SELECT COUNT(*) FROM url_queue WHERE execution_id = we.id AND status = 'failed') as urls_failed,
    (SELECT COUNT(*) FROM extracted_data WHERE execution_id = we.id) as products_extracted,
    EXTRACT(EPOCH FROM (NOW() - we.started_at)) as runtime_seconds
FROM workflow_executions we
JOIN workflows w ON w.id = we.workflow_id
WHERE we.id = '3f5fcbe4-49fb-4f36-bb85-7fceac65d532';

-- Query the view
SELECT * FROM crawl_progress;
```

## 🎯 Success Metrics

### What Indicates Success

- ✅ Queue stats show URLs being processed
- ✅ `extracted_data` table has records
- ✅ `completed` count is increasing
- ✅ `failed` count is low (< 5%)
- ✅ Execution is `running: true`

### Sample Successful Output

```json
{
  "execution_id": "3f5fcbe4-49fb-4f36-bb85-7fceac65d532",
  "running": true,
  "stats": {
    "pending": 23,
    "processing": 2,
    "completed": 125,
    "failed": 2
  }
}
```

## 📝 Workflow File

Complete workflow saved at:
```
examples/oxylabs_full_site_crawler.yaml
```

## 🚀 Next Steps

1. **Monitor Progress**: Use Postman or curl to check stats every 30 seconds
2. **View Data**: Query the database to see extracted products
3. **Export Results**: Export to CSV or JSON for analysis
4. **Analyze**: Use SQL queries to get insights
5. **Optimize**: Adjust settings based on results

## 📞 Quick Commands

```bash
# Status check
curl http://localhost:8080/api/v1/executions/3f5fcbe4-49fb-4f36-bb85-7fceac65d532/stats

# Database check
psql -h localhost -U postgres -d crawlify -c "SELECT COUNT(*) FROM extracted_data WHERE execution_id = '3f5fcbe4-49fb-4f36-bb85-7fceac65d532';"

# Stop crawl
curl -X DELETE http://localhost:8080/api/v1/executions/3f5fcbe4-49fb-4f36-bb85-7fceac65d532

# Export data
psql -h localhost -U postgres -d crawlify -c "\copy (SELECT * FROM extracted_data WHERE execution_id = '3f5fcbe4-49fb-4f36-bb85-7fceac65d532') TO 'oxylabs_export.csv' CSV HEADER;"
```

## ✅ Checklist

- [x] Workflow created
- [x] Workflow activated
- [x] Execution started
- [ ] Monitor progress (check stats)
- [ ] Verify data extraction (check database)
- [ ] Export results when complete
- [ ] Analyze data

---

**Happy Crawling!** 🕷️

Your Oxylabs sandbox crawl is now running. Monitor the progress and check the database to see the extracted products!
