<template>
  <div class="space-y-6">
    <!-- Basic Configuration (Common for all modes) -->
    <div class="bg-white rounded-lg border-2 border-gray-200 p-4 space-y-4">
      <div class="flex items-center justify-between border-b border-gray-300 pb-2">
        <h3 class="text-lg font-bold text-gray-800 flex items-center gap-2">
          <span v-if="props.field.mode === 'key-value-pairs'">🔗</span>
          <span v-else-if="props.field.mode === 'list'">📋</span>
          <span v-else>📄</span>
          {{ getModeTitle(props.field.mode) }}
        </h3>
        <button
          v-if="!props.editMode"
          @click="emit('enableEdit')"
          class="px-3 py-1.5 text-sm bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
        >
          ✏️ Edit
        </button>
        <div v-else class="flex gap-2">
          <button
            @click="handleSaveEdit"
            class="px-3 py-1.5 text-sm bg-green-500 text-white rounded-lg hover:bg-green-600 transition-colors"
          >
            ✓ Save
          </button>
          <button
            @click="emit('cancelEdit')"
            class="px-3 py-1.5 text-sm bg-gray-500 text-white rounded-lg hover:bg-gray-600 transition-colors"
          >
            ✕ Cancel
          </button>
        </div>
      </div>
      
      <!-- Field Name -->
      <div>
        <label class="block text-base font-medium text-gray-700 mb-2">Field Name</label>
        <input
          v-model="editedField.name"
          :disabled="!props.editMode"
          type="text"
          :class="[
            'w-full px-4 py-3 text-base border rounded-lg transition-all',
            props.editMode 
              ? 'border-gray-300 focus:ring-2 focus:ring-blue-500 focus:border-transparent' 
              : 'border-gray-200 bg-gray-50 cursor-not-allowed'
          ]"
          placeholder="Enter field name"
        />
      </div>

      <!-- Mode Badge -->
      <div class="flex items-center gap-2 text-sm">
        <span class="text-gray-600 font-medium">Mode:</span>
        <span 
          :class="[
            'px-2 py-1 rounded text-xs font-semibold',
            props.field.mode === 'key-value-pairs' ? 'bg-purple-100 text-purple-700' :
            props.field.mode === 'list' ? 'bg-blue-100 text-blue-700' :
            'bg-gray-100 text-gray-700'
          ]"
        >
          {{ getModeTitle(props.field.mode) }}
        </span>
      </div>

      <!-- Key-Value Pairs Configuration -->
      <div v-if="props.field.mode === 'key-value-pairs' && editedField.attributes?.extractions" class="space-y-4">
        <div class="flex items-center justify-between mb-2">
          <span class="font-semibold text-purple-900">Extraction Pairs ({{ editedField.attributes.extractions.length }})</span>
          <button
            v-if="props.editMode"
            @click="addNewExtractionPair"
            class="px-3 py-1 text-xs bg-purple-500 text-white rounded hover:bg-purple-600"
          >
            + Add Pair
          </button>
        </div>
        
        <div
          v-for="(extraction, idx) in editedField.attributes.extractions"
          :key="idx"
          class="bg-white rounded-lg border-2 border-purple-200 p-4 space-y-3"
        >
          <div class="flex items-center justify-between mb-2">
            <span class="font-semibold text-gray-800">Extraction Pair {{ idx + 1 }}</span>
            <button
              v-if="props.editMode && editedField.attributes.extractions.length > 1"
              @click="removeExtractionPair(idx)"
              class="text-red-500 hover:text-red-700 text-xs px-2 py-1 bg-red-50 hover:bg-red-100 rounded"
            >
              Remove
            </button>
          </div>
          
          <!-- Key Selector -->
          <div class="bg-green-50 border border-green-200 rounded p-3 space-y-2">
            <div class="text-xs font-semibold text-green-800 mb-1">🔑 Key Selector</div>
            <textarea
              v-if="props.editMode"
              v-model="extraction.key_selector"
              rows="2"
              class="w-full text-xs font-mono border border-green-300 rounded p-2 focus:ring-2 focus:ring-green-500"
            />
            <code v-else class="text-xs text-gray-800 break-all">{{ extraction.key_selector }}</code>
            
            <div class="grid grid-cols-2 gap-2">
              <div>
                <label class="text-xs text-gray-600">Type:</label>
                <select
                  v-if="props.editMode"
                  v-model="extraction.key_type"
                  class="w-full text-xs border border-green-300 rounded p-1"
                >
                  <option value="text">Text</option>
                  <option value="attribute">Attribute</option>
                  <option value="html">HTML</option>
                </select>
                <span v-else class="text-xs text-gray-700 ml-1">{{ extraction.key_type }}</span>
              </div>
              <div v-if="extraction.key_type === 'attribute'">
                <label class="text-xs text-gray-600">Attribute:</label>
                <input
                  v-if="props.editMode"
                  v-model="extraction.key_attribute"
                  type="text"
                  class="w-full text-xs border border-green-300 rounded p-1"
                  placeholder="e.g., href"
                />
                <span v-else class="text-xs text-gray-700 ml-1">{{ extraction.key_attribute }}</span>
              </div>
            </div>
          </div>
          
          <!-- Value Selector -->
          <div class="bg-blue-50 border border-blue-200 rounded p-3 space-y-2">
            <div class="text-xs font-semibold text-blue-800 mb-1">💎 Value Selector</div>
            <textarea
              v-if="props.editMode"
              v-model="extraction.value_selector"
              rows="2"
              class="w-full text-xs font-mono border border-blue-300 rounded p-2 focus:ring-2 focus:ring-blue-500"
            />
            <code v-else class="text-xs text-gray-800 break-all">{{ extraction.value_selector }}</code>
            
            <div class="grid grid-cols-2 gap-2">
              <div>
                <label class="text-xs text-gray-600">Type:</label>
                <select
                  v-if="props.editMode"
                  v-model="extraction.value_type"
                  class="w-full text-xs border border-blue-300 rounded p-1"
                >
                  <option value="text">Text</option>
                  <option value="attribute">Attribute</option>
                  <option value="html">HTML</option>
                </select>
                <span v-else class="text-xs text-gray-700 ml-1">{{ extraction.value_type }}</span>
              </div>
              <div v-if="extraction.value_type === 'attribute'">
                <label class="text-xs text-gray-600">Attribute:</label>
                <input
                  v-if="props.editMode"
                  v-model="extraction.value_attribute"
                  type="text"
                  class="w-full text-xs border border-blue-300 rounded p-1"
                  placeholder="e.g., href"
                />
                <span v-else class="text-xs text-gray-700 ml-1">{{ extraction.value_attribute }}</span>
              </div>
            </div>
          </div>

          <!-- Transform -->
          <div class="bg-gray-50 border border-gray-200 rounded p-2">
            <label class="text-xs font-semibold text-gray-700">Transform:</label>
            <input
              v-if="props.editMode"
              v-model="extraction.transform"
              type="text"
              class="w-full text-xs border border-gray-300 rounded p-1 mt-1"
              placeholder="e.g., trim"
            />
            <span v-else class="text-xs text-gray-700 ml-1">{{ extraction.transform || 'none' }}</span>
          </div>
        </div>
      </div>

      <!-- Regular Mode Selector (Single/List) -->
      <div v-if="props.field.mode !== 'key-value-pairs'" class="space-y-3">
        <!-- CSS Selector with Test Button -->
        <div>
          <div class="flex items-center justify-between mb-2">
            <label class="block text-base font-medium text-gray-700">
              CSS Selector
              <button
                type="button"
                class="ml-1 text-gray-400 hover:text-gray-600 text-sm"
                title="Enter a valid CSS selector to target elements on the page"
              >
                ⓘ
              </button>
            </label>
            <button
              @click="testCurrentSelector"
              class="px-4 py-2 text-sm bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors font-medium shadow-sm flex items-center gap-1"
            >
              <span>🔍</span>
              <span>Test Selector</span>
              <span v-if="props.testResults.length > 0" class="ml-1 px-2 py-0.5 bg-white text-blue-600 rounded-full text-xs font-bold">
                {{ props.testResults.length }}
              </span>
            </button>
          </div>
          <textarea
            v-model="editedField.selector"
            @change="handleFieldUpdate"
            rows="2"
            class="w-full px-4 py-3 text-base border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono transition-all"
            placeholder="e.g., .product-title, #main > div:first-child"
          />
        </div>

        <!-- Extraction Type -->
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block text-base font-medium text-gray-700 mb-2">Extraction Type</label>
            <select
              v-model="editedField.type"
              @change="handleFieldUpdate"
              class="w-full px-4 py-3 text-base border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="text">Text Content</option>
              <option value="attribute">Attribute</option>
              <option value="html">HTML</option>
            </select>
          </div>

          <div v-if="editedField.type === 'attribute'">
            <label class="block text-base font-medium text-gray-700 mb-2">Attribute Name</label>
            <input
              v-model="editedField.attribute"
              @change="handleFieldUpdate"
              type="text"
              placeholder="e.g., href, src"
              class="w-full px-4 py-3 text-base border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
        </div>
      </div>


      <!-- Info Note based on mode -->
      <div 
        :class="[
          'text-sm rounded p-3',
          props.field.mode === 'key-value-pairs' 
            ? 'text-purple-700 bg-purple-100 border border-purple-300'
            : props.field.mode === 'list'
              ? 'text-blue-700 bg-blue-100 border border-blue-300'
              : 'text-gray-700 bg-gray-100 border border-gray-300'
        ]"
      >
        <span v-if="props.field.mode === 'key-value-pairs'">
          💡 <strong>Key-Value Pairs:</strong> Extracts data from separate lists and pairs them by index position.
          Output: <code class="bg-white px-1 rounded">[{"key": "...", "value": "..."}]</code>
        </span>
        <span v-else-if="props.field.mode === 'list'">
          💡 <strong>Multiple Values:</strong> Extracts an array of values from all matching elements.
          Output: <code class="bg-white px-1 rounded">["value1", "value2", "value3"]</code>
        </span>
        <span v-else>
          💡 <strong>Single Value:</strong> Extracts one value from the first matching element.
          Output: <code class="bg-white px-1 rounded">"single value"</code>
        </span>
      </div>
    </div>

    <!-- Section 2: Transformations (Collapsible) - Now available for all modes -->
    <div class="bg-gradient-to-br from-purple-50 to-indigo-50 rounded-lg border-2 border-purple-200">
      <button
        @click="showTransformations = !showTransformations"
        class="w-full px-4 py-3 flex items-center justify-between hover:bg-purple-100 transition-colors rounded-t-lg"
      >
        <h3 class="text-lg font-bold text-gray-800 flex items-center gap-2">
          <span>{{ showTransformations ? '▼' : '▶' }}</span>
          <span>✨ Transformations</span>
          <span class="text-sm font-normal text-gray-600">(Optional)</span>
          <span v-if="activeTransformations.length > 0" class="ml-2 px-2 py-1 bg-purple-600 text-white rounded-full text-xs font-bold">
            {{ activeTransformations.length }} active
          </span>
        </h3>
        <span class="text-sm text-gray-600">{{ showTransformations ? 'Click to collapse' : 'Click to expand' }}</span>
      </button>
      
      <div v-if="showTransformations" class="p-4 space-y-4 border-t border-purple-200">
        <!-- Quick Text Operations -->
        <div>
          <label class="block text-sm font-semibold text-gray-700 mb-2 flex items-center gap-2">
            <span>📝</span> Quick Text Operations
          </label>
          <div class="flex flex-wrap gap-2">
            <button
              @click="applyTransformation('trim')"
              class="px-3 py-1.5 text-sm bg-white hover:bg-gray-100 border border-gray-300 rounded-full transition-colors shadow-sm"
              title="Remove leading/trailing whitespace"
            >
              Trim
            </button>
            <button
              @click="applyTransformation('lowercase')"
              class="px-3 py-1.5 text-sm bg-white hover:bg-gray-100 border border-gray-300 rounded-full transition-colors shadow-sm"
              title="Convert to lowercase"
            >
              Lowercase
            </button>
            <button
              @click="applyTransformation('uppercase')"
              class="px-3 py-1.5 text-sm bg-white hover:bg-gray-100 border border-gray-300 rounded-full transition-colors shadow-sm"
              title="Convert to uppercase"
            >
              Uppercase
            </button>
            <button
              @click="applyTransformation('slugify')"
              class="px-3 py-1.5 text-sm bg-white hover:bg-gray-100 border border-gray-300 rounded-full transition-colors shadow-sm"
              title="Convert to URL-friendly slug"
            >
              Slugify
            </button>
            <button
              @click="applyTransformation('remove-whitespace')"
              class="px-3 py-1.5 text-sm bg-white hover:bg-gray-100 border border-gray-300 rounded-full transition-colors shadow-sm"
              title="Remove all whitespace"
            >
              Remove Spaces
            </button>
          </div>
        </div>

        <!-- Data & Content Operations -->
        <div>
          <label class="block text-sm font-semibold text-gray-700 mb-2 flex items-center gap-2">
            <span>🔢</span> Data & Content Operations
          </label>
          <div class="flex flex-wrap gap-2">
            <button
              @click="applyTransformation('number')"
              class="px-3 py-1.5 text-sm bg-white hover:bg-gray-100 border border-gray-300 rounded-full transition-colors shadow-sm"
              title="Extract numbers only"
            >
              Extract Number
            </button>
            <button
              @click="applyTransformation('url')"
              class="px-3 py-1.5 text-sm bg-white hover:bg-gray-100 border border-gray-300 rounded-full transition-colors shadow-sm"
              title="Ensure absolute URL"
            >
              Absolute URL
            </button>
            <button
              @click="applyTransformation('decode-html')"
              class="px-3 py-1.5 text-sm bg-white hover:bg-gray-100 border border-gray-300 rounded-full transition-colors shadow-sm"
              title="Decode HTML entities"
            >
              Decode HTML
            </button>
            <button
              @click="applyTransformation('remove-html')"
              class="px-3 py-1.5 text-sm bg-white hover:bg-gray-100 border border-gray-300 rounded-full transition-colors shadow-sm"
              title="Strip HTML tags"
            >
              Strip HTML
            </button>
            <button
              @click="applyTransformation('parse-json')"
              class="px-3 py-1.5 text-sm bg-white hover:bg-gray-100 border border-gray-300 rounded-full transition-colors shadow-sm"
              title="Parse JSON string"
            >
              Parse JSON
            </button>
            <button
              @click="applyTransformation('format-date')"
              class="px-3 py-1.5 text-sm bg-white hover:bg-gray-100 border border-gray-300 rounded-full transition-colors shadow-sm"
              title="Format as ISO date"
            >
              Format Date
            </button>
          </div>
        </div>

        <!-- Regex Transformation -->
        <div class="mt-3 pt-3 border-t border-purple-200">
          <label class="block text-sm font-semibold text-gray-700 mb-2 flex items-center gap-2">
            <span>🔧</span> Custom Regex
            <button
              type="button"
              class="ml-1 text-gray-400 hover:text-gray-600 text-sm"
              title="Use regex patterns to match and replace text. Example: pattern='\d+' will extract numbers"
            >
              ⓘ
            </button>
          </label>
          <div class="space-y-2 bg-white p-3 rounded-lg border border-gray-300">
            <input
              v-model="customRegexPattern"
              type="text"
              placeholder="Regex pattern (e.g., \d+ to match numbers)"
              class="w-full px-4 py-2.5 text-sm border border-gray-300 rounded font-mono focus:ring-2 focus:ring-blue-500"
            />
            <input
              v-model="customRegexReplace"
              type="text"
              placeholder="Replacement (optional, e.g., $1 for first group)"
              class="w-full px-4 py-2.5 text-sm border border-gray-300 rounded font-mono focus:ring-2 focus:ring-blue-500"
            />
            <button
              @click="applyCustomRegex"
              class="w-full px-4 py-2.5 text-sm bg-purple-500 text-white rounded hover:bg-purple-600 transition-colors font-medium"
            >
              Apply Regex
            </button>
          </div>
        </div>

        <!-- Custom JS Code Button -->
        <div class="mt-3 pt-3 border-t border-purple-200">
          <button
            @click="showJsModal = true"
            class="w-full px-4 py-3 text-base bg-gradient-to-r from-purple-500 to-indigo-600 text-white rounded-lg hover:from-purple-600 hover:to-indigo-700 transition-colors font-medium shadow-md flex items-center justify-center gap-2"
          >
            <span>💻</span>
            <span>Custom JavaScript Code</span>
            <span class="text-xs">(Expert Mode)</span>
          </button>
        </div>

        <!-- Active Transformations -->
        <div v-if="activeTransformations.length > 0" class="mt-4 pt-4 border-t border-purple-200">
          <label class="block text-sm font-semibold text-gray-700 mb-2 flex items-center gap-2">
            <span>📋</span> Active Transformation Chain
          </label>
          <div class="space-y-2">
            <div
              v-for="(trans, idx) in activeTransformations"
              :key="idx"
              class="flex items-center justify-between bg-white border-2 border-purple-300 px-4 py-2.5 rounded-lg text-sm shadow-sm"
            >
              <div class="flex items-center gap-2">
                <span class="text-gray-500 font-bold">{{ idx + 1 }}.</span>
                <span class="font-mono text-purple-900">{{ trans.startsWith('js:') ? '💻 Custom JS' : trans.startsWith('regex:') ? '🔧 Regex' : trans }}</span>
              </div>
              <button
                @click="removeTransformation(idx)"
                class="text-red-600 hover:text-red-800 text-lg font-bold"
                title="Remove this transformation"
              >
                ✕
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Section 3: Live Preview (Always visible, directly after transformations) - Now for all modes -->
    <div class="bg-gradient-to-br from-green-50 to-emerald-50 rounded-lg border-2 border-green-300 p-5 shadow-md">
      <div class="flex items-center justify-between mb-3">
        <h3 class="text-lg font-bold text-gray-800 flex items-center gap-2">
          <span>👁️</span> Live Preview
        </h3>
        <button
          v-if="livePreviewOriginal && activeTransformations.length > 0"
          @click="comparisonView = comparisonView === 'both' ? 'transformed' : 'both'"
          class="text-sm px-3 py-1.5 bg-white border border-green-400 text-green-700 rounded-lg hover:bg-green-100 transition-colors font-medium shadow-sm"
        >
          {{ comparisonView === 'both' ? '📊 Side by Side' : '🔄 Show Comparison' }}
        </button>
      </div>

      <div v-if="livePreviewOriginal" class="space-y-3">
        <!-- Comparison View (Side by Side) -->
        <div v-if="comparisonView === 'both' && activeTransformations.length > 0" class="grid grid-cols-2 gap-3">
          <!-- Original -->
          <div class="bg-white border-2 border-gray-300 p-4 rounded-lg">
            <div class="text-sm text-gray-600 mb-2 font-semibold flex items-center gap-1">
              📝 Original
            </div>
            <div class="text-base text-gray-900 break-words">
              {{ livePreviewOriginal }}
            </div>
          </div>

          <!-- Transformed -->
          <div class="bg-white border-2 border-green-400 p-4 rounded-lg">
            <div class="text-sm text-green-700 mb-2 font-semibold flex items-center gap-1">
              ✨ Transformed
            </div>
            <div class="text-base text-gray-900 break-words font-medium">
              {{ livePreview }}
            </div>
          </div>
        </div>

        <!-- Single View (Transformed Only) -->
        <div v-else class="bg-white border-2 border-green-400 p-4 rounded-lg">
          <div class="text-sm text-green-700 mb-2 font-semibold">
            {{ activeTransformations.length > 0 ? '✨ Transformed Value:' : '📝 Extracted Value:' }}
          </div>
          <div class="text-base text-gray-900 break-words font-medium">
            {{ livePreview }}
          </div>
        </div>

        <!-- Transformation Stats -->
        <div v-if="activeTransformations.length > 0" class="text-sm text-gray-700 bg-blue-50 border border-blue-300 rounded-lg p-3">
          <strong>📊 Impact:</strong>
          Length: <span class="font-mono">{{ livePreviewOriginal.length }}</span> → <span class="font-mono">{{ livePreview.length }}</span> chars
          <span v-if="livePreviewOriginal.length !== livePreview.length" class="ml-2 font-semibold" :class="livePreview.length > livePreviewOriginal.length ? 'text-green-600' : 'text-orange-600'">
            ({{ livePreview.length > livePreviewOriginal.length ? '+' : '' }}{{ livePreview.length - livePreviewOriginal.length }})
          </span>
        </div>
      </div>

      <div v-else class="bg-white border-2 border-gray-300 p-4 rounded-lg text-base text-gray-500 italic text-center">
        💡 Test your selector to see a live preview of extracted data
      </div>
    </div>

    <!-- Section 4: Test Results (Collapsible) - Now for all modes -->
    <div class="bg-gradient-to-br from-blue-50 to-cyan-50 rounded-lg border-2 border-blue-200">
      <button
        @click="showTestResults = !showTestResults"
        class="w-full px-4 py-3 flex items-center justify-between hover:bg-blue-100 transition-colors rounded-t-lg"
      >
        <h3 class="text-lg font-bold text-gray-800 flex items-center gap-2">
          <span>{{ showTestResults ? '▼' : '▶' }}</span>
          <span>🔍 Test Results</span>
          <span v-if="props.testResults.length > 0" class="ml-2 px-2.5 py-1 bg-blue-600 text-white rounded-full text-sm font-bold">
            {{ props.testResults.length }} {{ props.testResults.length === 1 ? 'match' : 'matches' }}
          </span>
        </h3>
        <span class="text-sm text-gray-600">{{ showTestResults ? 'Click to collapse' : 'Click to expand' }}</span>
      </button>

      <!-- Summary (Always Visible) -->
      <div v-if="props.testResults.length > 0 && !showTestResults" class="px-4 py-3 bg-white border-t border-blue-200">
        <div class="text-sm text-gray-700">
          <span class="font-semibold">✅ {{ props.testResults.length }} elements found</span>
          <span class="text-gray-500 ml-2">• Click to expand and view all results</span>
        </div>
        <div class="mt-2 text-xs text-gray-600 bg-blue-50 p-2 rounded">
          💡 Hover over results to preview, click to highlight on page
        </div>
      </div>

      <!-- Detailed Results -->
      <div v-if="showTestResults && props.testResults.length > 0" class="border-t border-blue-200 bg-white">
        <!-- Results Header -->
        <div class="bg-gradient-to-r from-blue-100 to-cyan-100 px-4 py-3 border-b-2 border-blue-200">
          <div class="flex items-center justify-between">
            <span class="text-sm font-semibold text-blue-900">
              📊 Showing all {{ props.testResults.length }} matching element(s)
            </span>
            <span class="text-sm text-blue-700">
              {{ props.testResults.length > 5 ? '⬇️ Scroll panel to view all' : 'All results visible' }}
            </span>
          </div>
        </div>
          
        <!-- Results List -->
        <div class="p-3 space-y-3">
          <div
            v-for="result in props.testResults"
            :key="result.index"
            @click="emit('scrollToResult', result)"
            class="bg-white border-2 border-purple-200 p-4 rounded-lg hover:border-purple-500 hover:shadow-md transition-all cursor-pointer hover:scale-[1.02] active:scale-[0.98]"
            title="Click to scroll and highlight this element on the page"
          >
            <div class="flex items-center justify-between mb-3">
              <span class="text-base font-semibold text-purple-900">🔹 Element #{{ result.index + 1 }}</span>
              <span class="text-sm bg-purple-600 text-white px-2.5 py-1 rounded font-medium">
                {{ result.index + 1 }}/{{ props.testResults.length }}
              </span>
            </div>

            <!-- Show comparison if transformations are active -->
            <div v-if="activeTransformations.length > 0" class="space-y-2">
              <div class="text-gray-800 break-words bg-gray-100 p-3 rounded border border-gray-300">
                <div class="text-sm text-gray-500 mb-1 font-medium">📝 Original:</div>
                <div class="text-base">{{ result.value || '(empty content)' }}</div>
              </div>
              <div class="text-gray-800 break-words bg-green-50 p-3 rounded border border-green-300">
                <div class="text-sm text-green-700 mb-1 font-medium">✨ Transformed:</div>
                <div class="text-base font-medium">{{ applyTransformationsToValue(result.value || '') }}</div>
              </div>
            </div>

            <!-- Show only value if no transformations -->
            <div v-else class="text-base text-gray-800 break-words bg-gray-50 p-3 rounded border border-gray-200">
              {{ result.value || '(empty content)' }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- JavaScript Code Modal -->
    <div
      v-if="showJsModal"
      @click="showJsModal = false"
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[1000001] p-4"
    >
      <div
        @click.stop
        class="bg-white rounded-xl shadow-2xl max-w-3xl w-full max-h-[90vh] overflow-y-auto"
      >
        <!-- Modal Header -->
        <div class="bg-gradient-to-r from-purple-500 to-indigo-600 text-white px-6 py-4 flex items-center justify-between rounded-t-xl">
          <h3 class="text-xl font-bold flex items-center gap-2">
            <span>💻</span>
            <span>Custom JavaScript Code Editor</span>
          </h3>
          <button
            @click="showJsModal = false"
            class="text-white hover:bg-white hover:bg-opacity-20 rounded-full p-2 transition-colors"
          >
            <span class="text-2xl">×</span>
          </button>
        </div>

        <!-- Modal Content -->
        <div class="p-6 space-y-4">
          <div class="text-sm text-gray-700 bg-yellow-50 border-l-4 border-yellow-400 p-4 rounded">
            <strong>💡 How it works:</strong>
            <ul class="mt-2 space-y-1 ml-4 list-disc">
              <li>Use <code class="bg-yellow-100 px-2 py-0.5 rounded font-mono text-xs">value</code> variable to access the extracted data</li>
              <li>Write your transformation logic and <strong>return</strong> the result</li>
              <li>Example: <code class="bg-yellow-100 px-2 py-0.5 rounded font-mono text-xs">return value.split(',')[0]</code></li>
            </ul>
          </div>

          <div>
            <label class="block text-base font-semibold text-gray-700 mb-2">Your Code:</label>
            <textarea
              v-model="customJsCode"
              @input="validateAndApplyJsCode"
              rows="10"
              placeholder="// Your JavaScript code here&#10;// Access extracted value with 'value' variable&#10;// Example:&#10;return value.toUpperCase().replace(/[^a-z0-9]/gi, '-')"
              class="w-full px-4 py-3 border-2 border-gray-300 rounded-lg font-mono text-sm focus:ring-2 focus:ring-purple-500 focus:border-purple-500 bg-gray-900 text-green-400"
              spellcheck="false"
            />
          </div>

          <div v-if="jsCodeError" class="text-sm text-red-700 bg-red-50 border-l-4 border-red-500 p-4 rounded">
            <strong>⚠️ Error:</strong> {{ jsCodeError }}
          </div>

          <div v-if="customJsCode && !jsCodeError" class="text-sm text-green-700 bg-green-50 border-l-4 border-green-500 p-4 rounded">
            <strong>✅ Success:</strong> Your code is valid and will be applied to all extracted values
          </div>

          <!-- Modal Actions -->
          <div class="flex gap-3 pt-4 border-t">
            <button
              @click="applyJsCode(); showJsModal = false"
              :disabled="!customJsCode || !!jsCodeError"
              class="flex-1 px-6 py-3 text-base bg-gradient-to-r from-purple-500 to-indigo-600 text-white rounded-lg hover:from-purple-600 hover:to-indigo-700 transition-colors font-medium shadow-md disabled:opacity-50 disabled:cursor-not-allowed"
            >
              💾 Apply & Close
            </button>
            <button
              @click="clearJsCode(); showJsModal = false"
              class="px-6 py-3 text-base bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-colors font-medium"
            >
              Clear & Close
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { SelectedField, TestResult } from '../types'

interface Props {
  field: SelectedField
  tab: 'preview' | 'edit'
  editMode: boolean
  testResults: TestResult[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'switchTab': [tab: 'preview' | 'edit']
  'enableEdit': []
  'saveEdit': [field: Partial<SelectedField>]
  'cancelEdit': []
  'testSelector': [field: SelectedField]
  'scrollToResult': [result: TestResult]
}>()

const editedField = ref({
  name: props.field.name,
  selector: props.field.selector,
  type: props.field.type,
  attribute: props.field.attribute,
  attributes: props.field.attributes
})

const activeTransformations = ref<string[]>([])
const customRegexPattern = ref('')
const customRegexReplace = ref('')
const customJsCode = ref('')
const jsCodeError = ref('')
const showJsEditor = ref(false)
const comparisonView = ref<'transformed' | 'both'>('transformed')
const showTransformations = ref(false)
const showTestResults = ref(false)
const showJsModal = ref(false)

watch(() => props.field, (newField) => {
  editedField.value = {
    name: newField.name,
    selector: newField.selector,
    type: newField.type,
    attribute: newField.attribute,
    attributes: newField.attributes
  }
  activeTransformations.value = []
  customRegexPattern.value = ''
  customRegexReplace.value = ''
  customJsCode.value = ''
  jsCodeError.value = ''
  showJsEditor.value = false
  comparisonView.value = 'transformed'
  showTransformations.value = false
  showTestResults.value = false
  showJsModal.value = false
}, { immediate: true })

// Original extracted value (before transformations)
const livePreviewOriginal = computed(() => {
  try {
    // Handle key-value pairs mode
    if (props.field.mode === 'key-value-pairs' && editedField.value.attributes?.extractions) {
      const pairs = []
      for (const extraction of editedField.value.attributes.extractions) {
        // Skip if selectors are empty
        if (!extraction.key_selector || !extraction.value_selector) continue
        
        const keyElements = document.querySelectorAll(extraction.key_selector)
        const valueElements = document.querySelectorAll(extraction.value_selector)
        
        const minLength = Math.min(keyElements.length, valueElements.length)
        for (let i = 0; i < Math.min(minLength, 3); i++) { // Show first 3 pairs
          const keyEl = keyElements[i]
          const valueEl = valueElements[i]
          
          let key = ''
          let value = ''
          
          // Extract key
          if (extraction.key_type === 'text') {
            key = keyEl.textContent?.trim() || ''
          } else if (extraction.key_type === 'attribute' && extraction.key_attribute) {
            key = keyEl.getAttribute(extraction.key_attribute) || ''
          } else if (extraction.key_type === 'html') {
            key = keyEl.innerHTML
          }
          
          // Extract value
          if (extraction.value_type === 'text') {
            value = valueEl.textContent?.trim() || ''
          } else if (extraction.value_type === 'attribute' && extraction.value_attribute) {
            value = valueEl.getAttribute(extraction.value_attribute) || ''
          } else if (extraction.value_type === 'html') {
            value = valueEl.innerHTML
          }
          
          pairs.push({ key, value })
        }
      }
      
      return JSON.stringify(pairs, null, 2)
    }
    
    // Handle regular mode (single/list)
    if (!editedField.value.selector) return ''
    
    const elements = document.querySelectorAll(editedField.value.selector)
    if (elements.length === 0) return ''
    
    const firstEl = elements[0]
    let value = ''
    
    // Extract value based on type
    switch (editedField.value.type) {
      case 'text':
        value = firstEl.textContent?.trim() || ''
        break
      case 'attribute':
        value = editedField.value.attribute ? firstEl.getAttribute(editedField.value.attribute) || '' : ''
        break
      case 'html':
        value = firstEl.innerHTML
        break
    }
    
    return value
  } catch (error) {
    console.error('Error in livePreviewOriginal:', error)
    return ''
  }
})

// Live preview of extracted and transformed value
const livePreview = computed(() => {
  const original = livePreviewOriginal.value
  if (!original) return ''
  
  // Apply transformations
  return applyTransformationsToValue(original)
})

const applyTransformationsToValue = (value: string): string => {
  let result = value
  
  for (const trans of activeTransformations.value) {
    try {
      if (trans === 'trim') {
        result = result.trim()
      } else if (trans === 'lowercase') {
        result = result.toLowerCase()
      } else if (trans === 'uppercase') {
        result = result.toUpperCase()
      } else if (trans === 'number') {
        const match = result.match(/\d+\.?\d*/)
        result = match ? match[0] : ''
      } else if (trans === 'url') {
        if (result && !result.startsWith('http')) {
          result = new URL(result, window.location.href).href
        }
      } else if (trans === 'remove-whitespace') {
        result = result.replace(/\s+/g, '')
      } else if (trans === 'remove-html') {
        result = result.replace(/<[^>]*>/g, '')
      } else if (trans === 'parse-json') {
        try {
          const parsed = JSON.parse(result)
          result = typeof parsed === 'object' ? JSON.stringify(parsed, null, 2) : String(parsed)
        } catch {
          result = 'Invalid JSON'
        }
      } else if (trans === 'format-date') {
        const date = new Date(result)
        if (!isNaN(date.getTime())) {
          result = date.toISOString()
        }
      } else if (trans === 'decode-html') {
        const textarea = document.createElement('textarea')
        textarea.innerHTML = result
        result = textarea.value
      } else if (trans === 'slugify') {
        result = result
          .toLowerCase()
          .trim()
          .replace(/[^\w\s-]/g, '')
          .replace(/[\s_-]+/g, '-')
          .replace(/^-+|-+$/g, '')
      } else if (trans.startsWith('regex:')) {
        // Custom regex transformation
        const parts = trans.substring(6).split('|||')
        const pattern = parts[0]
        const replacement = parts[1] || ''
        const regex = new RegExp(pattern, 'g')
        if (replacement) {
          result = result.replace(regex, replacement)
        } else {
          const match = result.match(regex)
          result = match ? match[0] : ''
        }
      } else if (trans.startsWith('js:')) {
        // Custom JavaScript code transformation
        const code = trans.substring(3)
        try {
          const func = new Function('value', code)
          const transformed = func(result)
          result = transformed !== undefined ? String(transformed) : result
        } catch (e) {
          console.error('JS transformation error:', e)
          result = `[JS Error: ${(e as Error).message}]`
        }
      }
    } catch (e) {
      console.error(`Transformation error for ${trans}:`, e)
    }
  }
  
  return result
}

const applyTransformation = (type: string) => {
  if (!activeTransformations.value.includes(type)) {
    activeTransformations.value.push(type)
  }
}

const applyCustomRegex = () => {
  if (!customRegexPattern.value) return
  
  const transKey = `regex:${customRegexPattern.value}|||${customRegexReplace.value}`
  if (!activeTransformations.value.includes(transKey)) {
    activeTransformations.value.push(transKey)
  }
  customRegexPattern.value = ''
  customRegexReplace.value = ''
}

const removeTransformation = (index: number) => {
  activeTransformations.value.splice(index, 1)
}

const validateAndApplyJsCode = () => {
  if (!customJsCode.value.trim()) {
    jsCodeError.value = ''
    return
  }

  try {
    // Validate the code by trying to create a function
    const testFunc = new Function('value', customJsCode.value)
    // Test with sample data
    testFunc('test')
    jsCodeError.value = ''
  } catch (e) {
    jsCodeError.value = (e as Error).message
  }
}

const applyJsCode = () => {
  if (!customJsCode.value.trim()) return
  
  // Validate first
  try {
    const testFunc = new Function('value', customJsCode.value)
    testFunc('test')
    
    // If valid, add to transformations
    const jsKey = `js:${customJsCode.value}`
    
    // Remove any existing JS transformation
    activeTransformations.value = activeTransformations.value.filter(t => !t.startsWith('js:'))
    
    // Add new JS transformation
    activeTransformations.value.push(jsKey)
    
    jsCodeError.value = ''
  } catch (e) {
    jsCodeError.value = (e as Error).message
  }
}

const clearJsCode = () => {
  customJsCode.value = ''
  jsCodeError.value = ''
  // Remove JS transformation from active list
  activeTransformations.value = activeTransformations.value.filter(t => !t.startsWith('js:'))
}

const testCurrentSelector = () => {
  // Pass the complete field object so test results can extract based on type/attribute
  const fieldToTest: SelectedField = {
    id: props.field.id,
    name: editedField.value.name,
    selector: editedField.value.selector,
    type: editedField.value.type,
    attribute: editedField.value.attribute,
    timestamp: props.field.timestamp,
    sampleValue: props.field.sampleValue,
    matchCount: props.field.matchCount
  }
  emit('testSelector', fieldToTest)
}

const handleFieldUpdate = () => {
  // Auto-save changes
  try {
    const elements = document.querySelectorAll(editedField.value.selector)
    if (elements.length > 0) {
      const firstEl = elements[0]
      let sampleValue = ''
      
      switch (editedField.value.type) {
        case 'text':
          sampleValue = firstEl.textContent?.trim() || ''
          break
        case 'attribute':
          sampleValue = editedField.value.attribute ? firstEl.getAttribute(editedField.value.attribute) || '' : ''
          break
        case 'html':
          sampleValue = firstEl.innerHTML
          break
      }
      
      // Apply transformations to sample value
      sampleValue = applyTransformationsToValue(sampleValue)
      
      emit('saveEdit', {
        ...editedField.value,
        sampleValue,
        matchCount: elements.length,
        transformations: activeTransformations.value.length > 0 ? activeTransformations.value : undefined
      })
    } else {
      emit('saveEdit', editedField.value)
    }
  } catch (error) {
    console.error('Error updating field:', error)
    emit('saveEdit', editedField.value)
  }
}

const addNewExtractionPair = () => {
  if (!editedField.value.attributes) {
    editedField.value.attributes = { extractions: [] }
  }
  editedField.value.attributes.extractions.push({
    key_selector: '',
    value_selector: '',
    key_type: 'text',
    value_type: 'text',
    transform: 'trim'
  })
}

const removeExtractionPair = (index: number) => {
  if (editedField.value.attributes?.extractions) {
    editedField.value.attributes.extractions.splice(index, 1)
  }
}

const handleSaveEdit = () => {
  if (props.field.mode === 'key-value-pairs') {
    emit('saveEdit', {
      name: editedField.value.name,
      attributes: editedField.value.attributes
    })
  } else {
    handleFieldUpdate()
  }
}

const getModeTitle = (mode?: string) => {
  switch (mode) {
    case 'key-value-pairs':
      return 'Key-Value Pair Extraction'
    case 'list':
      return 'Multiple Values (Array)'
    case 'single':
    default:
      return 'Single Value Extraction'
  }
}

</script>
