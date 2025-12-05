package dom

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
)

type DOMBaseNode interface {
	ToJSON() map[string]any
	SetParent(parent *DOMElementNode)
}

type DOMTextNode struct {
	Text      string
	Type      string
	Parent    *DOMElementNode
	IsVisible bool
}

func (n *DOMTextNode) SetParent(parent *DOMElementNode) {
	n.Parent = parent
}

func (n *DOMTextNode) ToJSON() map[string]any {
	return map[string]any{
		"text": n.Text,
		"type": n.Type,
	}
}

type Coordinates struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type CoordinateSet struct {
	TopLeft     Coordinates `json:"topLeft"`
	TopRight    Coordinates `json:"topRight"`
	BottomLeft  Coordinates `json:"bottomLeft"`
	BottomRight Coordinates `json:"bottomRight"`
	Center      Coordinates `json:"center"`
	Width       int         `json:"width"`
	Height      int         `json:"height"`
}

type ViewportInfo struct {
	ScrollX int `json:"scrollX"`
	ScrollY int `json:"scrollY"`
	Width   int `json:"width"`
	Height  int `json:"height"`
}

type DOMElementNode struct {
	TagName             string
	Xpath               string
	Attributes          map[string]string
	Children            []DOMBaseNode
	IsInteractive       bool
	IsTopElement        bool
	IsInViewport        bool
	ShadowRoot          bool
	HighlightIndex      *int
	ViewportCoordinates *CoordinateSet
	PageCoordinates     *CoordinateSet
	ViewportInfo        *ViewportInfo
	Parent              *DOMElementNode
	IsVisible           bool
	IsNew               *bool
}

func (n *DOMElementNode) SetParent(parent *DOMElementNode) {
	n.Parent = parent
}

func (n *DOMElementNode) ToJSON() map[string]any {
	var children []map[string]any
	if n.Children != nil {
		for _, child := range n.Children {
			children = append(children, child.ToJSON())
		}
	}
	return map[string]any{
		"tagName":       n.TagName,
		"xpath":         n.Xpath,
		"attributes":    n.Attributes,
		"isVisible":     n.IsVisible,
		"isInteractive": n.IsInteractive,
		"children":      children,
	}
}

func (n *DOMElementNode) GetAllTextTillNextClickableElement(maxDepth int) string {
	var textParts []string
	var collectText func(node DOMBaseNode, currentDepth int)
	collectText = func(node DOMBaseNode, currentDepth int) {
		if maxDepth != -1 && currentDepth > maxDepth {
			return
		}
		if el, ok := node.(*DOMElementNode); ok && el != n && el.HighlightIndex != nil {
			return
		}
		switch t := node.(type) {
		case *DOMTextNode:
			textParts = append(textParts, t.Text)
		case *DOMElementNode:
			for _, child := range t.Children {
				collectText(child, currentDepth+1)
			}
		}
	}
	collectText(n, 0)
	return strings.TrimSpace(strings.Join(textParts, "\n"))
}

func (n *DOMElementNode) ClickableElementsToString(includeAttributes []string) string {
	var formattedText []string
	var processNode func(node DOMBaseNode, depth int)
	processNode = func(node DOMBaseNode, depth int) {
		nextDepth := depth
		depthStr := strings.Repeat("\t", depth)

		el, ok := node.(*DOMElementNode)
		if !ok {
			return
		}

		if el.HighlightIndex != nil {
			nextDepth++
			text := el.GetAllTextTillNextClickableElement(-1)
			var attributesHTML string

			if len(includeAttributes) > 0 {
				attributesToInclude := make(map[string]string)
				for key, value := range el.Attributes {
					if slices.Contains(includeAttributes, key) {
						attributesToInclude[key] = value
					}
				}

				if el.TagName == attributesToInclude["role"] {
					delete(attributesToInclude, "role")
				}
				if ariaLabel := attributesToInclude["aria-label"]; strings.TrimSpace(ariaLabel) == strings.TrimSpace(text) {
					delete(attributesToInclude, "aria-label")
				}

				if len(attributesToInclude) > 0 {
					var attributeStrs []string
					for k, v := range attributesToInclude {
						attributeStrs = append(attributeStrs, fmt.Sprintf("%s='%s'", k, v))
					}
					attributesHTML = strings.Join(attributeStrs, " ")
				}
			}

			highlightIndicator := fmt.Sprintf("[%d]", *el.HighlightIndex)
			if el.IsNew != nil && *el.IsNew {
				highlightIndicator = fmt.Sprintf("*[%d]", *el.HighlightIndex)
			}

			line := fmt.Sprintf("%s%s<%s", depthStr, highlightIndicator, el.TagName)
			if len(attributesHTML) > 0 {
				line += " " + attributesHTML
			}
			if len(text) > 0 {
				if attributesHTML == "" {
					line += " "
				}
				line += fmt.Sprintf(">%s", text)
			} else if attributesHTML == "" {
				line += " "
			}
			line += " />"
			formattedText = append(formattedText, line)
		}

		for _, child := range el.Children {
			processNode(child, nextDepth)
		}
	}

	processNode(n, 0)
	return strings.Join(formattedText, "\n")
}

type HashedDOMElement struct {
	BranchPathHash string
	AttributesHash string
	XpathHash      string
}

func (n *DOMElementNode) Hash() HashedDOMElement {
	return hashDOMElement(n)
}

func hashDOMElement(element *DOMElementNode) HashedDOMElement {
	parentBranchPath := getParentBranchPath(element)
	return HashedDOMElement{
		BranchPathHash: hashString(strings.Join(parentBranchPath, "/")),
		AttributesHash: hashAttributes(element.Attributes),
		XpathHash:      hashString(element.Xpath),
	}
}

func getParentBranchPath(element *DOMElementNode) []string {
	var parents []string
	current := element
	for current.Parent != nil {
		parents = append(parents, current.Parent.TagName)
		current = current.Parent
	}
	slices.Reverse(parents)
	return parents
}

func hashAttributes(attributes map[string]string) string {
	var attributesStr string
	for key, value := range attributes {
		attributesStr += fmt.Sprintf("%s=%s", key, value)
	}
	return hashString(attributesStr)
}

func hashString(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}

type SelectorMap map[int]*DOMElementNode

type DOMState struct {
	ElementTree *DOMElementNode
	SelectorMap *SelectorMap
}
