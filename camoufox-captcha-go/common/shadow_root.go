// Package common provides shared utilities for CAPTCHA solving.
package common

import (
	"fmt"
	"strings"

	"github.com/playwright-community/playwright-go"
)

// FindAllIframes finds all iframes matching the src filter.
// This searches both regular DOM and shadow DOM using Camoufox's shadowRootUnl.
func FindAllIframes(queryable PageOrFrame, srcFilter string) ([]playwright.Frame, error) {
	// Check if this is an ElementWrapper - needs special handling
	elemWrapper, isElement := queryable.(*ElementWrapper)
	if isElement {
		return findIframesInElement(elemWrapper.Element, srcFilter)
	}

	// For Page/Frame, we can use the full search
	seenURLs := make(map[string]bool)
	return findIframesInPageOrFrame(queryable, srcFilter, seenURLs)
}

// findIframesInElement finds iframes within an element (including its shadow DOM)
func findIframesInElement(element playwright.ElementHandle, srcFilter string) ([]playwright.Frame, error) {
	var matchedIframes []playwright.Frame
	seenURLs := make(map[string]bool)

	// First, find regular iframes within this element
	iframes, err := element.QuerySelectorAll("iframe")
	if err == nil {
		for _, iframe := range iframes {
			if frame := getMatchingFrame(iframe, srcFilter, seenURLs); frame != nil {
				matchedIframes = append(matchedIframes, frame)
			}
		}
	}

	// Get the owner frame to use for EvaluateHandle (element.EvaluateHandle panics in playwright-go)
	ownerFrame, err := element.OwnerFrame()
	if err != nil || ownerFrame == nil {
		return matchedIframes, nil
	}

	// Mark the element with a data attribute so we can reference it in JS
	_, err = element.Evaluate(`el => el.setAttribute('data-camoufox-captcha-target', 'true')`)
	if err != nil {
		return matchedIframes, nil
	}
	defer element.Evaluate(`el => el.removeAttribute('data-camoufox-captcha-target')`)

	escapedFilter := strings.ReplaceAll(srcFilter, "'", "\\'")

	// Count iframes in this element's shadow DOM using the owner frame
	countJS := fmt.Sprintf(`
		() => {
			const el = document.querySelector('[data-camoufox-captcha-target="true"]');
			if (!el) return 0;
			
			let count = 0;
			const srcFilter = '%s';
			
			function searchShadowDom(node) {
				if (!node) return;
				const shadowRoot = node.shadowRootUnl || node.shadowRoot;
				if (shadowRoot) {
					shadowRoot.querySelectorAll('iframe').forEach(iframe => {
						if (srcFilter === '' || (iframe.src && iframe.src.includes(srcFilter))) {
							count++;
						}
					});
					shadowRoot.querySelectorAll('*').forEach(child => searchShadowDom(child));
				}
			}
			
			searchShadowDom(el);
			el.querySelectorAll('*').forEach(child => searchShadowDom(child));
			return count;
		}
	`, escapedFilter)

	countResult, err := ownerFrame.Evaluate(countJS)
	if err != nil {
		return matchedIframes, nil
	}

	iframeCount := 0
	switch v := countResult.(type) {
	case int:
		iframeCount = v
	case float64:
		iframeCount = int(v)
	}

	// Get each iframe element directly using the owner frame
	for i := 0; i < iframeCount; i++ {
		getIframeJS := fmt.Sprintf(`
			() => {
				const el = document.querySelector('[data-camoufox-captcha-target="true"]');
				if (!el) return null;
				
				let index = 0;
				const targetIndex = %d;
				const srcFilter = '%s';
				
				function searchShadowDom(node) {
					if (!node) return null;
					const shadowRoot = node.shadowRootUnl || node.shadowRoot;
					if (shadowRoot) {
						const iframes = shadowRoot.querySelectorAll('iframe');
						for (const iframe of iframes) {
							if (srcFilter === '' || (iframe.src && iframe.src.includes(srcFilter))) {
								if (index === targetIndex) return iframe;
								index++;
							}
						}
						for (const child of shadowRoot.querySelectorAll('*')) {
							const result = searchShadowDom(child);
							if (result) return result;
						}
					}
					return null;
				}
				
				let result = searchShadowDom(el);
				if (result) return result;
				
				for (const child of el.querySelectorAll('*')) {
					result = searchShadowDom(child);
					if (result) return result;
				}
				return null;
			}
		`, i, escapedFilter)

		iframeHandle, err := ownerFrame.EvaluateHandle(getIframeJS)
		if err != nil {
			continue
		}

		iframeElement := iframeHandle.AsElement()
		if iframeElement == nil {
			continue
		}

		frame, err := iframeElement.ContentFrame()
		if err != nil || frame == nil || frame.IsDetached() {
			continue
		}

		frameURL := frame.URL()
		if !seenURLs[frameURL] {
			matchedIframes = append(matchedIframes, frame)
			seenURLs[frameURL] = true
		}
	}

	return matchedIframes, nil
}

