.PHONY: all build build-backend build-plugins build-frontend clean run dev help

# Directories
PLUGIN_SRC_DIR := examples/plugins
PLUGIN_OUT_DIR := plugins

# Dynamically find all plugin directories
PLUGIN_DIRS := $(shell find $(PLUGIN_SRC_DIR) -mindepth 1 -maxdepth 1 -type d)
PLUGIN_NAMES := $(notdir $(PLUGIN_DIRS))
PLUGIN_BINARIES := $(addprefix $(PLUGIN_OUT_DIR)/,$(addsuffix .so,$(PLUGIN_NAMES)))

# Default target
.PHONY: all
all: build

# Build everything (backend + all plugins)
.PHONY: build
build: build-backend build-plugins

# Clean everything
.PHONY: clean
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -f cmd/crawler/main
	@rm -f $(PLUGIN_OUT_DIR)/*.so
	@echo "✅ Clean complete"

# Build backend
.PHONY: build-backend
build-backend:
	@echo "🔨 Building backend..."
	@cd cmd/crawler && go build -o main
	@echo "✅ Backend built"

# Build all plugins dynamically
.PHONY: build-plugins
build-plugins: $(PLUGIN_BINARIES)
	@echo "✅ All plugins built"

# Pattern rule to build individual plugins
$(PLUGIN_OUT_DIR)/%.so: $(PLUGIN_SRC_DIR)/%/plugin.go
	@echo "  → Building $* plugin..."
	@mkdir -p $(PLUGIN_OUT_DIR)
	@cd $(PLUGIN_SRC_DIR)/$* && \
		go mod tidy > /dev/null 2>&1 && \
		GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o $(CURDIR)/$(PLUGIN_OUT_DIR)/$*.so
	@echo "  ✓ $* built"

# Build a specific plugin
.PHONY: build-plugin-%
build-plugin-%:
	@echo "🔨 Building $* plugin..."
	@mkdir -p $(PLUGIN_OUT_DIR)
	@cd $(PLUGIN_SRC_DIR)/$* && \
		go mod tidy && \
		GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o $(CURDIR)/$(PLUGIN_OUT_DIR)/$*.so
	@echo "✅ $* plugin built"

# List all discovered plugins
.PHONY: list-plugins
list-plugins:
	@echo "📦 Discovered plugins:"
	@for plugin in $(PLUGIN_NAMES); do \
		echo "  - $$plugin"; \
	done

# Run backend (builds first if needed)
.PHONY: run
run: build
	@echo "🚀 Starting Crawlify backend..."
	@cd cmd/crawler && ./main

# Development watch mode (requires entr: brew install entr)
.PHONY: watch
watch:
	@echo "👀 Watching for changes..."
	@find . -name '*.go' | entr -r make run

# Help
.PHONY: help
help:
	@echo "Crawlify Makefile Commands:"
	@echo ""
	@echo "  make build              Build backend and all plugins"
	@echo "  make build-backend      Build only the backend"
	@echo "  make build-plugins      Build all plugins"
	@echo "  make build-plugin-NAME  Build a specific plugin"
	@echo "  make list-plugins       List all discovered plugins"
	@echo "  make run                Build and run the backend"
	@echo "  make clean              Clean all build artifacts"
	@echo "  make watch              Watch for changes and rebuild"
	@echo "  make help               Show this help message"
	@echo ""
	@echo "Plugin directories: $(PLUGIN_SRC_DIR)"
	@echo "Plugin output: $(PLUGIN_OUT_DIR)"
