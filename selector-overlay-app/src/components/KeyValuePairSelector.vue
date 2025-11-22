<template>
  <div class="kv-selector space-y-4">
    <!-- Field Name Input -->
    <div>
      <label class="block text-sm font-medium text-gray-700 mb-1">
        Field Name <span class="text-red-500">*</span>
      </label>
      <input
        v-model="fieldName"
        type="text"
        placeholder="e.g., attributes, specifications"
        class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        @keydown.enter="handleAdd"
      />
    </div>

    <!-- Key Selector Section with improved design -->
    <div 
      class="selector-section key-section rounded-xl border-2 transition-all duration-300 shadow-sm hover:shadow-md"
      :class="keySectionClass"
    >
      <div class="section-header bg-gradient-to-br from-green-50 to-emerald-50 px-5 py-4 rounded-t-xl border-b-2 border-green-200">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <span class="text-3xl">🔑</span>
            <span class="font-bold text-gray-900 text-base tracking-tight">KEY SELECTOR</span>
            <span v-if="keyCount > 0" class="px-2.5 py-1 bg-gradient-to-r from-green-500 to-green-600 text-white text-xs font-bold rounded-full shadow-sm">
              {{ keyCount }}
            </span>
          </div>
        </div>
        <p class="text-xs text-gray-700 mt-2 font-medium">Select the labels/names (e.g., "Color", "Size")</p>
      </div>
      
      <div class="p-4 space-y-3">
        <!-- Selector Input/Display -->
        <div>
          <div class="flex gap-2">
            <button
              @click="handleStartKeySelection"
              :disabled="isSelectingValues"
              :class="[
                'flex-1 px-4 py-2 rounded-lg font-medium transition-all',
                isSelectingKeys
                  ? 'bg-green-600 text-white animate-pulse'
                  : keySelector
                    ? 'bg-green-100 text-green-700 hover:bg-green-200 border border-green-300'
                    : 'bg-green-500 text-white hover:bg-green-600',
                isSelectingValues && 'opacity-50 cursor-not-allowed'
              ]"
            >
              <span v-if="isSelectingKeys">🎯 Click an element...</span>
              <span v-else-if="keySelector">📍 Change Selection</span>
              <span v-else>📍 Click to Pick Keys</span>
            </button>
            <button
              v-if="isSelectingKeys"
              @click="handleCancelSelection"
              class="px-4 py-2 bg-red-100 text-red-700 rounded-lg hover:bg-red-200 font-medium"
            >
              ✕
            </button>
          </div>
        </div>
        
        <div v-if="keySelector" class="bg-gray-50 p-3 rounded border border-gray-200">
          <div class="text-xs text-gray-500 mb-1">CSS Selector:</div>
          <code class="text-xs text-gray-800 break-all font-mono">{{ keySelector }}</code>
        </div>
        
        <!-- Extraction Options -->
        <div class="flex gap-2">
          <div class="flex-1">
            <label class="block text-xs font-medium text-gray-700 mb-1">Extract:</label>
            <select
              v-model="keyType"
              @change="handleKeyTypeChange"
              class="w-full px-2 py-1.5 text-sm border border-gray-300 rounded focus:ring-2 focus:ring-green-500"
            >
              <option value="text">Text Content</option>
              <option value="attribute">Attribute</option>
              <option value="html">HTML</option>
            </select>
          </div>
          <div v-if="keyType === 'attribute'" class="flex-1">
            <label class="block text-xs font-medium text-gray-700 mb-1">Attribute:</label>
            <input
              v-model="keyAttribute"
              @input="handleKeyAttributeChange"
              type="text"
              placeholder="e.g., href, src"
              class="w-full px-2 py-1.5 text-sm border border-gray-300 rounded focus:ring-2 focus:ring-green-500"
            />
          </div>
        </div>
        
        <!-- Preview -->
        <div v-if="keyMatches.length > 0" class="bg-green-50 border border-green-200 rounded p-3">
          <div class="text-xs font-semibold text-green-800 mb-2">
            ✓ Preview: {{ keyMatches.length }} keys found
          </div>
          <div class="space-y-1 max-h-32 overflow-y-auto crawlify-scrollbar">
            <div
              v-for="(item, idx) in keyMatches.slice(0, 5)"
              :key="idx"
              class="flex items-start gap-2 text-xs"
            >
              <span class="inline-flex items-center justify-center w-5 h-5 bg-green-500 text-white rounded-full font-bold flex-shrink-0">
                {{ idx + 1 }}
              </span>
              <span class="text-gray-700 break-all">{{ item || '(empty)' }}</span>
            </div>
            <div v-if="keyMatches.length > 5" class="text-xs text-gray-500 italic pl-7">
              ... and {{ keyMatches.length - 5 }} more
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Value Selector Section with improved design -->
    <div 
      class="selector-section value-section rounded-xl border-2 transition-all duration-300 shadow-sm hover:shadow-md"
      :class="valueSectionClass"
    >
      <div class="section-header bg-gradient-to-br from-blue-50 to-indigo-50 px-5 py-4 rounded-t-xl border-b-2 border-blue-200">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <span class="text-3xl">💎</span>
            <span class="font-bold text-gray-900 text-base tracking-tight">VALUE SELECTOR</span>
            <span v-if="valueCount > 0" class="px-2.5 py-1 bg-gradient-to-r from-blue-500 to-blue-600 text-white text-xs font-bold rounded-full shadow-sm">
              {{ valueCount }}
            </span>
          </div>
        </div>
        <p class="text-xs text-gray-700 mt-2 font-medium">Select the data/values (e.g., "Black", "Large")</p>
      </div>
      
      <div class="p-4 space-y-3">
        <!-- Selector Input/Display -->
        <div>
          <div class="flex gap-2">
            <button
              @click="handleStartValueSelection"
              :disabled="isSelectingKeys"
              :class="[
                'flex-1 px-4 py-2 rounded-lg font-medium transition-all',
                isSelectingValues
                  ? 'bg-blue-600 text-white animate-pulse'
                  : valueSelector
                    ? 'bg-blue-100 text-blue-700 hover:bg-blue-200 border border-blue-300'
                    : 'bg-blue-500 text-white hover:bg-blue-600',
                isSelectingKeys && 'opacity-50 cursor-not-allowed'
              ]"
            >
              <span v-if="isSelectingValues">🎯 Click an element...</span>
              <span v-else-if="valueSelector">📍 Change Selection</span>
              <span v-else>📍 Click to Pick Values</span>
            </button>
            <button
              v-if="isSelectingValues"
              @click="handleCancelSelection"
              class="px-4 py-2 bg-red-100 text-red-700 rounded-lg hover:bg-red-200 font-medium"
            >
              ✕
            </button>
          </div>
        </div>
        
        <div v-if="valueSelector" class="bg-gray-50 p-3 rounded border border-gray-200">
          <div class="text-xs text-gray-500 mb-1">CSS Selector:</div>
          <code class="text-xs text-gray-800 break-all font-mono">{{ valueSelector }}</code>
        </div>
        
        <!-- Extraction Options -->
        <div class="flex gap-2">
          <div class="flex-1">
            <label class="block text-xs font-medium text-gray-700 mb-1">Extract:</label>
            <select
              v-model="valueType"
              @change="handleValueTypeChange"
              class="w-full px-2 py-1.5 text-sm border border-gray-300 rounded focus:ring-2 focus:ring-blue-500"
            >
              <option value="text">Text Content</option>
              <option value="attribute">Attribute</option>
              <option value="html">HTML</option>
            </select>
          </div>
          <div v-if="valueType === 'attribute'" class="flex-1">
            <label class="block text-xs font-medium text-gray-700 mb-1">Attribute:</label>
            <input
              v-model="valueAttribute"
              @input="handleValueAttributeChange"
              type="text"
              placeholder="e.g., href, src"
              class="w-full px-2 py-1.5 text-sm border border-gray-300 rounded focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>
        
        <!-- Preview -->
        <div v-if="valueMatches.length > 0" class="bg-blue-50 border border-blue-200 rounded p-3">
          <div class="text-xs font-semibold text-blue-800 mb-2">
            ✓ Preview: {{ valueMatches.length }} values found
          </div>
          <div class="space-y-1 max-h-32 overflow-y-auto crawlify-scrollbar">
            <div
              v-for="(item, idx) in valueMatches.slice(0, 5)"
              :key="idx"
              class="flex items-start gap-2 text-xs"
            >
              <span class="inline-flex items-center justify-center w-5 h-5 bg-blue-500 text-white rounded-full font-bold flex-shrink-0">
                {{ idx + 1 }}
              </span>
              <span class="text-gray-700 break-all">{{ item || '(empty)' }}</span>
            </div>
            <div v-if="valueMatches.length > 5" class="text-xs text-gray-500 italic pl-7">
              ... and {{ valueMatches.length - 5 }} more
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Pairing Preview with enhanced design -->
    <div v-if="pairs.length > 0" class="pairing-preview bg-gradient-to-br from-purple-50 to-indigo-50 border-2 border-purple-300 rounded-xl p-5 shadow-md">
      <div class="flex items-center justify-between mb-4">
        <div class="flex items-center gap-3">
          <span class="text-2xl">🔗</span>
          <span class="font-bold text-gray-900 text-base tracking-tight">PAIRING PREVIEW</span>
          <span class="text-xs text-gray-700 font-semibold bg-purple-200 px-2 py-1 rounded-full">({{ pairs.length }} pairs)</span>
        </div>
        <div v-if="hasCountMismatch" class="flex items-center gap-1.5 text-orange-600 text-xs font-bold bg-orange-100 px-2 py-1 rounded-full">
          <span>⚠️</span>
          <span>Mismatch</span>
        </div>
      </div>

      <!-- Mismatch Warning -->
      <div v-if="hasCountMismatch" class="bg-orange-50 border border-orange-300 rounded p-3 mb-3">
        <div class="text-sm font-medium text-orange-800 mb-1">⚠️ Count Mismatch Warning</div>
        <div class="text-xs text-orange-700">
          Keys: <strong>{{ keyCount }}</strong> | Values: <strong>{{ valueCount }}</strong>
          <br />
          Only <strong>{{ Math.min(keyCount, valueCount) }}</strong> pairs will be created.
          {{ keyCount > valueCount ? `${keyCount - valueCount} keys will be ignored.` : `${valueCount - keyCount} values will be ignored.` }}
        </div>
      </div>

      <!-- Pairing Table -->
      <div class="bg-white border border-purple-200 rounded overflow-hidden">
        <table class="w-full text-xs">
          <thead class="bg-purple-100">
            <tr>
              <th class="px-3 py-2 text-left font-semibold text-gray-700">Key</th>
              <th class="px-2 py-2 text-center w-8"></th>
              <th class="px-3 py-2 text-left font-semibold text-gray-700">Value</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-purple-100">
            <tr
              v-for="pair in pairs.slice(0, 5)"
              :key="pair.index"
              class="hover:bg-purple-50"
            >
              <td class="px-3 py-2">
                <div class="flex items-start gap-2">
                  <span class="inline-flex items-center justify-center w-4 h-4 bg-green-500 text-white rounded-full text-[10px] font-bold flex-shrink-0">
                    {{ pair.index }}
                  </span>
                  <span class="break-all">{{ pair.key || '(empty)' }}</span>
                </div>
              </td>
              <td class="px-2 py-2 text-center text-purple-500 font-bold">→</td>
              <td class="px-3 py-2">
                <div class="flex items-start gap-2">
                  <span class="inline-flex items-center justify-center w-4 h-4 bg-blue-500 text-white rounded-full text-[10px] font-bold flex-shrink-0">
                    {{ pair.index }}
                  </span>
                  <span class="break-all">{{ pair.value || '(empty)' }}</span>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
        <div v-if="pairs.length > 5" class="px-3 py-2 bg-purple-50 text-xs text-gray-600 italic text-center">
          ... and {{ pairs.length - 5 }} more pairs
        </div>
      </div>
    </div>

    <!-- Transformations -->
    <div class="transformations">
      <label class="flex items-center gap-2 text-sm text-gray-700 cursor-pointer">
        <input
          v-model="applyTrim"
          type="checkbox"
          class="w-4 h-4 text-blue-500 rounded border-gray-300 focus:ring-2 focus:ring-blue-500"
        />
        <span>Trim whitespace</span>
      </label>
    </div>

    <!-- Multiple Extraction Pairs Section -->
    <div v-if="extractionPairs.length > 0" class="bg-purple-50 border-2 border-purple-300 rounded-lg p-4">
      <div class="flex items-center justify-between mb-3">
        <span class="font-semibold text-purple-900">📦 Extraction Pairs ({{ extractionPairs.length }})</span>
      </div>
      
      <div class="space-y-3">
        <div
          v-for="(pair, idx) in extractionPairs"
          :key="idx"
          class="bg-white border border-purple-200 rounded-lg p-3"
        >
          <div class="flex items-center justify-between mb-2">
            <span class="text-sm font-semibold text-gray-800">Pair {{ idx + 1 }}</span>
            <button
              @click="removeExtractionPair(idx)"
              class="text-red-500 hover:text-red-700 text-xs px-2 py-1 bg-red-50 hover:bg-red-100 rounded"
            >
              Remove
            </button>
          </div>
          <div class="text-xs space-y-1">
            <div class="flex gap-2">
              <span class="text-green-700 font-mono">🔑 {{ pair.key_selector }}</span>
            </div>
            <div class="flex gap-2">
              <span class="text-blue-700 font-mono">💎 {{ pair.value_selector }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Tips -->
    <details class="bg-blue-50 border border-blue-200 rounded-lg p-3">
      <summary class="text-sm font-medium text-blue-900 cursor-pointer">
        💡 Tips
      </summary>
      <ul class="mt-2 space-y-1 text-xs text-blue-800 list-disc list-inside">
        <li>Select elements with the same parent structure for best results</li>
        <li>Keys and values are paired by their order (index position)</li>
        <li>The 1st key pairs with the 1st value, 2nd with 2nd, etc.</li>
        <li>You can add multiple extraction pairs to combine different data sources</li>
      </ul>
    </details>

    <!-- Add Buttons with improved styling -->
    <div class="flex gap-3">
      <button
        v-if="canAdd"
        @click="handleAddAnotherPair"
        class="flex-1 px-5 py-3.5 rounded-xl font-bold transition-all text-base bg-gradient-to-r from-purple-500 to-purple-600 text-white hover:from-purple-600 hover:to-purple-700 shadow-md hover:shadow-lg hover:scale-105"
      >
        <span class="flex items-center justify-center gap-2">
          <span class="text-lg">➕</span>
          <span>Add Another Pair</span>
        </span>
      </button>
      <button
        @click="handleAdd"
        :disabled="!canAddToField"
        :class="[
          'flex-1 px-5 py-3.5 rounded-xl font-bold transition-all text-base shadow-md',
          canAddToField
            ? props.editingFieldId
              ? 'bg-gradient-to-r from-green-500 to-green-600 text-white hover:from-green-600 hover:to-green-700 hover:shadow-lg hover:scale-105'
              : 'bg-gradient-to-r from-blue-500 to-blue-600 text-white hover:from-blue-600 hover:to-blue-700 hover:shadow-lg hover:scale-105'
            : 'bg-gray-200 text-gray-400 cursor-not-allowed opacity-60'
        ]"
      >
        <span v-if="props.editingFieldId" class="flex items-center justify-center gap-2">
          <span class="text-lg">💾</span>
          <span>Update Field</span>
        </span>
        <span v-else class="flex items-center justify-center gap-2">
          <span class="text-lg">✓</span>
          <span>Add to Field List</span>
        </span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, onMounted, onBeforeUnmount, ref } from 'vue'
