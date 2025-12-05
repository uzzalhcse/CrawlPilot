package controller

import (
	"context"
	"crawer-agent/exp/v2/pkg/browser"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/playwright-community/playwright-go"
)

type ActionResult struct {
	IsDone           *bool
	Success          *bool
	ExtractedContent *string
	Error            *string
	IncludeInMemory  bool
}

func NewActionResult() *ActionResult {
	return &ActionResult{
		IsDone:           playwright.Bool(false),
		Success:          playwright.Bool(false),
		ExtractedContent: nil,
		Error:            nil,
		IncludeInMemory:  false,
	}
}

type contextKey string

const (
	browserKey            contextKey = "browser"
	pageExtractionLlmKey  contextKey = "page_extraction_llm"
	availableFilePathsKey contextKey = "available_file_paths"
)

// ActionFunction defines the signature for action functions
type ActionFunction func(ctx context.Context, paramsJSON string) (*ActionResult, error)

type Registry struct {
	actions        map[string]ActionFunction
	actionsInfo    map[string]*schema.ToolInfo
	excludeActions []string
}

func NewRegistry() *Registry {
	return &Registry{
		actions:        make(map[string]ActionFunction),
		actionsInfo:    make(map[string]*schema.ToolInfo),
		excludeActions: []string{},
	}
}

func (r *Registry) RegisterAction(name, description string, function ActionFunction, parameters *schema.ToolInfo) error {
	if contains(r.excludeActions, name) {
		return fmt.Errorf("action %s is excluded", name)
	}

	r.actions[name] = function
	if parameters != nil {
		r.actionsInfo[name] = parameters
	}
	return nil
}

func (r *Registry) GetToolInfo() []*schema.ToolInfo {
	var tools []*schema.ToolInfo
	for _, info := range r.actionsInfo {
		tools = append(tools, info)
	}
	return tools
}

