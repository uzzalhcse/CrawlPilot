<script setup lang="ts">
import { ref, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Plus, ChevronDown, ChevronUp, MousePointerClick } from 'lucide-vue-next'
import FieldCard from './FieldCard.vue'
import { selectorApi, type SelectedField } from '@/api/selector'

interface ParamField {
  key: string
  label: string
  type: string
  required?: boolean
  placeholder?: string
  description?: string
  options?: Array<{label: string; value: string}>
  defaultValue?: any
}

interface Props {
  modelValue: Record<string, any>
  schema: ParamField[]
  paramKey: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, any>]
}>()

const collapsedFields = ref(new Set<string>())
const searchQuery = ref('')

// Visual selector state
const isVisualSelectorOpen = ref(false)
const visualSelectorSessionId = ref<string | null>(null)
const visualSelectorLoading = ref(false)
const visualSelectorError = ref<string | null>(null)
let stopPolling: (() => void) | null = null

const fields = computed(() => props.modelValue || {})

const visibleFields = computed(() => {
  if (!searchQuery.value) {
    return Object.keys(fields.value)
  }
  return Object.keys(fields.value).filter(name =>
    name.toLowerCase().includes(searchQuery.value.toLowerCase())
  )
})

function addField() {
  const newFields = { ...fields.value }
  const newFieldName = `field_${Object.keys(newFields).length + 1}`
  
  newFields[newFieldName] = {
    selector: '',
    type: 'text',
    transform: 'none',
    attribute: '',
    default_value: '',
    multiple: false,
    limit: 0,
    fields: '',
    extractions: ''
  }
  
  emit('update:modelValue', newFields)
  
  // Scroll to new field
  setTimeout(() => {
    const fieldCards = document.querySelectorAll('[data-field-card]')
    const lastCard = fieldCards[fieldCards.length - 1]
    if (lastCard) {
      lastCard.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
    }
  }, 100)
}

function removeField(fieldName: string) {
  const newFields = { ...fields.value }
  delete newFields[fieldName]
  emit('update:modelValue', newFields)
  
  // Also remove from collapsed set
  collapsedFields.value.delete(fieldName)
}

function duplicateField(fieldName: string) {
  const newFields = { ...fields.value }
  const originalField = newFields[fieldName]
  
  let newFieldName = `${fieldName}_copy`
  let counter = 1
  
  while (newFields[newFieldName]) {
    newFieldName = `${fieldName}_copy_${counter}`
    counter++
  }
  
  newFields[newFieldName] = JSON.parse(JSON.stringify(originalField))
  emit('update:modelValue', newFields)
}

function updateFieldData(fieldName: string, data: Record<string, any>) {
  const newFields = { ...fields.value }
  newFields[fieldName] = data
  emit('update:modelValue', newFields)
}

function renameField(oldName: string, newName: string) {
  if (!newName || oldName === newName) return
  
  const newFields = { ...fields.value }
  
  // Check if new name already exists
  if (newFields[newName]) {
    console.warn(`Field name "${newName}" already exists`)
    return
  }
  
  // Move field to new key
  newFields[newName] = newFields[oldName]
  delete newFields[oldName]
  
  emit('update:modelValue', newFields)
  
  // Update collapsed state
  if (collapsedFields.value.has(oldName)) {
    collapsedFields.value.delete(oldName)
    collapsedFields.value.add(newName)
  }
}

function toggleFieldCollapse(fieldName: string) {
  if (collapsedFields.value.has(fieldName)) {
    collapsedFields.value.delete(fieldName)
  } else {
    collapsedFields.value.add(fieldName)
  }
}

function expandAll() {
  collapsedFields.value.clear()
}

function collapseAll() {
  Object.keys(fields.value).forEach(name => {
    collapsedFields.value.add(name)
  })
}