import type { ExtractionPair } from '../types'

// Inject the key-value selection composable from parent
const kvSelection = inject<ReturnType<typeof import('../composables/useKeyValueSelection').useKeyValueSelection>>('kvSelection')

if (!kvSelection) {
  throw new Error('KeyValuePairSelector must be used within a component that provides kvSelection')
}

const {
  isSelectingKeys,
  isSelectingValues,
  keySelector,
  valueSelector,
  keyType,
  valueType,
  keyAttribute,
  valueAttribute,
  keyMatches,
  valueMatches,
  keyCount,
  valueCount,
  hasCountMismatch,
  pairs,
  applyTrim,
  startKeySelection,
  startValueSelection,
  cancelSelection,
  updateKeyType,
  updateValueType,
  updateKeyAttribute,
  updateValueAttribute,
  getExtractionData,
  reset
} = kvSelection

const props = withDefaults(defineProps<{
  editMode?: boolean
  editingFieldId?: string | null
}>(), {
  editMode: false,
  editingFieldId: null
})

const emit = defineEmits<{
  add: [data: {
    fieldName: string
    extractions: ExtractionPair[]
  }]
}>()

const fieldName = defineModel<string>('fieldName', { default: '' })

// Store multiple extraction pairs
const extractionPairs = ref<ExtractionPair[]>([])

