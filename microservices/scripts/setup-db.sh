#!/bin/bash
# Simple database setup for local development

echo "Setting up PostgreSQL database..."

# Create user and database
sudo -u postgres psql << 'EOF'
-- Drop existing if any
DROP DATABASE IF EXISTS crawlify;
DROP USER IF EXISTS crawlify;

-- Create user
CREATE USER crawlify WITH PASSWORD 'dev_password';

-- Create database
CREATE DATABASE crawlify OWNER crawlify;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE crawlify TO crawlify;
EOF

echo "✅ Database and user created"

# Apply schema
echo "Applying database schema..."
PGPASSWORD=dev_password psql -h localhost -U crawlify -d crawlify -f infrastructure/database/schema.sql

echo "✅ Schema applied successfully!"