func (r *Registry) ExecuteAction(
	actionName string,
	argumentsJSON string,
	browserCtx *browser.BrowserContext,
	pageExtractionLlm model.ToolCallingChatModel,
	sensitiveData map[string]string,
	availableFilePaths []string,
) (string, error) {
	action, ok := r.actions[actionName]
	if !ok {
		// Log available actions for debugging
		var availableActions []string
		for name := range r.actions {
			availableActions = append(availableActions, name)
		}
		log.Errorf("Action '%s' not found. Available actions: %v", actionName, availableActions)
		return "", fmt.Errorf("action '%s' not found. Available actions: %v", actionName, availableActions)
	}

	ctx := context.Background()
	if browserCtx != nil {
		ctx = context.WithValue(ctx, browserKey, browserCtx)
	}
	if pageExtractionLlm != nil {
		ctx = context.WithValue(ctx, pageExtractionLlmKey, pageExtractionLlm)
	}
	if availableFilePaths != nil {
		ctx = context.WithValue(ctx, availableFilePathsKey, availableFilePaths)
	}

	if len(sensitiveData) > 0 {
		argumentsJSON = r.replaceSensitiveData(argumentsJSON, sensitiveData)
	}

	result, err := action(ctx, argumentsJSON)
	if err != nil {
		return "", err
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(resultJSON), nil
}

func (r *Registry) replaceSensitiveData(argumentsJSON string, sensitiveData map[string]string) string {
	secretPattern := regexp.MustCompile(`<secret>(.*?)</secret>`)
	replaceSecrets := func(value string) string {
		if strings.Contains(value, "<secret>") {
			matches := secretPattern.FindAllStringSubmatch(value, -1)
			for _, match := range matches {
				placeholder := match[1]
				if replacement, ok := sensitiveData[placeholder]; ok {
					value = strings.ReplaceAll(value, fmt.Sprintf("<secret>%s</secret>", placeholder), replacement)
				}
			}
		}
		return value
	}
	return replaceSecrets(argumentsJSON)
}

func getBrowserContext(ctx context.Context) (*browser.BrowserContext, error) {
	if bc, ok := ctx.Value(browserKey).(*browser.BrowserContext); ok {
		return bc, nil
	}
	return nil, errors.New("browser context not found")
}

type Controller struct {
	Registry *Registry
}

func NewController() *Controller {
	c := &Controller{
		Registry: NewRegistry(),
	}

	// Register actions with proper tool info
	c.Registry.RegisterAction("done", "Complete task", c.wrapDone, &schema.ToolInfo{
		Name: "done",
		Desc: "Complete the task with result",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"text": {
				Type: schema.String,
				Desc: "The final result or answer",
			},
			"success": {
				Type: schema.Boolean,
				Desc: "Whether the task was successful",
			},
		}),
	})

	c.Registry.RegisterAction("click_element_by_index", "Click element by index", c.wrapClickElementByIndex, &schema.ToolInfo{
		Name: "click_element_by_index",
		Desc: "Click an element by its index number",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"index": {
				Type: schema.Integer,
				Desc: "The index number of the element to click",
			},
		}),
	})

	c.Registry.RegisterAction("input_text", "Input text into element", c.wrapInputText, &schema.ToolInfo{
		Name: "input_text",
		Desc: "Type text into an input field",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"index": {
				Type: schema.Integer,
				Desc: "The index number of the input element",
			},
			"text": {
				Type: schema.String,
				Desc: "The text to input",
			},
		}),
	})

	c.Registry.RegisterAction("search_google", "Search Google", c.wrapSearchGoogle, &schema.ToolInfo{
		Name: "search_google",
		Desc: "Perform a Google search",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type: schema.String,
				Desc: "The search query",
			},
		}),
	})

	c.Registry.RegisterAction("go_to_url", "Navigate to URL", c.wrapGoToURL, &schema.ToolInfo{
		Name: "go_to_url",
		Desc: "Navigate to a specific URL",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"url": {
				Type: schema.String,
				Desc: "The URL to navigate to",
			},
		}),
	})

	c.Registry.RegisterAction("go_back", "Go back", c.wrapGoBack, &schema.ToolInfo{
		Name:        "go_back",
		Desc:        "Go back to the previous page",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	})

	c.Registry.RegisterAction("wait", "Wait for seconds", c.wrapWait, &schema.ToolInfo{
		Name: "wait",
		Desc: "Wait for a specified number of seconds",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"seconds": {
				Type: schema.Integer,
				Desc: "Number of seconds to wait",
			},
		}),
	})

	c.Registry.RegisterAction("scroll_down", "Scroll down", c.wrapScrollDown, &schema.ToolInfo{
		Name: "scroll_down",
		Desc: "Scroll down the page",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"amount": {
				Type: schema.Integer,
				Desc: "Number of pixels to scroll (optional)",
			},
		}),
	})

	c.Registry.RegisterAction("scroll_up", "Scroll up", c.wrapScrollUp, &schema.ToolInfo{
		Name: "scroll_up",
		Desc: "Scroll up the page",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"amount": {
				Type: schema.Integer,
				Desc: "Number of pixels to scroll (optional)",
			},
		}),
	})

	c.Registry.RegisterAction("open_tab", "Open new tab", c.wrapOpenTab, &schema.ToolInfo{
		Name: "open_tab",
		Desc: "Open a new browser tab",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"url": {
				Type: schema.String,
				Desc: "The URL to open in the new tab",
			},
		}),
	})

	c.Registry.RegisterAction("close_tab", "Close tab", c.wrapCloseTab, &schema.ToolInfo{
		Name: "close_tab",
		Desc: "Close a browser tab",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"page_id": {
				Type: schema.Integer,
				Desc: "The ID of the tab to close",
			},
		}),
	})

	c.Registry.RegisterAction("switch_tab", "Switch tab", c.wrapSwitchTab, &schema.ToolInfo{
		Name: "switch_tab",
		Desc: "Switch to a different browser tab",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"page_id": {
				Type: schema.Integer,
				Desc: "The ID of the tab to switch to",
			},
		}),
	})

	return c
}

// Wrapper functions to convert typed params to ActionFunction signature

func (c *Controller) wrapDone(ctx context.Context, paramsJSON string) (*ActionResult, error) {
	var params struct {
		Text    string `json:"text"`
		Success bool   `json:"success"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return nil, err
	}
	return c.Done(ctx, params)
}

func (c *Controller) wrapClickElementByIndex(ctx context.Context, paramsJSON string) (*ActionResult, error) {
	var params struct {
		Index int `json:"index"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return nil, err
	}
	return c.ClickElementByIndex(ctx, params)
}

