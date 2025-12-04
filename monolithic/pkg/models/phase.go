package models

// WorkflowPhase represents a distinct phase in the workflow execution
type WorkflowPhase struct {
	ID         string           `json:"id" yaml:"id"`
	Type       PhaseType        `json:"type" yaml:"type"`
	Name       string           `json:"name,omitempty" yaml:"name,omitempty"`
	Nodes      []Node           `json:"nodes" yaml:"nodes"`
	URLFilter  *URLFilter       `json:"url_filter,omitempty" yaml:"url_filter,omitempty"`
	Transition *PhaseTransition `json:"transition,omitempty" yaml:"transition,omitempty"`
}

// PhaseType defines the type of workflow phase
type PhaseType string

const (
	PhaseTypeDiscovery  PhaseType = "discovery"
	PhaseTypeExtraction PhaseType = "extraction"
	PhaseTypeProcessing PhaseType = "processing"
	PhaseTypeCustom     PhaseType = "custom"
)

// URLFilter defines which URLs should be processed in this phase
type URLFilter struct {
	Markers  []string `json:"markers,omitempty" yaml:"markers,omitempty"`   // URLs with these markers
	Patterns []string `json:"patterns,omitempty" yaml:"patterns,omitempty"` // Regex patterns
	Depth    *int     `json:"depth,omitempty" yaml:"depth,omitempty"`       // Specific depth
}

// PhaseTransition defines when and how to move to the next phase
type PhaseTransition struct {
	Condition string                 `json:"condition" yaml:"condition"`                       // all_nodes_complete, url_count, custom_condition
	NextPhase string                 `json:"next_phase,omitempty" yaml:"next_phase,omitempty"` // ID of next phase (empty = end)
	Params    map[string]interface{} `json:"params,omitempty" yaml:"params,omitempty"`         // Parameters for condition
}

// PhaseExecutionState tracks the current state of phase execution
type PhaseExecutionState struct {
	CurrentPhaseID   string
	CompletedPhases  []string
	URLsInPhase      int
	CompletedURLs    int
	LastTransitionAt *string
}
