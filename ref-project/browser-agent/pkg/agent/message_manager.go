package agent

import (
	"crawer-agent/exp/v2/internal/controller"
	"crawer-agent/exp/v2/pkg/browser"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino/schema"
)

type MessageManagerSettings struct {
	MaxInputTokens              int
	EstimatedCharactersPerToken int
	ImageTokens                 int
	IncludeAttributes           []string
	MessageContext              *string
	SensitiveData               map[string]string
}

type MessageManagerConfig map[string]interface{}

func NewMessageManagerSettings(config MessageManagerConfig) *MessageManagerSettings {
	return &MessageManagerSettings{
		MaxInputTokens:              getDefaultValue(config, "max_input_tokens", 128000),
		EstimatedCharactersPerToken: getDefaultValue(config, "estimated_characters_per_token", 3),
		ImageTokens:                 getDefaultValue(config, "image_tokens", 800),
		IncludeAttributes:           getDefaultValue(config, "include_attributes", []string{}),
		MessageContext:              getOptional[string](config, "message_context"),
		SensitiveData:               getDefaultValue(config, "sensitive_data", map[string]string{}),
	}
}

type MessageManager struct {
	Task         string
	SystemPrompt *schema.Message
	Settings     *MessageManagerSettings
	State        *MessageManagerState
}

func NewMessageManager(
	task string,
	systemPrompt *schema.Message,
	settings *MessageManagerSettings,
	state *MessageManagerState,
) *MessageManager {
	if settings == nil {
		settings = NewMessageManagerSettings(MessageManagerConfig{})
	}
	if state == nil {
		state = NewMessageManagerState()
	}

	manager := &MessageManager{
		Task:         task,
		SystemPrompt: systemPrompt,
		Settings:     settings,
		State:        state,
	}

	if len(state.History.Messages) == 0 {
		manager.initMessages()
	}

	return manager
}

func (m *MessageManager) initMessages() {
	initStr := "init"
	m.AddMessageWithTokens(m.SystemPrompt, nil, &initStr)

	if m.Settings.MessageContext != nil {
		contextMessage := &schema.Message{
			Role:    schema.User,
			Content: "Context: " + *m.Settings.MessageContext,
		}
		m.AddMessageWithTokens(contextMessage, nil, &initStr)
	}

	taskMessage := &schema.Message{
		Role:    schema.User,
		Content: fmt.Sprintf("Your task is: \"%s\"", m.Task),
	}
	m.AddMessageWithTokens(taskMessage, nil, &initStr)
}

func (m *MessageManager) AddStateMessage(
	state *browser.BrowserState,
	result []*controller.ActionResult,
	stepInfo *AgentStepInfo,
	useVision bool,
) {
	for _, r := range result {
		if r.IncludeInMemory {
			if r.ExtractedContent != nil {
				msg := &schema.Message{
					Role:    schema.User,
					Content: "Action result: " + *r.ExtractedContent,
				}
				m.AddMessageWithTokens(msg, nil, nil)
			}
			if r.Error != nil {
				errStr := strings.TrimSuffix(*r.Error, "\n")
				splitted := strings.Split(errStr, "\n")
				lastLine := splitted[len(splitted)-1]
				msg := &schema.Message{
					Role:    schema.User,
					Content: "Action error: " + lastLine,
				}
				m.AddMessageWithTokens(msg, nil, nil)
			}
			result = nil
		}
	}

	stateMessage := NewAgentMessagePrompt(state, result, m.Settings.IncludeAttributes, stepInfo).
		GetUserMessage(useVision)
	m.AddMessageWithTokens(stateMessage, nil, nil)
}

func (m *MessageManager) AddModelOutput(output *AgentOutput) {
	toolCalls := []schema.ToolCall{
		{
			ID:   strconv.Itoa(m.State.ToolID),
			Type: "tool_call",
			Function: schema.FunctionCall{
				Name:      "AgentOutput",
				Arguments: output.ToString(),
			},
		},
	}

	msg := &schema.Message{
		Role:      schema.Assistant,
		Content:   "",
		ToolCalls: toolCalls,
	}
	m.AddMessageWithTokens(msg, nil, nil)
	m.State.ToolID++

	toolMsg := &schema.Message{
		Role:       schema.Tool,
		Content:    "tool executed",
		ToolCallID: strconv.Itoa(m.State.ToolID - 1),
	}
	m.AddMessageWithTokens(toolMsg, nil, nil)
}