const keySectionClass = computed(() => {
  if (isSelectingKeys.value) return 'border-green-500 shadow-lg'
  if (keyCount.value > 0) return 'border-green-300'
  return 'border-gray-200'
})

const valueSectionClass = computed(() => {
  if (isSelectingValues.value) return 'border-blue-500 shadow-lg'
  if (valueCount.value > 0) return 'border-blue-300'
  return 'border-gray-200'
})

const canAdd = computed(() => {
  return keySelector.value !== '' &&
         valueSelector.value !== '' &&
         keyCount.value > 0 &&
         valueCount.value > 0
})

const canAddToField = computed(() => {
  // Can add if field name is filled AND either:
  // 1. There's a current valid pair (has both key and value selectors)
  // 2. There are already saved pairs in the array
  const hasCurrentPair = keySelector.value && valueSelector.value
  const hasSavedPairs = extractionPairs.value.length > 0
  
  return fieldName.value.trim() !== '' && (hasCurrentPair || hasSavedPairs)
})

function handleStartKeySelection() {
  startKeySelection()
}

function handleStartValueSelection() {
  startValueSelection()
}

function handleCancelSelection() {
  cancelSelection()
}

function handleKeyTypeChange() {
  updateKeyType(keyType.value)
}

function handleValueTypeChange() {
  updateValueType(valueType.value)
}