func (c *Controller) wrapInputText(ctx context.Context, paramsJSON string) (*ActionResult, error) {
	var params struct {
		Index int    `json:"index"`
		Text  string `json:"text"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return nil, err
	}
	return c.InputText(ctx, params)
}

func (c *Controller) wrapSearchGoogle(ctx context.Context, paramsJSON string) (*ActionResult, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return nil, err
	}
	return c.SearchGoogle(ctx, params)
}

func (c *Controller) wrapGoToURL(ctx context.Context, paramsJSON string) (*ActionResult, error) {
	var params struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return nil, err
	}
	return c.GoToURL(ctx, params)
}

func (c *Controller) wrapGoBack(ctx context.Context, paramsJSON string) (*ActionResult, error) {
	var params struct{}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return nil, err
	}
	return c.GoBack(ctx, params)
}

func (c *Controller) wrapWait(ctx context.Context, paramsJSON string) (*ActionResult, error) {
	var params struct {
		Seconds int `json:"seconds"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return nil, err
	}
	return c.Wait(ctx, params)
}

func (c *Controller) wrapScrollDown(ctx context.Context, paramsJSON string) (*ActionResult, error) {
	var params struct {
		Amount *int `json:"amount,omitempty"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return nil, err
	}
	return c.ScrollDown(ctx, params)
}

func (c *Controller) wrapScrollUp(ctx context.Context, paramsJSON string) (*ActionResult, error) {
	var params struct {
		Amount *int `json:"amount,omitempty"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return nil, err
	}
	return c.ScrollUp(ctx, params)
}

func (c *Controller) wrapOpenTab(ctx context.Context, paramsJSON string) (*ActionResult, error) {
	var params struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return nil, err
	}
	return c.OpenTab(ctx, params)
}

func (c *Controller) wrapCloseTab(ctx context.Context, paramsJSON string) (*ActionResult, error) {
	var params struct {
		PageID int `json:"page_id"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return nil, err
	}
	return c.CloseTab(ctx, params)
}

func (c *Controller) wrapSwitchTab(ctx context.Context, paramsJSON string) (*ActionResult, error) {
	var params struct {
		PageID int `json:"page_id"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return nil, err
	}
	return c.SwitchTab(ctx, params)
}

func (c *Controller) Done(_ context.Context, params struct {
	Text    string `json:"text"`
	Success bool   `json:"success"`
}) (*ActionResult, error) {
	log.Debug("Done action called")
	return &ActionResult{
		IsDone:           playwright.Bool(true),
		Success:          &params.Success,
		ExtractedContent: &params.Text,
	}, nil
}

func (c *Controller) ClickElementByIndex(ctx context.Context, params struct {
	Index int `json:"index"`
}) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}

	elementNode, err := bc.GetDOMElementByIndex(params.Index)
	if err != nil {
		return nil, err
	}

	downloadPath, err := bc.ClickElementNode(elementNode)
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Clicked element at index %d", params.Index)
	if downloadPath != nil {
		msg = fmt.Sprintf("Downloaded file to %s", *downloadPath)
	}

	return &ActionResult{
		ExtractedContent: &msg,
		IncludeInMemory:  true,
	}, nil
}

func (c *Controller) InputText(ctx context.Context, params struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
}) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}

	elementNode, err := bc.GetDOMElementByIndex(params.Index)
	if err != nil {
		return nil, err
	}

	if err := bc.InputTextElementNode(elementNode, params.Text); err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Input '%s' into index %d", params.Text, params.Index)
	return &ActionResult{
		ExtractedContent: &msg,
		IncludeInMemory:  true,
	}, nil
}

func (c *Controller) SearchGoogle(ctx context.Context, params struct {
	Query string `json:"query"`
}) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}

	page := bc.GetCurrentPage()
	page.Goto(fmt.Sprintf("https://www.google.com/search?q=%s&udm=14", params.Query))
	page.WaitForLoadState()

	msg := fmt.Sprintf("Searched for '%s' in Google", params.Query)
	return &ActionResult{
		ExtractedContent: &msg,
		IncludeInMemory:  true,
	}, nil
}

func (c *Controller) GoToURL(ctx context.Context, params struct {
	URL string `json:"url"`
}) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := bc.NavigateTo(params.URL); err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Navigated to %s", params.URL)
	return &ActionResult{
		ExtractedContent: &msg,
		IncludeInMemory:  true,
	}, nil
}

func (c *Controller) GoBack(ctx context.Context, params struct{}) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := bc.GoBack(); err != nil {
		return nil, err
	}

	msg := "Navigated back"
	return &ActionResult{
		ExtractedContent: &msg,
		IncludeInMemory:  true,
	}, nil
}

