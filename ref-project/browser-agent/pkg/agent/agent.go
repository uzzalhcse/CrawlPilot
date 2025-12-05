package agent

import (
	"context"
	"crawer-agent/exp/v2/internal/controller"
	"crawer-agent/exp/v2/internal/dom"
	"crawer-agent/exp/v2/pkg/browser"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type Agent struct {
	Task                   string
	LLM                    model.ToolCallingChatModel
	Controller             *controller.Controller
	SensitiveData          map[string]string
	Settings               *AgentSettings
	State                  *AgentState
	Browser                *browser.Browser
	BrowserContext         *browser.BrowserContext
	MessageManager         *MessageManager
	InjectedBrowser        bool
	InjectedBrowserContext bool
}

type AgentOption func(*AgentOptions)

type AgentOptions struct {
	settings           *AgentSettings
	browserInst        *browser.Browser
	browserContext     *browser.BrowserContext
	controller         *controller.Controller
	sensitiveData      map[string]string
	initialActions     []map[string]interface{}
	injectedAgentState *AgentState
}

func WithAgentSettings(settings AgentSettingsConfig) AgentOption {
	return func(o *AgentOptions) {
		o.settings = NewAgentSettings(settings)
	}
}

func WithBrowser(b *browser.Browser) AgentOption {
	return func(o *AgentOptions) {
		o.browserInst = b
	}
}

func WithBrowserConfig(b browser.BrowserConfig) AgentOption {
	return func(o *AgentOptions) {
		o.browserInst = browser.NewBrowser(b)
	}
}

func WithBrowserContext(b *browser.BrowserContext) AgentOption {
	return func(o *AgentOptions) {
		o.browserContext = b
	}
}

func WithController(c *controller.Controller) AgentOption {
	return func(o *AgentOptions) {
		o.controller = c
	}
}

func WithSensitiveData(data map[string]string) AgentOption {
	return func(o *AgentOptions) {
		o.sensitiveData = data
	}
}

func WithInitialActions(actions []map[string]interface{}) AgentOption {
	return func(o *AgentOptions) {
		o.initialActions = actions
	}
}

func NewAgent(task string, llm model.ToolCallingChatModel, options ...AgentOption) *Agent {
	opts := &AgentOptions{
		settings: NewAgentSettings(AgentSettingsConfig{}),
	}
	for _, opt := range options {
		opt(opts)
	}

	agent := &Agent{
		Task:          task,
		LLM:           llm,
		Controller:    opts.controller,
		SensitiveData: opts.sensitiveData,
		Settings:      opts.settings,
	}

	if agent.Controller == nil {
		agent.Controller = controller.NewController()
	}

	state := opts.injectedAgentState
	if state == nil {
		state = NewAgentState()
	}
	agent.State = state

	systemPrompt := NewSystemPrompt(agent.Settings.MaxActionsPerStep)
	agent.MessageManager = NewMessageManager(
		task,
		systemPrompt.SystemMessage,
		NewMessageManagerSettings(MessageManagerConfig{
			"max_input_tokens":   agent.Settings.MaxInputTokens,
			"include_attributes": agent.Settings.IncludeAttributes,
			"message_context":    agent.Settings.MessageContext,
			"sensitive_data":     agent.SensitiveData,
		}),
		agent.State.MessageManagerState,
	)

	agent.InjectedBrowser = opts.browserInst != nil
	agent.InjectedBrowserContext = opts.browserContext != nil

	if opts.browserInst == nil {
		opts.browserInst = browser.NewBrowser(browser.BrowserConfig{})
	}
	agent.Browser = opts.browserInst

	if opts.browserContext == nil {
		opts.browserContext = opts.browserInst.NewContext()
	}
	agent.BrowserContext = opts.browserContext

	if len(opts.initialActions) > 0 {
		initialActions := make([]*controller.ActModel, len(opts.initialActions))
		for i, action := range opts.initialActions {
			actModel := controller.ActModel(action)
			initialActions[i] = &actModel
		}
		result, _ := agent.multiAct(initialActions, false)
		agent.State.LastResult = result
	}

	return agent
}

func (ag *Agent) Run(opts ...AgentRunOption) (*AgentHistoryList, error) {
	options := agentRunOptions{
		maxSteps:  10,
		autoClose: true,
	}
	for _, opt := range opts {
		opt(&options)
	}

	if options.autoClose {
		defer ag.Close()
	}

	log.Infof("Starting task: %s", ag.Task)

	for step := 0; step < options.maxSteps; step++ {
		if ag.State.ConsecutiveFailures >= ag.Settings.MaxFailures {
			log.Errorf("Stopping due to %d consecutive failures", ag.Settings.MaxFailures)
			break
		}

		if ag.State.Stopped || ag.State.Paused {
			break
		}

		if options.onStepStart != nil {
			options.onStepStart(ag)
		}

		stepInfo := &AgentStepInfo{
			StepNumber: step,
			MaxSteps:   options.maxSteps,
		}

		if err := ag.step(stepInfo); err != nil {
			log.Errorf("Step %d failed: %s", step, err)
			return nil, err
		}

		if options.onStepEnd != nil {
			options.onStepEnd(ag)
		}

		if ag.State.History.IsDone() {
			ag.logCompletion()
			break
		}
	}

	return ag.State.History, nil
}

func (ag *Agent) step(stepInfo *AgentStepInfo) error {
	log.Infof("Step %d", ag.State.NSteps)
	stepStartTime := time.Now().UnixNano()

	browserState := ag.BrowserContext.GetState(true)

	ag.MessageManager.AddStateMessage(browserState, ag.State.LastResult, stepInfo, ag.Settings.UseVision)

	if stepInfo != nil && stepInfo.IsLastStep() {
		msg := "This is your last step. Use only the 'done' action."
		ag.MessageManager.AddMessageWithTokens(&schema.Message{
			Role:    schema.User,
			Content: msg,
		}, nil, nil)
	}

	inputMessages := ag.MessageManager.GetMessages()
	tokens := ag.MessageManager.State.History.CurrentTokens

	modelOutput, err := ag.getNextAction(inputMessages)
	if err != nil {
		ag.MessageManager.RemoveLastStateMessage()
		return errors.New("failed to get next action")
	}

	ag.State.NSteps++

	ag.MessageManager.RemoveLastStateMessage()
	ag.MessageManager.AddModelOutput(modelOutput)

	result, err := ag.multiAct(modelOutput.Actions, true)
	if err != nil {
		errStr := err.Error()
		ag.State.LastResult = []*controller.ActionResult{
			{Error: &errStr, IncludeInMemory: false},
		}
		return err
	}

	ag.State.LastResult = result
	ag.State.ConsecutiveFailures = 0

	if len(result) > 0 {
		lastResult := result[len(result)-1]
		if lastResult.IsDone != nil && *lastResult.IsDone && lastResult.ExtractedContent != nil {
			log.Infof("Result: %s", *lastResult.ExtractedContent)
		}
	}

	if browserState != nil {
		metadata := &StepMetadata{
			StepNumber:    ag.State.NSteps,
			StepStartTime: float64(stepStartTime),
			StepEndTime:   float64(time.Now().UnixNano()),
			InputTokens:   tokens,
		}
		ag.makeHistoryItem(modelOutput, browserState, result, metadata)
	}

	return nil
}

func (ag *Agent) getNextAction(inputMessages []*schema.Message) (*AgentOutput, error) {
	ctx := context.Background()

	// Get available tools from controller
	tools := ag.Controller.Registry.GetToolInfo()

	// Create options with tools
	options := model.WithTools(tools)

	response, err := ag.LLM.Generate(ctx, inputMessages, options)
	if err != nil {
		log.Errorf("LLM generation failed: %v", err)
		return nil, err
	}

	// If we have proper tool calls, use them
	if len(response.ToolCalls) > 0 {
		return ag.parseToolCalls(response.ToolCalls)
	}

	// If no tool calls but has content, try to parse it
	if response.Content != "" {
		return ag.parseContentAsAction(response.Content)
	}

	log.Warnf("No tool calls and no content in response")
	return nil, errors.New("no tool calls returned from LLM")
}

func (ag *Agent) parseToolCalls(toolCalls []schema.ToolCall) (*AgentOutput, error) {
	var actions []*controller.ActModel

	for i, toolCall := range toolCalls {
		log.Debugf("Tool call %d: %s with args: %s", i, toolCall.Function.Name, toolCall.Function.Arguments)

		// Special case: if the tool call is "AgentOutput", extract the nested actions
		if toolCall.Function.Name == "AgentOutput" {
			var nestedOutput AgentOutput
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &nestedOutput); err != nil {
				log.Errorf("Failed to parse nested AgentOutput: %v", err)
				continue
			}

			// Extract actions from the nested output
			if nestedOutput.Actions != nil {
				log.Debugf("Extracted %d actions from nested AgentOutput", len(nestedOutput.Actions))
				actions = append(actions, nestedOutput.Actions...)
			}
			continue
		}

		// Normal tool call processing
		var actionParams map[string]interface{}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &actionParams); err != nil {
			log.Errorf("Failed to parse tool call arguments: %v", err)
			continue
		}

		actModel := controller.ActModel{
			toolCall.Function.Name: actionParams,
		}
		actions = append(actions, &actModel)
	}

	if len(actions) == 0 {
		return nil, errors.New("no valid actions parsed from tool calls")
	}

	return &AgentOutput{
		CurrentState: &AgentBrain{
			EvaluationPreviousGoal: "Processing action",
			Memory:                 "",
			NextGoal:               "Execute action",
		},
		Actions: actions,
	}, nil
}

