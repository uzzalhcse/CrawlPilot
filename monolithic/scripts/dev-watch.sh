#!/bin/bash
# Development watch script - automatically rebuilds on file changes
# Requires: inotifywait (install with: sudo apt-get install inotify-tools)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "👀 Crawlify Development Watch Mode"
echo "=================================="
echo ""
echo "Watching for .go file changes..."
echo "Press Ctrl+C to stop"
echo ""

# Initial build
echo "🔨 Initial build..."
"$SCRIPT_DIR/build-all.sh"
echo ""

# Watch for changes
while true; do
    # Wait for any .go file to change
    inotifywait -r -e modify,create,delete \
        "$PROJECT_ROOT/cmd" \
        "$PROJECT_ROOT/internal" \
        "$PROJECT_ROOT/pkg" \
        "$PROJECT_ROOT/api" \
        "$PROJECT_ROOT/examples/plugins" \
        --exclude '\.git|node_modules|\.so$' \
        2>/dev/null
    
    echo ""
    echo "🔄 Files changed, rebuilding..."
    "$SCRIPT_DIR/build-all.sh"
    echo ""
    echo "👀 Watching for changes..."
done