func (c *Controller) Wait(_ context.Context, params struct {
	Seconds int `json:"seconds"`
}) (*ActionResult, error) {
	time.Sleep(time.Duration(params.Seconds) * time.Second)
	msg := fmt.Sprintf("Waited for %d seconds", params.Seconds)
	return &ActionResult{
		ExtractedContent: &msg,
		IncludeInMemory:  true,
	}, nil
}

func (c *Controller) ScrollDown(ctx context.Context, params struct {
	Amount *int `json:"amount,omitempty"`
}) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}

	page := bc.GetCurrentPage()
	amount := "one page"
	if params.Amount != nil {
		page.Evaluate(fmt.Sprintf("window.scrollBy(0, %d);", *params.Amount))
		amount = fmt.Sprintf("%d pixels", *params.Amount)
	} else {
		page.Evaluate("window.scrollBy(0, window.innerHeight);")
	}

	msg := fmt.Sprintf("Scrolled down by %s", amount)
	return &ActionResult{
		ExtractedContent: &msg,
		IncludeInMemory:  true,
	}, nil
}

func (c *Controller) ScrollUp(ctx context.Context, params struct {
	Amount *int `json:"amount,omitempty"`
}) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}

	page := bc.GetCurrentPage()
	amount := "one page"
	if params.Amount != nil {
		page.Evaluate(fmt.Sprintf("window.scrollBy(0, -%d);", *params.Amount))
		amount = fmt.Sprintf("%d pixels", *params.Amount)
	} else {
		page.Evaluate("window.scrollBy(0, -window.innerHeight);")
	}

	msg := fmt.Sprintf("Scrolled up by %s", amount)
	return &ActionResult{
		ExtractedContent: &msg,
		IncludeInMemory:  true,
	}, nil
}

func (c *Controller) OpenTab(ctx context.Context, params struct {
	URL string `json:"url"`
}) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := bc.CreateNewTab(params.URL); err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Opened new tab with %s", params.URL)
	return &ActionResult{
		ExtractedContent: &msg,
		IncludeInMemory:  true,
	}, nil
}

func (c *Controller) CloseTab(ctx context.Context, params struct {
	PageID int `json:"page_id"`
}) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}

	bc.SwitchToTab(params.PageID)
	page := bc.GetCurrentPage()
	page.WaitForLoadState()
	url := page.URL()

	if err := page.Close(); err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Closed tab %d with url %s", params.PageID, url)
	return &ActionResult{
		ExtractedContent: &msg,
		IncludeInMemory:  true,
	}, nil
}

func (c *Controller) SwitchTab(ctx context.Context, params struct {
	PageID int `json:"page_id"`
}) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := bc.SwitchToTab(params.PageID); err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Switched to tab %d", params.PageID)
	return &ActionResult{
		ExtractedContent: &msg,
		IncludeInMemory:  true,
	}, nil
}

type ActModel map[string]interface{}

func (am *ActModel) GetIndex() *int {
	for _, params := range *am {
		paramJSON, ok := params.(map[string]interface{})
		if !ok {
			continue
		}
		if index, ok := paramJSON["index"]; ok {
			if indexFloat, ok := index.(float64); ok {
				indexInt := int(indexFloat)
				return &indexInt
			}
		}
	}
	return nil
}

func (c *Controller) ExecuteAction(
	action *ActModel,
	browserContext *browser.BrowserContext,
	pageExtractionLlm model.ToolCallingChatModel,
	sensitiveData map[string]string,
	availableFilePaths []string,
) (*ActionResult, error) {
	for actionName, actionParams := range *action {
		log.Debugf("Executing action: %s with params: %v", actionName, actionParams)

		paramsJSON, err := json.Marshal(actionParams)
		if err != nil {
			errStr := fmt.Sprintf("Failed to marshal action params: %v", err)
			return &ActionResult{
				Error:           &errStr,
				IncludeInMemory: false,
			}, err
		}

		result, err := c.Registry.ExecuteAction(
			actionName,
			string(paramsJSON),
			browserContext,
			pageExtractionLlm,
			sensitiveData,
			availableFilePaths,
		)
		if err != nil {
			errStr := fmt.Sprintf("Action execution failed: %v", err)
			return &ActionResult{
				Error:           &errStr,
				IncludeInMemory: false,
			}, err
		}

		var actionResult ActionResult
		if err := json.Unmarshal([]byte(result), &actionResult); err != nil {
			errStr := fmt.Sprintf("Failed to unmarshal action result: %v", err)
			return &ActionResult{
				Error:           &errStr,
				IncludeInMemory: false,
			}, err
		}

		return &actionResult, nil
	}

	return NewActionResult(), nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
