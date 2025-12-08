#!/bin/bash

# Exit on error
set -e

echo "🚀 Starting Crawlify Deployment on GCP VM..."

# 1. Update Codebase
echo "📥 Pulling latest code..."
git fetch origin
git checkout microservice/v1
git pull origin microservice/v1

# 2. Setup Infrastructure (Docker, Migrations, PubSub)
echo "🏗️  Setting up infrastructure..."
cd microservices
# make dev runs: docker-up migrate-up setup-pubsub-local
make dev

# 3. Build SelectFlow (Must be done before Orchestrator build)
echo "🎨 Building SelectFlow..."
cd ../SelectFlow
npm install
npm run build
# Verify build output exists
if [ ! -f "../microservices/orchestrator/assets/selectflow.js" ]; then
    echo "❌ SelectFlow build failed - assets/selectflow.js not found!"
    exit 1
fi

# 4. Build Frontend
echo "🖥️  Building Frontend..."
cd ../frontend
npm install
npm run build

# 5. Build Backends
echo "⚙️  Building Microservices..."
cd ../microservices
make build-all

# 6. Setup PM2
echo "🚀 Starting services with PM2..."

# Check if PM2 is installed
if ! command -v pm2 &> /dev/null; then
    echo "📦 Installing PM2..."
    npm install -g pm2
fi

# Stop existing processes
pm2 delete all || true

# Start Orchestrator
# Env vars should be loaded from .env or set here. Assuming .env exists or defaults are fine.
# We need to ensure SELECTFLOW_SCRIPT_PATH is set correctly relative to the binary or absolute
export SELECTFLOW_SCRIPT_PATH="$(pwd)/orchestrator/assets/selectflow.js"

pm2 start ./orchestrator/bin/orchestrator --name "crawlify-orchestrator" --cwd ./orchestrator

# Start Worker
pm2 start ./worker/bin/worker --name "crawlify-worker" --cwd ./worker

# Start Frontend (Serving static files or using preview)
# For production, usually served via Nginx, but user asked for PM2.
# We can use 'serve' or 'npm run preview'
cd ../frontend
pm2 start "npm run preview -- --host 0.0.0.0 --port 3131" --name "crawlify-frontend"

# Save PM2 list
pm2 save

echo "✅ Deployment Complete!"
echo "--------------------------------"
echo "Orchestrator: http://<VM_IP>:8181"
echo "Worker:       http://<VM_IP>:8282"
echo "Frontend:     http://<VM_IP>:3131"
echo "--------------------------------"
pm2 status
