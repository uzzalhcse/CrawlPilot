package recovery

import (
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
)

// ProbeBaseline manages baseline comparisons for probe results
type ProbeBaseline struct{}

// NewProbeBaseline creates a new probe baseline comparison service
func NewProbeBaseline() *ProbeBaseline {
	return &ProbeBaseline{}
}

// NodeDeviation describes how a node deviated from baseline
type NodeDeviation struct {
	NodeID           string  `json:"node_id"`
	NodeName         string  `json:"node_name"`
	NodeType         string  `json:"node_type"`
	DeviationType    string  `json:"deviation_type"` // "element_count_drop", "links_missing", "new_failure", "recovered"
	ExpectedValue    int     `json:"expected_value"`
	ActualValue      int     `json:"actual_value"`
	PercentageChange float64 `json:"percentage_change"` // Negative = decrease
	Severity         string  `json:"severity"`          // "minor", "major", "critical"
	Selector         string  `json:"selector,omitempty"`
	Error            string  `json:"error,omitempty"`
}

// DeviationSummary provides an overview of all deviations
type DeviationSummary struct {
	TotalDeviations    int             `json:"total_deviations"`
	CriticalCount      int             `json:"critical_count"`
	MajorCount         int             `json:"major_count"`
	MinorCount         int             `json:"minor_count"`
	NewFailures        int             `json:"new_failures"`         // Nodes that passed before but failed now
	RecoveredNodes     int             `json:"recovered_nodes"`      // Nodes that failed before but passed now
	OverallHealthDelta float64         `json:"overall_health_delta"` // -1.0 to +1.0
	Deviations         []NodeDeviation `json:"deviations"`
}

// Compare compares current probe result with baseline and returns deviations
func (pb *ProbeBaseline) Compare(current, baseline *models.ProbeResult) *DeviationSummary {
	if baseline == nil {
		// No baseline = no deviations to report
		return &DeviationSummary{
			Deviations: []NodeDeviation{},
		}
	}

	summary := &DeviationSummary{
		Deviations: []NodeDeviation{},
	}

	// Build lookup maps for baseline nodes
	baselineNodes := make(map[string]map[string]*models.NodeProbeResult)
	for _, phase := range baseline.Phases {
		if baselineNodes[phase.PhaseID] == nil {
			baselineNodes[phase.PhaseID] = make(map[string]*models.NodeProbeResult)
		}
		for i := range phase.Nodes {
			node := &phase.Nodes[i]
			baselineNodes[phase.PhaseID][node.NodeID] = node
		}
	}

	// Compare each node in current result with baseline
	for _, phase := range current.Phases {
		phaseBaseline, hasPhase := baselineNodes[phase.PhaseID]
		if !hasPhase {
			// New phase, skip comparison
			continue
		}

		for _, node := range phase.Nodes {
			baselineNode, hasNode := phaseBaseline[node.NodeID]
			if !hasNode {
				// New node, skip comparison
				continue
			}

			deviation := pb.compareNodes(&node, baselineNode)
			if deviation != nil {
				summary.Deviations = append(summary.Deviations, *deviation)
				summary.TotalDeviations++

				switch deviation.Severity {
				case "critical":
					summary.CriticalCount++
				case "major":
					summary.MajorCount++
				case "minor":
					summary.MinorCount++
				}

				if deviation.DeviationType == "new_failure" {
					summary.NewFailures++
				} else if deviation.DeviationType == "recovered" {
					summary.RecoveredNodes++
				}
			}
		}
	}

	// Calculate overall health delta
	if len(summary.Deviations) > 0 {
		// Negative = degraded, positive = improved
		summary.OverallHealthDelta = float64(summary.RecoveredNodes-summary.NewFailures) /
			float64(len(summary.Deviations)+1)
	}

	return summary
}