// Visual Selector Methods
async function openVisualSelector() {
  const url = prompt('Enter the URL to open for element selection:')
  if (!url) return
  
  visualSelectorLoading.value = true
  visualSelectorError.value = null
  
  try {
    // Convert existing fields to SelectedField format
    const existingFields = convertToSelectedFields(fields.value)
    
    const session = await selectorApi.createSession(url, existingFields)
    visualSelectorSessionId.value = session.session_id
    isVisualSelectorOpen.value = true
    
    // Start polling for selected fields
    stopPolling = await selectorApi.pollForFields(
      session.session_id,
      2000,
      (selectedFields: SelectedField[]) => {
        if (selectedFields.length > 0) {
          importFromVisualSelector(selectedFields)
        }
      },
      (error) => {
        console.error('Visual selector error:', error)
        visualSelectorError.value = 'Session closed or connection lost'
        closeVisualSelector()
      }
    )
    
    visualSelectorLoading.value = false
  } catch (error: any) {
    visualSelectorError.value = error.response?.data?.error || 'Failed to open visual selector'
    visualSelectorLoading.value = false
  }
}

function convertToSelectedFields(nodeFields: Record<string, any>): SelectedField[] {
  const selected: SelectedField[] = []
  
  for (const [fieldName, fieldConfig] of Object.entries(nodeFields)) {
    const config = fieldConfig as any
    
    // Handle key-value pairs
    if (config.extractions && Array.isArray(config.extractions)) {
      selected.push({
        name: fieldName,
        selector: '',
        type: 'text',
        mode: 'key-value-pairs',
        multiple: false,
        attributes: { extractions: config.extractions }
      })
      continue
    }
    
    // Regular field
    selected.push({
      name: fieldName,
      selector: config.selector || '',
      type: config.type === 'attr' ? 'attribute' : config.type || 'text',
      attribute: config.attribute || '',
      multiple: config.multiple || false,
      mode: config.multiple ? 'list' : 'single'
    })
  }
  
  return selected
}

function importFromVisualSelector(selectedFields: SelectedField[]) {
  const newFields = { ...fields.value }
  
  selectedFields.forEach(field => {
    if (field.mode === 'key-value-pairs' && field.attributes?.extractions) {
      // Key-value pairs
      newFields[field.name] = {
        selector: '',
        type: 'text',
        attribute: '',
        transform: 'none',
        default_value: '',
        multiple: false,
        limit: 0,
        fields: '',
        extractions: [...field.attributes.extractions]
      }
    } else {
      // Regular field
      newFields[field.name] = {
        selector: field.selector,
        type: field.type === 'attribute' ? 'attr' : field.type,
        attribute: field.attribute || '',
        transform: 'none',
        default_value: '',
        multiple: field.multiple || false,
        limit: 0,
        fields: '',
        extractions: ''
      }
    }
  })
  
  emit('update:modelValue', newFields)
}

async function closeVisualSelector() {
  if (stopPolling) {
    stopPolling()
    stopPolling = null
  }
  
  if (visualSelectorSessionId.value) {
    try {
      await selectorApi.closeSession(visualSelectorSessionId.value)
    } catch (error) {
      console.error('Error closing selector session:', error)
    }
    visualSelectorSessionId.value = null
  }
  
  isVisualSelectorOpen.value = false
  visualSelectorError.value = null
}
</script>

