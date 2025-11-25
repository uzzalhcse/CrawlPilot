#!/bin/bash

# Dynamic Plugin Builder
# Automatically discovers and builds all plugins in examples/plugins/

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Directories
PLUGIN_SRC_DIR="examples/plugins"
PLUGIN_OUT_DIR="plugins"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Change to project root
cd "$PROJECT_ROOT"

echo -e "${BLUE}🔌 Dynamic Plugin Builder${NC}"
echo ""

# Create output directory if it doesn't exist
mkdir -p "$PLUGIN_OUT_DIR"

# Find all plugin directories
PLUGIN_DIRS=$(find "$PLUGIN_SRC_DIR" -mindepth 1 -maxdepth 1 -type d)

if [ -z "$PLUGIN_DIRS" ]; then
    echo -e "${YELLOW}⚠️  No plugin directories found in $PLUGIN_SRC_DIR${NC}"
    exit 0
fi

# Count plugins
PLUGIN_COUNT=$(echo "$PLUGIN_DIRS" | wc -l | tr -d ' ')
echo -e "${BLUE}📦 Discovered $PLUGIN_COUNT plugin(s)${NC}"
echo ""

# Track successes and failures
SUCCESS_COUNT=0
FAIL_COUNT=0
FAILED_PLUGINS=()

# Build each plugin
for plugin_dir in $PLUGIN_DIRS; do
    PLUGIN_NAME=$(basename "$plugin_dir")
    PLUGIN_FILE="$plugin_dir/plugin.go"
    OUTPUT_FILE="$PLUGIN_OUT_DIR/$PLUGIN_NAME.so"
    
    echo -e "${BLUE}  → Building ${YELLOW}$PLUGIN_NAME${NC} plugin..."
    
    # Check if plugin.go exists
    if [ ! -f "$PLUGIN_FILE" ]; then
        echo -e "${RED}    ✗ plugin.go not found, skipping${NC}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        FAILED_PLUGINS+=("$PLUGIN_NAME (no plugin.go)")
        echo ""
        continue
    fi
    
    # Build the plugin
    if (cd "$plugin_dir" && \
        go mod tidy > /dev/null 2>&1 && \
        GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o "$PROJECT_ROOT/$OUTPUT_FILE" 2>&1); then
        
        # Get file size
        if [ -f "$OUTPUT_FILE" ]; then
            SIZE=$(du -h "$OUTPUT_FILE" | cut -f1)
            echo -e "${GREEN}    ✓ $PLUGIN_NAME built successfully ($SIZE)${NC}"
            SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        else
            echo -e "${RED}    ✗ Build succeeded but output file not found${NC}"
            FAIL_COUNT=$((FAIL_COUNT + 1))
            FAILED_PLUGINS+=("$PLUGIN_NAME (missing output)")
        fi
    else
        echo -e "${RED}    ✗ Build failed${NC}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        FAILED_PLUGINS+=("$PLUGIN_NAME (build error)")
    fi
    
    echo ""
done

# Summary
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}📊 Build Summary${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  ✓ Successful: $SUCCESS_COUNT${NC}"
if [ $FAIL_COUNT -gt 0 ]; then
    echo -e "${RED}  ✗ Failed: $FAIL_COUNT${NC}"
    echo ""
    echo -e "${RED}Failed plugins:${NC}"
    for failed in "${FAILED_PLUGINS[@]}"; do
        echo -e "${RED}  - $failed${NC}"
    done
fi
echo ""

# List built plugins
if [ $SUCCESS_COUNT -gt 0 ]; then
    echo -e "${BLUE}📦 Built plugins in $PLUGIN_OUT_DIR/:${NC}"
    ls -lh "$PLUGIN_OUT_DIR"/*.so 2>/dev/null | awk '{printf "  %s  %s\n", $9, $5}'
    echo ""
fi

# Exit code
if [ $FAIL_COUNT -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Some plugins failed to build${NC}"
    exit 1
else
    echo -e "${GREEN}✅ All plugins built successfully!${NC}"
    exit 0
fi
