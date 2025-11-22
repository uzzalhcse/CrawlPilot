<template>
  <div class="kv-selector space-y-3">
    <!-- Field Name Input -->
    <div>
      <label class="block text-xs font-medium text-gray-700 mb-1.5">
        Field Name <span class="text-red-600">*</span>
      </label>
      <input
        v-model="fieldName"
        type="text"
        placeholder="e.g., attributes, specifications"
        class="w-full h-9 px-3 text-sm border border-gray-300 rounded focus:ring-2 focus:ring-gray-400 focus:border-gray-400"
        @keydown.enter="handleAdd"
      />
    </div>

    <!-- Key Selector Section -->
    <div 
      class="selector-section key-section rounded border transition-all"
      :class="keySectionClass"
    >
      <div class="section-header bg-gray-50 px-4 py-3 rounded-t border-b border-gray-200">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="font-semibold text-gray-900 text-sm">Key Selector</span>
            <span v-if="keyCount > 0" class="px-2 py-0.5 bg-gray-900 text-white text-[10px] font-semibold rounded">
              {{ keyCount }}
            </span>
          </div>
        </div>
        <p class="text-[11px] text-gray-600 mt-1">Select the labels/names (e.g., "Color", "Size")</p>
      </div>
      
      <div class="p-3 space-y-2.5">
        <!-- Selector Input/Display -->
        <div>
          <div class="flex gap-2">
            <button
              @click="handleStartKeySelection"
              :disabled="isSelectingValues"
              :class="[
                'flex-1 h-9 px-3 rounded text-sm font-medium transition-all',
                isSelectingKeys
                  ? 'bg-gray-900 text-white'
                  : keySelector
                    ? 'bg-gray-100 text-gray-900 hover:bg-gray-200 border border-gray-300'
                    : 'bg-gray-900 text-white hover:bg-gray-800',
                isSelectingValues && 'opacity-50 cursor-not-allowed'
              ]"
            >
              <span v-if="isSelectingKeys">Click an element...</span>
              <span v-else-if="keySelector">Change Selection</span>
              <span v-else>Click to Pick Keys</span>
            </button>
            <button
              v-if="isSelectingKeys"
              @click="handleCancelSelection"
              class="h-9 px-3 bg-gray-100 text-gray-700 rounded hover:bg-gray-200 text-sm font-medium"
            >
              Cancel
            </button>
          </div>
        </div>
        
        <div v-if="keySelector" class="bg-gray-50 p-2 rounded border border-gray-200">
          <div class="text-[11px] text-gray-500 mb-0.5">CSS Selector:</div>
          <code class="text-[11px] text-gray-800 break-all font-mono">{{ keySelector }}</code>
        </div>
        
        <!-- Extraction Options -->
        <div class="flex gap-2">
          <div class="flex-1">
            <label class="block text-[11px] font-medium text-gray-700 mb-1">Extract:</label>
            <select
              v-model="keyType"
              @change="handleKeyTypeChange"
              class="w-full h-8 px-2 text-xs border border-gray-300 rounded focus:ring-2 focus:ring-gray-400"
            >
              <option value="text">Text Content</option>
              <option value="attribute">Attribute</option>
              <option value="html">HTML</option>
            </select>
          </div>
          <div v-if="keyType === 'attribute'" class="flex-1">
            <label class="block text-[11px] font-medium text-gray-700 mb-1">Attribute:</label>
            <input
              v-model="keyAttribute"
              @input="handleKeyAttributeChange"
              type="text"
              placeholder="e.g., href, src"
              class="w-full h-8 px-2 text-xs border border-gray-300 rounded focus:ring-2 focus:ring-gray-400"
            />
          </div>
        </div>
        
        <!-- Preview -->
        <div v-if="keyMatches.length > 0" class="bg-gray-50 border border-gray-200 rounded p-2.5">
          <div class="text-[11px] font-semibold text-gray-900 mb-2">
            Preview: {{ keyMatches.length }} keys found
          </div>
          <div class="space-y-1 max-h-32 overflow-y-auto crawlify-scrollbar">
            <div
              v-for="(item, idx) in keyMatches.slice(0, 5)"
              :key="idx"
              class="flex items-start gap-2 text-[11px]"
            >
              <span class="inline-flex items-center justify-center w-4 h-4 bg-gray-700 text-white rounded text-[10px] font-semibold flex-shrink-0">
                {{ idx + 1 }}
              </span>
              <span class="text-gray-700 break-all">{{ item || '(empty)' }}</span>
            </div>
            <div v-if="keyMatches.length > 5" class="text-[11px] text-gray-500 pl-6">
              ... and {{ keyMatches.length - 5 }} more
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Value Selector Section -->
    <div 
      class="selector-section value-section rounded border transition-all"
      :class="valueSectionClass"
    >
      <div class="section-header bg-gray-50 px-4 py-3 rounded-t border-b border-gray-200">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="font-semibold text-gray-900 text-sm">Value Selector</span>
            <span v-if="valueCount > 0" class="px-2 py-0.5 bg-gray-900 text-white text-[10px] font-semibold rounded">
              {{ valueCount }}
            </span>
          </div>
        </div>
        <p class="text-[11px] text-gray-600 mt-1">Select the data/values (e.g., "Black", "Large")</p>
      </div>
      
      <div class="p-3 space-y-2.5">
        <!-- Selector Input/Display -->
        <div>
          <div class="flex gap-2">
            <button
              @click="handleStartValueSelection"
              :disabled="isSelectingKeys"
              :class="[
                'flex-1 h-9 px-3 rounded text-sm font-medium transition-all',
                isSelectingValues
                  ? 'bg-gray-900 text-white'
                  : valueSelector
                    ? 'bg-gray-100 text-gray-900 hover:bg-gray-200 border border-gray-300'
                    : 'bg-gray-900 text-white hover:bg-gray-800',
                isSelectingKeys && 'opacity-50 cursor-not-allowed'
              ]"
            >
              <span v-if="isSelectingValues">Click an element...</span>
              <span v-else-if="valueSelector">Change Selection</span>
              <span v-else>Click to Pick Values</span>
            </button>
            <button
              v-if="isSelectingValues"
              @click="handleCancelSelection"
              class="h-9 px-3 bg-gray-100 text-gray-700 rounded hover:bg-gray-200 text-sm font-medium"
            >
              Cancel
            </button>
          </div>
        </div>
        
        <div v-if="valueSelector" class="bg-gray-50 p-2 rounded border border-gray-200">
          <div class="text-[11px] text-gray-500 mb-0.5">CSS Selector:</div>
          <code class="text-[11px] text-gray-800 break-all font-mono">{{ valueSelector }}</code>
        </div>
        
        <!-- Extraction Options -->
        <div class="flex gap-2">
          <div class="flex-1">
            <label class="block text-[11px] font-medium text-gray-700 mb-1">Extract:</label>
            <select
              v-model="valueType"
              @change="handleValueTypeChange"
              class="w-full h-8 px-2 text-xs border border-gray-300 rounded focus:ring-2 focus:ring-gray-400"
            >
              <option value="text">Text Content</option>
              <option value="attribute">Attribute</option>
              <option value="html">HTML</option>
            </select>
          </div>
          <div v-if="valueType === 'attribute'" class="flex-1">
            <label class="block text-[11px] font-medium text-gray-700 mb-1">Attribute:</label>
            <input
              v-model="valueAttribute"
              @input="handleValueAttributeChange"
              type="text"
              placeholder="e.g., href, src"
              class="w-full h-8 px-2 text-xs border border-gray-300 rounded focus:ring-2 focus:ring-gray-400"
            />
          </div>
        </div>
        
        <!-- Preview -->
        <div v-if="valueMatches.length > 0" class="bg-gray-50 border border-gray-200 rounded p-2.5">
          <div class="text-[11px] font-semibold text-gray-900 mb-2">
            Preview: {{ valueMatches.length }} values found
          </div>
          <div class="space-y-1 max-h-32 overflow-y-auto crawlify-scrollbar">
            <div
              v-for="(item, idx) in valueMatches.slice(0, 5)"
              :key="idx"
              class="flex items-start gap-2 text-[11px]"
            >
              <span class="inline-flex items-center justify-center w-4 h-4 bg-gray-700 text-white rounded text-[10px] font-semibold flex-shrink-0">
                {{ idx + 1 }}
              </span>
              <span class="text-gray-700 break-all">{{ item || '(empty)' }}</span>
            </div>
            <div v-if="valueMatches.length > 5" class="text-[11px] text-gray-500 pl-6">
              ... and {{ valueMatches.length - 5 }} more
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Pairing Preview -->
    <div v-if="pairs.length > 0" class="pairing-preview bg-white border border-gray-200 rounded p-4">
      <div class="flex items-center justify-between mb-3">
        <div class="flex items-center gap-2">
          <span class="font-semibold text-gray-900 text-sm">Pairing Preview</span>
          <span class="text-[10px] text-gray-600 font-medium bg-gray-100 px-2 py-0.5 rounded">({{ pairs.length }} pairs)</span>
        </div>
        <div v-if="hasCountMismatch" class="flex items-center gap-1 text-orange-600 text-[10px] font-semibold bg-orange-50 px-2 py-0.5 rounded border border-orange-200">
          <span>Mismatch</span>
        </div>
      </div>

      <!-- Mismatch Warning -->
      <div v-if="hasCountMismatch" class="bg-orange-50 border border-orange-200 rounded p-2.5 mb-3">
        <div class="text-xs font-semibold text-orange-900 mb-1">Count Mismatch Warning</div>
        <div class="text-[11px] text-orange-800">
          Keys: <strong>{{ keyCount }}</strong> | Values: <strong>{{ valueCount }}</strong>
          <br />
          Only <strong>{{ Math.min(keyCount, valueCount) }}</strong> pairs will be created.
          {{ keyCount > valueCount ? `${keyCount - valueCount} keys will be ignored.` : `${valueCount - keyCount} values will be ignored.` }}
        </div>
      </div>

      <!-- Pairing Table -->
      <div class="bg-white border border-gray-200 rounded overflow-hidden">
        <table class="w-full text-[11px]">
          <thead class="bg-gray-50">
            <tr>
              <th class="px-3 py-2 text-left font-semibold text-gray-700">Key</th>
              <th class="px-2 py-2 text-center w-6"></th>
              <th class="px-3 py-2 text-left font-semibold text-gray-700">Value</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100">
            <tr
              v-for="pair in pairs.slice(0, 5)"
              :key="pair.index"
              class="hover:bg-gray-50"
            >
              <td class="px-3 py-2">
                <div class="flex items-start gap-2">
                  <span class="inline-flex items-center justify-center w-4 h-4 bg-gray-700 text-white rounded text-[10px] font-semibold flex-shrink-0">
                    {{ pair.index }}
                  </span>
                  <span class="break-all">{{ pair.key || '(empty)' }}</span>
                </div>
              </td>
              <td class="px-2 py-2 text-center text-gray-400 font-medium">→</td>
              <td class="px-3 py-2">
                <div class="flex items-start gap-2">
                  <span class="inline-flex items-center justify-center w-4 h-4 bg-gray-700 text-white rounded text-[10px] font-semibold flex-shrink-0">
                    {{ pair.index }}
                  </span>
                  <span class="break-all">{{ pair.value || '(empty)' }}</span>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
        <div v-if="pairs.length > 5" class="px-3 py-2 bg-gray-50 text-[11px] text-gray-500 text-center">
          ... and {{ pairs.length - 5 }} more pairs
        </div>
      </div>
    </div>

    <!-- Transformations -->
    <div class="transformations">
      <label class="flex items-center gap-2 text-xs text-gray-700 cursor-pointer">
        <input
          v-model="applyTrim"
          type="checkbox"
          class="w-3.5 h-3.5 text-gray-900 rounded border-gray-300 focus:ring-2 focus:ring-gray-400"
        />
        <span>Trim whitespace</span>
      </label>
    </div>

    <!-- All Extraction Pairs Section (Shows Current + Saved) -->
    <div v-if="extractionPairs.length > 0 || (keySelector && valueSelector)" class="bg-gray-50 border border-gray-200 rounded p-3">
      <div class="flex items-center justify-between mb-2.5">
        <span class="font-semibold text-gray-900 text-sm">All Extraction Pairs ({{ totalPairsCount }})</span>
        <span class="text-[11px] text-gray-600">{{ editingPairIndex !== null ? 'Editing in form above' : 'Configure new pair above' }}</span>
      </div>
      
      <div class="space-y-3">
        <!-- Current pair being configured (if exists and not editing a saved pair) -->
        <div
          v-if="keySelector && valueSelector && editingPairIndex === null"
          class="bg-white border border-gray-300 rounded overflow-hidden"
        >
          <!-- Pair Header -->
          <div class="flex items-center justify-between px-3 py-2 bg-gray-100 border-b border-gray-300">
            <div class="flex items-center gap-2">
              <span class="text-xs font-semibold text-gray-900">Pair {{ extractionPairs.length + 1 }}</span>
              <span class="text-[10px] px-2 py-0.5 bg-gray-900 text-white rounded font-semibold">CURRENT</span>
            </div>
            <span class="text-[11px] text-gray-600">Unsaved - click button below to save</span>
          </div>
          
          <!-- Pair Details -->
          <div class="p-2.5 space-y-2 text-[11px]">
            <!-- Key Info -->
            <div class="bg-gray-50 border border-gray-200 rounded p-2">
              <div class="font-semibold text-gray-900 mb-1">
                Key Selector
              </div>
              <div class="font-mono text-gray-800 break-all">{{ keySelector }}</div>
              <div class="mt-1 text-gray-600">
                <span class="font-medium">Type:</span> {{ keyType }}
                <span v-if="keyType === 'attribute' && keyAttribute" class="ml-2">
                  <span class="font-medium">Attr:</span> {{ keyAttribute }}
                </span>
              </div>
            </div>
            
            <!-- Value Info -->
            <div class="bg-gray-50 border border-gray-200 rounded p-2">
              <div class="font-semibold text-gray-900 mb-1">
                Value Selector
              </div>
              <div class="font-mono text-gray-800 break-all">{{ valueSelector }}</div>
              <div class="mt-1 text-gray-600">
                <span class="font-medium">Type:</span> {{ valueType }}
                <span v-if="valueType === 'attribute' && valueAttribute" class="ml-2">
                  <span class="font-medium">Attr:</span> {{ valueAttribute }}
                </span>
              </div>
            </div>
          </div>
        </div>
        
        <!-- Saved pairs -->
        <div
          v-for="(pair, idx) in extractionPairs"
          :key="idx"
          :class="[
            'bg-white border rounded overflow-hidden transition-all',
            editingPairIndex === idx 
              ? 'border-gray-400 ring-2 ring-gray-300' 
              : 'border-gray-200 hover:border-gray-300'
          ]"
        >
          <!-- Pair Header -->
          <div :class="[
            'flex items-center justify-between px-3 py-2 border-b',
            editingPairIndex === idx
              ? 'bg-gray-100 border-gray-400'
              : 'bg-gray-50 border-gray-200'
          ]">
            <div class="flex items-center gap-2">
              <span :class="[
                'text-xs font-semibold',
                editingPairIndex === idx ? 'text-gray-900' : 'text-gray-900'
              ]">Pair {{ idx + 1 }}</span>
              <span v-if="editingPairIndex === idx" class="text-[10px] px-2 py-0.5 bg-gray-900 text-white rounded font-semibold">
                EDITING
              </span>
            </div>
            <div class="flex gap-1.5">
              <button
                v-if="editingPairIndex !== idx"
                @click="editExtractionPair(idx)"
                class="text-[10px] px-2 py-1 bg-gray-900 hover:bg-gray-800 text-white rounded font-medium transition-colors"
                title="Edit this pair"
              >
                Edit
              </button>
              <button
                v-else
                @click="cancelEditPair"
                class="text-[10px] px-2 py-1 bg-gray-500 hover:bg-gray-600 text-white rounded font-medium transition-colors"
                title="Cancel editing"
              >
                Cancel
              </button>
              <button
                @click="removeExtractionPair(idx)"
                class="text-[10px] px-2 py-1 bg-red-600 hover:bg-red-700 text-white rounded font-medium transition-colors"
                title="Delete this pair"
              >
                Delete
              </button>
            </div>
          </div>
          
          <!-- Pair Details -->
          <div class="p-2.5 space-y-2 text-[11px]">
            <!-- Key Info -->
            <div class="bg-gray-50 border border-gray-200 rounded p-2">
              <div class="font-semibold text-gray-900 mb-1">
                Key Selector
              </div>
              <div class="font-mono text-gray-800 break-all">{{ pair.key_selector }}</div>
              <div class="mt-1 text-gray-600">
                <span class="font-medium">Type:</span> {{ pair.key_type }}
                <span v-if="pair.key_attribute" class="ml-2">
                  <span class="font-medium">Attr:</span> {{ pair.key_attribute }}
                </span>
              </div>
            </div>
            
            <!-- Value Info -->
            <div class="bg-gray-50 border border-gray-200 rounded p-2">
              <div class="font-semibold text-gray-900 mb-1">
                Value Selector
              </div>
              <div class="font-mono text-gray-800 break-all">{{ pair.value_selector }}</div>
              <div class="mt-1 text-gray-600">
                <span class="font-medium">Type:</span> {{ pair.value_type }}
                <span v-if="pair.value_attribute" class="ml-2">
                  <span class="font-medium">Attr:</span> {{ pair.value_attribute }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Tips -->
    <details class="bg-gray-50 border border-gray-200 rounded p-2.5">
      <summary class="text-xs font-medium text-gray-900 cursor-pointer">
        Tips
      </summary>
      <ul class="mt-2 space-y-1 text-[11px] text-gray-600 list-disc list-inside">
        <li>Select elements with the same parent structure for best results</li>
        <li>Keys and values are paired by their order (index position)</li>
        <li>The 1st key pairs with the 1st value, 2nd with 2nd, etc.</li>
        <li>You can add multiple extraction pairs to combine different data sources</li>
      </ul>
    </details>

    <!-- Add Buttons -->
    <div class="flex gap-2">
      <!-- Cancel button when editing -->
      <button
        v-if="editingPairIndex !== null && !canAdd"
        @click="cancelEditPair"
        class="flex-1 h-9 px-3 rounded font-medium transition-all text-sm bg-white border border-gray-300 text-gray-700 hover:bg-gray-50"
      >
        Cancel Edit
      </button>
      
      <!-- Add/Save button -->
      <button
        v-if="canAdd || editingPairIndex !== null"
        @click="handleAddAnotherPair"
        :disabled="!canAdd"
        :class="[
          'flex-1 h-9 px-3 rounded font-medium transition-all text-sm',
          canAdd
            ? 'bg-gray-900 text-white hover:bg-gray-800'
            : 'bg-gray-200 text-gray-400 cursor-not-allowed'
        ]"
      >
        <span v-if="editingPairIndex !== null">Save Changes</span>
        <span v-else>Add Another Pair</span>
      </button>
      <button
        @click="handleAdd"
        :disabled="!canAddToField"
        :class="[
          'flex-1 h-9 px-3 rounded font-medium transition-all text-sm',
          canAddToField
            ? 'bg-gray-900 text-white hover:bg-gray-800'
            : 'bg-gray-200 text-gray-400 cursor-not-allowed'
        ]"
      >
        <span v-if="props.editingFieldId">Update Field</span>
        <span v-else>Add to Field List</span>
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