// findIframesInPageOrFrame finds iframes in a Page or Frame context
func findIframesInPageOrFrame(queryable PageOrFrame, srcFilter string, seenURLs map[string]bool) ([]playwright.Frame, error) {
	var matchedIframes []playwright.Frame

	// First, find all regular iframes in the DOM
	iframes, err := queryable.QuerySelectorAll("iframe")
	if err == nil {
		for _, iframe := range iframes {
			if frame := getMatchingFrame(iframe, srcFilter, seenURLs); frame != nil {
				matchedIframes = append(matchedIframes, frame)
			}
		}
	}

	escapedFilter := strings.ReplaceAll(srcFilter, "'", "\\'")

	// Count iframes in shadow DOM
	countJS := fmt.Sprintf(`
		() => {
			let count = 0;
			const srcFilter = '%s';
			
			function searchShadowDom(node) {
				if (!node) return;
				const shadowRoot = node.shadowRootUnl || node.shadowRoot;
				if (shadowRoot) {
					shadowRoot.querySelectorAll('iframe').forEach(iframe => {
						if (srcFilter === '' || (iframe.src && iframe.src.includes(srcFilter))) {
							count++;
						}
					});
					shadowRoot.querySelectorAll('*').forEach(el => searchShadowDom(el));
				}
			}
			
			document.querySelectorAll('*').forEach(el => searchShadowDom(el));
			return count;
		}
	`, escapedFilter)

	countResult, err := queryable.Evaluate(countJS)
	if err != nil {
		return matchedIframes, nil
	}

	iframeCount := 0
	switch v := countResult.(type) {
	case int:
		iframeCount = v
	case float64:
		iframeCount = int(v)
	}

	// Get each iframe element directly
	for i := 0; i < iframeCount; i++ {
		getIframeJS := fmt.Sprintf(`
			() => {
				let index = 0;
				const targetIndex = %d;
				const srcFilter = '%s';
				
				function searchShadowDom(node) {
					if (!node) return null;
					const shadowRoot = node.shadowRootUnl || node.shadowRoot;
					if (shadowRoot) {
						const iframes = shadowRoot.querySelectorAll('iframe');
						for (const iframe of iframes) {
							if (srcFilter === '' || (iframe.src && iframe.src.includes(srcFilter))) {
								if (index === targetIndex) return iframe;
								index++;
							}
						}
						for (const child of shadowRoot.querySelectorAll('*')) {
							const result = searchShadowDom(child);
							if (result) return result;
						}
					}
					return null;
				}
				
				for (const el of document.querySelectorAll('*')) {
					const result = searchShadowDom(el);
					if (result) return result;
				}
				return null;
			}
		`, i, escapedFilter)

		iframeHandle, err := queryable.EvaluateHandle(getIframeJS)
		if err != nil {
			continue
		}

		iframeElement := iframeHandle.AsElement()
		if iframeElement == nil {
			continue
		}

		frame, err := iframeElement.ContentFrame()
		if err != nil || frame == nil || frame.IsDetached() {
			continue
		}

		frameURL := frame.URL()
		if !seenURLs[frameURL] {
			matchedIframes = append(matchedIframes, frame)
			seenURLs[frameURL] = true
		}
	}

	return matchedIframes, nil
}

