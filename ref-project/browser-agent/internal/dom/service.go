package dom

import (
	"crawer-agent/exp/v2/internal/utils"
	"embed"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/playwright-community/playwright-go"
)

//go:embed buildDomTree.js
var domScripts embed.FS

type Service struct {
	Page   playwright.Page
	JsCode string
}

func NewService(page playwright.Page) *Service {
	jsCode, err := domScripts.ReadFile("buildDomTree.js")
	if err != nil {
		panic(err)
	}
	return &Service{
		Page:   page,
		JsCode: string(jsCode),
	}
}

func (s *Service) GetClickableElements(highlightElements bool, focusElement, viewportExpansion int) (*DOMState, error) {
	elementTree, selectorMap, err := s.buildDOMTree(highlightElements, focusElement, viewportExpansion)
	if err != nil {
		return nil, err
	}

	return &DOMState{
		ElementTree: elementTree,
		SelectorMap: selectorMap,
	}, nil
}

func (s *Service) buildDOMTree(highlightElements bool, focusElement, viewportExpansion int) (*DOMElementNode, *SelectorMap, error) {
	if _, err := s.Page.Evaluate("1+1"); err != nil {
		return nil, nil, errors.New("page evaluation failed")
	}

	if s.Page.URL() == "about:blank" {
		return &DOMElementNode{
			TagName:    "body",
			Xpath:      "",
			Attributes: map[string]string{},
			Children:   []DOMBaseNode{},
			IsVisible:  false,
		}, &SelectorMap{}, nil
	}

	debugMode := log.GetLevel() == log.DebugLevel
	args := map[string]interface{}{
		"doHighlightElements": highlightElements,
		"focusHighlightIndex": focusElement,
		"viewportExpansion":   viewportExpansion,
		"debugMode":           debugMode,
	}

	evalPage, err := s.Page.Evaluate(s.JsCode, args)
	if err != nil {
		return nil, nil, err
	}

	evalPageMap, ok := evalPage.(map[string]any)
	if !ok {
		return nil, nil, errors.New("invalid evaluation result")
	}

	if debugMode && evalPageMap["perfMetrics"] != nil {
		metrics, _ := json.MarshalIndent(evalPageMap["perfMetrics"], "", "  ")
		log.Debugf("DOM Tree Building Performance Metrics:\n%s", string(metrics))
	}

	return s.constructDOMTree(evalPageMap)
}

func (s *Service) constructDOMTree(evalPage map[string]any) (*DOMElementNode, *SelectorMap, error) {
	jsNodeMap, ok := evalPage["map"].(map[string]any)
	if !ok {
		return nil, nil, errors.New("invalid node map")
	}

	jsRootID, err := strconv.Atoi(evalPage["rootId"].(string))
	if err != nil {
		return nil, nil, err
	}

	selectorMap := &SelectorMap{}
	nodeMap := make(map[int]DOMBaseNode)
	type recheck struct {
		node        DOMBaseNode
		childrenIDs []int
	}
	var rechecks []recheck

	for id, nodeData := range jsNodeMap {
		node, childrenIDs := s.parseNode(nodeData.(map[string]any))
		if node == nil {
			continue
		}

		idInt, err := strconv.Atoi(id)
		if err != nil {
			return nil, nil, err
		}
		nodeMap[idInt] = node

		if el, ok := node.(*DOMElementNode); ok && el.HighlightIndex != nil {
			(*selectorMap)[*el.HighlightIndex] = el
		}

		rechecks = append(rechecks, recheck{node, childrenIDs})
	}

	for _, r := range rechecks {
		el, ok := r.node.(*DOMElementNode)
		if !ok {
			continue
		}

		for _, childID := range r.childrenIDs {
			if childNode, exists := nodeMap[childID]; exists {
				childNode.SetParent(el)
				el.Children = append(el.Children, childNode)
			}
		}
	}

	rootNode, ok := nodeMap[jsRootID].(*DOMElementNode)
	if !ok {
		return nil, nil, errors.New("invalid root node")
	}

	return rootNode, selectorMap, nil
}

func (s *Service) parseNode(nodeData map[string]any) (DOMBaseNode, []int) {
	if nodeData == nil {
		return nil, []int{}
	}

	if nodeData["type"] == "TEXT_NODE" {
		return &DOMTextNode{
			Text:      nodeData["text"].(string),
			IsVisible: nodeData["isVisible"].(bool),
		}, []int{}
	}

	var viewportInfo *ViewportInfo
	if viewport := nodeData["viewport"]; viewport != nil {
		v := viewport.(map[string]any)
		viewportInfo = &ViewportInfo{
			Width:  utils.ConvertToInt(v["width"]),
			Height: utils.ConvertToInt(v["height"]),
		}
	}

	elementNode := &DOMElementNode{
		TagName:        nodeData["tagName"].(string),
		Xpath:          nodeData["xpath"].(string),
		Attributes:     utils.ConvertToStringMap(nodeData["attributes"].(map[string]any)),
		Children:       []DOMBaseNode{},
		IsVisible:      utils.GetDefaultValue(nodeData, "isVisible", false),
		IsInteractive:  utils.GetDefaultValue(nodeData, "isInteractive", false),
		IsTopElement:   utils.GetDefaultValue(nodeData, "isTopElement", false),
		IsInViewport:   utils.GetDefaultValue(nodeData, "isInViewport", false),
		HighlightIndex: convertToOptionalInt(nodeData["highlightIndex"]),
		ShadowRoot:     utils.GetDefaultValue(nodeData, "shadowRoot", false),
		ViewportInfo:   viewportInfo,
	}

	childrenIDs, err := utils.ConvertToSliceOfInt(nodeData["children"])
	if err != nil {
		return elementNode, []int{}
	}

	return elementNode, childrenIDs
}

// Keep this helper in service.go as it's DOM-specific
func convertToOptionalInt(value interface{}) *int {
	if value == nil {
		return nil
	}
	result := utils.ConvertToInt(value)
	return &result
}

func (s *Service) GetCrossOriginIframes() []string {
	hiddenFrameURLs, _ := s.Page.Locator("iframe").Filter(playwright.LocatorFilterOptions{
		Visible: playwright.Bool(false),
	}).EvaluateAll("e => e.map(e => e.src)")

	adDomains := []string{"doubleclick.net", "adroll.com", "googletagmanager.com"}
	isAdURL := func(url string) bool {
		for _, domain := range adDomains {
			if strings.Contains(url, domain) {
				return true
			}
		}
		return false
	}

	pageURL := s.Page.URL()
	var crossOriginIframes []string

	for _, frame := range s.Page.Frames() {
		frameURL := frame.URL()
		if frameURL == "" || frameURL == pageURL {
			continue
		}

		if _, hidden := hiddenFrameURLs.(map[string]any)[frameURL]; hidden {
			continue
		}

		if isAdURL(frameURL) {
			continue
		}

		crossOriginIframes = append(crossOriginIframes, frameURL)
	}

	return crossOriginIframes
}
