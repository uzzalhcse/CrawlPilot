package nodes

import (
	"fmt"
	"sync"
)

// Registry manages node executors
type Registry struct {
	executors map[string]NodeExecutor
	mu        sync.RWMutex
}

// NewRegistry creates a new node registry
func NewRegistry() *Registry {
	registry := &Registry{
		executors: make(map[string]NodeExecutor),
	}

	// Register built-in executors
	registry.Register(NewNavigateNode())
	registry.Register(NewClickNode())
	registry.Register(NewTypeNode())
	registry.Register(NewWaitNode())
	registry.Register(NewExtractNode())
	registry.Register(NewDiscoverLinksNode())
	registry.Register(NewExtractLinksNode()) // Add extract_links support
	registry.Register(NewScriptNode())

	return registry
}

// Register adds a node executor to the registry
func (r *Registry) Register(executor NodeExecutor) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.executors[executor.Type()] = executor
}

// Get retrieves a node executor by type
func (r *Registry) Get(nodeType string) (NodeExecutor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	executor, ok := r.executors[nodeType]
	if !ok {
		return nil, fmt.Errorf("no executor found for node type: %s", nodeType)
	}

	return executor, nil
}

// List returns all registered node types
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.executors))
	for nodeType := range r.executors {
		types = append(types, nodeType)
	}

	return types
}