// getMatchingFrame checks if an iframe element matches the filter and returns its frame
func getMatchingFrame(iframeElement playwright.ElementHandle, srcFilter string, seenURLs map[string]bool) playwright.Frame {
	if iframeElement == nil {
		return nil
	}

	srcProp, err := iframeElement.GetProperty("src")
	if err != nil {
		return nil
	}

	srcVal, err := srcProp.JSONValue()
	if err != nil {
		return nil
	}

	src, ok := srcVal.(string)
	if !ok {
		return nil
	}

	if srcFilter != "" && !strings.Contains(src, srcFilter) {
		return nil
	}

	frame, err := iframeElement.ContentFrame()
	if err != nil || frame == nil || frame.IsDetached() {
		return nil
	}

	frameURL := frame.URL()
	if seenURLs[frameURL] {
		return nil
	}
	seenURLs[frameURL] = true

	return frame
}

// FindAllIframesMultiple is an alias for FindAllIframes for backwards compatibility.
func FindAllIframesMultiple(queryable PageOrFrame, srcFilter string) ([]playwright.Frame, error) {
	return FindAllIframes(queryable, srcFilter)
}

// SearchShadowRootElements searches for elements by selector within shadow DOM.
func SearchShadowRootElements(queryable PageOrFrame, selector string) ([]playwright.ElementHandle, error) {
	var elements []playwright.ElementHandle

	escapedSelector := strings.ReplaceAll(selector, "'", "\\'")
	escapedSelector = strings.ReplaceAll(escapedSelector, "\"", "\\\"")

	regularElements, err := queryable.QuerySelectorAll(selector)
	if err == nil {
		elements = append(elements, regularElements...)
	}

	countJS := fmt.Sprintf(`
		() => {
			let count = 0;
			const selector = '%s';
			
			function searchShadowDom(node) {
				if (!node) return;
				const shadowRoot = node.shadowRootUnl || node.shadowRoot;
				if (shadowRoot) {
					count += shadowRoot.querySelectorAll(selector).length;
					shadowRoot.querySelectorAll('*').forEach(el => searchShadowDom(el));
				}
			}
			
			document.querySelectorAll('*').forEach(el => searchShadowDom(el));
			return count;
		}
	`, escapedSelector)

	countResult, err := queryable.Evaluate(countJS)
	if err != nil {
		return elements, nil
	}

	shadowCount := 0
	switch v := countResult.(type) {
	case int:
		shadowCount = v
	case float64:
		shadowCount = int(v)
	}

	for i := 0; i < shadowCount; i++ {
		getElementJS := fmt.Sprintf(`
			() => {
				let index = 0;
				const targetIndex = %d;
				const selector = '%s';
				
				function searchShadowDom(node) {
					if (!node) return null;
					const shadowRoot = node.shadowRootUnl || node.shadowRoot;
					if (shadowRoot) {
						const found = shadowRoot.querySelectorAll(selector);
						for (const el of found) {
							if (index === targetIndex) return el;
							index++;
						}
						for (const child of shadowRoot.querySelectorAll('*')) {
							const result = searchShadowDom(child);
							if (result) return result;
						}
					}
					return null;
				}
				
				for (const el of document.querySelectorAll('*')) {
					const result = searchShadowDom(el);
					if (result) return result;
				}
				return null;
			}
		`, i, escapedSelector)

		elemHandle, err := queryable.EvaluateHandle(getElementJS)
		if err != nil {
			continue
		}

		elem := elemHandle.AsElement()
		if elem != nil {
			elements = append(elements, elem)
		}
	}

	return elements, nil
}

// SearchShadowRootIframes searches for iframes within shadow DOM whose src contains the filter string.
func SearchShadowRootIframes(queryable PageOrFrame, srcFilter string) ([]playwright.Frame, error) {
	return FindAllIframes(queryable, srcFilter)
}
