// Package bayesian provides Bayesian network implementation for browserforge-go.
// It enables probabilistic sampling of browser fingerprints and headers
// using conditional probability distributions.
package bayesian

import (
	"math/rand"
)

// BayesianNode represents a single node in a Bayesian network.
// It supports sampling from its conditional probability distribution.
type BayesianNode struct {
	NodeDefinition map[string]interface{}
}

// NewBayesianNode creates a new BayesianNode from a node definition.
func NewBayesianNode(nodeDefinition map[string]interface{}) *BayesianNode {
	return &BayesianNode{
		NodeDefinition: nodeDefinition,
	}
}

// Name returns the name of this node.
func (n *BayesianNode) Name() string {
	if name, ok := n.NodeDefinition["name"].(string); ok {
		return name
	}
	return ""
}

// ParentNames returns the names of parent nodes.
func (n *BayesianNode) ParentNames() []string {
	if parents, ok := n.NodeDefinition["parentNames"].([]interface{}); ok {
		result := make([]string, 0, len(parents))
		for _, p := range parents {
			if s, ok := p.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// PossibleValues returns all possible values for this node.
func (n *BayesianNode) PossibleValues() []string {
	if values, ok := n.NodeDefinition["possibleValues"].([]interface{}); ok {
		result := make([]string, 0, len(values))
		for _, v := range values {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// GetProbabilitiesGivenKnownValues extracts unconditional probabilities
// of node values given the values of parent nodes.
func (n *BayesianNode) GetProbabilitiesGivenKnownValues(parentValues map[string]interface{}) map[string]float64 {
	probs, ok := n.NodeDefinition["conditionalProbabilities"].(map[string]interface{})
	if !ok {
		return nil
	}

	for _, parentName := range n.ParentNames() {
		parentValue, exists := parentValues[parentName]
		if !exists {
			// Try skip
			if skipProbs, ok := probs["skip"].(map[string]interface{}); ok {
				probs = skipProbs
			}
			continue
		}

		// Convert parent value to string for lookup
		parentValueStr := ""
		switch v := parentValue.(type) {
		case string:
			parentValueStr = v
		default:
			// Try skip if value type doesn't match
			if skipProbs, ok := probs["skip"].(map[string]interface{}); ok {
				probs = skipProbs
			}
			continue
		}

		// Navigate deeper into the probability tree
		if deeper, ok := probs["deeper"].(map[string]interface{}); ok {
			if childProbs, ok := deeper[parentValueStr].(map[string]interface{}); ok {
				probs = childProbs
			} else if skipProbs, ok := probs["skip"].(map[string]interface{}); ok {
				probs = skipProbs
			}
		} else if skipProbs, ok := probs["skip"].(map[string]interface{}); ok {
			probs = skipProbs
		}
	}

	// Convert to map[string]float64
	result := make(map[string]float64)
	for k, v := range probs {
		if k == "deeper" || k == "skip" {
			continue
		}
		switch val := v.(type) {
		case float64:
			result[k] = val
		case int:
			result[k] = float64(val)
		}
	}
	return result
}

// SampleRandomValueFromPossibilities randomly samples from the given values
// using the provided probabilities. Uses weighted random sampling.
func (n *BayesianNode) SampleRandomValueFromPossibilities(possibleValues []string, probabilities map[string]float64) string {
	if len(possibleValues) == 0 {
		return ""
	}

	anchor := rand.Float64()
	cumulativeProbability := 0.0

	for _, value := range possibleValues {
		if prob, ok := probabilities[value]; ok {
			cumulativeProbability += prob
			if cumulativeProbability > anchor {
				return value
			}
		}
	}

	// Default to first item
	return possibleValues[0]
}

// Sample randomly samples from the conditional distribution of this node
// given values of parent nodes.
func (n *BayesianNode) Sample(parentValues map[string]interface{}) string {
	probabilities := n.GetProbabilitiesGivenKnownValues(parentValues)
	if probabilities == nil || len(probabilities) == 0 {
		// Return first possible value as fallback
		possibleValues := n.PossibleValues()
		if len(possibleValues) > 0 {
			return possibleValues[0]
		}
		return ""
	}

	// Get keys from probabilities as possible values
	possibleValues := make([]string, 0, len(probabilities))
	for k := range probabilities {
		possibleValues = append(possibleValues, k)
	}

	return n.SampleRandomValueFromPossibilities(possibleValues, probabilities)
}

// SampleAccordingToRestrictions randomly samples from the conditional distribution
// of this node given restrictions on possible values and parent values.
// Returns empty string if no valid sample can be generated.
func (n *BayesianNode) SampleAccordingToRestrictions(
	parentValues map[string]interface{},
	valuePossibilities []string,
	bannedValues []string,
) string {
	probabilities := n.GetProbabilitiesGivenKnownValues(parentValues)
	if probabilities == nil {
		return ""
	}

	// Create banned set for O(1) lookup
	bannedSet := make(map[string]bool, len(bannedValues))
	for _, v := range bannedValues {
		bannedSet[v] = true
	}

	// Filter valid values
	validValues := make([]string, 0)
	for _, value := range valuePossibilities {
		if !bannedSet[value] {
			if _, hasProbability := probabilities[value]; hasProbability {
				validValues = append(validValues, value)
			}
		}
	}

	if len(validValues) == 0 {
		return ""
	}

	return n.SampleRandomValueFromPossibilities(validValues, probabilities)
}
