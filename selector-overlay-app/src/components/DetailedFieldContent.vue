<template>
  <div class="space-y-4">
    <!-- Tabs -->
    <div class="flex border-b border-gray-200 -mx-5 px-5">
      <button
        @click="emit('switchTab', 'preview')"
        :class="[
          'flex-1 px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          props.tab === 'preview'
            ? 'border-blue-500 text-blue-600'
            : 'border-transparent text-gray-600 hover:text-gray-900'
        ]"
      >
        Preview
      </button>
      <button
        @click="emit('switchTab', 'edit')"
        :class="[
          'flex-1 px-4 py-3 text-sm font-medium border-b-2 transition-colors',
          props.tab === 'edit'
            ? 'border-blue-500 text-blue-600'
            : 'border-transparent text-gray-600 hover:text-gray-900'
        ]"
      >
        Edit
      </button>
    </div>

    <!-- Preview Tab -->
    <div v-if="props.tab === 'preview'" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">Field Name</label>
        <div class="text-gray-900">{{ props.field.name }}</div>
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">Selector</label>
        <code class="block bg-gray-50 p-3 rounded text-sm font-mono break-all">
          {{ props.field.selector }}
        </code>
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">Type</label>
        <div class="text-gray-900 capitalize">{{ props.field.type }}</div>
      </div>

      <div v-if="props.field.attribute">
        <label class="block text-sm font-medium text-gray-700 mb-1">Attribute</label>
        <div class="text-gray-900">{{ props.field.attribute }}</div>
      </div>

      <div v-if="props.field.sampleValue">
        <label class="block text-sm font-medium text-gray-700 mb-1">Sample Value</label>
        <div class="bg-gray-50 p-3 rounded text-sm text-gray-900">
          {{ props.field.sampleValue }}
        </div>
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-2">
          Test Selector
          <span v-if="props.testResults.length > 0" class="ml-2 px-2 py-1 text-xs bg-green-100 text-green-800 rounded-full font-semibold">
            {{ props.testResults.length }} {{ props.testResults.length === 1 ? 'match' : 'matches' }}
          </span>
        </label>
        <button
          @click="emit('testSelector', props.field.selector)"
          class="w-full px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors font-medium shadow-sm"
        >
          🔍 Run Test
        </button>

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
                <div class="text-gray-800 break-words bg-gray-50 p-2 rounded border border-gray-200">
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

    <!-- Edit Tab -->
    <div v-if="props.tab === 'edit'" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">Field Name</label>
        <input
          v-model="editedField.name"
          type="text"
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">Selector</label>
        <textarea
          v-model="editedField.selector"
          rows="3"
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono text-sm"
        />
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">Type</label>
        <select
          v-model="editedField.type"
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
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
          type="text"
          placeholder="e.g., href, src"
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
      </div>

      <div class="flex gap-2 pt-4">
        <button
          @click="handleSave"
          class="flex-1 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
        >
          Save Changes
        </button>
        <button
          @click="handleCancel"
          class="flex-1 px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-colors"
        >
          Cancel
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
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

watch(() => props.field, (newField) => {
  editedField.value = {
    name: newField.name,
    selector: newField.selector,
    type: newField.type,
    attribute: newField.attribute
  }
}, { immediate: true })

const handleSave = () => {
  // Update sample value based on new selector
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
      
      emit('saveEdit', {
        ...editedField.value,
        sampleValue,
        matchCount: elements.length
      })
    } else {
      emit('saveEdit', editedField.value)
    }
  } catch (error) {
    console.error('Error updating field:', error)
    emit('saveEdit', editedField.value)
  }
}

const handleCancel = () => {
  editedField.value = {
    name: props.field.name,
    selector: props.field.selector,
    type: props.field.type,
    attribute: props.field.attribute
  }
  emit('cancelEdit')
}
</script>