func (ag *Agent) parseContentAsAction(content string) (*AgentOutput, error) {
	content = strings.TrimSpace(content)

	// Try to parse as JSON first
	if strings.HasPrefix(content, "{") || strings.HasPrefix(content, "```json") {
		// Remove markdown code blocks
		if strings.HasPrefix(content, "```json") {
			content = strings.TrimPrefix(content, "```json")
			content = strings.TrimPrefix(content, "```")
			content = strings.TrimSuffix(content, "```")
			content = strings.TrimSpace(content)
		}

		var output AgentOutput
		if err := json.Unmarshal([]byte(content), &output); err == nil {
			return &output, nil
		}
	}

	// Try to parse tool_call format: tool_name(param='value')
	if strings.Contains(content, "```tool_call") || strings.Contains(content, "(") {
		return ag.parseToolCallFormat(content)
	}

	log.Errorf("Failed to parse content: %s", content)
	return nil, errors.New("no valid actions from LLM")
}

func (ag *Agent) parseToolCallFormat(content string) (*AgentOutput, error) {
	// Remove markdown code blocks
	content = strings.TrimPrefix(content, "```tool_call")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	log.Debugf("Parsing tool call format from content: %s", content)

	// Check for nested function calls like print(default_api.go_to_url(...))
	// or just default_api.go_to_url(...)
	content = unwrapNestedFunctions(content)
	log.Debugf("After unwrapping nested functions: %s", content)

	// Find function name and parameters
	parenIdx := strings.Index(content, "(")
	if parenIdx == -1 {
		log.Errorf("Invalid tool call format: no opening parenthesis in: %s", content)
		return nil, errors.New("invalid tool call format: no opening parenthesis")
	}

	functionName := content[:parenIdx]
	// Remove prefix if present (default_api., etc.)
	if dotIdx := strings.LastIndex(functionName, "."); dotIdx != -1 {
		functionName = functionName[dotIdx+1:]
	}
	functionName = strings.TrimSpace(functionName)

	// Normalize the action name
	normalizedName := normalizeActionName(functionName)
	log.Debugf("Extracted function name: %s -> normalized to: %s", functionName, normalizedName)

	// Extract parameters
	paramsStr := content[parenIdx+1:]
	if closeIdx := strings.LastIndex(paramsStr, ")"); closeIdx != -1 {
		paramsStr = paramsStr[:closeIdx]
	}

	log.Debugf("Extracted parameters string: %s", paramsStr)

	// Parse simple key='value' or key="value" patterns
	params := make(map[string]interface{})
	if paramsStr != "" {
		parts := splitParams(paramsStr)
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if eqIdx := strings.Index(part, "="); eqIdx != -1 {
				key := strings.TrimSpace(part[:eqIdx])
				value := strings.TrimSpace(part[eqIdx+1:])

				// Remove quotes
				value = strings.Trim(value, "'\"")
				params[key] = value
			}
		}
	}

	log.Debugf("Parsed parameters: %v", params)

	actModel := controller.ActModel{
		normalizedName: params,
	}

	return &AgentOutput{
		CurrentState: &AgentBrain{
			EvaluationPreviousGoal: "Processing action",
			Memory:                 "",
			NextGoal:               "Execute action",
		},
		Actions: []*controller.ActModel{&actModel},
	}, nil
}

