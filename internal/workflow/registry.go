package workflow

import (
	"fmt"
	"sync"

	"github.com/uzzalhcse/crawlify/internal/plugins"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes/discovery"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes/extraction"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes/interaction"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// NodeRegistry manages registered node executors
type NodeRegistry struct {
	executors       map[models.NodeType]nodes.NodeExecutor
	pluginExecutors map[string]*plugins.PluginExecutor // pluginID -> executor
	pluginLoader    *plugins.PluginLoader
	mu              sync.RWMutex
	logger          *zap.Logger
}

// NewNodeRegistry creates a new node registry
func NewNodeRegistry() *NodeRegistry {
	return NewNodeRegistryWithLogger(zap.NewNop())
}

// NewNodeRegistryWithLogger creates a new node registry with logger
func NewNodeRegistryWithLogger(logger *zap.Logger) *NodeRegistry {
	return &NodeRegistry{
		executors:       make(map[models.NodeType]nodes.NodeExecutor),
		pluginExecutors: make(map[string]*plugins.PluginExecutor),
		pluginLoader:    plugins.NewPluginLoader(logger),
		logger:          logger,
	}
}

// Register registers a node executor
func (r *NodeRegistry) Register(executor nodes.NodeExecutor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	nodeType := executor.Type()
	if _, exists := r.executors[nodeType]; exists {
		return fmt.Errorf("executor for type %s already registered", nodeType)
	}
	r.executors[nodeType] = executor
	return nil
}

// Get retrieves a registered executor by type
func (r *NodeRegistry) Get(nodeType models.NodeType) (nodes.NodeExecutor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	executor, exists := r.executors[nodeType]
	if !exists {
		return nil, fmt.Errorf("no executor registered for type: %s", nodeType)
	}
	return executor, nil
}

// IsRegistered checks if a node type is registered
func (r *NodeRegistry) IsRegistered(nodeType models.NodeType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.executors[nodeType]
	return exists
}

// ListRegistered returns all registered node types
func (r *NodeRegistry) ListRegistered() []models.NodeType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]models.NodeType, 0, len(r.executors))
	for nodeType := range r.executors {
		types = append(types, nodeType)
	}
	return types
}

// RegisterDefaultNodes registers all built-in node executors
func (r *NodeRegistry) RegisterDefaultNodes() error {
	// Discovery nodes
	if err := r.Register(discovery.NewExtractLinksExecutor()); err != nil {
		return err
	}
	if err := r.Register(discovery.NewNavigateExecutor()); err != nil {
		return err
	}
	if err := r.Register(discovery.NewPaginateExecutor()); err != nil {
		return err
	}

	// Extraction nodes
	if err := r.Register(extraction.NewExtractExecutor()); err != nil {
		return err
	}

	// Interaction nodes
	if err := r.Register(interaction.NewClickExecutor()); err != nil {
		return err
	}
	if err := r.Register(interaction.NewWaitExecutor()); err != nil {
		return err
	}

	// Plugin executor
	if err := r.Register(nodes.NewPluginNodeExecutor(r.logger)); err != nil {
		return err
	}

	return nil
}

// RegisterPlugin registers a compiled plugin from file
func (r *NodeRegistry) RegisterPlugin(pluginPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load the plugin
	loaded, err := r.pluginLoader.LoadPlugin(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	// Create plugin executor wrapper
	pluginExec := plugins.NewPluginExecutor(loaded, r.logger)

	// Register as node executor
	nodeType := pluginExec.Type()
	if _, exists := r.executors[nodeType]; exists {
		return fmt.Errorf("plugin executor for type %s already registered", nodeType)
	}

	r.executors[nodeType] = pluginExec
	r.pluginExecutors[loaded.Info.ID] = pluginExec

	r.logger.Info("Registered plugin",
		zap.String("id", loaded.Info.ID),
		zap.String("name", loaded.Info.Name),
		zap.String("version", loaded.Info.Version))

	return nil
}

// UnregisterPlugin unregisters a plugin
func (r *NodeRegistry) UnregisterPlugin(pluginID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pluginExec, exists := r.pluginExecutors[pluginID]
	if !exists {
		return fmt.Errorf("plugin %s not registered", pluginID)
	}

	// Remove from executors
	delete(r.executors, pluginExec.Type())
	delete(r.pluginExecutors, pluginID)

	// Unload plugin
	if err := r.pluginLoader.UnloadPlugin(pluginID); err != nil {
		r.logger.Warn("Failed to unload plugin",
			zap.String("id", pluginID),
			zap.Error(err))
	}

	return nil
}

// GetPluginExecutor retrieves a plugin executor by plugin ID
func (r *NodeRegistry) GetPluginExecutor(pluginID string) (*plugins.PluginExecutor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pluginExec, exists := r.pluginExecutors[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin %s not registered", pluginID)
	}

	return pluginExec, nil
}

// ListPlugins returns all registered plugins
func (r *NodeRegistry) ListPlugins() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]string, 0, len(r.pluginExecutors))
	for pluginID := range r.pluginExecutors {
		plugins = append(plugins, pluginID)
	}
	return plugins
}

// ReloadPlugin hot-reloads a plugin with new version
func (r *NodeRegistry) ReloadPlugin(pluginID, newPath string) error {
	// Unregister existing
	if err := r.UnregisterPlugin(pluginID); err != nil {
		r.logger.Warn("Failed to unregister plugin during reload",
			zap.String("id", pluginID),
			zap.Error(err))
	}

	// Register new version
	return r.RegisterPlugin(newPath)
}
