// Package common provides shared utilities for CAPTCHA solving.
package common

import (
	"github.com/playwright-community/playwright-go"
)

// PageOrFrame can be either a playwright.Page or playwright.Frame
type PageOrFrame interface {
	QuerySelector(selector string) (playwright.ElementHandle, error)
	QuerySelectorAll(selector string) ([]playwright.ElementHandle, error)
	EvaluateHandle(expression string, arg ...interface{}) (playwright.JSHandle, error)
	Evaluate(expression string, arg ...interface{}) (interface{}, error)
}

// PageWrapper wraps a playwright.Page to implement PageOrFrame
type PageWrapper struct {
	Page playwright.Page
}

func (p *PageWrapper) QuerySelector(selector string) (playwright.ElementHandle, error) {
	return p.Page.QuerySelector(selector)
}

func (p *PageWrapper) QuerySelectorAll(selector string) ([]playwright.ElementHandle, error) {
	return p.Page.QuerySelectorAll(selector)
}

func (p *PageWrapper) EvaluateHandle(expression string, arg ...interface{}) (playwright.JSHandle, error) {
	return p.Page.EvaluateHandle(expression, arg...)
}

func (p *PageWrapper) Evaluate(expression string, arg ...interface{}) (interface{}, error) {
	return p.Page.Evaluate(expression, arg...)
}

// FrameWrapper wraps a playwright.Frame to implement PageOrFrame
type FrameWrapper struct {
	Frame playwright.Frame
}

func (f *FrameWrapper) QuerySelector(selector string) (playwright.ElementHandle, error) {
	return f.Frame.QuerySelector(selector)
}

func (f *FrameWrapper) QuerySelectorAll(selector string) ([]playwright.ElementHandle, error) {
	return f.Frame.QuerySelectorAll(selector)
}

func (f *FrameWrapper) EvaluateHandle(expression string, arg ...interface{}) (playwright.JSHandle, error) {
	return f.Frame.EvaluateHandle(expression, arg...)
}

func (f *FrameWrapper) Evaluate(expression string, arg ...interface{}) (interface{}, error) {
	return f.Frame.Evaluate(expression, arg...)
}

// ElementWrapper wraps a playwright.ElementHandle to implement PageOrFrame
type ElementWrapper struct {
	Element playwright.ElementHandle
}

func (e *ElementWrapper) QuerySelector(selector string) (playwright.ElementHandle, error) {
	return e.Element.QuerySelector(selector)
}

func (e *ElementWrapper) QuerySelectorAll(selector string) ([]playwright.ElementHandle, error) {
	return e.Element.QuerySelectorAll(selector)
}

func (e *ElementWrapper) EvaluateHandle(expression string, arg ...interface{}) (playwright.JSHandle, error) {
	return e.Element.EvaluateHandle(expression, arg...)
}

func (e *ElementWrapper) Evaluate(expression string, arg ...interface{}) (interface{}, error) {
	return e.Element.Evaluate(expression, arg...)
}

// WrapPage wraps a playwright.Page
func WrapPage(page playwright.Page) PageOrFrame {
	return &PageWrapper{Page: page}
}

// WrapFrame wraps a playwright.Frame
func WrapFrame(frame playwright.Frame) PageOrFrame {
	return &FrameWrapper{Frame: frame}
}

// WrapElement wraps a playwright.ElementHandle
func WrapElement(element playwright.ElementHandle) PageOrFrame {
	return &ElementWrapper{Element: element}
}

// DetectExpectedContent checks if the expected content is present on the page.
//
// Parameters:
//   - queryable: PageOrFrame to search in
//   - selector: CSS selector for the expected content (empty returns false)
//
// Returns:
//   - bool: true if element is found, false otherwise
//   - error: any error that occurred during detection
func DetectExpectedContent(queryable PageOrFrame, selector string) (bool, error) {
	if selector == "" {
		return false, nil
	}

	element, err := queryable.QuerySelector(selector)
	if err != nil {
		return false, err
	}

	return element != nil, nil
}
