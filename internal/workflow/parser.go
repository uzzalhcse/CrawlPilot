package workflow

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/uzzalhcse/crawlify/pkg/models"
	"gopkg.in/yaml.v3"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

// ParseFromFile parses a workflow from a file (JSON or YAML)
func (p *Parser) ParseFromFile(filePath string) (*models.WorkflowConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.ParseFromBytes(data, filePath)
}

// ParseFromBytes parses a workflow from bytes
func (p *Parser) ParseFromBytes(data []byte, filename string) (*models.WorkflowConfig, error) {
	var config models.WorkflowConfig

	// Try YAML first
	if err := yaml.Unmarshal(data, &config); err == nil {
		return &config, p.Validate(&config)
	}

	// Try JSON
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse workflow (tried both YAML and JSON): %w", err)
	}

	return &config, p.Validate(&config)
}

// Validate validates a workflow configuration
func (p *Parser) Validate(config *models.WorkflowConfig) error {
	if len(config.StartURLs) == 0 {
		return fmt.Errorf("workflow must have at least one start URL")
	}

	if len(config.Phases) == 0 {
		return fmt.Errorf("workflow must have at least one phase")
	}

	// Validate each phase
	for i, phase := range config.Phases {
		if phase.ID == "" {
			return fmt.Errorf("phase %d must have an ID", i)
		}
		if phase.Type == "" {
			return fmt.Errorf("phase '%s' must have a type", phase.ID)
		}

		// Validate nodes in this phase
		if err := p.validateNodes(phase.Nodes); err != nil {
			return fmt.Errorf("phase '%s' validation failed: %w", phase.ID, err)
		}
	}

	// Check for circular dependencies across all phases
	if err := p.checkCircularDependencies(config); err != nil {
		return err
	}

	return nil
}

// validateNodes validates a list of nodes
func (p *Parser) validateNodes(nodes []models.Node) error {
	nodeIDs := make(map[string]bool)

	for _, node := range nodes {
		if node.ID == "" {
			return fmt.Errorf("node must have an ID")
		}

		if node.Type == "" {
			return fmt.Errorf("node '%s' must have a type", node.ID)
		}

		// Check for duplicate node IDs
		if nodeIDs[node.ID] {
			return fmt.Errorf("duplicate node ID: %s", node.ID)
		}
		nodeIDs[node.ID] = true

		// Validate node type
		if !p.isValidNodeType(node.Type) {
			return fmt.Errorf("node '%s' has invalid type: %s", node.ID, node.Type)
		}
	}

	// Validate dependencies exist
	for _, node := range nodes {
		for _, depID := range node.Dependencies {
			if !nodeIDs[depID] {
				return fmt.Errorf("node '%s' depends on non-existent node '%s'", node.ID, depID)
			}
		}
	}

	return nil
}

// isValidNodeType checks if a node type is valid
func (p *Parser) isValidNodeType(nodeType models.NodeType) bool {
	validTypes := []models.NodeType{
		models.NodeTypeFetch,
		models.NodeTypeExtractLinks,
		models.NodeTypeFilterURLs,
		models.NodeTypeNavigate,
		models.NodeTypePaginate,
		models.NodeTypeClick,
		models.NodeTypeScroll,
		models.NodeTypeType,
		models.NodeTypeHover,
		models.NodeTypeWait,
		models.NodeTypeWaitFor,
		models.NodeTypeScreenshot,
		models.NodeTypeExtract,
		models.NodeTypeExtractText,
		models.NodeTypeExtractAttr,
		models.NodeTypeExtractJSON,
		models.NodeTypeTransform,
		models.NodeTypeFilter,
		models.NodeTypeMap,
		models.NodeTypeValidate,
		models.NodeTypeSequence, // NEW
		models.NodeTypeConditional,
		models.NodeTypeLoop,
		models.NodeTypeParallel,
	}

	for _, validType := range validTypes {
		if nodeType == validType {
			return true
		}
	}
	return false
}

// checkCircularDependencies checks for circular dependencies in the DAG
func (p *Parser) checkCircularDependencies(config *models.WorkflowConfig) error {
	// Collect all nodes from all phases
	var allNodes []models.Node
	for _, phase := range config.Phases {
		allNodes = append(allNodes, phase.Nodes...)
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(nodeID string) bool
	dfs = func(nodeID string) bool {
		visited[nodeID] = true
		recStack[nodeID] = true

		// Find the node
		var node *models.Node
		for i := range allNodes {
			if allNodes[i].ID == nodeID {
				node = &allNodes[i]
				break
			}
		}

		if node == nil {
			return false
		}

		// Visit all dependencies
		for _, depID := range node.Dependencies {
			if !visited[depID] {
				if dfs(depID) {
					return true
				}
			} else if recStack[depID] {
				return true
			}
		}

		recStack[nodeID] = false
		return false
	}

	for _, node := range allNodes {
		if !visited[node.ID] {
			if dfs(node.ID) {
				return fmt.Errorf("circular dependency detected involving node: %s", node.ID)
			}
		}
	}

	return nil
}

// BuildDAG builds a directed acyclic graph from nodes
func (p *Parser) BuildDAG(nodes []models.Node) (*DAG, error) {
	dag := NewDAG()

	// Add all nodes
	for _, node := range nodes {
		dag.AddNode(node)
	}

	// Add edges based on dependencies
	for _, node := range nodes {
		for _, depID := range node.Dependencies {
			if err := dag.AddEdge(depID, node.ID); err != nil {
				return nil, err
			}
		}
	}

	return dag, nil
}