// unwrapNestedFunctions extracts the innermost/actual function call from wrappers
// e.g., "print(default_api.go_to_url(url='...'))" -> "default_api.go_to_url(url='...')"
func unwrapNestedFunctions(content string) string {
	content = strings.TrimSpace(content)

	// Common wrapper functions to ignore
	wrappers := []string{"print", "execute", "run", "call"}

	for _, wrapper := range wrappers {
		// Check if content starts with wrapper(
		if strings.HasPrefix(content, wrapper+"(") {
			// Find matching closing parenthesis
			if closeIdx := findMatchingParen(content, len(wrapper)); closeIdx != -1 {
				// Extract the content between wrapper( and )
				inner := content[len(wrapper)+1 : closeIdx]
				inner = strings.TrimSpace(inner)

				// Recursively unwrap in case of multiple layers
				return unwrapNestedFunctions(inner)
			}
		}
	}

	return content
}

// findMatchingParen finds the matching closing parenthesis for the opening one at startIdx
func findMatchingParen(s string, startIdx int) int {
	if startIdx >= len(s) || s[startIdx] != '(' {
		return -1
	}

	depth := 0
	for i := startIdx; i < len(s); i++ {
		if s[i] == '(' {
			depth++
		} else if s[i] == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}

// Helper function to split parameters respecting quotes
func splitParams(s string) []string {
	var result []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, char := range s {
		if char == '\'' || char == '"' {
			if !inQuote {
				inQuote = true
				quoteChar = char
			} else if char == quoteChar {
				inQuote = false
				quoteChar = 0
			}
			current.WriteRune(char)
		} else if char == ',' && !inQuote {
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

func (ag *Agent) multiAct(actions []*controller.ActModel, checkForNewElements bool) ([]*controller.ActionResult, error) {
	var results []*controller.ActionResult

	ag.BrowserContext.RemoveHighlights()

	for i, action := range actions {
		result, err := ag.Controller.ExecuteAction(
			action,
			ag.BrowserContext,
			nil,
			ag.SensitiveData,
			nil,
		)
		if err != nil {
			return nil, err
		}

		results = append(results, result)
		log.Debugf("Executed action %d / %d", i+1, len(actions))

		if result.IsDone != nil && *result.IsDone || result.Error != nil || i == len(actions)-1 {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	return results, nil
}

func (ag *Agent) makeHistoryItem(
	modelOutput *AgentOutput,
	browserState *browser.BrowserState,
	result []*controller.ActionResult,
	metadata *StepMetadata,
) {
	interactedElements := GetInteractedElement(modelOutput, browserState.SelectorMap)

	var domElements []*dom.DOMElementNode
	for _, el := range interactedElements {
		domElements = append(domElements, el)
	}

	stateHistory := &browser.BrowserStateHistory{
		URL:   browserState.URL,
		Title: browserState.Title,
		Tabs:  browserState.Tabs,
	}

	historyItem := &AgentHistory{
		ModelOutput: modelOutput,
		Result:      result,
		State:       stateHistory,
		Metadata:    metadata,
	}

	ag.State.History.History = append(ag.State.History.History, historyItem)
}

func (ag *Agent) logCompletion() {
	log.Info("Task completed")
	if success := ag.State.History.IsSuccessful(); success != nil && *success {
		log.Info("Successfully")
	} else {
		log.Info("Unfinished")
	}

	totalTokens := ag.State.History.TotalInputTokens()
	log.Infof("Total input tokens used: %d", totalTokens)
}

func (ag *Agent) Close() {
	if ag.BrowserContext != nil && !ag.InjectedBrowserContext {
		ag.BrowserContext.Close()
	}
	if ag.Browser != nil && !ag.InjectedBrowser {
		ag.Browser.Close()
	}
}

type AgentRunOption func(*agentRunOptions)

type agentRunOptions struct {
	maxSteps    int
	onStepStart func(*Agent)
	onStepEnd   func(*Agent)
	autoClose   bool
}

func WithMaxSteps(n int) AgentRunOption {
	return func(o *agentRunOptions) {
		o.maxSteps = n
	}
}

func WithOnStepStart(cb func(*Agent)) AgentRunOption {
	return func(o *agentRunOptions) {
		o.onStepStart = cb
	}
}

func WithOnStepEnd(cb func(*Agent)) AgentRunOption {
	return func(o *agentRunOptions) {
		o.onStepEnd = cb
	}
}

func WithAutoClose(autoClose bool) AgentRunOption {
	return func(o *agentRunOptions) {
		o.autoClose = autoClose
	}
}

// normalizeActionName converts various action name formats to the canonical format
func normalizeActionName(name string) string {
	// Remove common prefixes
	name = strings.TrimPrefix(name, "default_api.")
	name = strings.TrimPrefix(name, "agent.")
	name = strings.TrimPrefix(name, "browser.")

	// Convert camelCase to snake_case if needed
	// e.g., goToUrl -> go_to_url
	var result strings.Builder
	for i, r := range name {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}

	normalized := strings.ToLower(result.String())

	// Map common variations
	actionMap := map[string]string{
		"goto_url":     "go_to_url",
		"gotourl":      "go_to_url",
		"navigate":     "go_to_url",
		"click":        "click_element_by_index",
		"clickelement": "click_element_by_index",
		"type":         "input_text",
		"typetext":     "input_text",
		"input":        "input_text",
		"search":       "search_google",
		"googlesearch": "search_google",
		"scrolldown":   "scroll_down",
		"scrollup":     "scroll_up",
		"goback":       "go_back",
		"back":         "go_back",
		"opentab":      "open_tab",
		"newtab":       "open_tab",
		"closetab":     "close_tab",
		"switchtab":    "switch_tab",
		"finish":       "done",
		"complete":     "done",
	}

	if mapped, ok := actionMap[normalized]; ok {
		return mapped
	}

	return normalized
}
