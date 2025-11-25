# Plugin Development Workflow Guide

## 🚨 The Plugin Version Problem

Go plugins (.so files) must be built with the **exact same version** of all dependencies as the main application. When you rebuild the backend, all plugins become incompatible and need to be rebuilt.

**Error you'll see:**
```
plugin was built with a different version of package github.com/uzzalhcse/crawlify/pkg/models
```

---

## ✅ Solution 1: Makefile (RECOMMENDED)

Use the provided `Makefile` to build everything together:

```bash
# Build backend + all plugins
make build

# Just rebuild backend
make build-backend

# Just rebuild all plugins
make build-plugins

# Build and run
make run

# Clean all build artifacts
make clean
```

### Adding a New Plugin to Makefile

Edit `Makefile` and add:

```makefile
build-plugin-your-plugin:
	@echo "  → Building your-plugin..."
	@cd examples/plugins/your-plugin && \
		GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o your-plugin.so && \
		cp your-plugin.so ../../../plugins/
	@echo "  ✓ your-plugin built"
```

Then add it to the `build-plugins` target:
```makefile
build-plugins:
	@echo "🔌 Building plugins..."
	@$(MAKE) -s build-plugin-ecommerce-discovery
	@$(MAKE) -s build-plugin-aqua-extractor
	@$(MAKE) -s build-plugin-your-plugin  # Add this
	@echo "✅ All plugins built"
```

---

## ✅ Solution 2: Build Script

Use the automated build script:

```bash
# Build everything
./scripts/build-all.sh

# Watch for changes and auto-rebuild
./scripts/dev-watch.sh
```

### Adding a New Plugin to Build Script

Edit `scripts/build-all.sh` and add:

```bash
# Plugin N: Your Plugin
echo "  → Building your-plugin..."
cd "$PROJECT_ROOT/examples/plugins/your-plugin"
GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o your-plugin.so
cp your-plugin.so "$PROJECT_ROOT/plugins/"
echo "  ✓ your-plugin built"
```

---

## ✅ Solution 3: Auto-Watch Mode

For continuous development, use the watch script:

```bash
# Terminal 1: Watch and auto-rebuild
./scripts/dev-watch.sh

# Terminal 2: Run the application
cd cmd/crawler && ./main
```

**Requires:** `inotify-tools` (install with `sudo apt-get install inotify-tools`)

---

## 🎯 Recommended Development Workflow

### Option A: Manual Control (Makefile)
```bash
# When you make changes:
make build    # Rebuilds backend + all plugins
make run      # Runs the application
```

### Option B: Auto-Rebuild (Watch Script)
```bash
# Terminal 1
./scripts/dev-watch.sh    # Auto-rebuilds on file changes

# Terminal 2  
cd cmd/crawler && ./main  # Run manually after rebuild
```

### Option C: Quick Development Cycle
```bash
# One command to build and run
make run
```

---

## 🔄 Alternative Approaches (Future Considerations)

### 1. **gRPC-Based Plugins** (Most Flexible)
Instead of .so files, use gRPC for plugins:
- ✅ No version compatibility issues
- ✅ Plugins can be in any language
- ✅ Plugins can run as separate processes
- ✅ Hot-reload without restarting
- ❌ More complex setup
- ❌ Network overhead

### 2. **HashiCorp go-plugin** (Industry Standard)
Uses gRPC under the hood with nice abstractions:
- ✅ Battle-tested (used by Terraform, Vault, etc.)
- ✅ Clean plugin interface
- ✅ No version issues
- ❌ Requires additional dependency

### 3. **WebAssembly (WASM)** (Emerging)
Compile plugins to WASM:
- ✅ Language agnostic
- ✅ Sandboxed execution
- ✅ No version issues
- ❌ Limited Go support currently
- ❌ Performance overhead

---

## 📝 Best Practices

### During Development
1. **Always use `make build`** instead of rebuilding backend manually
2. **Use `make clean`** if you encounter weird errors
3. **Commit both backend and plugin changes together**

### For Production
1. **Build all plugins with the same CI/CD pipeline** as the backend
2. **Version plugins alongside the main application**
3. **Store plugin binaries with release artifacts**

### Testing New Plugins
```bash
# 1. Create your plugin
cd examples/plugins/my-plugin

# 2. Add to Makefile
# (edit Makefile to include your plugin)

# 3. Build everything
cd ../../../
make build

# 4. Test
cd cmd/crawler && ./main
```

---

## 🐛 Troubleshooting

### "plugin was built with a different version"
**Fix:** Rebuild everything together
```bash
make clean
make build
```

### Plugin not loading
**Check:**
1. Plugin file exists in `plugins/` directory
2. Plugin has correct constructor function (`NewExtractionPlugin` or `NewDiscoveryPlugin`)
3. Plugin implements the correct interface

### Build errors
**Fix:**
```bash
# Clean and rebuild
make clean
cd examples/plugins/your-plugin
go mod tidy
cd ../../..
make build
```

---

## 📚 Quick Reference

| Command | Description |
|---------|-------------|
| `make` | Build everything |
| `make build` | Build backend + plugins |
| `make run` | Build and run |
| `make clean` | Remove build artifacts |
| `make dev` | Development mode |
| `make build-backend` | Build only backend |
| `make build-plugins` | Build only plugins |
| `./scripts/build-all.sh` | Alternative build script |
| `./scripts/dev-watch.sh` | Watch mode |

---

## 🎓 Summary

**The solution:** Use the Makefile or build script to always build the backend and plugins together. This ensures version compatibility and saves you from manual rebuilding headaches.

**Development workflow:**
```bash
# Edit code → make build → test
```

**That's it!** 🎉
