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
          Test Selector ({{ props.testResults.length }} results)
        </label>
        <button
          @click="emit('testSelector', props.field.selector)"
          class="w-full px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
        >
          Run Test
        </button>

        <div v-if="props.testResults.length > 0" class="mt-3 space-y-2">
          <div
            v-for="result in props.testResults"
            :key="result.index"
            class="bg-purple-50 p-2 rounded text-sm"
          >
            <div class="font-medium text-purple-900">#{{ result.index + 1 }}</div>
            <div class="text-gray-700 mt-1">{{ result.value }}</div>
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