<template>
  <div class="space-y-3">
    <!-- Help Banner (when no fields) -->
    <div 
      v-if="Object.keys(fields).length === 0" 
      class="p-3 bg-gradient-to-r from-purple-50 to-blue-50 dark:from-purple-950/30 dark:to-blue-950/30 border-2 border-purple-200 dark:border-purple-800 rounded-lg"
    >
      <div class="text-sm font-semibold text-purple-900 dark:text-purple-100 mb-2">
        ✨ Array Field Extraction
      </div>
      <div class="text-xs text-purple-700 dark:text-purple-300 space-y-1">
        <div>• <strong>Single values:</strong> Extract one value per field (default)</div>
        <div>• <strong>Simple arrays:</strong> Enable "multiple" to extract arrays</div>
        <div>• <strong>Object arrays:</strong> Use "fields" to extract structured data</div>
      </div>
    </div>
    
    <!-- Visual Selector Status Banner -->
    <div v-if="isVisualSelectorOpen" class="p-3 bg-blue-50 dark:bg-blue-950/30 border-2 border-blue-200 dark:border-blue-800 rounded-lg">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <div class="h-2 w-2 bg-blue-500 rounded-full animate-pulse"></div>
          <span class="text-sm font-medium text-blue-900 dark:text-blue-100">Visual Selector Active</span>
        </div>
        <Button
          type="button"
          size="sm"
          variant="outline"
          @click="closeVisualSelector"
          class="border-blue-300 hover:bg-blue-100"
        >
          Close Session
        </Button>
      </div>
      <p class="text-xs text-blue-700 dark:text-blue-300 mt-2">
        Select elements in the browser window. Selected fields will automatically appear below.
      </p>
    </div>
    
    <!-- Error Message -->
    <div v-if="visualSelectorError" class="p-3 bg-red-50 dark:bg-red-950/30 border-2 border-red-200 dark:border-red-800 rounded-lg">
      <p class="text-sm text-red-900 dark:text-red-100">{{ visualSelectorError }}</p>
    </div>
    
    <!-- Toolbar -->
    <div class="sticky top-0 z-10 bg-background pb-2 space-y-2 border-b border-border">
      <div class="flex items-center gap-2">
        <Button
          type="button"
          size="sm"
          variant="default"
          @click="openVisualSelector"
          :disabled="visualSelectorLoading || isVisualSelectorOpen"
          class="flex-1"
        >
          <MousePointerClick class="h-4 w-4 mr-2" />
          {{ visualSelectorLoading ? 'Opening Browser...' : 'Visual Selector' }}
        </Button>
        
        <Button
          type="button"
          size="sm"
          variant="outline"
          @click="addField"
        >
          <Plus class="h-4 w-4" />
        </Button>
        
        <Button
          v-if="Object.keys(fields).length > 0"
          type="button"
          size="sm"
          variant="outline"
          @click="expandAll"
          title="Expand all fields"
        >
          <ChevronDown class="h-4 w-4" />
        </Button>
        
        <Button
          v-if="Object.keys(fields).length > 0"
          type="button"
          size="sm"
          variant="outline"
          @click="collapseAll"
          title="Collapse all fields"
        >
          <ChevronUp class="h-4 w-4" />
        </Button>
      </div>
      
      <!-- Search Bar (show when > 3 fields) -->
      <div v-if="Object.keys(fields).length > 3" class="relative">
        <Input
          v-model="searchQuery"
          placeholder="Search fields by name..."
          class="pr-16"
        />
        <div class="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">
          {{ visibleFields.length }}/{{ Object.keys(fields).length }}
        </div>
      </div>
    </div>
    
    <!-- Field Cards -->
    <div class="space-y-3">
      <FieldCard
        v-for="(fieldName, index) in visibleFields"
        :key="`field-${index}`"
        :field-name="fieldName"
        :field-data="fields[fieldName]"
        :schema="schema"
        :collapsed="collapsedFields.has(fieldName)"
        :index="index"
        @update:field-data="updateFieldData(fieldName, $event)"
        @update:field-name="renameField"
        @delete="removeField(fieldName)"
        @duplicate="duplicateField(fieldName)"
        @toggle-collapse="toggleFieldCollapse(fieldName)"
      />
    </div>
    
    <!-- Empty state after search -->
    <div 
      v-if="searchQuery && visibleFields.length === 0"
      class="p-6 text-center text-muted-foreground"
    >
      No fields found matching "{{ searchQuery }}"
    </div>
  </div>
</template>
