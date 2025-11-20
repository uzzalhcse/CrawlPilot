#!/bin/bash
set -e

DIST_DIR="internal/browser/dist"
REQUIRED_FILES=("selector-overlay.js" "selector-overlay.css")

echo "Checking for selector overlay build artifacts..."

for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$DIST_DIR/$file" ]; then
        echo "❌ Missing: $DIST_DIR/$file"
        echo "Run: cd selector-overlay-app && npm install && npm run build"
        exit 1
    fi
done

echo "✅ All selector overlay files present"