func (m *MessageManager) GetMessages() []*schema.Message {
	totalInputTokens := 0
	messages := make([]*schema.Message, len(m.State.History.Messages))

	for i, mm := range m.State.History.Messages {
		messages[i] = mm.Message
		totalInputTokens += mm.Metadata.Tokens
		log.Debugf("%s - Token count: %d", mm.Message.Role, mm.Metadata.Tokens)
	}

	log.Debugf("Total input tokens: %d", totalInputTokens)
	return messages
}

func (m *MessageManager) AddMessageWithTokens(
	message *schema.Message,
	position *int,
	messageType *string,
) {
	tokenCount := m.countTokens(message)
	metadata := &MessageMetadata{
		Tokens:      tokenCount,
		MessageType: messageType,
	}
	m.State.History.AddMessage(message, metadata, position)
}

func (m *MessageManager) countTokens(message *schema.Message) int {
	tokens := 0
	if len(message.MultiContent) > 0 {
		for _, part := range message.MultiContent {
			if part.Type == schema.ChatMessagePartTypeImageURL {
				tokens += m.Settings.ImageTokens
			} else if part.Type == schema.ChatMessagePartTypeText {
				tokens += m.countTextTokens(part.Text)
			}
		}
	} else {
		tokens += m.countTextTokens(message.Content)
	}
	return tokens
}

func (m *MessageManager) countTextTokens(text string) int {
	return int(math.Round(float64(len(text)) / float64(m.Settings.EstimatedCharactersPerToken)))
}

func (m *MessageManager) RemoveLastStateMessage() {
	m.State.History.RemoveLastStateMessage()
}

type AgentMessagePrompt struct {
	State             *browser.BrowserState
	Result            []*controller.ActionResult
	IncludeAttributes []string
	StepInfo          *AgentStepInfo
}

func NewAgentMessagePrompt(
	state *browser.BrowserState,
	result []*controller.ActionResult,
	includeAttributes []string,
	stepInfo *AgentStepInfo,
) *AgentMessagePrompt {
	return &AgentMessagePrompt{
		State:             state,
		Result:            result,
		IncludeAttributes: includeAttributes,
		StepInfo:          stepInfo,
	}
}

func (amp *AgentMessagePrompt) GetUserMessage(useVision bool) *schema.Message {
	elementText := amp.State.ElementTree.ClickableElementsToString(amp.IncludeAttributes)

	if elementText != "" {
		if amp.State.PixelAbove > 0 {
			elementText = fmt.Sprintf("... %d pixels above ...\n%s", amp.State.PixelAbove, elementText)
		} else {
			elementText = fmt.Sprintf("[Start of page]\n%s", elementText)
		}

		if amp.State.PixelBelow > 0 {
			elementText = fmt.Sprintf("%s\n... %d pixels below ...", elementText, amp.State.PixelBelow)
		} else {
			elementText = fmt.Sprintf("%s\n[End of page]", elementText)
		}
	} else {
		elementText = "empty page"
	}

	var stepInfoDescription string
	if amp.StepInfo != nil {
		current := amp.StepInfo.StepNumber + 1
		max := amp.StepInfo.MaxSteps
		stepInfoDescription = fmt.Sprintf("Current step: %d/%d", current, max)
	}
	timeStr := time.Now().Format("2006-01-02 15:04")
	stepInfoDescription += fmt.Sprintf("\nCurrent date and time: %s", timeStr)

	tabsStr := make([]string, len(amp.State.Tabs))
	for i, tab := range amp.State.Tabs {
		tabsStr[i] = tab.String()
	}

	stateDescription := fmt.Sprintf(`
Current state:
URL: %s
Available tabs: %s
Interactive elements:
%s
%s`,
		amp.State.URL,
		strings.Join(tabsStr, ", "),
		elementText,
		stepInfoDescription,
	)

	if amp.Result != nil {
		for i, result := range amp.Result {
			if result.ExtractedContent != nil {
				stateDescription += fmt.Sprintf("\nAction result %d: %s", i+1, *result.ExtractedContent)
			}
			if result.Error != nil {
				errStr := *result.Error
				splitted := strings.Split(errStr, "\n")
				lastLine := splitted[len(splitted)-1]
				stateDescription += fmt.Sprintf("\nAction error %d: %s", i+1, lastLine)
			}
		}
	}

	if amp.State.Screenshot != nil && useVision {
		return &schema.Message{
			Role: schema.User,
			MultiContent: []schema.ChatMessagePart{
				{
					Type: schema.ChatMessagePartTypeText,
					Text: stateDescription,
				},
				{
					Type: schema.ChatMessagePartTypeImageURL,
					ImageURL: &schema.ChatMessageImageURL{
						URL: "data:image/png;base64," + *amp.State.Screenshot,
					},
				},
			},
		}
	}

	return &schema.Message{
		Role:    schema.User,
		Content: stateDescription,
	}
}

