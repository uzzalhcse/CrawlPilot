<template>
  <div class="fixed top-5 right-5 bg-white rounded-xl shadow-2xl p-6 min-w-[400px] max-w-[480px] max-h-[90vh] overflow-y-auto pointer-events-auto z-[1000000] border border-gray-200 crawlify-scrollbar" @click.stop>
    <!-- Header -->
    <div class="mb-4">
      <div class="flex items-start justify-between">
        <div>
          <h2 class="text-2xl font-bold text-gray-900">Element Selector</h2>
          <p class="text-base text-gray-600 mt-1">
            {{ props.detailedViewField ? 'Field Details' : 'Click elements to select them' }}
          </p>
        </div>
        <button
          v-if="props.detailedViewField"
          @click="emit('closeDetailedView')"
          class="text-gray-400 hover:text-gray-600 text-2xl leading-none ml-2"
          title="Back to list (ESC)"
        >
          ←
        </button>
      </div>
      
      <!-- Keyboard hints -->
      <div v-if="!props.detailedViewField" class="mt-3 text-sm text-gray-600 bg-gray-50 p-3 rounded border border-gray-200">
        <div><kbd class="px-2 py-1 bg-white border border-gray-300 rounded text-gray-700 font-mono text-sm">ESC</kbd> Clear selection</div>
        <div class="mt-1"><kbd class="px-2 py-1 bg-white border border-gray-300 rounded text-gray-700 font-mono text-sm">Enter</kbd> Add field</div>
      </div>

      <!-- Color Legend -->
      <details v-if="!props.detailedViewField && props.selectedFields.length > 0" class="mt-3">
        <summary class="text-sm font-medium text-gray-700 cursor-pointer hover:text-gray-900">
          Color Legend
        </summary>
        <div class="mt-2 space-y-2 text-sm">
          <div class="flex items-center gap-2">
            <div class="w-4 h-4 border-2 border-blue-500 bg-blue-500/15 rounded"></div>
            <span>Text content</span>
          </div>
          <div class="flex items-center gap-2">
            <div class="w-4 h-4 border-2 border-purple-500 bg-purple-500/15 rounded"></div>
            <span>Attribute value</span>
          </div>
          <div class="flex items-center gap-2">
            <div class="w-4 h-4 border-2 border-pink-500 bg-pink-500/15 rounded"></div>
            <span>HTML content</span>
          </div>
        </div>
      </details>
    </div>

    <!-- Mode Selector -->
    <div v-if="!props.detailedViewField" class="mb-4">
      <label class="block text-sm font-medium text-gray-700 mb-2">Selection Mode</label>
      <div class="flex gap-2">
        <button
          @click="emit('update:mode', 'single')"
          :class="[
            'flex-1 px-3 py-2 text-sm font-medium rounded-lg transition-colors',
            props.mode === 'single'
              ? 'bg-blue-500 text-white'
              : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
          ]"
        >
          Single
        </button>
        <button
          @click="emit('update:mode', 'list')"
          :class="[
            'flex-1 px-3 py-2 text-sm font-medium rounded-lg transition-colors',
            props.mode === 'list'
              ? 'bg-blue-500 text-white'
              : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
          ]"
        >
          List
        </button>
      </div>
    </div>

    <!-- Field Configuration -->
    <div v-if="!props.detailedViewField" class="space-y-3 mb-4">
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">Field Name</label>
        <input
          :value="props.fieldName"
          @input="emit('update:fieldName', ($event.target as HTMLInputElement).value)"
          @keydown.enter="canAddField && emit('addField')"
          type="text"
          placeholder="e.g., title, price"
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          autofocus
        />
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">Extract</label>
        <select
          :value="props.fieldType"
          @change="emit('update:fieldType', ($event.target as HTMLSelectElement).value)"
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        >
          <option value="text">Text Content</option>
          <option value="attribute">Attribute</option>
          <option value="html">HTML</option>
        </select>
      </div>

      <div v-if="props.fieldType === 'attribute'">
        <label class="block text-sm font-medium text-gray-700 mb-1">Attribute Name</label>
        <input
          :value="props.fieldAttribute"
          @input="emit('update:fieldAttribute', ($event.target as HTMLInputElement).value)"
          type="text"
          placeholder="e.g., href, src"
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
      </div>

      <!-- Validation Message -->
      <div v-if="props.hoveredElementValidation" class="text-sm">
        <div
          :class="[
            'flex items-center gap-2 px-3 py-2 rounded-lg',
            props.hoveredElementValidation.isValid
              ? 'bg-green-50 text-green-700 border border-green-200'
              : 'bg-red-50 text-red-700 border border-red-200'
          ]"
        >
          <span class="text-lg">{{ props.hoveredElementValidation.isValid ? '✓' : '✗' }}</span>
          <span class="font-medium">{{ props.hoveredElementValidation.message }}</span>
        </div>
      </div>

      <button
        @click="emit('addField')"
        :disabled="!canAddField"
        :class="[
          'w-full px-4 py-2 rounded-lg font-medium transition-colors',
          canAddField
            ? 'bg-blue-500 text-white hover:bg-blue-600 active:bg-blue-700'
            : 'bg-gray-200 text-gray-400 cursor-not-allowed'
        ]"
      >
        {{ canAddField ? '✓ Add Field' : 'Add Field' }}
      </button>
    </div>

    <!-- Selected Fields List -->
    <div v-if="!props.detailedViewField" class="mt-6">
      <h3 class="text-sm font-semibold text-gray-900 mb-3">
        Selected Fields ({{ props.selectedFields.length }})
      </h3>
      
      <div v-if="props.selectedFields.length === 0" class="text-sm text-gray-500 text-center py-4">
        No fields selected yet
      </div>

      <div v-else class="space-y-2">
        <div
          v-for="field in props.selectedFields"
          :key="field.id"
          class="bg-gray-50 rounded-lg p-3 hover:bg-gray-100 transition-all cursor-pointer border-l-4 border-gray-200 hover:shadow-sm"
          :class="getFieldBorderClass(field)"
          @click="emit('openDetailedView', field)"
        >
          <div class="flex items-start justify-between">
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <div class="font-medium text-gray-900 truncate">{{ field.name }}</div>
                <span class="text-xs px-1.5 py-0.5 rounded font-medium" :class="getFieldTypeBadge(field)">
                  {{ field.type }}
                </span>
              </div>
              <div class="text-xs text-gray-500 font-mono truncate mt-1">
                {{ field.selector }}
              </div>
              <div v-if="field.matchCount" class="text-xs text-blue-600 mt-1">
                {{ field.matchCount }} {{ field.matchCount === 1 ? 'match' : 'matches' }}
              </div>
              <div v-if="field.sampleValue" class="text-xs text-gray-600 truncate mt-1 italic">
                "{{ field.sampleValue }}"
              </div>
            </div>
            <button
              @click.stop="emit('removeField', field.id)"
              class="ml-2 text-red-500 hover:text-red-700 hover:bg-red-50 p-1.5 rounded transition-colors"
              title="Remove field"
            >
              ✕
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Detailed View Content (inside panel) -->
    <DetailedFieldContent
      v-if="props.detailedViewField"
      :field="props.detailedViewField"
      :tab="props.detailedViewTab"
      :edit-mode="props.editMode"
      :test-results="props.testResults"
      @switch-tab="emit('switchTab', $event)"
      @enable-edit="emit('enableEditMode')"
      @save-edit="emit('saveEdit', $event)"
      @cancel-edit="emit('cancelEdit')"
      @test-selector="emit('testSelector', $event)"
      @scroll-to-result="emit('scrollToResult', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { SelectedField, SelectionMode, FieldType, ValidationResult, TestResult } from '../types'
