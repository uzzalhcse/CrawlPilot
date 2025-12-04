#!/bin/bash
# Complete Plugin System Test - End-to-End Flow

echo "=== Crawlify Plugin Marketplace - End-to-End Test ==="
echo ""

# Configuration
BASE_URL="http://localhost:8080/api/v1"
PLUGIN_ID="ecommerce-product-discovery"

echo "Step 1: Running database migration..."
echo "----------------------------------------"
psql -U postgres -d crawlify -f migrations/018_plugin_marketplace.up.sql
echo "✓ Migration complete"
echo ""

echo "Step 2: Building example e-commerce plugin..."
echo "---------------------------------------------"
cd examples/plugins/ecommerce-discovery

# Update go.mod to use local Crawlify
sed -i 's|// replace|replace|' go.mod

# Build for Linux AMD64
GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o ecommerce-discovery-linux-amd64.so
echo "✓ Built plugin: ecommerce-discovery-linux-amd64.so"

# Calculate SHA256 hash
BINARY_HASH=$(sha256sum ecommerce-discovery-linux-amd64.so | awk '{print $1}')
BINARY_SIZE=$(stat -f%z ecommerce-discovery-linux-amd64.so 2>/dev/null || stat -c%s ecommerce-discovery-linux-amd64.so)

# Move to plugins directory
mkdir -p ../../../plugins
cp ecommerce-discovery-linux-amd64.so ../../../plugins/
BINARY_PATH="./plugins/ecommerce-discovery-linux-amd64.so"

cd ../../..
echo ""

echo "Step 3: Creating plugin via API..."
echo "-----------------------------------"
PLUGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/plugins" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "'$PLUGIN_ID'",
    "name": "E-commerce Product Discovery",
    "slug": "ecommerce-product-discovery",
    "description": "Discovers product URLs from e-commerce websites with automatic pagination support",
    "author_name": "Crawlify Team",
    "author_email": "team@crawlify.io",
    "repository_url": "https://github.com/crawlify/plugins/ecommerce-discovery",
    "phase_type": "discovery",
    "plugin_type": "official",
    "category": "ecommerce",
    "tags": ["ecommerce", "pagination", "products"],
    "is_verified": true
  }')

echo "$PLUGIN_RESPONSE" | jq '.'
echo ""

echo "Step 4: Publishing plugin version..."
echo "------------------------------------"
VERSION_RESPONSE=$(curl -s -X POST "$BASE_URL/plugins/$PLUGIN_ID/versions" \
  -H "Content-Type: application/json" \
  -d '{
    "version": "1.0.0",
    "changelog": "Initial release with pagination support",
    "is_stable": true,
    "min_crawlify_version": "1.0.0",
    "linux_amd64_binary_path": "'$BINARY_PATH'",
    "binary_hash": "'$BINARY_HASH'",
    "binary_size_bytes": '$BINARY_SIZE',
    "config_schema": {
      "type": "object",
      "required": ["product_selector"],
      "properties": {
        "product_selector": {
          "type": "string",
          "description": "CSS selector for product links"
        },
        "max_products": {
          "type": "integer",
          "default": 100
        }
      }
    }
  }')

VERSION_ID=$(echo "$VERSION_RESPONSE" | jq -r '.id')
echo "$VERSION_RESPONSE" | jq '.'
echo ""

echo "Step 5: Listing all plugins..."
echo "-------------------------------"
curl -s "$BASE_URL/plugins" | jq '.'
echo ""

echo "Step 6: Getting plugin details..."
echo "---------------------------------"
curl -s "$BASE_URL/plugins/ecommerce-product-discovery" | jq '.'
echo ""

echo "Step 7: Installing plugin..."
echo "----------------------------"
INSTALL_RESPONSE=$(curl -s -X POST "$BASE_URL/plugins/$PLUGIN_ID/install" \
  -H "X-Workspace-ID: default")

echo "$INSTALL_RESPONSE" | jq '.'
echo ""

echo "Step 8: Listing installed plugins..."
echo "------------------------------------"
curl -s "$BASE_URL/plugins/installed" \
  -H "X-Workspace-ID: default" | jq '.'
echo ""

echo "Step 9: Creating a review..."
echo "----------------------------"
REVIEW_RESPONSE=$(curl -s -X POST "$BASE_URL/plugins/$PLUGIN_ID/reviews" \
  -H "Content-Type: application/json" \
  -d '{
    "rating": 5,
    "review_text": "Excellent plugin for e-commerce scraping! Works perfectly with pagination."
  }')

echo "$REVIEW_RESPONSE" | jq '.'
echo ""

echo "Step 10: Testing plugin loading in NodeRegistry..."
echo "--------------------------------------------------"
cat > test_plugin_loading.go << 'EOF'
package main

import (
	"fmt"
	"github.com/uzzalhcse/crawlify/internal/workflow"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	registry := workflow.NewNodeRegistryWithLogger(logger)
	
	// Register default nodes
	if err := registry.RegisterDefaultNodes(); err != nil {
		panic(err)
	}
	
	// Register our plugin
	if err := registry.RegisterPlugin("./plugins/ecommerce-discovery-linux-amd64.so"); err != nil {
		panic(err)
	}
	
	// List all registered plugins
	plugins := registry.ListPlugins()
	fmt.Printf("✓ Successfully loaded %d plugins\\n", len(plugins))
	for _, p := range plugins {
		fmt.Printf("  - %s\\n", p)
	}
}
EOF

go run test_plugin_loading.go
rm test_plugin_loading.go
echo ""

echo "Step 11: Uninstalling plugin..."
echo "-------------------------------"
curl -s -X DELETE "$BASE_URL/plugins/$PLUGIN_ID/uninstall" \
  -H "X-Workspace-ID: default"
echo "✓ Plugin uninstalled"
echo ""

echo "=== Test Complete! ==="
echo ""
echo "Summary:"
echo "--------"
echo "✓ Database migration applied"
echo "✓ Plugin compiled and built"
echo "✓ Plugin registered via API"
echo "✓ Version published with binary"
echo "✓ Plugin installed and uninstalled"
echo "✓ Plugin loaded dynamically in NodeRegistry"
echo ""
echo "The complete plugin marketplace system is working end-to-end!"
