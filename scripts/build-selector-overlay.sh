#!/bin/bash
set -e

echo "Building Selector Overlay..."

# Navigate to selector overlay app directory
cd selector-overlay-app

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install
fi

# Build the app
echo "Building Vue app..."
npm run build

echo "✅ Selector overlay built successfully!"
echo "Output: internal/browser/dist/selector-overlay.js"