import DetailedFieldContent from './DetailedFieldContent.vue'
import { getElementColor } from '../utils/elementColors'

interface Props {
  mode: SelectionMode
  fieldName: string
  fieldType: FieldType
  fieldAttribute: string
  selectedFields: SelectedField[]
  hoveredElementCount: number
  hoveredElementValidation: ValidationResult | null
  detailedViewField: SelectedField | null
  detailedViewTab: 'preview' | 'edit'
  editMode: boolean
  testResults: TestResult[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:mode': [mode: SelectionMode]
  'update:fieldName': [name: string]
  'update:fieldType': [type: FieldType]
  'update:fieldAttribute': [attr: string]
  'addField': []
  'removeField': [id: string]
  'openDetailedView': [field: SelectedField]
  'closeDetailedView': []
  'switchTab': [tab: 'preview' | 'edit']
  'enableEditMode': []
  'saveEdit': [field: Partial<SelectedField>]
  'cancelEdit': []
  'testSelector': [field: SelectedField]
  'scrollToResult': [result: TestResult]
}>()

const canAddField = computed(() => {
  if (!props.fieldName.trim()) return false
  if (props.hoveredElementCount === 0) return false
  if (props.fieldType === 'attribute' && !props.fieldAttribute.trim()) return false
  return true
})

const getFieldBorderClass = (field: SelectedField) => {
  const colors = getElementColor(field.type)
  return `!border-l-${colors.border}`
}

const getFieldTypeBadge = (field: SelectedField) => {
  const colors = getElementColor(field.type)
  if (field.type === 'text') return 'bg-blue-100 text-blue-700'
  if (field.type === 'attribute') return 'bg-purple-100 text-purple-700'
  if (field.type === 'html') return 'bg-pink-100 text-pink-700'
  return 'bg-gray-100 text-gray-700'
}
</script>