type SystemPrompt struct {
	SystemMessage     *schema.Message
	MaxActionsPerStep int
}

func NewSystemPrompt(maxActionsPerStep int) *SystemPrompt {
	prompt := fmt.Sprintf(`You are an AI agent designed to automate browser tasks.

CRITICAL: You MUST ONLY use tool functions. NEVER write explanations, analysis, or commentary.

Your ONLY allowed responses:
1. Call tool functions with proper arguments
2. Nothing else - no text, no analysis, no explanations

Your workflow:
1. Analyze the current browser state shown in the user message
2. Decide which action(s) to take from the available tools
3. Call the appropriate tool function(s) with the required parameters
4. STOP - do not explain what you did

Available actions (call these as tool functions):
- go_to_url: Navigate to a URL
- click_element_by_index: Click an element by its [index]
- input_text: Type text into an input field
- search_google: Perform a Google search
- scroll_down/scroll_up: Scroll the page
- open_tab/close_tab/switch_tab: Manage browser tabs
- go_back: Navigate back
- wait: Wait for seconds
- done: Complete the task with results

Rules:
1. Use element indexes shown in brackets [index] for interactions
2. You can perform up to %d actions in sequence
3. When task is complete, call the "done" tool with success=true and the extracted information
4. If stuck, try alternative approaches

EXAMPLES OF CORRECT BEHAVIOR:

Example 1 - Extract product data when visible:
USER: Task: Get product name and price from current page
      Page shows: [15] Product: Sony Headphones, [16] Price: $299
CORRECT RESPONSE: 
<tool_call>done(success=true, text="Product: Sony Headphones, Price: $299")</tool_call>

WRONG RESPONSE: "I can see the product is Sony Headphones priced at $299. **Note:** All information is visible."

Example 2 - Need to scroll for missing data:
USER: Task: Get product price
      Page shows: [10] Product: Laptop, [11] Description...
      (Price not visible)
CORRECT RESPONSE: 
<tool_call>scroll_down()</tool_call>

WRONG RESPONSE: "The price is not visible yet. I should scroll down to find it."

Example 3 - Multiple data points with some missing:
USER: Task: Extract name, price, rating
      Page shows: [5] Name: Phone, [6] Rating: 4.5 stars
      (Price not visible)
CORRECT RESPONSE: 
<tool_call>scroll_down()</tool_call>

WRONG RESPONSE: "**Product Name:** Phone, **Rating:** 4.5 stars, **Price:** [Not visible, need to scroll]"

FORBIDDEN BEHAVIORS:
❌ DO NOT write analyses like "The provided text appears to be..."
❌ DO NOT explain the page structure
❌ DO NOT describe technical details
❌ DO NOT provide use cases or next steps
❌ DO NOT generate markdown formatting like **bold** or bullets
❌ DO NOT write anything except tool function calls
❌ DO NOT add notes like "**Note:** The price is not visible"
❌ DO NOT explain what you're about to do

REMEMBER: 
- If data is complete: Call done() immediately
- If data is incomplete: Call scroll_down() or other action
- NEVER explain, NEVER add notes, ONLY call tools`, maxActionsPerStep)

	return &SystemPrompt{
		SystemMessage: &schema.Message{
			Role:    schema.System,
			Content: prompt,
		},
		MaxActionsPerStep: maxActionsPerStep,
	}
}
