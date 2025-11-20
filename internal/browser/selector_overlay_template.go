package browser

// selectorOverlayTemplate contains the JavaScript/Vue.js code for the selector overlay
const selectorOverlayTemplate = `
(function() {
	// Create overlay container
	const overlayContainer = document.createElement('div');
	overlayContainer.id = 'crawlify-selector-overlay';
	document.body.appendChild(overlayContainer);
	
	// Add styles
	const style = document.createElement('style');
	style.textContent = ` + "`" + `
		#crawlify-selector-overlay {
			position: fixed;
			top: 0;
			left: 0;
			width: 100%;
			height: 100%;
			pointer-events: none;
			z-index: 999999;
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
		}
		
		.crawlify-highlight {
			position: fixed;
			pointer-events: none;
			border: 3px solid #3b82f6;
			background: rgba(59, 130, 246, 0.15);
			z-index: 999998;
			transition: all 0.15s cubic-bezier(0.4, 0, 0.2, 1);
			box-shadow: 0 0 0 1px rgba(59, 130, 246, 0.3), 0 4px 12px rgba(59, 130, 246, 0.2);
			animation: crawlify-pulse 2s ease-in-out infinite;
		}
		
		@keyframes crawlify-pulse {
			0%, 100% { box-shadow: 0 0 0 1px rgba(59, 130, 246, 0.3), 0 4px 12px rgba(59, 130, 246, 0.2); }
			50% { box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.5), 0 6px 16px rgba(59, 130, 246, 0.3); }
		}
		
		.crawlify-highlight-parent {
			position: fixed;
			pointer-events: none;
			border: 1px dashed #94a3b8;
			background: rgba(148, 163, 184, 0.05);
			z-index: 999997;
		}
		
		.crawlify-element-tag {
			position: fixed;
			background: #1e293b;
			color: #f1f5f9;
			padding: 4px 8px;
			border-radius: 4px;
			font-size: 11px;
			font-family: 'Courier New', monospace;
			z-index: 1000001;
			pointer-events: none;
			box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
			max-width: 300px;
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;
		}
		
		.crawlify-selected {
			border-color: #10b981;
			background: rgba(16, 185, 129, 0.2);
			animation: crawlify-success-pulse 0.5s ease-out;
		}
		
		@keyframes crawlify-success-pulse {
			0% { transform: scale(1); }
			50% { transform: scale(1.02); }
			100% { transform: scale(1); }
		}
		
		.crawlify-control-panel {
			position: fixed;
			top: 20px;
			right: 20px;
			background: white;
			border-radius: 12px;
			box-shadow: 0 10px 40px rgba(0, 0, 0, 0.25);
			padding: 20px;
			min-width: 360px;
			max-width: 420px;
			max-height: 85vh;
			overflow-y: auto;
			pointer-events: auto;
			z-index: 1000000;
			backdrop-filter: blur(10px);
			border: 1px solid rgba(0, 0, 0, 0.1);
		}
		
		.crawlify-control-panel::-webkit-scrollbar {
			width: 6px;
		}
		
		.crawlify-control-panel::-webkit-scrollbar-track {
			background: #f1f5f9;
			border-radius: 3px;
		}
		
		.crawlify-control-panel::-webkit-scrollbar-thumb {
			background: #cbd5e1;
			border-radius: 3px;
		}
		
		.crawlify-control-panel::-webkit-scrollbar-thumb:hover {
			background: #94a3b8;
		}
		
		.crawlify-header {
			display: flex;
			justify-content: space-between;
			align-items: center;
			margin-bottom: 16px;
			padding-bottom: 12px;
			border-bottom: 2px solid #e5e7eb;
		}
		
		.crawlify-title {
			font-size: 18px;
			font-weight: 700;
			color: #1f2937;
		}
		
		.crawlify-close-btn {
			background: #ef4444;
			color: white;
			border: none;
			border-radius: 6px;
			padding: 6px 12px;
			cursor: pointer;
			font-size: 14px;
			font-weight: 600;
			transition: background 0.2s;
		}
		
		.crawlify-close-btn:hover {
			background: #dc2626;
		}
		
		.crawlify-info {
			background: #dbeafe;
			border-left: 4px solid #3b82f6;
			padding: 12px;
			margin-bottom: 16px;
			border-radius: 4px;
			font-size: 13px;
			color: #1e40af;
		}
		
		.crawlify-mode-toggle {
			margin-bottom: 16px;
		}
		
		.crawlify-toggle-btn {
			width: 100%;
			padding: 10px;
			background: #f3f4f6;
			border: 2px solid #d1d5db;
			border-radius: 6px;
			cursor: pointer;
			font-size: 14px;
			font-weight: 600;
			color: #374151;
			transition: all 0.2s;
		}
		
		.crawlify-toggle-btn.active {
			background: #3b82f6;
			border-color: #3b82f6;
			color: white;
		}
		
		.crawlify-fields {
			margin-bottom: 16px;
		}
		
		.crawlify-field-item {
			background: #f9fafb;
			border: 1px solid #e5e7eb;
			border-radius: 6px;
			padding: 12px;
			margin-bottom: 8px;
			position: relative;
			transition: all 0.2s;
		}
		
		.crawlify-field-item:hover {
			background: #f3f4f6;
			border-color: #d1d5db;
			transform: translateX(2px);
			box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
		}
		
		.crawlify-field-name {
			font-weight: 600;
			color: #1f2937;
			margin-bottom: 4px;
			font-size: 14px;
		}
		
		.crawlify-field-selector {
			font-family: 'Courier New', monospace;
			font-size: 12px;
			color: #6b7280;
			word-break: break-all;
			margin-bottom: 4px;
		}
		
		.crawlify-field-preview {
			font-size: 12px;
			color: #10b981;
			font-style: italic;
			margin-top: 6px;
			padding: 6px;
			background: #ecfdf5;
			border-radius: 4px;
			max-height: 60px;
			overflow-y: auto;
		}
		
		.crawlify-field-count {
			display: inline-block;
			background: #3b82f6;
			color: white;
			padding: 2px 8px;
			border-radius: 12px;
			font-size: 11px;
			font-weight: 600;
			margin-left: 8px;
		}
		
		.crawlify-field-validation {
			display: flex;
			align-items: center;
			gap: 6px;
			margin-top: 6px;
			padding: 6px 8px;
			border-radius: 4px;
			font-size: 11px;
		}
		
		.crawlify-field-validation.valid {
			background: #d1fae5;
			color: #065f46;
		}
		
		.crawlify-field-validation.warning {
			background: #fef3c7;
			color: #92400e;
		}
		
		.crawlify-field-validation.error {
			background: #fee2e2;
			color: #991b1b;
		}
		
		.crawlify-field-validation-icon {
			font-weight: bold;
		}
		
		.crawlify-field-remove {
			position: absolute;
			top: 8px;
			right: 8px;
			background: #ef4444;
			color: white;
			border: none;
			border-radius: 4px;
			width: 24px;
			height: 24px;
			cursor: pointer;
			font-size: 12px;
			display: flex;
			align-items: center;
			justify-content: center;
		}
		
		.crawlify-add-field {
			margin-bottom: 16px;
		}
		
		.crawlify-input {
			width: 100%;
			padding: 8px 12px;
			border: 2px solid #d1d5db;
			border-radius: 6px;
			font-size: 14px;
			margin-bottom: 8px;
			box-sizing: border-box;
		}
		
		.crawlify-input:focus {
			outline: none;
			border-color: #3b82f6;
		}
		
		.crawlify-btn {
			padding: 10px 16px;
			background: #3b82f6;
			color: white;
			border: none;
			border-radius: 6px;
			cursor: pointer;
			font-size: 14px;
			font-weight: 600;
			transition: background 0.2s;
			width: 100%;
		}
		
		.crawlify-btn:hover {
			background: #2563eb;
		}
		
		.crawlify-btn:disabled {
			background: #9ca3af;
			cursor: not-allowed;
		}
		
		.crawlify-btn-success {
			background: #10b981;
			margin-top: 8px;
		}
		
		.crawlify-btn-success:hover {
			background: #059669;
		}
		
		.crawlify-status {
			margin-top: 16px;
			padding: 12px;
			border-radius: 6px;
			font-size: 13px;
			text-align: center;
		}
		
		.crawlify-status.success {
			background: #d1fae5;
			color: #065f46;
		}
		
		.crawlify-status.error {
			background: #fee2e2;
			color: #991b1b;
		}
		
		.crawlify-checkbox-group {
			display: flex;
			align-items: center;
			gap: 8px;
			margin: 8px 0;
		}
		
		.crawlify-checkbox {
			width: 18px;
			height: 18px;
			cursor: pointer;
		}
		
		.crawlify-label {
			font-size: 13px;
			color: #374151;
			cursor: pointer;
		}
		
		.crawlify-select {
			width: 100%;
			padding: 8px 12px;
			border: 2px solid #d1d5db;
			border-radius: 6px;
			font-size: 14px;
			margin-bottom: 8px;
			background: white;
			cursor: pointer;
		}
		
		.crawlify-empty {
			text-align: center;
			color: #9ca3af;
			font-size: 14px;
			padding: 20px;
		}
		
		.crawlify-keyboard-hints {
			background: #f8fafc;
			border: 1px solid #e2e8f0;
			border-radius: 6px;
			padding: 10px;
			margin-top: 12px;
			font-size: 11px;
			color: #64748b;
		}
		
		.crawlify-keyboard-hint {
			display: flex;
			align-items: center;
			justify-content: space-between;
			margin: 4px 0;
		}
		
		.crawlify-kbd {
			background: #ffffff;
			border: 1px solid #cbd5e1;
			border-radius: 3px;
			padding: 2px 6px;
			font-family: monospace;
			font-size: 10px;
			box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
		}
		
		.crawlify-stats {
			display: flex;
			gap: 12px;
			margin: 12px 0;
			padding: 10px;
			background: #f1f5f9;
			border-radius: 6px;
			font-size: 12px;
		}
		
		.crawlify-stat {
			flex: 1;
			text-align: center;
		}
		
		.crawlify-stat-value {
			display: block;
			font-size: 18px;
			font-weight: 700;
			color: #1e293b;
		}
		
		.crawlify-stat-label {
			display: block;
			color: #64748b;
			font-size: 11px;
			margin-top: 2px;
		}
		
		.crawlify-detailed-view {
			animation: slideIn 0.3s ease-out;
		}
		
		@keyframes slideIn {
			from {
				opacity: 0;
				transform: translateX(20px);
			}
			to {
				opacity: 1;
				transform: translateX(0);
			}
		}
		
		.crawlify-detailed-header {
			display: flex;
			align-items: center;
			gap: 12px;
			margin-bottom: 16px;
			padding-bottom: 12px;
			border-bottom: 2px solid #e5e7eb;
		}
		
		.crawlify-back-button {
			background: #f3f4f6;
			color: #374151;
			border: none;
			border-radius: 6px;
			padding: 6px 12px;
			font-size: 13px;
			font-weight: 600;
			cursor: pointer;
			transition: background 0.2s;
		}
		
		.crawlify-back-button:hover {
			background: #e5e7eb;
		}
		
		.crawlify-detailed-title {
			font-size: 16px;
			font-weight: 700;
			color: #1f2937;
			flex: 1;
		}
		
		.crawlify-tabs {
			display: flex;
			gap: 4px;
			margin-bottom: 16px;
			background: #f3f4f6;
			padding: 4px;
			border-radius: 8px;
		}
		
		.crawlify-tab {
			flex: 1;
			background: transparent;
			color: #6b7280;
			border: none;
			border-radius: 6px;
			padding: 8px 12px;
			font-size: 13px;
			font-weight: 600;
			cursor: pointer;
			transition: all 0.2s;
		}
		
		.crawlify-tab:hover {
			background: #e5e7eb;
			color: #374151;
		}
		
		.crawlify-tab.active {
			background: white;
			color: #1f2937;
			box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		}
		
		.crawlify-tab-content {
			animation: fadeIn 0.3s ease-out;
		}
		
		@keyframes fadeIn {
			from {
				opacity: 0;
			}
			to {
				opacity: 1;
			}
		}
		
		.crawlify-config-section {
			background: #f9fafb;
			border: 1px solid #e5e7eb;
			border-radius: 6px;
			padding: 12px;
			margin-bottom: 12px;
		}
		
		.crawlify-config-label {
			font-size: 11px;
			font-weight: 600;
			color: #6b7280;
			text-transform: uppercase;
			letter-spacing: 0.5px;
			margin-bottom: 4px;
		}
		
		.crawlify-config-value {
			font-size: 13px;
			color: #1f2937;
			font-weight: 500;
		}
		
		.crawlify-test-results {
			position: fixed;
			top: 50%;
			left: 50%;
			transform: translate(-50%, -50%);
			background: white;
			border-radius: 12px;
			box-shadow: 0 20px 60px rgba(0, 0, 0, 0.4);
			padding: 24px;
			max-width: 600px;
			max-height: 80vh;
			overflow-y: auto;
			z-index: 1000002;
			pointer-events: auto;
		}
		
		.crawlify-test-results::-webkit-scrollbar {
			width: 6px;
		}
		
		.crawlify-test-results::-webkit-scrollbar-track {
			background: #f1f5f9;
			border-radius: 3px;
		}
		
		.crawlify-test-results::-webkit-scrollbar-thumb {
			background: #cbd5e1;
			border-radius: 3px;
		}
		
		.crawlify-test-overlay {
			position: fixed;
			top: 0;
			left: 0;
			width: 100%;
			height: 100%;
			background: rgba(0, 0, 0, 0.5);
			z-index: 1000001;
			pointer-events: auto;
			backdrop-filter: blur(2px);
		}
		
		.crawlify-test-header {
			display: flex;
			justify-content: space-between;
			align-items: center;
			margin-bottom: 16px;
			padding-bottom: 12px;
			border-bottom: 2px solid #e5e7eb;
		}
		
		.crawlify-test-title {
			font-size: 18px;
			font-weight: 700;
			color: #1f2937;
		}
		
		.crawlify-test-close {
			background: #ef4444;
			color: white;
			border: none;
			border-radius: 6px;
			padding: 6px 12px;
			cursor: pointer;
			font-size: 14px;
			font-weight: 600;
		}
		
		.crawlify-test-summary {
			background: #f0fdf4;
			border-left: 4px solid #10b981;
			padding: 12px;
			margin-bottom: 16px;
			border-radius: 4px;
		}
		
		.crawlify-test-summary-title {
			font-weight: 600;
			color: #065f46;
			margin-bottom: 8px;
		}
		
		.crawlify-test-summary-detail {
			font-size: 13px;
			color: #047857;
			margin: 4px 0;
		}
		
		.crawlify-test-element {
			background: #f9fafb;
			border: 1px solid #e5e7eb;
			border-radius: 6px;
			padding: 12px;
			margin-bottom: 8px;
		}
		
		.crawlify-test-element-header {
			display: flex;
			justify-content: space-between;
			align-items: center;
			margin-bottom: 8px;
		}
		
		.crawlify-test-element-index {
			background: #10b981;
			color: white;
			padding: 2px 8px;
			border-radius: 12px;
			font-size: 11px;
			font-weight: 600;
		}
		
		.crawlify-test-element-tag {
			font-family: 'Courier New', monospace;
			font-size: 12px;
			color: #6b7280;
		}
		
		.crawlify-test-element-value {
			font-size: 13px;
			color: #1f2937;
			padding: 8px;
			background: white;
			border-radius: 4px;
			word-break: break-word;
			max-height: 100px;
			overflow-y: auto;
		}
		
		.crawlify-test-error {
			background: #fee2e2;
			border-left: 4px solid #ef4444;
			padding: 12px;
			border-radius: 4px;
			color: #991b1b;
		}
	` + "`" + `;
	document.head.appendChild(style);
	
	// Wait for Vue to be available
	const waitForVue = setInterval(() => {
		if (typeof Vue !== 'undefined') {
			clearInterval(waitForVue);
			initVueApp();
		}
	}, 100);
	
	function initVueApp() {
		const { createApp } = Vue;
		
		// Prevent default link navigation and form submissions during selection
		const preventNavigation = (event) => {
			// Check if we're in the control panel or test modal
			if (event.target.closest('#crawlify-selector-overlay .crawlify-control-panel') ||
				event.target.closest('#crawlify-selector-overlay .crawlify-test-results') ||
				event.target.closest('#crawlify-selector-overlay .crawlify-test-overlay')) {
				return; // Allow interactions within control panel and test modal
			}
			
			// Prevent navigation for links and form submissions
			if (event.target.tagName === 'A' || event.target.closest('a')) {
				event.preventDefault();
				event.stopPropagation();
			}
			if (event.target.tagName === 'FORM' || event.target.closest('form')) {
				event.preventDefault();
				event.stopPropagation();
			}
			if (event.target.tagName === 'BUTTON' || event.target.tagName === 'INPUT' && 
				(event.target.type === 'submit' || event.target.type === 'button')) {
				event.preventDefault();
				event.stopPropagation();
			}
		};
		
		// Add event listeners to prevent navigation
		document.addEventListener('click', preventNavigation, true);
		document.addEventListener('submit', preventNavigation, true);
		
		// Store the cleanup function
		window.__crawlifyCleanupNavigation = () => {
			document.removeEventListener('click', preventNavigation, true);
			document.removeEventListener('submit', preventNavigation, true);
		};
		
		const app = createApp({
			data() {
				return {
					mode: 'single', // 'single' or 'multiple'
					selectionActive: true,
					selectedFields: [],
					currentFieldName: '',
					currentFieldType: 'text',
					currentFieldAttribute: '',
					hoveredElement: null,
					hoveredElementSelector: null,
					hoveredElementCount: 0,
					hoveredElementValidation: null,
					status: null,
					statusType: 'success',
					testingSelector: null,
					testResults: null,
					detailedViewField: null,
					detailedViewTab: 'test', // 'test' or 'config'
					lockedElement: null,
					lockedElementSelector: null
				};
			},
			methods: {
				toggleMode() {
					this.mode = this.mode === 'single' ? 'multiple' : 'single';
					this.updateElementCount();
					this.showStatus('Mode: ' + (this.mode === 'single' ? 'Single Element' : 'Multiple Elements'), 'success');
				},
				
				updateElementCount() {
					if (!this.hoveredElement || !this.hoveredElementSelector) {
						this.hoveredElementCount = 0;
						return;
					}
					
					try {
						const elements = document.querySelectorAll(this.hoveredElementSelector);
						this.hoveredElementCount = elements.length;
						this.validateSelector(this.hoveredElementSelector, elements.length);
					} catch (e) {
						this.hoveredElementCount = 0;
						this.hoveredElementValidation = {
							type: 'error',
							message: 'Invalid selector'
						};
					}
				},
				
				validateSelector(selector, count) {
					if (count === 0) {
						this.hoveredElementValidation = {
							type: 'error',
							message: '⚠ No elements match this selector'
						};
					} else if (this.mode === 'single' && count > 1) {
						this.hoveredElementValidation = {
							type: 'warning',
							message: '⚠ ' + count + ' elements found. Will use first match.'
						};
					} else if (this.mode === 'multiple' && count === 1) {
						this.hoveredElementValidation = {
							type: 'warning',
							message: '⚠ Only 1 element found. Consider using single mode.'
						};
					} else if (this.mode === 'multiple' && count > 100) {
						this.hoveredElementValidation = {
							type: 'warning',
							message: '⚠ ' + count + ' elements found. This may be too many.'
						};
					} else {
						this.hoveredElementValidation = {
							type: 'valid',
							message: '✓ ' + count + ' element' + (count > 1 ? 's' : '') + ' found'
						};
					}
				},
				
				addCurrentSelection() {
					if (!this.currentFieldName.trim()) {
						this.showStatus('Please enter a field name', 'error');
						return;
					}
					
					if (!this.hoveredElement || !this.hoveredElementSelector) {
						this.showStatus('Please hover over an element first', 'error');
						return;
					}
					
					// Check for duplicate field names
					if (this.selectedFields.some(f => f.name === this.currentFieldName.trim())) {
						this.showStatus('Field name already exists. Please use a unique name.', 'error');
						return;
					}
					
					// Use the cached selector to ensure consistency
					const selector = this.hoveredElementSelector;
					const preview = this.getElementPreview(this.hoveredElement);
					
					// Get actual element count (should match what we already calculated)
					let elementCount = this.hoveredElementCount || 1;
					try {
						elementCount = document.querySelectorAll(selector).length;
					} catch (e) {
						console.error('Error counting elements:', e);
					}
					
					// Create validation for the field
					const validation = this.createFieldValidation(selector, elementCount);
					
					this.selectedFields.push({
						name: this.currentFieldName.trim(),
						selector: selector,
						type: this.currentFieldType,
						attribute: this.currentFieldAttribute,
						multiple: this.mode === 'multiple',
						preview: preview,
						elementCount: elementCount,
						validation: validation
					});
					
					this.currentFieldName = '';
					this.currentFieldAttribute = '';
					this.showStatus('Field "' + this.selectedFields[this.selectedFields.length - 1].name + '" added successfully!', 'success');
					this.saveToWindow();
				},
				
				createFieldValidation(selector, count) {
					if (count === 0) {
						return {
							type: 'error',
							message: 'No elements found'
						};
					} else if (count > 0 && count <= 100) {
						return {
							type: 'valid',
							message: 'Selector is valid'
						};
					} else {
						return {
							type: 'warning',
							message: 'Very high element count (' + count + ')'
						};
					}
				},
				
				removeField(index) {
					this.selectedFields.splice(index, 1);
					this.saveToWindow();
				},
				
				generateSelector(element) {
					// Enhanced selector generation with priority scoring
					const selectorStrategies = [
						// Strategy 1: ID selector (highest priority)
						() => {
							if (element.id && /^[a-zA-Z][\w-]*$/.test(element.id)) {
								const selector = '#' + CSS.escape(element.id);
								if (this.isUniqueSelector(selector)) {
									return { selector, score: 100, type: 'id' };
								}
							}
							return null;
						},
						
						// Strategy 2: Data attributes (high priority for stability)
						() => {
							const dataAttrs = ['data-testid', 'data-test', 'data-id', 'data-cy', 'data-automation'];
							for (const attr of dataAttrs) {
								const value = element.getAttribute(attr);
								if (value) {
									const selector = element.tagName.toLowerCase() + '[' + attr + '="' + CSS.escape(value) + '"]';
									if (this.isUniqueSelector(selector)) {
										return { selector, score: 95, type: 'data-attribute' };
									}
								}
							}
							return null;
						},
						
						// Strategy 3: ARIA labels (good for accessibility)
						() => {
							const ariaLabel = element.getAttribute('aria-label');
							if (ariaLabel) {
								const selector = element.tagName.toLowerCase() + '[aria-label="' + CSS.escape(ariaLabel) + '"]';
								if (this.isUniqueSelector(selector)) {
									return { selector, score: 90, type: 'aria-label' };
								}
							}
							return null;
						},
						
						// Strategy 4: Name attribute (forms)
						() => {
							const name = element.getAttribute('name');
							if (name) {
								const selector = element.tagName.toLowerCase() + '[name="' + CSS.escape(name) + '"]';
								if (this.isUniqueSelector(selector)) {
									return { selector, score: 85, type: 'name' };
								}
							}
							return null;
						},
						
						// Strategy 5: Semantic classes (meaningful names)
						() => {
							if (element.className && typeof element.className === 'string') {
								const classes = element.className.split(' ').filter(c => c.trim() && this.isSemanticClass(c));
								if (classes.length > 0) {
									// Try individual semantic classes
									for (const cls of classes) {
										const selector = element.tagName.toLowerCase() + '.' + CSS.escape(cls);
										if (this.isUniqueSelector(selector)) {
											return { selector, score: 80, type: 'semantic-class' };
										}
									}
									// Try combination
									const selector = element.tagName.toLowerCase() + '.' + classes.map(c => CSS.escape(c)).join('.');
									if (this.isUniqueSelector(selector)) {
										return { selector, score: 75, type: 'semantic-class-combo' };
									}
								}
							}
							return null;
						},
						
						// Strategy 6: Tag with unique type
						() => {
							const type = element.getAttribute('type');
							if (type) {
								const selector = element.tagName.toLowerCase() + '[type="' + CSS.escape(type) + '"]';
								if (this.isUniqueSelector(selector)) {
									return { selector, score: 70, type: 'type-attribute' };
								}
							}
							return null;
						}
					];
					
					// Try strategies in order
					for (const strategy of selectorStrategies) {
						const result = strategy();
						if (result) {
							return result.selector;
						}
					}
					
					// Fallback: Build path-based selector
					return this.buildPathSelector(element);
				},
				
				isUniqueSelector(selector) {
					try {
						return document.querySelectorAll(selector).length === 1;
					} catch (e) {
						return false;
					}
				},
				
				isSemanticClass(className) {
					// Check if class name is semantic (not generated/utility)
					const semanticPatterns = [
						/^(title|heading|content|body|header|footer|nav|menu|button|link|item|card|list)/i,
						/^(article|section|main|sidebar|container|wrapper|form|input|label)/i
					];
					const nonSemanticPatterns = [
						/^(col|row|xs|sm|md|lg|xl|offset|pull|push|hidden|visible)/i, // Bootstrap/Grid
						/^(p|m|pt|pb|pl|pr|mt|mb|ml|mr|w|h|text|bg|border|rounded|flex|grid)-/i, // Utility classes
						/^[a-f0-9]{6,}$/i, // Hash-like classes
						/^_[a-zA-Z0-9]+$/  // CSS modules
					];
					
					for (const pattern of nonSemanticPatterns) {
						if (pattern.test(className)) return false;
					}
					
					for (const pattern of semanticPatterns) {
						if (pattern.test(className)) return true;
					}
					
					return className.length > 3 && /^[a-zA-Z]/.test(className);
				},
				
				buildPathSelector(element) {
					// Build path from parent with smart strategies
					let current = element;
					const path = [];
					const maxDepth = 5;
					let depth = 0;
					
					while (current && current !== document.body && depth < maxDepth) {
						let selector = current.tagName.toLowerCase();
						
						// Stop at parent with ID
						if (current.id && /^[a-zA-Z][\w-]*$/.test(current.id)) {
							selector = '#' + CSS.escape(current.id);
							path.unshift(selector);
							break;
						}
						
						// Add semantic classes if available
						if (current.className && typeof current.className === 'string') {
							const classes = current.className.split(' ')
								.filter(c => c.trim() && this.isSemanticClass(c))
								.slice(0, 2); // Limit to 2 classes
							if (classes.length > 0) {
								selector += '.' + classes.map(c => CSS.escape(c)).join('.');
							}
						}
						
						// Add nth-of-type only if necessary
						const parent = current.parentElement;
						if (parent && depth === 0) { // Only for target element
							const siblings = Array.from(parent.children).filter(
								child => child.tagName === current.tagName
							);
							if (siblings.length > 1) {
								const index = siblings.indexOf(current) + 1;
								selector += ':nth-of-type(' + index + ')';
							}
						}
						
						path.unshift(selector);
						current = current.parentElement;
						depth++;
					}
					
					return path.join(' > ');
				},
				
				getElementPreview(element) {
					if (this.currentFieldType === 'text') {
						return element.textContent.trim().substring(0, 50);
					} else if (this.currentFieldType === 'attribute' && this.currentFieldAttribute) {
						return element.getAttribute(this.currentFieldAttribute) || '';
					} else if (this.currentFieldType === 'html') {
						return element.innerHTML.substring(0, 50);
					}
					return '';
				},
				
				handleMouseMove(event) {
					if (!this.selectionActive) return;
					
					// Don't update hover when element is locked or in detailed view
					if (this.lockedElement || this.detailedViewField) return;
					
					// Find the actual target element (not our overlay)
					const elements = document.elementsFromPoint(event.clientX, event.clientY);
					const targetElement = elements.find(el => 
						!el.closest('#crawlify-selector-overlay')
					);
					
					if (targetElement && targetElement !== this.hoveredElement) {
						this.hoveredElement = targetElement;
						// Cache the selector for this element to ensure consistency
						this.hoveredElementSelector = this.generateSelector(targetElement);
						this.highlightElement(targetElement);
						this.updateElementCount();
					}
				},
				
				lockElement() {
					if (!this.hoveredElement || !this.hoveredElementSelector) return;
					
					this.lockedElement = this.hoveredElement;
					this.lockedElementSelector = this.hoveredElementSelector;
					this.selectionActive = false;
					
					// Show locked highlight
					this.showLockedHighlight();
					this.showStatus('Element locked. Click anywhere on page to unlock.', 'success');
				},
				
				unlockElement() {
					this.lockedElement = null;
					this.lockedElementSelector = null;
					this.selectionActive = true;
					
					// Remove locked highlights
					document.querySelectorAll('.crawlify-locked-highlight, .crawlify-element-tooltip').forEach(el => el.remove());
					this.showStatus('Element unlocked. Resume selection.', 'success');
				},
				
				handlePageClick(event) {
					// Ignore clicks on overlay
					if (event.target.closest('#crawlify-selector-overlay')) {
						return;
					}
					
					// If element is locked, unlock it
					if (this.lockedElement) {
						this.unlockElement();
						event.preventDefault();
						event.stopPropagation();
						return;
					}
					
					// If hovering over an element, lock it
					if (this.hoveredElement && !this.detailedViewField) {
						this.lockElement();
						event.preventDefault();
						event.stopPropagation();
					}
				},
				
				highlightElement(element) {
					// Remove old highlights
					document.querySelectorAll('.crawlify-highlight, .crawlify-highlight-parent, .crawlify-element-tag').forEach(el => el.remove());
					
					const rect = element.getBoundingClientRect();
					
					// Highlight parent element for context
					if (element.parentElement && element.parentElement !== document.body) {
						const parentRect = element.parentElement.getBoundingClientRect();
						const parentHighlight = document.createElement('div');
						parentHighlight.className = 'crawlify-highlight-parent';
						parentHighlight.style.top = parentRect.top + 'px';
						parentHighlight.style.left = parentRect.left + 'px';
						parentHighlight.style.width = parentRect.width + 'px';
						parentHighlight.style.height = parentRect.height + 'px';
						document.body.appendChild(parentHighlight);
					}
					
					// Main highlight
					const highlight = document.createElement('div');
					highlight.className = 'crawlify-highlight';
					highlight.style.top = rect.top + 'px';
					highlight.style.left = rect.left + 'px';
					highlight.style.width = rect.width + 'px';
					highlight.style.height = rect.height + 'px';
					document.body.appendChild(highlight);
					
					// Element tag label - show the actual generated selector
					const tag = document.createElement('div');
					tag.className = 'crawlify-element-tag';
					// Show the cached selector if available, otherwise show basic tag info
					if (this.hoveredElementSelector) {
						tag.textContent = this.hoveredElementSelector;
					} else {
						const tagName = element.tagName.toLowerCase();
						const elementId = element.id ? '#' + element.id : '';
						const elementClass = element.className && typeof element.className === 'string' 
							? '.' + element.className.split(' ').filter(c => c.trim()).slice(0, 2).join('.') 
							: '';
						tag.textContent = tagName + elementId + elementClass;
					}
					
					// Position tag above element, or below if not enough space
					const tagTop = rect.top > 30 ? rect.top - 24 : rect.bottom + 4;
					tag.style.top = tagTop + 'px';
					tag.style.left = rect.left + 'px';
					
					document.body.appendChild(tag);
				},
				
				showLockedHighlight() {
					if (!this.lockedElement) return;
					
					// Remove old locked highlights
					document.querySelectorAll('.crawlify-locked-highlight, .crawlify-element-tooltip').forEach(el => el.remove());
					
					const rect = this.lockedElement.getBoundingClientRect();
					
					// Locked highlight (thicker border, different color)
					const highlight = document.createElement('div');
					highlight.className = 'crawlify-locked-highlight';
					highlight.style.position = 'fixed';
					highlight.style.top = rect.top + 'px';
					highlight.style.left = rect.left + 'px';
					highlight.style.width = rect.width + 'px';
					highlight.style.height = rect.height + 'px';
					highlight.style.border = '4px solid #f59e0b';
					highlight.style.background = 'rgba(245, 158, 11, 0.1)';
					highlight.style.pointerEvents = 'none';
					highlight.style.zIndex = '999999';
					highlight.style.boxShadow = '0 0 0 2px rgba(245, 158, 11, 0.3), 0 4px 12px rgba(245, 158, 11, 0.4)';
					highlight.style.borderRadius = '4px';
					document.body.appendChild(highlight);
					
					// Element tooltip with type and selector
					const tooltip = document.createElement('div');
					tooltip.className = 'crawlify-element-tooltip';
					tooltip.style.position = 'fixed';
					tooltip.style.background = '#1f2937';
					tooltip.style.color = '#f9fafb';
					tooltip.style.padding = '8px 12px';
					tooltip.style.borderRadius = '6px';
					tooltip.style.fontSize = '12px';
					tooltip.style.zIndex = '1000000';
					tooltip.style.pointerEvents = 'none';
					tooltip.style.boxShadow = '0 4px 12px rgba(0, 0, 0, 0.3)';
					tooltip.style.maxWidth = '300px';
					tooltip.style.wordWrap = 'break-word';
					
					// Get element type
					const tagName = this.lockedElement.tagName.toLowerCase();
					let elementType = tagName;
					if (tagName === 'a') elementType = 'Link';
					else if (tagName === 'button') elementType = 'Button';
					else if (tagName === 'input') elementType = 'Input (' + (this.lockedElement.type || 'text') + ')';
					else if (tagName === 'select') elementType = 'Select';
					else if (tagName === 'textarea') elementType = 'Textarea';
					else if (tagName === 'img') elementType = 'Image';
					else if (tagName === 'div') elementType = 'Div';
					else if (tagName === 'span') elementType = 'Span';
					
					tooltip.innerHTML = ` + "`" + `
						<div style="font-weight: 600; margin-bottom: 4px; color: #fbbf24;">
							🔒 Locked Element
						</div>
						<div style="margin-bottom: 4px;">
							<span style="color: #9ca3af;">Type:</span> 
							<span style="color: #60a5fa;">${elementType}</span>
						</div>
						<div style="font-family: monospace; font-size: 11px; color: #d1d5db;">
							${this.lockedElementSelector}
						</div>
						<div style="margin-top: 6px; font-size: 10px; color: #9ca3af; font-style: italic;">
							Click anywhere to unlock
						</div>
					` + "`" + `;
					
					// Position tooltip near the element
					const tooltipTop = rect.top > 120 ? rect.top - 100 : rect.bottom + 10;
					const tooltipLeft = Math.min(rect.left, window.innerWidth - 320);
					tooltip.style.top = tooltipTop + 'px';
					tooltip.style.left = tooltipLeft + 'px';
					
					document.body.appendChild(tooltip);
				},
				
				showStatus(message, type) {
					this.status = message;
					this.statusType = type;
					setTimeout(() => {
						this.status = null;
					}, 3000);
				},
				
				saveToWindow() {
					// Save selections to window object so backend can retrieve them
					window.__crawlifySelections = this.selectedFields;
				},
				
				handleKeyDown(event) {
					// Escape key to close
					if (event.key === 'Escape') {
						event.preventDefault();
						this.closeOverlay();
					}
					
					// Enter key to add field (if input has value)
					if (event.key === 'Enter' && this.currentFieldName.trim() && this.hoveredElement) {
						event.preventDefault();
						this.addCurrentSelection();
					}
					
					// Delete/Backspace to remove last field (if not in input)
					if ((event.key === 'Delete' || event.key === 'Backspace') && 
						event.target.tagName !== 'INPUT' && 
						this.selectedFields.length > 0) {
						event.preventDefault();
						this.removeField(this.selectedFields.length - 1);
					}
					
					// Tab key to toggle mode
					if (event.key === 'Tab' && event.target.tagName !== 'INPUT') {
						event.preventDefault();
						this.toggleMode();
					}
				},
				
				openDetailedView(field) {
					this.detailedViewField = field;
					this.detailedViewTab = 'test';
					
					// Run test automatically when opening detailed view
					this.testSelectorInline(field);
				},
				
				closeDetailedView() {
					this.detailedViewField = null;
					this.testResults = null;
					document.querySelectorAll('.crawlify-test-highlight').forEach(el => el.remove());
				},
				
				switchTab(tab) {
					this.detailedViewTab = tab;
				},
				
				testSelectorInline(field) {
					this.testingSelector = field.name;
					this.testResults = null;
					
					try {
						const elements = document.querySelectorAll(field.selector);
						const results = {
							count: elements.length,
							elements: [],
							selector: field.selector,
							type: field.type,
							attribute: field.attribute
						};
						
						// Collect data from matched elements
						Array.from(elements).slice(0, 10).forEach((el, index) => {
							let value = '';
							if (field.type === 'text') {
								value = el.textContent.trim();
							} else if (field.type === 'attribute' && field.attribute) {
								value = el.getAttribute(field.attribute) || '';
							} else if (field.type === 'html') {
								value = el.innerHTML.substring(0, 200);
							}
							
							results.elements.push({
								index: index + 1,
								value: value.substring(0, 100),
								tagName: el.tagName.toLowerCase(),
								classes: el.className && typeof el.className === 'string' ? el.className : ''
							});
						});
						
						this.testResults = results;
						
						// Highlight all matching elements
						this.highlightTestResults(elements);
						
					} catch (error) {
						this.testResults = {
							error: 'Invalid selector or error: ' + error.message,
							selector: field.selector
						};
					}
				},
				
				highlightTestResults(elements) {
					// Remove old test highlights
					document.querySelectorAll('.crawlify-test-highlight').forEach(el => el.remove());
					
					// Add highlights for each matched element
					Array.from(elements).forEach((element, index) => {
						const rect = element.getBoundingClientRect();
						const highlight = document.createElement('div');
						highlight.className = 'crawlify-test-highlight';
						highlight.style.position = 'fixed';
						highlight.style.top = rect.top + 'px';
						highlight.style.left = rect.left + 'px';
						highlight.style.width = rect.width + 'px';
						highlight.style.height = rect.height + 'px';
						highlight.style.border = '2px solid #10b981';
						highlight.style.background = 'rgba(16, 185, 129, 0.1)';
						highlight.style.pointerEvents = 'none';
						highlight.style.zIndex = '999996';
						highlight.style.boxShadow = '0 0 0 1px rgba(16, 185, 129, 0.3)';
						
						// Add index label
						const label = document.createElement('div');
						label.style.position = 'absolute';
						label.style.top = '2px';
						label.style.left = '2px';
						label.style.background = '#10b981';
						label.style.color = 'white';
						label.style.padding = '2px 6px';
						label.style.borderRadius = '3px';
						label.style.fontSize = '11px';
						label.style.fontWeight = 'bold';
						label.textContent = (index + 1).toString();
						highlight.appendChild(label);
						
						document.body.appendChild(highlight);
					});
				},
				
				closeTestResults() {
					this.testingSelector = null;
					this.testResults = null;
					document.querySelectorAll('.crawlify-test-highlight').forEach(el => el.remove());
				},
				
				closeOverlay() {
					// Clean up navigation prevention
					if (window.__crawlifyCleanupNavigation) {
						window.__crawlifyCleanupNavigation();
					}
					
					// Signal backend that we're done
					window.__crawlifyClosed = true;
					document.getElementById('crawlify-selector-overlay').remove();
					document.querySelectorAll('.crawlify-highlight, .crawlify-highlight-parent, .crawlify-element-tag, .crawlify-test-highlight').forEach(el => el.remove());
					document.removeEventListener('mousemove', this.handleMouseMove);
					document.removeEventListener('keydown', this.handleKeyDown);
				}
			},
			mounted() {
				document.addEventListener('mousemove', this.handleMouseMove);
				document.addEventListener('click', this.handlePageClick, true);
				
				// Keyboard shortcuts
				document.addEventListener('keydown', this.handleKeyDown);
				
				// Make selections available to backend
				window.__crawlifyGetSelections = () => {
					return this.selectedFields;
				};
			},
			beforeUnmount() {
				document.removeEventListener('mousemove', this.handleMouseMove);
				document.removeEventListener('click', this.handlePageClick, true);
				document.removeEventListener('keydown', this.handleKeyDown);
			},
			computed: {
				totalElementsSelected() {
					return this.selectedFields.reduce((sum, field) => {
						return sum + (field.elementCount || 1);
					}, 0);
				},
				validFieldsCount() {
					return this.selectedFields.filter(f => 
						f.validation && f.validation.type === 'valid'
					).length;
				}
			},
			template: ` + "`" + `
				<div>
					<div class="crawlify-control-panel">
						<div class="crawlify-header">
							<div class="crawlify-title">🎯 Element Selector</div>
							<button class="crawlify-close-btn" @click="closeOverlay">Done</button>
						</div>
						
						<div class="crawlify-info">
							Hover over elements to select them. Click "Add Field" to save the selector.
						</div>
						
						<!-- Stats Display -->
						<div v-if="selectedFields.length > 0" class="crawlify-stats">
							<div class="crawlify-stat">
								<span class="crawlify-stat-value">{{ selectedFields.length }}</span>
								<span class="crawlify-stat-label">Fields</span>
							</div>
							<div class="crawlify-stat">
								<span class="crawlify-stat-value">{{ validFieldsCount }}</span>
								<span class="crawlify-stat-label">Valid</span>
							</div>
							<div class="crawlify-stat">
								<span class="crawlify-stat-value">{{ totalElementsSelected }}</span>
								<span class="crawlify-stat-label">Elements</span>
							</div>
						</div>
						
						<!-- Real-time validation display -->
						<div v-if="hoveredElement && hoveredElementValidation" 
							class="crawlify-field-validation"
							:class="hoveredElementValidation.type">
							<span class="crawlify-field-validation-icon">
								{{ hoveredElementValidation.type === 'valid' ? '✓' : hoveredElementValidation.type === 'warning' ? '⚠' : '✗' }}
							</span>
							<span>{{ hoveredElementValidation.message }}</span>
						</div>
						
						<div class="crawlify-mode-toggle">
							<button 
								class="crawlify-toggle-btn"
								:class="{ active: mode === 'multiple' }"
								@click="toggleMode">
								{{ mode === 'single' ? '📄 Single Element' : '📑 Multiple Elements' }}
								<span v-if="hoveredElementCount > 0" class="crawlify-field-count">
									{{ hoveredElementCount }}
								</span>
							</button>
						</div>
						
						<div class="crawlify-add-field">
							<input 
								v-model="currentFieldName" 
								class="crawlify-input" 
								placeholder="Field name (e.g., 'title')"
								@keyup.enter="addCurrentSelection">
							
							<select v-model="currentFieldType" class="crawlify-select">
								<option value="text">Text Content</option>
								<option value="attribute">Attribute</option>
								<option value="html">HTML</option>
							</select>
							
							<input 
								v-if="currentFieldType === 'attribute'"
								v-model="currentFieldAttribute" 
								class="crawlify-input" 
								placeholder="Attribute name (e.g., 'href')">
							
							<button 
								class="crawlify-btn" 
								@click="addCurrentSelection"
								:disabled="!currentFieldName.trim() || !hoveredElement">
								➕ Add Field
							</button>
						</div>
						
						<!-- Fields List View -->
						<div v-if="!detailedViewField" class="crawlify-fields">
							<div v-if="selectedFields.length > 0">
								<div style="font-size: 14px; font-weight: 600; margin-bottom: 8px; color: #374151;">
									Selected Fields ({{ selectedFields.length }})
								</div>
								<div 
									v-for="(field, index) in selectedFields" 
									:key="index"
									class="crawlify-field-item"
									@click="openDetailedView(field)"
									style="cursor: pointer;">
									<button class="crawlify-field-remove" @click.stop="removeField(index)">×</button>
									<div class="crawlify-field-name">
										{{ field.name }}
										<span v-if="field.multiple" class="crawlify-field-count">
											{{ field.elementCount || '?' }} elements
										</span>
										<span v-else style="color: #6b7280; font-size: 12px;"> (single)</span>
									</div>
									<div class="crawlify-field-selector">{{ field.selector }}</div>
									<div v-if="field.preview" class="crawlify-field-preview">
										Preview: "{{ field.preview }}"
									</div>
									<div v-if="field.validation" 
										class="crawlify-field-validation"
										:class="field.validation.type">
										<span class="crawlify-field-validation-icon">
											{{ field.validation.type === 'valid' ? '✓' : field.validation.type === 'warning' ? '⚠' : '✗' }}
										</span>
										<span>{{ field.validation.message }}</span>
									</div>
								</div>
							</div>
						</div>
						
						<!-- Detailed View -->
						<div v-if="detailedViewField" class="crawlify-detailed-view">
							<div class="crawlify-detailed-header">
								<button class="crawlify-back-button" @click="closeDetailedView">
									← Back
								</button>
								<div class="crawlify-detailed-title">{{ detailedViewField.name }}</div>
							</div>
							
							<!-- Tabs -->
							<div class="crawlify-tabs">
								<button 
									class="crawlify-tab"
									:class="{ active: detailedViewTab === 'test' }"
									@click="switchTab('test')">
									🧪 Test Results
								</button>
								<button 
									class="crawlify-tab"
									:class="{ active: detailedViewTab === 'config' }"
									@click="switchTab('config')">
									⚙️ Configuration
								</button>
							</div>
							
							<!-- Test Results Tab -->
							<div v-if="detailedViewTab === 'test'" class="crawlify-tab-content">
								<div v-if="testResults && !testResults.error">
									<div class="crawlify-test-summary">
										<div class="crawlify-test-summary-title">Summary</div>
										<div class="crawlify-test-summary-detail">
											<strong>Selector:</strong> <code style="background: white; padding: 2px 4px; border-radius: 2px;">{{ testResults.selector }}</code>
										</div>
										<div class="crawlify-test-summary-detail">
											<strong>Total matches:</strong> {{ testResults.count }} element(s)
										</div>
										<div class="crawlify-test-summary-detail">
											<strong>Extraction type:</strong> {{ testResults.type }}
											<span v-if="testResults.attribute"> ({{ testResults.attribute }})</span>
										</div>
										<div class="crawlify-test-summary-detail" style="margin-top: 8px; color: #059669;">
											✓ All matching elements are highlighted on the page
										</div>
									</div>
									
									<div style="font-weight: 600; margin: 12px 0 8px 0; color: #374151;">
										Sample Data (first {{ Math.min(10, testResults.elements.length) }} of {{ testResults.count }})
									</div>
									
									<div v-for="element in testResults.elements" :key="element.index" class="crawlify-test-element">
										<div class="crawlify-test-element-header">
											<span class="crawlify-test-element-index">#{{ element.index }}</span>
											<span class="crawlify-test-element-tag">
												&lt;{{ element.tagName }}&gt;
												<span v-if="element.classes" style="color: #9ca3af;">{{ element.classes.substring(0, 30) }}</span>
											</span>
										</div>
										<div class="crawlify-test-element-value">
											{{ element.value || '(empty)' }}
										</div>
									</div>
									
									<div v-if="testResults.count > 10" style="text-align: center; color: #6b7280; font-size: 12px; margin-top: 12px; padding: 8px; background: #f9fafb; border-radius: 4px;">
										+ {{ testResults.count - 10 }} more elements
									</div>
								</div>
								
								<div v-if="testResults && testResults.error" class="crawlify-test-error">
									{{ testResults.error }}
									<div style="margin-top: 8px; font-family: monospace; font-size: 11px;">
										{{ testResults.selector }}
									</div>
								</div>
							</div>
							
							<!-- Configuration Tab -->
							<div v-if="detailedViewTab === 'config'" class="crawlify-tab-content">
								<div class="crawlify-config-section">
									<div class="crawlify-config-label">Field Name</div>
									<div class="crawlify-config-value">{{ detailedViewField.name }}</div>
								</div>
								
								<div class="crawlify-config-section">
									<div class="crawlify-config-label">CSS Selector</div>
									<div class="crawlify-config-value" style="font-family: monospace; font-size: 11px; word-break: break-all;">
										{{ detailedViewField.selector }}
									</div>
								</div>
								
								<div class="crawlify-config-section">
									<div class="crawlify-config-label">Extraction Type</div>
									<div class="crawlify-config-value">{{ detailedViewField.type }}</div>
								</div>
								
								<div v-if="detailedViewField.type === 'attribute'" class="crawlify-config-section">
									<div class="crawlify-config-label">Attribute Name</div>
									<div class="crawlify-config-value">{{ detailedViewField.attribute }}</div>
								</div>
								
								<div class="crawlify-config-section">
									<div class="crawlify-config-label">Selection Mode</div>
									<div class="crawlify-config-value">
										{{ detailedViewField.multiple ? 'Multiple Elements' : 'Single Element' }}
									</div>
								</div>
								
								<div class="crawlify-config-section">
									<div class="crawlify-config-label">Element Count</div>
									<div class="crawlify-config-value">{{ detailedViewField.elementCount || '?' }}</div>
								</div>
								
								<div v-if="detailedViewField.preview" class="crawlify-config-section">
									<div class="crawlify-config-label">Preview</div>
									<div class="crawlify-config-value" style="font-style: italic; color: #6b7280;">
										"{{ detailedViewField.preview }}"
									</div>
								</div>
							</div>
						</div>
						
						<div v-else class="crawlify-empty">
							No fields selected yet
						</div>
						
						<!-- Keyboard shortcuts help -->
						<div v-if="!detailedViewField" class="crawlify-keyboard-hints">
							<div style="font-weight: 600; margin-bottom: 6px; color: #475569;">⌨️ Keyboard Shortcuts</div>
							<div class="crawlify-keyboard-hint">
								<span>Add current field</span>
								<span class="crawlify-kbd">Enter</span>
							</div>
							<div class="crawlify-keyboard-hint">
								<span>Toggle mode</span>
								<span class="crawlify-kbd">Tab</span>
							</div>
							<div class="crawlify-keyboard-hint">
								<span>Remove last field</span>
								<span class="crawlify-kbd">Delete</span>
							</div>
							<div class="crawlify-keyboard-hint">
								<span>Close selector</span>
								<span class="crawlify-kbd">Esc</span>
							</div>
						</div>
						
						<div v-if="status" class="crawlify-status" :class="statusType">
							{{ status }}
						</div>
					</div>
				</div>
			` + "`" + `
		});
		
		app.mount('#crawlify-selector-overlay');
	}
})();
`
