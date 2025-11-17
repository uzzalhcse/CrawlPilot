package workflow

import (
	"fmt"

	"github.com/uzzalhcse/crawlify/pkg/models"
)

// DAG represents a Directed Acyclic Graph
type DAG struct {
	Nodes map[string]*models.Node
	Edges map[string][]string // nodeID -> list of dependent node IDs
}

func NewDAG() *DAG {
	return &DAG{
		Nodes: make(map[string]*models.Node),
		Edges: make(map[string][]string),
	}
}

// AddNode adds a node to the DAG
func (d *DAG) AddNode(node models.Node) {
	d.Nodes[node.ID] = &node
	if _, exists := d.Edges[node.ID]; !exists {
		d.Edges[node.ID] = []string{}
	}
}

// AddEdge adds an edge from one node to another
func (d *DAG) AddEdge(from, to string) error {
	if _, exists := d.Nodes[from]; !exists {
		return fmt.Errorf("source node does not exist: %s", from)
	}
	if _, exists := d.Nodes[to]; !exists {
		return fmt.Errorf("destination node does not exist: %s", to)
	}

	d.Edges[from] = append(d.Edges[from], to)
	return nil
}

// TopologicalSort returns nodes in topological order
func (d *DAG) TopologicalSort() ([]*models.Node, error) {
	inDegree := make(map[string]int)

	// Initialize in-degrees
	for nodeID := range d.Nodes {
		inDegree[nodeID] = 0
	}

	// Calculate in-degrees
	for _, dependencies := range d.Edges {
		for _, depID := range dependencies {
			inDegree[depID]++
		}
	}

	// Find nodes with no incoming edges
	queue := []*models.Node{}
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, d.Nodes[nodeID])
		}
	}

	result := []*models.Node{}

	for len(queue) > 0 {
		// Dequeue
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// Reduce in-degree for dependent nodes
		for _, depID := range d.Edges[node.ID] {
			inDegree[depID]--
			if inDegree[depID] == 0 {
				queue = append(queue, d.Nodes[depID])
			}
		}
	}

	// Check if all nodes were processed
	if len(result) != len(d.Nodes) {
		return nil, fmt.Errorf("cycle detected in DAG")
	}

	return result, nil
}

// GetRootNodes returns nodes with no dependencies
func (d *DAG) GetRootNodes() []*models.Node {
	roots := []*models.Node{}

	hasIncoming := make(map[string]bool)
	for _, dependencies := range d.Edges {
		for _, depID := range dependencies {
			hasIncoming[depID] = true
		}
	}

	for nodeID, node := range d.Nodes {
		if !hasIncoming[nodeID] {
			roots = append(roots, node)
		}
	}

	return roots
}

// GetDependents returns all nodes that depend on the given node
func (d *DAG) GetDependents(nodeID string) []*models.Node {
	dependents := []*models.Node{}

	for _, depID := range d.Edges[nodeID] {
		if node, exists := d.Nodes[depID]; exists {
			dependents = append(dependents, node)
		}
	}

	return dependents
}

// GetDependencies returns all nodes that the given node depends on
func (d *DAG) GetDependencies(nodeID string) []*models.Node {
	node, exists := d.Nodes[nodeID]
	if !exists {
		return []*models.Node{}
	}

	dependencies := []*models.Node{}
	for _, depID := range node.Dependencies {
		if depNode, exists := d.Nodes[depID]; exists {
			dependencies = append(dependencies, depNode)
		}
	}

	return dependencies
}
