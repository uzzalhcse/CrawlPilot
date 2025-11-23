package workflow

import (
	"fmt"
	"sync"

	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes/discovery"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes/extraction"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes/interaction"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// NodeRegistry manages registered node executors
type NodeRegistry struct {
	executors map[models.NodeType]nodes.NodeExecutor
	mu        sync.RWMutex
}

// NewNodeRegistry creates a new node registry
func NewNodeRegistry() *NodeRegistry {
	return &NodeRegistry{
		executors: make(map[models.NodeType]nodes.NodeExecutor),
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

// RegisterDefaultNodes registers all built-in node types
func (r *NodeRegistry) RegisterDefaultNodes() error {
	// Discovery nodes
	if err := r.Register(discovery.NewExtractLinksExecutor()); err != nil {
		return fmt.Errorf("failed to register extract_links: %w", err)
	}
	if err := r.Register(discovery.NewNavigateExecutor()); err != nil {
		return fmt.Errorf("failed to register navigate: %w", err)
	}
	if err := r.Register(discovery.NewPaginateExecutor()); err != nil {
		return fmt.Errorf("failed to register paginate: %w", err)
	}

	// Extraction nodes
	if err := r.Register(extraction.NewExtractExecutor()); err != nil {
		return fmt.Errorf("failed to register extract: %w", err)
	}
	if err := r.Register(extraction.NewExtractJSONExecutor()); err != nil {
		return fmt.Errorf("failed to register extract_json: %w", err)
	}

	// Interaction nodes
	if err := r.Register(interaction.NewClickExecutor()); err != nil {
		return fmt.Errorf("failed to register click: %w", err)
	}
	if err := r.Register(interaction.NewScrollExecutor()); err != nil {
		return fmt.Errorf("failed to register scroll: %w", err)
	}
	if err := r.Register(interaction.NewTypeExecutor()); err != nil {
		return fmt.Errorf("failed to register type: %w", err)
	}
	if err := r.Register(interaction.NewHoverExecutor()); err != nil {
		return fmt.Errorf("failed to register hover: %w", err)
	}
	if err := r.Register(interaction.NewWaitExecutor()); err != nil {
		return fmt.Errorf("failed to register wait: %w", err)
	}
	if err := r.Register(interaction.NewWaitForExecutor()); err != nil {
		return fmt.Errorf("failed to register wait_for: %w", err)
	}
	if err := r.Register(interaction.NewScreenshotExecutor()); err != nil {
		return fmt.Errorf("failed to register screenshot: %w", err)
	}

	// Advanced nodes (need registry reference)
	if err := r.Register(interaction.NewSequenceExecutor(r)); err != nil {
		return fmt.Errorf("failed to register sequence: %w", err)
	}
	if err := r.Register(interaction.NewConditionalExecutor(r)); err != nil {
		return fmt.Errorf("failed to register conditional: %w", err)
	}

	return nil
}
