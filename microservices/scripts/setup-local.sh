# Local Development Setup Script
# This script initializes the database and starts all services

echo "🚀 Starting Crawlify Microservices Local Environment"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

# Create network if it doesn't exist
docker network inspect crawlify_network >/dev/null 2>&1 || \
    docker network create crawlify_network

echo "📦 Starting infrastructure services..."
cd infrastructure/docker-compose

# Start PostgreSQL and Redis
docker-compose up -d postgres redis

echo "⏳ Waiting for PostgreSQL to be ready..."
until docker-compose exec -T postgres pg_isready -U crawlify > /dev/null 2>&1; do
    sleep 1
done

echo "📊 Running database migrations..."
docker-compose exec -T postgres psql -U crawlify -d crawlify < ../database/schema.sql

echo "⏳ Waiting for Redis to be ready..."
until docker-compose exec -T redis redis-cli ping > /dev/null 2>&1; do
    sleep 1
done

echo "🔧 Starting Pub/Sub emulator..."
docker-compose up -d pubsub-emulator

sleep 3

echo "🎯 Setting up Pub/Sub topic and subscription..."
# Create topic and subscription
docker-compose exec -T pubsub-emulator sh -c '
    curl -X PUT http://localhost:8085/v1/projects/crawlify-local/topics/crawlify-tasks
    curl -X PUT http://localhost:8085/v1/projects/crawlify-local/subscriptions/crawlify-tasks-sub \
        -H "Content-Type: application/json" \
        -d "{\"topic\": \"projects/crawlify-local/topics/crawlify-tasks\"}"
'

echo "🎨 Starting Orchestrator..."
docker-compose up -d orchestrator

echo "👷 Starting Worker..."
docker-compose up -d worker

echo ""
echo "✅ All services started successfully!"
echo ""
echo "📍 Service URLs:"
echo "   Orchestrator API: http://localhost:8080"
echo "   Health Check: http://localhost:8080/health"
echo "   PostgreSQL: localhost:5432"
echo "   Redis: localhost:6379"
echo "   Pub/Sub Emulator: localhost:8085"
echo ""
echo "📝 Useful commands:"
echo "   View logs: make docker-logs"
echo "   Stop all: make docker-down"
echo "   Restart: make docker-restart"
echo ""
echo "🎉 Ready to crawl!"