// compareNodes compares a current node with its baseline version
func (pb *ProbeBaseline) compareNodes(current, baseline *models.NodeProbeResult) *NodeDeviation {
	// Check for status change
	if baseline.Status == "passed" && current.Status == "failed" {
		return &NodeDeviation{
			NodeID:        current.NodeID,
			NodeName:      current.NodeName,
			NodeType:      current.NodeType,
			DeviationType: "new_failure",
			Severity:      "critical",
			Selector:      current.Selector,
			Error:         current.Error,
		}
	}

	if baseline.Status == "failed" && current.Status == "passed" {
		return &NodeDeviation{
			NodeID:        current.NodeID,
			NodeName:      current.NodeName,
			NodeType:      current.NodeType,
			DeviationType: "recovered",
			Severity:      "minor", // Good news
		}
	}

	// Both passed - check for element count changes
	if current.Status == "passed" && baseline.ElementCount > 0 {
		if current.ElementCount == 0 {
			return &NodeDeviation{
				NodeID:           current.NodeID,
				NodeName:         current.NodeName,
				NodeType:         current.NodeType,
				DeviationType:    "element_count_drop",
				ExpectedValue:    baseline.ElementCount,
				ActualValue:      current.ElementCount,
				PercentageChange: -100.0,
				Severity:         "critical",
				Selector:         current.Selector,
			}
		}

		percentChange := (float64(current.ElementCount) - float64(baseline.ElementCount)) / float64(baseline.ElementCount) * 100
		if percentChange <= -50 {
			return &NodeDeviation{
				NodeID:           current.NodeID,
				NodeName:         current.NodeName,
				NodeType:         current.NodeType,
				DeviationType:    "element_count_drop",
				ExpectedValue:    baseline.ElementCount,
				ActualValue:      current.ElementCount,
				PercentageChange: percentChange,
				Severity:         "major",
				Selector:         current.Selector,
			}
		}
	}

	// Check for links found changes
	if current.Status == "passed" && baseline.LinksFound > 0 {
		if current.LinksFound == 0 {
			return &NodeDeviation{
				NodeID:           current.NodeID,
				NodeName:         current.NodeName,
				NodeType:         current.NodeType,
				DeviationType:    "links_missing",
				ExpectedValue:    baseline.LinksFound,
				ActualValue:      current.LinksFound,
				PercentageChange: -100.0,
				Severity:         "critical",
				Selector:         current.Selector,
			}
		}

		percentChange := (float64(current.LinksFound) - float64(baseline.LinksFound)) / float64(baseline.LinksFound) * 100
		if percentChange <= -50 {
			return &NodeDeviation{
				NodeID:           current.NodeID,
				NodeName:         current.NodeName,
				NodeType:         current.NodeType,
				DeviationType:    "links_missing",
				ExpectedValue:    baseline.LinksFound,
				ActualValue:      current.LinksFound,
				PercentageChange: percentChange,
				Severity:         "major",
				Selector:         current.Selector,
			}
		}
	}

	return nil
}

// PopulateNodeBaselines fills in ExpectedElementCount and ExpectedLinksFound from baseline
func (pb *ProbeBaseline) PopulateNodeBaselines(current *models.ProbeResult, baseline *models.ProbeResult) {
	if baseline == nil {
		return
	}

	// Build lookup maps for baseline nodes
	baselineNodes := make(map[string]map[string]*models.NodeProbeResult)
	for _, phase := range baseline.Phases {
		if baselineNodes[phase.PhaseID] == nil {
			baselineNodes[phase.PhaseID] = make(map[string]*models.NodeProbeResult)
		}
		for i := range phase.Nodes {
			node := &phase.Nodes[i]
			baselineNodes[phase.PhaseID][node.NodeID] = node
		}
	}

	// Populate baseline values in current result
	for i := range current.Phases {
		phase := &current.Phases[i]
		phaseBaseline, hasPhase := baselineNodes[phase.PhaseID]
		if !hasPhase {
			continue
		}

		for j := range phase.Nodes {
			node := &phase.Nodes[j]
			baselineNode, hasNode := phaseBaseline[node.NodeID]
			if !hasNode {
				continue
			}

			node.ExpectedElementCount = baselineNode.ElementCount
			node.ExpectedLinksFound = baselineNode.LinksFound

			// Calculate deviation
			node.Deviation = pb.calculateDeviation(node, baselineNode)
		}
	}
}

// calculateDeviation returns a deviation string for a node
func (pb *ProbeBaseline) calculateDeviation(current, baseline *models.NodeProbeResult) string {
	if baseline.Status == "passed" && current.Status == "failed" {
		return "missing"
	}
	if baseline.Status == "failed" && current.Status == "passed" {
		return "recovered"
	}

	// Check element count changes
	if baseline.ElementCount > 0 && current.ElementCount < baseline.ElementCount {
		return "decreased"
	}
	if baseline.ElementCount > 0 && current.ElementCount > baseline.ElementCount {
		return "increased"
	}

	// Check links found changes
	if baseline.LinksFound > 0 && current.LinksFound < baseline.LinksFound {
		return "decreased"
	}
	if baseline.LinksFound > 0 && current.LinksFound > baseline.LinksFound {
		return "increased"
	}

	return "none"
}
