#!/bin/bash
# Local Development Setup Script
# Run this to set up local development environment

set -e

echo "🚀 Setting up Crawlify Microservices - Local Development"
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check dependencies
echo "📋 Checking dependencies..."

check_installed() {
    if ! command -v $1 &> /dev/null; then
        echo -e "${RED}❌ $1 is not installed${NC}"
        echo "   Install it: $2"
        exit 1
    else
        echo -e "${GREEN}✓${NC} $1"
    fi
}

check_installed "psql" "sudo apt-get install postgresql-client"
check_installed "redis-cli" "sudo apt-get install redis-tools"
check_installed "go" "https://golang.org/doc/install"

echo ""
echo "📦 Starting services..."

# Start PostgreSQL (assuming it's installed)
if sudo systemctl is-active --quiet postgresql; then
    echo -e "${GREEN}✓${NC} PostgreSQL is running"
else
    echo "Starting PostgreSQL..."
    sudo systemctl start postgresql
fi

# Start Redis (assuming it's installed)
if sudo systemctl is-active --quiet redis-server || sudo systemctl is-active --quiet redis; then
    echo -e "${GREEN}✓${NC} Redis is running"
else
    echo "Starting Redis..."
    sudo systemctl start redis-server 2>/dev/null || sudo systemctl start redis
fi

echo ""
echo "💾 Setting up database..."

# Create database and user
sudo -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname = 'crawlify'" | grep -q 1 || \
sudo -u postgres psql <<EOF
CREATE DATABASE crawlify;
CREATE USER crawlify WITH PASSWORD 'dev_password';
GRANT ALL PRIVILEGES ON DATABASE crawlify TO crawlify;
EOF

echo -e "${GREEN}✓${NC} Database 'crawlify' created"

# Run migrations
PGPASSWORD=dev_password psql -h localhost -U crawlify -d crawlify < infrastructure/database/schema.sql
echo -e "${GREEN}✓${NC} Database schema applied"

echo ""
echo "🔧 Installing Pub/Sub emulator..."

# Check if gcloud is installed
if command -v gcloud &> /dev/null; then
    # Install Pub/Sub emulator component
    gcloud components install pubsub-emulator --quiet 2>/dev/null || true
    echo -e "${GREEN}✓${NC} Pub/Sub emulator ready"
else
    echo -e "${YELLOW}⚠${NC}  gcloud not found. Install from: https://cloud.google.com/sdk/docs/install"
    echo "   For now, you can run without Pub/Sub (worker won't process tasks)"
fi

echo ""
echo "📥 Installing Go dependencies..."

cd orchestrator && go mod download && cd ..
echo -e "${GREEN}✓${NC} Orchestrator dependencies"

cd worker && go mod download && cd ..
echo -e "${GREEN}✓${NC} Worker dependencies"

cd shared && go mod download && cd ..
echo -e "${GREEN}✓${NC} Shared dependencies"

echo ""
echo "🎭 Installing Playwright browsers..."
cd worker && npx playwright install chromium && cd ..
echo -e "${GREEN}✓${NC} Playwright installed"

echo ""
echo -e "${GREEN}✅ Setup complete!${NC}"
echo ""
echo "📍 Next steps:"
echo "   1. Start Pub/Sub emulator (in a separate terminal):"
echo "      make pubsub-local"
echo ""
echo "   2. Start Orchestrator (in a separate terminal):"
echo "      make run-orchestrator"
echo ""
echo "   3. Start Worker (in a separate terminal):"
echo "      make run-worker"
echo ""
echo "   4. Test the system:"
echo "      make test-workflow"
echo ""
echo "🎉 Happy crawling!"