function handleKeyAttributeChange() {
  updateKeyAttribute(keyAttribute.value)
}

function handleValueAttributeChange() {
  updateValueAttribute(valueAttribute.value)
}

function handleAddAnotherPair() {
  if (!canAdd.value) return
  
  const data = getExtractionData()
  extractionPairs.value.push(data)
  
  // Reset the selection for next pair
  reset()
}

function removeExtractionPair(index: number) {
  extractionPairs.value.splice(index, 1)
}

function handleAdd() {
  // In edit mode, emit only the current selection (single pair)
  if (props.editMode) {
    if (!canAdd.value) return
    
    const data = getExtractionData()
    emit('add', {
      fieldName: fieldName.value,
      extractions: [data] // Single pair for edit mode
    })
    
    // Reset everything
    fieldName.value = ''
    extractionPairs.value = []
    reset()
    return
  }
  
  // In add mode, collect all pairs
  // If current selection is valid, add it first
  if (canAdd.value) {
    const data = getExtractionData()
    extractionPairs.value.push(data)
  }
  
  if (!canAddToField.value) return
  
  emit('add', {
    fieldName: fieldName.value,
    extractions: [...extractionPairs.value]
  })
  
  // Reset everything
  fieldName.value = ''
  extractionPairs.value = []
  reset()
}

