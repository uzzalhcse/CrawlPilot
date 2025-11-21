<template>
  <div class="fixed inset-0 bg-black/50 z-[1000001] flex items-center justify-center p-4 pointer-events-auto" @click.self="emit('close')">
    <div class="bg-white rounded-xl shadow-2xl w-full max-w-2xl max-h-[90vh] flex flex-col">
      <!-- Header -->
      <div class="flex items-center justify-between p-5 border-b border-gray-200">
        <h3 class="text-lg font-semibold text-gray-900">{{ props.field.name }}</h3>
        <button
          @click="emit('close')"
          class="text-gray-400 hover:text-gray-600 text-2xl leading-none"
        >
          ×
        </button>
      </div>

      <!-- Tabs -->
      <div class="flex border-b border-gray-200">
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

      <!-- Content -->
      <div class="flex-1 overflow-y-auto p-5 crawlify-scrollbar">
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

            <!-- Array Detection Banner -->
            <div v-if="props.testResults.length > 1" class="mt-3 p-3 bg-purple-50 border-2 border-purple-200 rounded-lg">
              <div class="flex items-start gap-2">
                <span class="text-purple-600 text-lg">📋</span>
                <div class="flex-1">
                  <div class="font-semibold text-purple-900 text-sm mb-1">
                    Multiple Elements Detected
                  </div>
                  <p class="text-xs text-purple-700 mb-2">
                    This selector matches {{ props.testResults.length }} elements. You can extract all of them as an array.
                  </p>
                  <div class="text-xs text-purple-600 bg-purple-100 p-2 rounded border border-purple-200">
                    💡 <strong>Tip:</strong> Enable "Extract Multiple Values" in the workflow builder to get an array like: 
                    <code class="bg-white px-1 py-0.5 rounded text-purple-800">["value1", "value2", "value3"]</code>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="props.testResults.length > 0" class="mt-3">
              <!-- Results Header -->
              <div class="bg-gradient-to-b from-blue-50 to-blue-100 px-3 py-2 border-2 border-blue-200 rounded-t-lg">
                <div class="flex items-center justify-between">
                  <span class="text-xs font-semibold text-blue-900">
                    📊 {{ props.testResults.length }} matching element(s)
                  </span>
                  <span class="text-xs text-blue-700">
                    {{ props.testResults.length > 5 ? '⬇️ Scroll modal to view all' : 'All results visible' }}
                  </span>
                </div>
              </div>
              
              <!-- Results Container (no fixed height, flows naturally) -->
              <div class="border-x-2 border-b-2 border-gray-300 rounded-b-lg bg-gray-50 shadow-inner">
                
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
  'close': []
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
