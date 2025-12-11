package bayesian

import (
	"fmt"
)

// BayesianNetwork represents a Bayesian network that can generate random samples.
type BayesianNetwork struct {
	NodesInSamplingOrder []*BayesianNode
	NodesByName          map[string]*BayesianNode
}

// NewBayesianNetwork creates a new BayesianNetwork from a network definition.
func NewBayesianNetwork(networkDef map[string]interface{}) (*BayesianNetwork, error) {
	nodesRaw, ok := networkDef["nodes"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("network definition missing 'nodes' array")
	}

	nodes := make([]*BayesianNode, 0, len(nodesRaw))
	nodesByName := make(map[string]*BayesianNode)

	for _, nodeRaw := range nodesRaw {
		nodeDef, ok := nodeRaw.(map[string]interface{})
		if !ok {
			continue
		}
		node := NewBayesianNode(nodeDef)
		nodes = append(nodes, node)
		nodesByName[node.Name()] = node
	}

	return &BayesianNetwork{
		NodesInSamplingOrder: nodes,
		NodesByName:          nodesByName,
	}, nil
}

// GenerateSample randomly samples from the distribution represented by the Bayesian network.
func (bn *BayesianNetwork) GenerateSample(inputValues map[string]interface{}) map[string]interface{} {
	sample := make(map[string]interface{})
	for k, v := range inputValues {
		sample[k] = v
	}

	for _, node := range bn.NodesInSamplingOrder {
		nodeName := node.Name()
		if _, exists := sample[nodeName]; !exists {
			sample[nodeName] = node.Sample(sample)
		}
	}

	return sample
}

// GenerateConsistentSampleWhenPossible randomly samples values from the distribution,
// ensuring the sample is consistent with the provided restrictions on value possibilities.
// Returns nil if no such sample can be generated.
func (bn *BayesianNetwork) GenerateConsistentSampleWhenPossible(
	valuePossibilities map[string][]string,
) map[string]interface{} {
	return bn.recursivelyGenerateConsistentSampleWhenPossible(
		make(map[string]interface{}),
		valuePossibilities,
		0,
	)
}

// recursivelyGenerateConsistentSampleWhenPossible recursively generates a random sample
// consistent with the given restrictions on possible values.
func (bn *BayesianNetwork) recursivelyGenerateConsistentSampleWhenPossible(
	sampleSoFar map[string]interface{},
	valuePossibilities map[string][]string,
	depth int,
) map[string]interface{} {
	if depth == len(bn.NodesInSamplingOrder) {
		return sampleSoFar
	}

	node := bn.NodesInSamplingOrder[depth]
	bannedValues := make([]string, 0)

	for {
		// Get possibilities for this node
		possibilities := valuePossibilities[node.Name()]
		if possibilities == nil {
			possibilities = node.PossibleValues()
		}

		sampleValue := node.SampleAccordingToRestrictions(
			sampleSoFar,
			possibilities,
			bannedValues,
		)

		if sampleValue == "" {
			break
		}

		sampleSoFar[node.Name()] = sampleValue

		nextSample := bn.recursivelyGenerateConsistentSampleWhenPossible(
			sampleSoFar,
			valuePossibilities,
			depth+1,
		)

		if nextSample != nil {
			return nextSample
		}

		bannedValues = append(bannedValues, sampleValue)
		delete(sampleSoFar, node.Name())
	}

	return nil
}

// GetPossibleValues computes extended constraints induced by the original constraints
// and network structure.
func GetPossibleValues(
	network *BayesianNetwork,
	possibleValues map[string][]string,
) (map[string][]string, error) {
	sets := make([]map[string][]string, 0)

	for key, values := range possibleValues {
		if len(values) == 0 {
			return nil, fmt.Errorf("no possible values for key: %s", key)
		}

		node, exists := network.NodesByName[key]
		if !exists {
			continue
		}

		// Get the conditional probabilities tree without deeper/skip
		condProbs, ok := node.NodeDefinition["conditionalProbabilities"].(map[string]interface{})
		if !ok {
			continue
		}

		tree := undeeper(condProbs)
		zippedValues := filterByLastLevelKeys(tree, values)

		setDict := make(map[string][]string)
		parentNames := node.ParentNames()
		for i, parentName := range parentNames {
			if i < len(zippedValues) {
				setDict[parentName] = zippedValues[i]
			}
		}
		setDict[key] = values

		sets = append(sets, setDict)
	}

	// Compute intersection of all possible values for each node
	result := make(map[string][]string)
	for _, setDict := range sets {
		for key, values := range setDict {
			if existing, exists := result[key]; exists {
				intersected := arrayIntersection(values, existing)
				if len(intersected) == 0 {
					return nil, fmt.Errorf("constraints too restrictive for key: %s", key)
				}
				result[key] = intersected
			} else {
				result[key] = values
			}
		}
	}

	return result, nil
}

// undeeper removes the "deeper/skip" structures from the conditional probability table
func undeeper(obj map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range obj {
		if key == "skip" {
			continue
		}
		if key == "deeper" {
			if deeperMap, ok := value.(map[string]interface{}); ok {
				deeper := undeeper(deeperMap)
				for k, v := range deeper {
					result[k] = v
				}
			}
		} else {
			if nestedMap, ok := value.(map[string]interface{}); ok {
				result[key] = undeeper(nestedMap)
			} else {
				result[key] = value
			}
		}
	}

	return result
}

// filterByLastLevelKeys performs DFS on the tree and returns values of nodes
// on paths that end with the given keys (stored by levels)
func filterByLastLevelKeys(tree map[string]interface{}, validKeys []string) [][]string {
	validKeySet := make(map[string]bool)
	for _, k := range validKeys {
		validKeySet[k] = true
	}

	var out [][]string

	var recurse func(t map[string]interface{}, acc []string)
	recurse = func(t map[string]interface{}, acc []string) {
		for key, value := range t {
			if nestedMap, ok := value.(map[string]interface{}); ok && nestedMap != nil {
				recurse(nestedMap, append(acc, key))
			} else {
				if validKeySet[key] {
					if len(out) == 0 {
						out = make([][]string, len(acc))
						for i, v := range acc {
							out[i] = []string{v}
						}
					} else {
						out = arrayZip(out, acc)
					}
				}
			}
		}
	}

	recurse(tree, nil)
	return out
}

// arrayZip combines two arrays using set union
func arrayZip(a [][]string, b []string) [][]string {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	result := make([][]string, minLen)
	for i := 0; i < minLen; i++ {
		// Create union of a[i] and b[i]
		seen := make(map[string]bool)
		combined := make([]string, 0)
		for _, v := range a[i] {
			if !seen[v] {
				seen[v] = true
				combined = append(combined, v)
			}
		}
		if !seen[b[i]] {
			combined = append(combined, b[i])
		}
		result[i] = combined
	}
	return result
}

// arrayIntersection performs set intersection on two arrays
func arrayIntersection(a, b []string) []string {
	setB := make(map[string]bool)
	for _, v := range b {
		setB[v] = true
	}

	result := make([]string, 0)
	for _, v := range a {
		if setB[v] {
			result = append(result, v)
		}
	}
	return result
}