// Expose methods for parent component
function initializeWithData(data: {
  key_selector: string
  value_selector: string
  key_type: any
  value_type: any
  key_attribute?: string
  value_attribute?: string
}) {
  // Reset first to clear any previous state
  kvSelection.reset()
  // Clear extraction pairs array (important for edit mode)
  extractionPairs.value = []
  // Then initialize with the provided data
  kvSelection.initialize(data)
}

// Load existing field data for editing
function loadFieldData(extractions: any[]) {
  console.log('🔍 loadFieldData called with:', extractions)
  
  if (!extractions || extractions.length === 0) {
    console.log('❌ No extractions provided')
    return
  }
  
  // Load the first pair to get started
  const firstPair = extractions[0]
  console.log('📝 First pair:', firstPair)
  
  // Initialize with the first pair - this will set selectors and load elements
  const initData = {
    key_selector: firstPair.key_selector,
    value_selector: firstPair.value_selector,
    key_type: firstPair.key_type,
    value_type: firstPair.value_type,
    key_attribute: firstPair.key_attribute,
    value_attribute: firstPair.value_attribute
  }
  console.log('🚀 Calling initialize with:', initData)
  kvSelection.initialize(initData)
  
  // Log state after initialization
  console.log('✅ After initialize:')
  console.log('  - keySelector:', keySelector.value)
  console.log('  - valueSelector:', valueSelector.value)
  console.log('  - keyCount:', keyCount.value)
  console.log('  - valueCount:', valueCount.value)
  console.log('  - keyMatches:', keyMatches.value)
  console.log('  - valueMatches:', valueMatches.value)
  
  // If there are multiple pairs, load them into extractionPairs
  if (extractions.length > 1) {
    console.log('📦 Loading additional pairs:', extractions.length - 1)
    // Load remaining pairs (skip first one since we're showing it in the form)
    extractionPairs.value = extractions.slice(1).map(ext => ({
      key_selector: ext.key_selector,
      value_selector: ext.value_selector,
      key_type: ext.key_type,
      value_type: ext.value_type,
      key_attribute: ext.key_attribute,
      value_attribute: ext.value_attribute,
      transform: ext.transform
    }))
  } else {
    console.log('📦 Only one pair, clearing extractionPairs')
    // Only one pair, clear extraction pairs array
    extractionPairs.value = []
  }
  
  console.log('🎉 loadFieldData complete')
}

defineExpose({
  isSelectingKeys,
  isSelectingValues,
  initializeWithData,
  loadFieldData
})
</script>

<style scoped>
.crawlify-scrollbar::-webkit-scrollbar {
  width: 8px;
}

.crawlify-scrollbar::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 4px;
}

.crawlify-scrollbar::-webkit-scrollbar-thumb {
  background: #888;
  border-radius: 4px;
}

.crawlify-scrollbar::-webkit-scrollbar-thumb:hover {
  background: #555;
}
</style>
