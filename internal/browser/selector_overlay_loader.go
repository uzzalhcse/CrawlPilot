package browser

import (
	_ "embed"
)

//go:embed dist/selector-overlay.js
var selectorOverlayJS string

//go:embed dist/selector-overlay.css
var selectorOverlayCSS string

// getSelectorOverlayJS returns the compiled Vue.js selector overlay with injected CSS
func getSelectorOverlayJS() string {
	// Inject CSS into the JavaScript
	cssInjectionCode := `
(function() {
	const style = document.createElement('style');
	style.textContent = ` + "`" + selectorOverlayCSS + "`" + `;
	document.head.appendChild(style);
})();
`
	return cssInjectionCode + "\n" + selectorOverlayJS
}