// Store ALL extraction pairs (including the one being edited)
const extractionPairs = ref<ExtractionPair[]>([])
const editingPairIndex = ref<number | null>(null) // Track which pair index is being edited (-1 = new pair)

const keySectionClass = computed(() => {
  if (isSelectingKeys.value) return 'border-gray-400'
  if (keyCount.value > 0) return 'border-gray-300'
  return 'border-gray-200'
})

const valueSectionClass = computed(() => {
  if (isSelectingValues.value) return 'border-gray-400'
  if (valueCount.value > 0) return 'border-gray-300'
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

const totalPairsCount = computed(() => {
  let count = extractionPairs.value.length
  // Add 1 if there's a current pair being configured (and not editing an existing one)
  if (keySelector.value && valueSelector.value && editingPairIndex.value === null) {
    count += 1
  }
  return count
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
  
  if (editingPairIndex.value !== null) {
    // Update the existing pair
    extractionPairs.value[editingPairIndex.value] = data
    editingPairIndex.value = null
  } else {
    // Add as a new pair
    extractionPairs.value.push(data)
  }
  
  // Reset the selection for next pair
  reset()
}

function removeExtractionPair(index: number) {
  extractionPairs.value.splice(index, 1)
}

function editExtractionPair(index: number) {
  // Get the pair to edit
  const pairToEdit = extractionPairs.value[index]
  
  // Set editing index
  editingPairIndex.value = index
  
  // Load it into the form
  kvSelection.initialize({
    key_selector: pairToEdit.key_selector,
    value_selector: pairToEdit.value_selector,
    key_type: pairToEdit.key_type,
    value_type: pairToEdit.value_type,
    key_attribute: pairToEdit.key_attribute,
    value_attribute: pairToEdit.value_attribute
  })
  
  // Scroll to top to show the form
  const container = document.querySelector('.kv-selector')
  if (container) {
    container.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
}

function cancelEditPair() {
  // Reset editing index
  editingPairIndex.value = null
  
  // Clear the form
  reset()
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
  if (!extractions || extractions.length === 0) {
    return
  }
  
  // Load ALL pairs into the array
  extractionPairs.value = extractions.map(ext => ({
    key_selector: ext.key_selector,
    value_selector: ext.value_selector,
    key_type: ext.key_type,
    value_type: ext.value_type,
    key_attribute: ext.key_attribute,
    value_attribute: ext.value_attribute,
    transform: ext.transform
  }))
  
  // Set the first pair as being edited
  editingPairIndex.value = 0
  
  // Load the first pair into the form
  const firstPair = extractions[0]
  kvSelection.initialize({
    key_selector: firstPair.key_selector,
    value_selector: firstPair.value_selector,
    key_type: firstPair.key_type,
    value_type: firstPair.value_type,
    key_attribute: firstPair.key_attribute,
    value_attribute: firstPair.value_attribute
  })
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
