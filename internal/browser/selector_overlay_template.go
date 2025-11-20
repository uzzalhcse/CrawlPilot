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
			position: absolute;
			pointer-events: none;
			border: 2px solid #3b82f6;
			background: rgba(59, 130, 246, 0.1);
			z-index: 999998;
			transition: all 0.1s ease;
		}
		
		.crawlify-selected {
			border-color: #10b981;
			background: rgba(16, 185, 129, 0.2);
		}
		
		.crawlify-control-panel {
			position: fixed;
			top: 20px;
			right: 20px;
			background: white;
			border-radius: 12px;
			box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
			padding: 20px;
			min-width: 320px;
			max-width: 400px;
			max-height: 80vh;
			overflow-y: auto;
			pointer-events: auto;
			z-index: 1000000;
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
					status: null,
					statusType: 'success'
				};
			},
			methods: {
				toggleMode() {
					this.mode = this.mode === 'single' ? 'multiple' : 'single';
					this.showStatus('Mode: ' + (this.mode === 'single' ? 'Single Element' : 'Multiple Elements'), 'success');
				},
				
				addCurrentSelection() {
					if (!this.currentFieldName.trim()) {
						this.showStatus('Please enter a field name', 'error');
						return;
					}
					
					if (!this.hoveredElement) {
						this.showStatus('Please hover over an element first', 'error');
						return;
					}
					
					const selector = this.generateSelector(this.hoveredElement);
					const preview = this.getElementPreview(this.hoveredElement);
					
					this.selectedFields.push({
						name: this.currentFieldName,
						selector: selector,
						type: this.currentFieldType,
						attribute: this.currentFieldAttribute,
						multiple: this.mode === 'multiple',
						preview: preview
					});
					
					this.currentFieldName = '';
					this.currentFieldAttribute = '';
					this.showStatus('Field added successfully!', 'success');
					this.saveToWindow();
				},
				
				removeField(index) {
					this.selectedFields.splice(index, 1);
					this.saveToWindow();
				},
				
				generateSelector(element) {
					// Generate optimal CSS selector
					const selectors = [];
					
					// Try ID first
					if (element.id) {
						return '#' + element.id;
					}
					
					// Try unique class
					if (element.className && typeof element.className === 'string') {
						const classes = element.className.split(' ').filter(c => c.trim());
						if (classes.length > 0) {
							const classSelector = '.' + classes.join('.');
							if (document.querySelectorAll(classSelector).length === 1) {
								return classSelector;
							}
						}
					}
					
					// Build path from parent
					let current = element;
					const path = [];
					
					while (current && current !== document.body) {
						let selector = current.tagName.toLowerCase();
						
						if (current.id) {
							selector = '#' + current.id;
							path.unshift(selector);
							break;
						}
						
						if (current.className && typeof current.className === 'string') {
							const classes = current.className.split(' ').filter(c => c.trim());
							if (classes.length > 0) {
								selector += '.' + classes.join('.');
							}
						}
						
						// Add nth-child if needed for uniqueness
						const parent = current.parentElement;
						if (parent) {
							const siblings = Array.from(parent.children).filter(
								child => child.tagName === current.tagName
							);
							if (siblings.length > 1) {
								const index = siblings.indexOf(current) + 1;
								selector += ':nth-child(' + index + ')';
							}
						}
						
						path.unshift(selector);
						current = current.parentElement;
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
					
					// Find the actual target element (not our overlay)
					const elements = document.elementsFromPoint(event.clientX, event.clientY);
					const targetElement = elements.find(el => 
						!el.closest('#crawlify-selector-overlay')
					);
					
					if (targetElement && targetElement !== this.hoveredElement) {
						this.hoveredElement = targetElement;
						this.highlightElement(targetElement);
					}
				},
				
				highlightElement(element) {
					// Remove old highlights
					document.querySelectorAll('.crawlify-highlight').forEach(el => el.remove());
					
					const rect = element.getBoundingClientRect();
					const highlight = document.createElement('div');
					highlight.className = 'crawlify-highlight';
					highlight.style.top = rect.top + window.scrollY + 'px';
					highlight.style.left = rect.left + window.scrollX + 'px';
					highlight.style.width = rect.width + 'px';
					highlight.style.height = rect.height + 'px';
					
					document.body.appendChild(highlight);
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
				
				closeOverlay() {
					// Signal backend that we're done
					window.__crawlifyClosed = true;
					document.getElementById('crawlify-selector-overlay').remove();
					document.querySelectorAll('.crawlify-highlight').forEach(el => el.remove());
					document.removeEventListener('mousemove', this.handleMouseMove);
				}
			},
			mounted() {
				document.addEventListener('mousemove', this.handleMouseMove);
				
				// Make selections available to backend
				window.__crawlifyGetSelections = () => {
					return this.selectedFields;
				};
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
						
						<div class="crawlify-mode-toggle">
							<button 
								class="crawlify-toggle-btn"
								:class="{ active: mode === 'multiple' }"
								@click="toggleMode">
								{{ mode === 'single' ? '📄 Single Element' : '📑 Multiple Elements' }}
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
								:disabled="!currentFieldName.trim()">
								➕ Add Field
							</button>
						</div>
						
						<div class="crawlify-fields" v-if="selectedFields.length > 0">
							<div style="font-size: 14px; font-weight: 600; margin-bottom: 8px; color: #374151;">
								Selected Fields ({{ selectedFields.length }})
							</div>
							<div 
								v-for="(field, index) in selectedFields" 
								:key="index"
								class="crawlify-field-item">
								<button class="crawlify-field-remove" @click="removeField(index)">×</button>
								<div class="crawlify-field-name">
									{{ field.name }}
									<span v-if="field.multiple" style="color: #3b82f6; font-size: 12px;"> (multiple)</span>
								</div>
								<div class="crawlify-field-selector">{{ field.selector }}</div>
								<div v-if="field.preview" class="crawlify-field-preview">
									Preview: "{{ field.preview }}"
								</div>
							</div>
						</div>
						
						<div v-else class="crawlify-empty">
							No fields selected yet
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
