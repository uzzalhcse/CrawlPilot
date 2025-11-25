package plugins

import (
	"fmt"
	"plugin"
	"sync"

	"github.com/uzzalhcse/crawlify/pkg/plugins"
	"go.uber.org/zap"
)

// LoadedPlugin represents a loaded plugin with its instance
type LoadedPlugin struct {
	Info     plugins.PluginInfo
	Instance interface{} // DiscoveryPlugin or ExtractionPlugin
	Plugin   *plugin.Plugin
	Path     string
}

// PluginLoader manages loading and unloading of compiled plugins
type PluginLoader struct {
	loadedPlugins map[string]*LoadedPlugin
	mu            sync.RWMutex
	logger        *zap.Logger
}

// NewPluginLoader creates a new plugin loader
func NewPluginLoader(logger *zap.Logger) *PluginLoader {
	return &PluginLoader{
		loadedPlugins: make(map[string]*LoadedPlugin),
		logger:        logger,
	}
}

// LoadPlugin loads a compiled plugin from file
func (pl *PluginLoader) LoadPlugin(path string) (*LoadedPlugin, error) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	pl.logger.Info("Loading plugin", zap.String("path", path))

	// Open the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	// Try loading as discovery plugin first
	discoverySymbol, err := p.Lookup("NewDiscoveryPlugin")
	if err == nil {
		// It's a discovery plugin
		newPlugin, ok := discoverySymbol.(func() plugins.DiscoveryPlugin)
		if !ok {
			return nil, fmt.Errorf("NewDiscoveryPlugin has wrong signature")
		}

		instance := newPlugin()
		info := instance.Info()

		loaded := &LoadedPlugin{
			Info:     info,
			Instance: instance,
			Plugin:   p,
			Path:     path,
		}

		pl.loadedPlugins[info.ID] = loaded
		pl.logger.Info("Loaded discovery plugin",
			zap.String("id", info.ID),
			zap.String("name", info.Name),
			zap.String("version", info.Version))

		return loaded, nil
	}

	// Try loading as extraction plugin
	extractionSymbol, err := p.Lookup("NewExtractionPlugin")
	if err == nil {
		// It's an extraction plugin
		newPlugin, ok := extractionSymbol.(func() plugins.ExtractionPlugin)
		if !ok {
			return nil, fmt.Errorf("NewExtractionPlugin has wrong signature")
		}

		instance := newPlugin()
		info := instance.Info()

		loaded := &LoadedPlugin{
			Info:     info,
			Instance: instance,
			Plugin:   p,
			Path:     path,
		}

		pl.loadedPlugins[info.ID] = loaded
		pl.logger.Info("Loaded extraction plugin",
			zap.String("id", info.ID),
			zap.String("name", info.Name),
			zap.String("version", info.Version))

		return loaded, nil
	}

	return nil, fmt.Errorf("plugin must export either NewDiscoveryPlugin or NewExtractionPlugin function")
}

// GetPlugin retrieves a loaded plugin by ID
func (pl *PluginLoader) GetPlugin(pluginID string) (*LoadedPlugin, error) {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	loaded, exists := pl.loadedPlugins[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin %s not loaded", pluginID)
	}

	return loaded, nil
}

// UnloadPlugin unloads a plugin
func (pl *PluginLoader) UnloadPlugin(pluginID string) error {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	loaded, exists := pl.loadedPlugins[pluginID]
	if !exists {
		return fmt.Errorf("plugin %s not loaded", pluginID)
	}

	delete(pl.loadedPlugins, pluginID)

	pl.logger.Info("Unloaded plugin",
		zap.String("id", pluginID),
		zap.String("name", loaded.Info.Name))

	// Note: Go's plugin package doesn't support actual unloading
	// The plugin stays in memory, we just remove our reference
	return nil
}

// ListLoadedPlugins returns all loaded plugins
func (pl *PluginLoader) ListLoadedPlugins() []plugins.PluginInfo {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	infos := make([]plugins.PluginInfo, 0, len(pl.loadedPlugins))
	for _, loaded := range pl.loadedPlugins {
		infos = append(infos, loaded.Info)
	}

	return infos
}

// IsLoaded checks if a plugin is loaded
func (pl *PluginLoader) IsLoaded(pluginID string) bool {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	_, exists := pl.loadedPlugins[pluginID]
	return exists
}

// ReloadPlugin reloads a plugin (by path)
func (pl *PluginLoader) ReloadPlugin(pluginID string, newPath string) error {
	// Unload existing
	if err := pl.UnloadPlugin(pluginID); err != nil {
		pl.logger.Warn("Failed to unload plugin during reload",
			zap.String("id", pluginID),
			zap.Error(err))
	}

	// Load new version
	_, err := pl.LoadPlugin(newPath)
	return err
}

// ValidatePlugin checks if a plugin file is valid before loading
func (pl *PluginLoader) ValidatePlugin(path string) error {
	// Try to open the plugin without loading it into our map
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("invalid plugin file: %w", err)
	}

	// Check for required exports
	hasDiscovery := false
	hasExtraction := false

	if _, err := p.Lookup("NewDiscoveryPlugin"); err == nil {
		hasDiscovery = true
	}

	if _, err := p.Lookup("NewExtractionPlugin"); err == nil {
		hasExtraction = true
	}

	if !hasDiscovery && !hasExtraction {
		return fmt.Errorf("plugin must export either NewDiscoveryPlugin or NewExtractionPlugin")
	}

	if hasDiscovery && hasExtraction {
		return fmt.Errorf("plugin cannot export both NewDiscoveryPlugin and NewExtractionPlugin")
	}

	return nil
}
