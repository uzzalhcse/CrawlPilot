#!/bin/bash
# Automated build script for Crawlify backend and all plugins
# This ensures version compatibility by building everything together

set -e  # Exit on error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🔨 Crawlify Build Script"
echo "========================"
echo ""

# Build backend
echo "📦 Building backend..."
cd "$PROJECT_ROOT/cmd/crawler"
go build -o main
echo "✅ Backend built successfully"
echo ""

# Build plugins
echo "🔌 Building plugins..."
cd "$PROJECT_ROOT"

# Plugin 1: E-commerce Discovery
echo "  → Building ecommerce-discovery plugin..."
cd "$PROJECT_ROOT/examples/plugins/ecommerce-discovery"
GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o ecommerce-discovery-linux-amd64.so
cp ecommerce-discovery-linux-amd64.so "$PROJECT_ROOT/plugins/"
echo "  ✓ ecommerce-discovery built ($(du -h "$PROJECT_ROOT/plugins/ecommerce-discovery-linux-amd64.so" | cut -f1))"

# Plugin 2: Aqua Product Extractor
echo "  → Building aqua-product-extractor plugin..."
cd "$PROJECT_ROOT/examples/plugins/aqua-product-extractor"
GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o aqua-product-extractor.so
cp aqua-product-extractor.so "$PROJECT_ROOT/plugins/"
echo "  ✓ aqua-product-extractor built ($(du -h "$PROJECT_ROOT/plugins/aqua-product-extractor.so" | cut -f1))"

echo ""
echo "✅ All builds completed successfully!"
echo ""
echo "Plugin directory:"
ls -lh "$PROJECT_ROOT/plugins/"*.so
echo ""
echo "🚀 Ready to run: cd cmd/crawler && ./main"
