<template>
  <div class="space-y-4">
    <!-- Field Name (Editable) -->
    <div>
      <label class="block text-sm font-medium text-gray-700 mb-1">Field Name</label>
      <input
        v-model="editedField.name"
        @change="handleFieldUpdate"
        type="text"
        class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
        placeholder="Enter field name"
      />
    </div>

    <!-- CSS Selector (Editable) -->
    <div>
      <label class="block text-sm font-medium text-gray-700 mb-1">
        CSS Selector
        <button
          @click="testCurrentSelector"
          class="ml-2 text-xs text-blue-600 hover:text-blue-700 underline"
        >
          Test
        </button>
      </label>
      <textarea
        v-model="editedField.selector"
        @change="handleFieldUpdate"
        rows="2"
        class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono text-sm transition-all"
        placeholder="Enter CSS selector"
      />
    </div>

    <!-- Extraction Type -->
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">Extraction Type</label>
        <select
          v-model="editedField.type"
          @change="handleFieldUpdate"
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
        >
          <option value="text">Text Content</option>
          <option value="attribute">Attribute</option>
          <option value="html">HTML</option>
        </select>
      </div>

      <div v-if="editedField.type === 'attribute'">
        <label class="block text-sm font-medium text-gray-700 mb-1">Attribute Name</label>
        <input
          v-model="editedField.attribute"
          @change="handleFieldUpdate"
          type="text"
          placeholder="e.g., href, src"
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
        />
      </div>
    </div>

    <!-- Transformations Section -->
    <div class="border-t pt-4">
      <label class="block text-sm font-medium text-gray-700 mb-2">
        ✨ Transformations (Optional)
      </label>
      
      <!-- Predefined Transformations -->
      <div class="space-y-2">
        <label class="block text-xs text-gray-600 mb-1">Quick Transformations</label>
        <div class="flex flex-wrap gap-2">
          <button
            @click="applyTransformation('trim')"
            class="px-3 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded-full transition-colors"
            title="Remove leading/trailing whitespace"
          >
            Trim
          </button>
          <button
            @click="applyTransformation('lowercase')"
            class="px-3 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded-full transition-colors"
            title="Convert to lowercase"
          >
            Lowercase
          </button>
          <button
            @click="applyTransformation('uppercase')"
            class="px-3 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded-full transition-colors"
            title="Convert to uppercase"
          >
            Uppercase
          </button>
          <button
            @click="applyTransformation('number')"
            class="px-3 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded-full transition-colors"
            title="Extract numbers only"
          >
            Extract Number
          </button>
          <button
            @click="applyTransformation('url')"
            class="px-3 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded-full transition-colors"
            title="Ensure absolute URL"
          >
            Absolute URL
          </button>
        </div>
      </div>

      <!-- More Quick Transformations -->
      <div class="mt-3">
        <label class="block text-xs text-gray-600 mb-1">More Transformations</label>
        <div class="flex flex-wrap gap-2">
          <button
            @click="applyTransformation('remove-whitespace')"
            class="px-3 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded-full transition-colors"
            title="Remove all whitespace"
          >
            Remove Spaces
          </button>
          <button
            @click="applyTransformation('remove-html')"
            class="px-3 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded-full transition-colors"
            title="Strip HTML tags"
          >
            Strip HTML
          </button>
          <button
            @click="applyTransformation('parse-json')"
            class="px-3 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded-full transition-colors"
            title="Parse JSON string"
          >
            Parse JSON
          </button>
          <button
            @click="applyTransformation('format-date')"
            class="px-3 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded-full transition-colors"
            title="Format as ISO date"
          >
            Format Date
          </button>
          <button
            @click="applyTransformation('decode-html')"
            class="px-3 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded-full transition-colors"
            title="Decode HTML entities"
          >
            Decode HTML
          </button>
          <button
            @click="applyTransformation('slugify')"
            class="px-3 py-1 text-xs bg-gray-100 hover:bg-gray-200 rounded-full transition-colors"
            title="Convert to URL-friendly slug"
          >
            Slugify
          </button>
        </div>
      </div>

      <!-- Custom Regex Transformation -->
      <div class="mt-3">
        <label class="block text-xs text-gray-600 mb-1">Custom Regex (Advanced)</label>
        <div class="space-y-2">
          <input
            v-model="customRegexPattern"
            type="text"
            placeholder="Regex pattern (e.g., \d+)"
            class="w-full px-3 py-1.5 border border-gray-300 rounded text-xs font-mono focus:ring-2 focus:ring-blue-500"
          />
          <input
            v-model="customRegexReplace"
            type="text"
            placeholder="Replacement (optional, e.g., $1)"
            class="w-full px-3 py-1.5 border border-gray-300 rounded text-xs font-mono focus:ring-2 focus:ring-blue-500"
          />
          <button
            @click="applyCustomRegex"
            class="w-full px-3 py-1.5 text-xs bg-purple-500 text-white rounded hover:bg-purple-600 transition-colors"
          >
            Apply Regex
          </button>
        </div>
      </div>

      <!-- Custom JavaScript Code -->
      <div class="mt-4 border-t pt-3">
        <div class="flex items-center justify-between mb-2">
          <label class="block text-xs text-gray-600">
            💻 Custom JavaScript Code (Expert Mode)
          </label>
          <button
            @click="showJsEditor = !showJsEditor"
            class="text-xs text-blue-600 hover:text-blue-700 underline"
          >
            {{ showJsEditor ? 'Hide' : 'Show' }} Editor
          </button>
        </div>
        
        <div v-if="showJsEditor" class="space-y-2">
          <div class="text-xs text-gray-500 bg-yellow-50 border border-yellow-200 rounded p-2 mb-2">
            <strong>💡 Tip:</strong> Use <code class="bg-yellow-100 px-1 rounded">value</code> variable to access extracted data. 
            Return the transformed result. Example: <code class="bg-yellow-100 px-1 rounded">return value.split(',')[0]</code>
          </div>
          
          <textarea
            v-model="customJsCode"
            @input="validateAndApplyJsCode"
            rows="6"
            placeholder="// Your JavaScript code here&#10;// Access extracted value with 'value' variable&#10;// Example:&#10;return value.toUpperCase().replace(/[^a-z0-9]/gi, '-')"
            class="w-full px-3 py-2 border border-gray-300 rounded font-mono text-xs focus:ring-2 focus:ring-blue-500 bg-gray-900 text-green-400"
            spellcheck="false"
          />
          
          <div v-if="jsCodeError" class="text-xs text-red-600 bg-red-50 border border-red-200 rounded p-2">
            ⚠️ <strong>Error:</strong> {{ jsCodeError }}
          </div>
          
          <div v-if="customJsCode && !jsCodeError" class="text-xs text-green-600 bg-green-50 border border-green-200 rounded p-2">
            ✅ Code is valid and will be applied to all extracted values
          </div>
          
          <div class="flex gap-2">
            <button
              @click="applyJsCode"
              class="flex-1 px-3 py-1.5 text-xs bg-gradient-to-r from-purple-500 to-indigo-600 text-white rounded hover:from-purple-600 hover:to-indigo-700 transition-colors font-medium"
            >
              💾 Apply JS Code
            </button>
            <button
              @click="clearJsCode"
              class="px-3 py-1.5 text-xs bg-gray-200 text-gray-700 rounded hover:bg-gray-300 transition-colors"
            >
              Clear
            </button>
          </div>
        </div>
      </div>

      <!-- Active Transformations -->
      <div v-if="activeTransformations.length > 0" class="mt-3">
        <label class="block text-xs text-gray-600 mb-1">Active Transformations</label>
        <div class="space-y-1">
          <div
            v-for="(trans, idx) in activeTransformations"
            :key="idx"
            class="flex items-center justify-between bg-blue-50 px-2 py-1 rounded text-xs"
          >
            <span class="font-mono">{{ trans }}</span>
            <button
              @click="removeTransformation(idx)"
              class="text-red-600 hover:text-red-700"
              title="Remove"
            >
              ✕
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Live Preview Section with Comparison -->
    <div class="border-t pt-4">
      <div class="flex items-center justify-between mb-2">
        <label class="block text-sm font-medium text-gray-700">
          👁️ Live Preview
        </label>
        <button
          v-if="livePreviewOriginal && activeTransformations.length > 0"
          @click="comparisonView = comparisonView === 'both' ? 'transformed' : 'both'"
          class="text-xs px-2 py-1 bg-blue-100 text-blue-700 rounded hover:bg-blue-200 transition-colors"
        >
          {{ comparisonView === 'both' ? '📊 Side by Side' : '🔄 Show Comparison' }}
        </button>
      </div>

      <div v-if="livePreviewOriginal" class="space-y-2">
        <!-- Comparison View (Side by Side) -->
        <div v-if="comparisonView === 'both' && activeTransformations.length > 0" class="grid grid-cols-2 gap-2">
          <!-- Original -->
          <div class="bg-gradient-to-br from-gray-50 to-gray-100 border-2 border-gray-300 p-3 rounded-lg">
            <div class="text-xs text-gray-600 mb-1 font-semibold flex items-center">
              📝 Original
            </div>
            <div class="text-sm text-gray-900 break-words">
              {{ livePreviewOriginal }}
            </div>
          </div>

          <!-- Transformed -->
          <div class="bg-gradient-to-br from-green-50 to-green-100 border-2 border-green-400 p-3 rounded-lg">
            <div class="text-xs text-green-700 mb-1 font-semibold flex items-center">
              ✨ Transformed
            </div>
            <div class="text-sm text-gray-900 break-words font-medium">
              {{ livePreview }}
            </div>
          </div>
        </div>

        <!-- Single View (Transformed Only) -->
        <div v-else class="bg-gradient-to-br from-green-50 to-green-100 border-2 border-green-300 p-3 rounded-lg">
          <div class="text-xs text-green-700 mb-1 font-semibold">
            {{ activeTransformations.length > 0 ? '✨ Transformed Value:' : '📝 Extracted Value:' }}
          </div>
          <div class="text-sm text-gray-900 break-words font-medium">
            {{ livePreview }}
          </div>
        </div>

        <!-- Transformation Stats -->
        <div v-if="activeTransformations.length > 0" class="text-xs text-gray-600 bg-blue-50 border border-blue-200 rounded p-2">
          <strong>📊 Changes:</strong>
          Length: {{ livePreviewOriginal.length }} → {{ livePreview.length }} chars
          <span v-if="livePreviewOriginal.length !== livePreview.length" class="ml-2">
            ({{ livePreview.length > livePreviewOriginal.length ? '+' : '' }}{{ livePreview.length - livePreviewOriginal.length }})
          </span>
        </div>
      </div>

      <div v-else class="bg-gray-50 p-3 rounded-lg text-sm text-gray-500 italic">
        No preview available. Test selector to see extracted value.
      </div>
    </div>

    <!-- Test Results Section -->
    <div class="border-t pt-4">
      <div class="flex items-center justify-between mb-2">
        <label class="block text-sm font-medium text-gray-700">
          🔍 Test Results
          <span v-if="props.testResults.length > 0" class="ml-2 px-2 py-1 text-xs bg-green-100 text-green-800 rounded-full font-semibold">
            {{ props.testResults.length }} {{ props.testResults.length === 1 ? 'match' : 'matches' }}
          </span>
        </label>
        <button
          @click="testCurrentSelector"
          class="px-4 py-1.5 text-sm bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors font-medium shadow-sm"
        >
          Run Test
        </button>
      </div>

        <div v-if="props.testResults.length > 0" class="mt-3 relative">
          <!-- Scrollable Results Container -->
          <div class="max-h-96 overflow-y-auto border-2 border-gray-300 rounded-lg bg-gray-50 shadow-inner">
            <!-- Sticky Header -->
            <div class="sticky top-0 z-10 bg-gradient-to-b from-blue-50 to-blue-100 px-3 py-2 border-b-2 border-blue-200">
              <div class="flex items-center justify-between">
                <span class="text-xs font-semibold text-blue-900">
                  📊 {{ props.testResults.length }} matching element(s)
                </span>
                <span class="text-xs text-blue-700">
                  ⬇️ Scroll to view all
                </span>
              </div>
            </div>
            
            <!-- Results List -->
            <div class="p-3 space-y-2">
              <div
                v-for="result in props.testResults"
                :key="result.index"
                @click="emit('scrollToResult', result)"
                class="bg-white border-2 border-purple-200 p-3 rounded-lg text-sm hover:border-purple-500 hover:shadow-md transition-all cursor-pointer hover:scale-[1.02] active:scale-[0.98]"
                title="Click to scroll and highlight this element on the page"
              >
                <div class="flex items-center justify-between mb-2">
                  <span class="font-semibold text-purple-900">🔹 Element #{{ result.index + 1 }}</span>
                  <span class="text-xs bg-purple-600 text-white px-2 py-1 rounded font-medium">
                    {{ result.index + 1 }}/{{ props.testResults.length }}
                  </span>
                </div>

                <!-- Show comparison if transformations are active -->
                <div v-if="activeTransformations.length > 0" class="space-y-1">
                  <div class="text-gray-800 break-words bg-gray-100 p-2 rounded border border-gray-300">
                    <div class="text-xs text-gray-500 mb-1">📝 Original:</div>
                    <div>{{ result.value || '(empty content)' }}</div>
                  </div>
                  <div class="text-gray-800 break-words bg-green-50 p-2 rounded border border-green-300">
                    <div class="text-xs text-green-700 mb-1">✨ Transformed:</div>
                    <div class="font-medium">{{ applyTransformationsToValue(result.value || '') }}</div>
                  </div>
                </div>

                <!-- Show only value if no transformations -->
                <div v-else class="text-gray-800 break-words bg-gray-50 p-2 rounded border border-gray-200">
                  {{ result.value || '(empty content)' }}
                </div>
              </div>
            </div>
            
            <!-- Bottom Indicator -->
            <div class="sticky bottom-0 bg-gradient-to-t from-gray-100 to-transparent px-3 py-2 text-center">
              <span class="text-xs text-gray-600 font-medium">
                End of results
              </span>
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
  'testSelector': [selector: string]
  'scrollToResult': [result: TestResult]
}>()

const editedField = ref({
  name: props.field.name,
  selector: props.field.selector,
  type: props.field.type,
  attribute: props.field.attribute
})

const activeTransformations = ref<string[]>([])
const customRegexPattern = ref('')
const customRegexReplace = ref('')
const customJsCode = ref('')
const jsCodeError = ref('')
const showJsEditor = ref(false)
const comparisonView = ref<'transformed' | 'both'>('transformed')

watch(() => props.field, (newField) => {
  editedField.value = {
    name: newField.name,
    selector: newField.selector,
    type: newField.type,
    attribute: newField.attribute
  }
  activeTransformations.value = []
  customRegexPattern.value = ''
  customRegexReplace.value = ''
  customJsCode.value = ''
  jsCodeError.value = ''
  showJsEditor.value = false
  comparisonView.value = 'transformed'
}, { immediate: true })

// Original extracted value (before transformations)
const livePreviewOriginal = computed(() => {
  try {
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
  emit('testSelector', editedField.value.selector)
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
</script>
