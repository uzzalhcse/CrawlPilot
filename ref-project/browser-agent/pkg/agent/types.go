package agent

import (
	"crawer-agent/exp/v2/internal/controller"
	"crawer-agent/exp/v2/internal/dom"
	"crawer-agent/exp/v2/pkg/browser"
	"encoding/json"

	"github.com/cloudwego/eino/schema"
)

type ToolCallingMethod string

const (
	FunctionCalling ToolCallingMethod = "function_calling"
	JSONMode        ToolCallingMethod = "json_mode"
	Raw             ToolCallingMethod = "raw"
	Auto            ToolCallingMethod = "auto"
)

type AgentSettings struct {
	UseVision            bool
	SaveConversationPath *string
	MaxFailures          int
	RetryDelay           int
	MaxInputTokens       int
	MessageContext       *string
	IncludeAttributes    []string
	MaxActionsPerStep    int
	ToolCallingMethod    *ToolCallingMethod
}

type AgentSettingsConfig map[string]interface{}

func NewAgentSettings(config AgentSettingsConfig) *AgentSettings {
	return &AgentSettings{
		UseVision:            getDefaultValue(config, "use_vision", true),
		SaveConversationPath: getOptional[string](config, "save_conversation_path"),
		MaxFailures:          getDefaultValue(config, "max_failures", 3),
		RetryDelay:           getDefaultValue(config, "retry_delay", 10),
		MaxInputTokens:       getDefaultValue(config, "max_input_tokens", 128000),
		MessageContext:       getOptional[string](config, "message_context"),
		IncludeAttributes: getDefaultValue(config, "include_attributes", []string{
			"title", "type", "name", "role", "aria-label", "placeholder", "value", "alt",
		}),
		MaxActionsPerStep: getDefaultValue(config, "max_actions_per_step", 10),
		ToolCallingMethod: getOptional[ToolCallingMethod](config, "tool_calling_method"),
	}
}

type AgentState struct {
	AgentID             string
	NSteps              int
	ConsecutiveFailures int
	LastResult          []*controller.ActionResult
	History             *AgentHistoryList
	Paused              bool
	Stopped             bool
	MessageManagerState *MessageManagerState
}

func NewAgentState() *AgentState {
	return &AgentState{
		AgentID:             generateUUID(),
		NSteps:              1,
		ConsecutiveFailures: 0,
		History:             &AgentHistoryList{History: []*AgentHistory{}},
		MessageManagerState: NewMessageManagerState(),
	}
}

type AgentBrain struct {
	EvaluationPreviousGoal string `json:"evaluation_previous_goal"`
	Memory                 string `json:"memory"`
	NextGoal               string `json:"next_goal"`
}

type AgentOutput struct {
	CurrentState *AgentBrain            `json:"current_state"`
	Actions      []*controller.ActModel `json:"actions"`
}

func (ao *AgentOutput) ToString() string {
	data, err := json.Marshal(ao)
	if err != nil {
		return "{}"
	}
	return string(data)
}

type StepMetadata struct {
	StepStartTime float64
	StepEndTime   float64
	InputTokens   int
	StepNumber    int
}

func (sm *StepMetadata) DurationSeconds() float64 {
	return sm.StepEndTime - sm.StepStartTime
}

type AgentHistory struct {
	ModelOutput *AgentOutput
	Result      []*controller.ActionResult
	State       *browser.BrowserStateHistory
	Metadata    *StepMetadata
}

type AgentHistoryList struct {
	History []*AgentHistory
}

func (ahl *AgentHistoryList) LastResult() *controller.ActionResult {
	if len(ahl.History) == 0 {
		return nil
	}
	lastHistory := ahl.History[len(ahl.History)-1]
	if len(lastHistory.Result) == 0 {
		return nil
	}
	return lastHistory.Result[len(lastHistory.Result)-1]
}

func (ahl *AgentHistoryList) IsDone() bool {
	lastResult := ahl.LastResult()
	if lastResult == nil || lastResult.IsDone == nil {
		return false
	}
	return *lastResult.IsDone
}

func (ahl *AgentHistoryList) IsSuccessful() *bool {
	lastResult := ahl.LastResult()
	if lastResult != nil && lastResult.IsDone != nil && *lastResult.IsDone {
		return lastResult.Success
	}
	return nil
}

func (ahl *AgentHistoryList) TotalInputTokens() int {
	total := 0
	for _, history := range ahl.History {
		if history.Metadata != nil {
			total += history.Metadata.InputTokens
		}
	}
	return total
}

type AgentStepInfo struct {
	StepNumber int
	MaxSteps   int
}

func (asi *AgentStepInfo) IsLastStep() bool {
	return asi.StepNumber >= asi.MaxSteps-1
}

type MessageMetadata struct {
	Tokens      int
	MessageType *string
}

type ManagedMessage struct {
	Message  *schema.Message
	Metadata *MessageMetadata
}

type MessageHistory struct {
	Messages      []ManagedMessage
	CurrentTokens int
}

func (m *MessageHistory) AddMessage(message *schema.Message, metadata *MessageMetadata, position *int) {
	managed := ManagedMessage{Message: message, Metadata: metadata}

	if position == nil {
		m.Messages = append(m.Messages, managed)
	} else {
		idx := *position
		if idx < 0 {
			idx = len(m.Messages) + idx
		}
		m.Messages = append(m.Messages[:idx], append([]ManagedMessage{managed}, m.Messages[idx:]...)...)
	}
	m.CurrentTokens += metadata.Tokens
}

func (m *MessageHistory) RemoveLastStateMessage() {
	if len(m.Messages) > 0 {
		lastMsg := m.Messages[len(m.Messages)-1]
		if lastMsg.Message.Role == schema.User {
			m.CurrentTokens -= lastMsg.Metadata.Tokens
			m.Messages = m.Messages[:len(m.Messages)-1]
		}
	}
}

type MessageManagerState struct {
	History *MessageHistory
	ToolID  int
}

func NewMessageManagerState() *MessageManagerState {
	return &MessageManagerState{
		History: &MessageHistory{
			Messages:      make([]ManagedMessage, 0),
			CurrentTokens: 0,
		},
		ToolID: 1,
	}
}

func getDefaultValue[T any](config map[string]interface{}, key string, defaultValue T) T {
	if value, ok := config[key]; ok {
		if typedValue, ok := value.(T); ok {
			return typedValue
		}
	}
	return defaultValue
}

func getOptional[T any](config map[string]interface{}, key string) *T {
	if value, ok := config[key]; ok {
		if typedValue, ok := value.(T); ok {
			return &typedValue
		}
	}
	return nil
}

func generateUUID() string {
	// Simple UUID generation
	return "agent-" + "uuid"
}

func GetInteractedElement(modelOutput *AgentOutput, selectorMap *dom.SelectorMap) []*dom.DOMElementNode {
	var elements []*dom.DOMElementNode
	for _, action := range modelOutput.Actions {
		index := action.GetIndex()
		if index != nil && selectorMap != nil {
			if el := (*selectorMap)[*index]; el != nil {
				elements = append(elements, el)
			} else {
				elements = append(elements, nil)
			}
		} else {
			elements = append(elements, nil)
		}
	}
	return elements
}
