.PHONY: all build build-backend build-plugins build-frontend clean run dev help

# Default target
all: build

# Build everything (backend + all plugins)
build: build-backend build-plugins
	@echo "✅ Build complete!"

# Build only the backend
build-backend:
	@echo "🔨 Building backend..."
	@cd cmd/crawler && go build -o main
	@echo "✅ Backend built"

# Build all plugins
build-plugins:
	@echo "🔌 Building plugins..."
	@$(MAKE) -s build-plugin-ecommerce-discovery
	@$(MAKE) -s build-plugin-aqua-extractor
	@echo "✅ All plugins built"

# Build individual plugins
build-plugin-ecommerce-discovery:
	@echo "  → Building ecommerce-discovery plugin..."
	@cd examples/plugins/ecommerce-discovery && \
		GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o ecommerce-discovery-linux-amd64.so && \
		cp ecommerce-discovery-linux-amd64.so ../../../plugins/
	@echo "  ✓ ecommerce-discovery built"

build-plugin-aqua-extractor:
	@echo "  → Building aqua-product-extractor plugin..."
	@cd examples/plugins/aqua-product-extractor && \
		GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o aqua-product-extractor.so && \
		cp aqua-product-extractor.so ../../../plugins/
	@echo "  ✓ aqua-product-extractor built"

# Build frontend
build-frontend:
	@echo "🎨 Building frontend..."
	@cd frontend && npm run build
	@echo "✅ Frontend built"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -f cmd/crawler/main
	@rm -f plugins/*.so
	@rm -f examples/plugins/*/*.so
	@echo "✅ Clean complete"

# Run the application (builds first if needed)
run: build
	@echo "🚀 Starting Crawlify..."
	@cd cmd/crawler && ./main

# Development mode: build and run with auto-reload
dev:
	@echo "🔧 Starting development mode..."
	@$(MAKE) build
	@echo "💡 Tip: Use 'make build' to rebuild when you make changes"
	@cd cmd/crawler && ./main

# Watch mode - rebuild on file changes (requires entr or similar)
watch:
	@echo "👀 Watching for changes..."
	@echo "Press Ctrl+C to stop"
	@find . -name '*.go' | entr -r make build run

# Help
help:
	@echo "Crawlify Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make              - Build backend and all plugins"
	@echo "  make build        - Build backend and all plugins"
	@echo "  make build-backend - Build only the backend"
	@echo "  make build-plugins - Build all plugins"
	@echo "  make build-frontend - Build frontend"
	@echo "  make run          - Build and run the application"
	@echo "  make dev          - Development mode (build + run)"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make watch        - Auto-rebuild on file changes (requires entr)"
	@echo "  make help         - Show this help message"
	@echo ""
	@echo "Individual plugin builds:"
	@echo "  make build-plugin-ecommerce-discovery"
	@echo "  make build-plugin-aqua-extractor"
