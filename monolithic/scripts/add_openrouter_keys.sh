#!/bin/bash
# Script to seed OpenRouter API keys from openrouter.txt file

# Database connection
DB_HOST="localhost"
DB_PORT="5432"
DB_USER="postgres"
DB_PASS="root"
DB_NAME="crawlify"

# Path to keys file
KEYS_FILE="/home/uzzalh/Workplace/github/uzzalhcse/Crawlify/openrouter.txt"

# Check if keys file exists
if [ ! -f "$KEYS_FILE" ]; then
    echo "Error: openrouter.txt not found at $KEYS_FILE"
    exit 1
fi

echo "Reading OpenRouter API keys from $KEYS_FILE..."

# Counter
count=0
skipped=0

# Read each line from openrouter.txt
while IFS= read -r api_key || [ -n "$api_key" ]; do
    # Skip empty lines and comments
    if [ -z "$api_key" ] || [[ "$api_key" =~ ^# ]]; then
        continue
    fi
    
    # Trim whitespace
    api_key=$(echo "$api_key" | xargs)
    
    # Skip if still empty after trim
    if [ -z "$api_key" ]; then
        continue
    fi
    
    # Generate name
    count=$((count + 1))
    name="OpenRouter-Key-$count"
    
    # Insert key with provider='openrouter'
    result=$(PGPASSWORD=$DB_PASS psql -h $DB_HOST -U $DB_USER -d $DB_NAME -t -c \
        "INSERT INTO ai_api_keys (name, api_key, provider) VALUES ('$name', '$api_key', 'openrouter') ON CONFLICT (api_key) DO NOTHING RETURNING id;" 2>&1)
    
    if [ -z "$result" ]; then
        echo "  ⚠️  Skipped duplicate: $name (key already exists)"
        skipped=$((skipped + 1))
    else
        echo "  ✅ Added: $name"
    fi
    
done < "$KEYS_FILE"

echo ""
echo "=========================================="
echo "OpenRouter Seeding Complete!"
echo "=========================================="
echo "Keys processed: $count"
echo "Keys added: $((count - skipped))"
echo "Keys skipped (duplicates): $skipped"
echo ""

# Show current status
echo "Current database status:"
PGPASSWORD=$DB_PASS psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c \
    "SELECT 
        provider,
        COUNT(*) as total_keys,
        COUNT(*) FILTER (WHERE is_active = true) as active_keys,
        COUNT(*) FILTER (WHERE is_rate_limited = true) as rate_limited_keys
     FROM ai_api_keys
     GROUP BY provider;"
