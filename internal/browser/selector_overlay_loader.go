package browser

import (
	_ "embed"
)

//go:embed dist/selectflow.js
var selectorOverlayJS string

////go:embed dist/selector-overlay.css
//var selectorOverlayCSS string

// getSelectorOverlayJS returns the compiled Vue.js selector overlay with injected CSS
func getSelectorOverlayJS() string {
	// Inject CSS into the JavaScript
	cssInjectionCode := ``
	return cssInjectionCode + "\n" + selectorOverlayJS
}
