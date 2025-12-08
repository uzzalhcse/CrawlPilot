<script setup lang="ts">
import { ref, onMounted, onUnmounted, provide, computed } from 'vue'
import { useFieldManager } from './composables/useFieldManager'
import { useElementSelection } from './composables/useElementSelection'
import { useHighlightManager } from './composables/useHighlightManager'
import { highlightElement, removeHighlight } from './utils/selector'
import FieldBadge from './components/FieldBadge.vue'
import FieldModal from './components/FieldModal.vue'
import PairModeToggle from './components/PairModeToggle.vue'
import PairStatusIndicator from './components/PairStatusIndicator.vue'
import PairConfigModal from './components/PairConfigModal.vue'
import PairBadge from './components/PairBadge.vue'
import Toolbar from './components/Toolbar.vue'
import DataPreview from './components/DataPreview.vue'
import { Toaster } from '@/components/ui/sonner'
import { toast } from 'vue-sonner'
import type { Field, FieldConfig, KeyValuePairConfig, KeyValueGroup, KeyValuePair } from './types'
import 'vue-sonner/style.css'

const { fields, keyValueGroups, addField, updateField, removeField, addKeyValuePair, undoLastPair, exportToJSON, removeKeyValuePair } = useFieldManager()
const { selectedElement, selectElement, clearSelection } = useElementSelection()
const { refreshSavedHighlights, cleanupPairHighlights } = useHighlightManager(fields, keyValueGroups)

// Provide fields to child components for validation
provide('fields', fields)

// Modal State
const showModal = ref(false)
const modalMode = ref<'add' | 'edit'>('add')
const editingField = ref<Field | null>(null)
const clickPosition = ref({ x: 0, y: 0 })

// Preview Drawer State
const showPreview = ref(false)

// Pair Mode State
const selectionMode = ref<'single' | 'pair'>('single')
const showPairModal = ref(false)
const pairState = ref<{
  step: 'idle' | 'awaiting_label' | 'awaiting_value'
  labelElement: HTMLElement | null
  valueElement: HTMLElement | null
  currentGroup: string
}>({
  step: 'idle',
  labelElement: null,
  valueElement: null,
  currentGroup: 'attributes'
})

// Computed
const allFields = computed<Field[]>(() => {
  // Convert KeyValueGroups to KeyValueGroupFields
  const groupFields: Field[] = keyValueGroups.value.map(group => ({
    id: group.id,
    name: group.name,
    fieldType: 'keyvalue',
    config: {
      output_format: group.output_format
    },
    pairs: group.pairs
  }))
  
  return [...fields.value, ...groupFields]
})

const pairCount = computed(() => {
  const group = keyValueGroups.value.find(g => g.name === pairState.value.currentGroup)
  return group ? group.pairs.length : 0
})

const labelPreviewText = computed(() => {
  if (!pairState.value.labelElement) return ''
  const text = pairState.value.labelElement.textContent?.trim() || ''
  return text.length > 20 ? text.substring(0, 20) + '...' : text
})

function handleClick(event: MouseEvent) {
  const target = event.target as HTMLElement
  
  // Check if clicking on our UI components (FieldBadge, Toolbar, etc)
  if (target.closest('[data-selectflow-ui]')) {
    return  // Don't interfere with our own UI
  }
  
  // If any modal is open, don't intercept clicks
  if (showModal.value || showPairModal.value) {
    return
  }
  
  // Ignore clicks on body or html (user likely missed the target)
  if (target === document.body || target === document.documentElement) {
    return
  }
  
  // Prevent default and stop propagation
  event.preventDefault()
  event.stopPropagation()
  
  // Route to appropriate handler based on mode
  if (selectionMode.value === 'pair') {
    handlePairModeClick(target, event)
  } else {
    handleSingleModeClick(target, event)
  }
}

function handleSingleModeClick(target: HTMLElement, event: MouseEvent) {
  selectElement(target)
  
  // Open Add Modal
  if (selectedElement.value) {
    // Capture click position for Popover positioning
    clickPosition.value = { x: event.clientX, y: event.clientY }
    
    modalMode.value = 'add'
    editingField.value = null
    showModal.value = true
  }
}

function handlePairModeClick(target: HTMLElement, event: MouseEvent) {
  if (pairState.value.step === 'idle' || pairState.value.step === 'awaiting_label') {
    // Step 1: Capture label
    pairState.value.labelElement = target
    pairState.value.step = 'awaiting_value'
    
    // Add visual indicator to label
    target.classList.add('selectflow-highlight-label')
    
    toast.info('Label selected. Now click the value element.')
  } else if (pairState.value.step === 'awaiting_value' && pairState.value.labelElement) {
    // Step 2: Capture value, open configuration modal
    pairState.value.valueElement = target
    
    // Add visual indicator to value
    target.classList.add('selectflow-highlight-value')
    
    // Capture click position and open pair config modal
    clickPosition.value = { x: event.clientX, y: event.clientY }
    showPairModal.value = true
  }
}

let hoveredElement: HTMLElement | null = null

function handleMouseOver(event: MouseEvent) {
  // Don't highlight if any modal is open
  if (showModal.value || showPairModal.value) {
    if (hoveredElement) {
      removeHighlight(hoveredElement)
      hoveredElement = null
    }
    return
  }

  const target = event.target as HTMLElement
  
  // FIRST: Ignore if hovering over our UI components (FieldBadge, Toolbar, PairModeToggle, etc.)
  if (target.closest('[data-selectflow-ui]')) {
    if (hoveredElement) {
      removeHighlight(hoveredElement)
      hoveredElement = null
    }
    return
  }
  
  // Also exclude body and html
  if (target === document.body || target === document.documentElement) {
    if (hoveredElement) {
      removeHighlight(hoveredElement)
      hoveredElement = null
    }
    return
  }
  
  // Clear previous highlight if hovering over a different element
  if (hoveredElement && hoveredElement !== target) {
    // Remove hover classes from previous element
    hoveredElement.classList.remove('selectflow-highlight-hover', 'selectflow-highlight-hover-label', 'selectflow-highlight-hover-value')
  }
  
  hoveredElement = target
  
  // Apply different highlight based on mode
  if (selectionMode.value === 'pair') {
    // In pair mode, show different color based on what we're selecting
    if (pairState.value.step === 'awaiting_label') {
      // Blue for label
      target.classList.add('selectflow-highlight-hover-label')
    } else if (pairState.value.step === 'awaiting_value') {
      // Green for value
      target.classList.add('selectflow-highlight-hover-value')
    }
  } else {
    // Normal highlight for single mode
    highlightElement(target)
  }
}

function handleMouseOut(event: MouseEvent) {
  const target = event.target as HTMLElement
  if (target === hoveredElement) {
    // Remove all hover classes
    target.classList.remove('selectflow-highlight-hover', 'selectflow-highlight-hover-label', 'selectflow-highlight-hover-value')
    hoveredElement = null
  }
}

onMounted(() => {
  // Use capture phase to intercept clicks BEFORE they trigger new element selection
  // This prevents the modal from reopening when closing
  document.addEventListener('click', handleClick, true)
  document.addEventListener('mouseover', handleMouseOver, true)
  document.addEventListener('mouseout', handleMouseOut, true)
  document.addEventListener('keydown', handleKeyPress)
  
  // Initial highlight
  refreshSavedHighlights()
})

onUnmounted(() => {
  document.removeEventListener('click', handleClick, true)
  document.removeEventListener('mouseover', handleMouseOver, true)
  document.removeEventListener('mouseout', handleMouseOut, true)
  document.removeEventListener('keydown', handleKeyPress)
})

function openEditModal(field: Field) {
  // Close any existing selection/modal
  clearSelection()
  
  modalMode.value = 'edit'
  editingField.value = field
  showModal.value = true
}

function handleSave(data: { name: string, config: FieldConfig }) {
  if (modalMode.value === 'add' && selectedElement.value) {
    addField(data.name, selectedElement.value, data.config)
    toast.success(`Field "${data.name}" added successfully`)
  } else if (modalMode.value === 'edit' && editingField.value) {
    updateField(editingField.value.id, {
      name: data.name,
      config: data.config
    })
    toast.success(`Field "${data.name}" updated successfully`)
  }
  
  closeModal()
  refreshSavedHighlights()
}

function closeModal() {
  showModal.value = false
  editingField.value = null
  clearSelection()
}

function handleCopyJSON() {
  const json = exportToJSON()
  navigator.clipboard.writeText(json).then(() => {
    toast.success('Configuration copied to clipboard!')
  })
}

function handleClearAll() {
  if (confirm('Are you sure you want to clear all fields? This cannot be undone.')) {
    fields.value = []
    toast.success('All fields cleared')
  }
}

function handlePreview() {
  showPreview.value = true
}

// Pair Mode Handlers
function toggleSelectionMode() {
  if (selectionMode.value === 'single') {
    // Switch to pair mode
    selectionMode.value = 'pair'
    pairState.value.step = 'awaiting_label'
    toast.info('Pair Mode activated. Click a label element to start.')
  } else {
    // Switch back to single mode
    cancelPairMode()
  }
}

function handlePairSave(data: { groupName: string, config: KeyValuePairConfig, outputFormat: 'object' | 'array' }) {
  if (editingPair.value) {
    // Update existing pair
    const { group, pair } = editingPair.value
    
    // Update pair config
    pair.config = data.config
    
    // Update group name and output format if changed
    if (group.name !== data.groupName) {
      group.name = data.groupName
    }
    group.output_format = data.outputFormat
    
    toast.success('Pair updated')
    
    // Reset editing state
    editingPair.value = null
    showPairModal.value = false
    
    // Refresh highlights
    refreshSavedHighlights()
  } else {
    // Add new pair
    addKeyValuePair(data.groupName, pairState.value.labelElement!, pairState.value.valueElement!, data.config, data.outputFormat)
    toast.success('Pair added')
    
    // Close modal and reset state
    showPairModal.value = false
    handlePairDone()
  }
}


function handleAddAnotherPair() {
  // Clean up highlights from previous pair
  cleanupPairHighlights()
  
  // Reset pair state for next selection
  pairState.value.step = 'awaiting_label'
  pairState.value.labelElement = null
  pairState.value.valueElement = null
  
  // Close modal to allow selection
  showPairModal.value = false
  
  toast.info('Select next label element')
}

function handleRemovePair(groupId: string, pairId: string) {
  removeKeyValuePair(groupId, pairId)
  refreshSavedHighlights()
  toast.success('Pair removed')
}

// State for editing pair
const editingPair = ref<{
  group: KeyValueGroup
  pair: KeyValuePair
} | null>(null)

function handleEditPair(group: KeyValueGroup, pair: KeyValuePair) {
  // Set editing state
  editingPair.value = { group, pair }
  
  // Set pair state to show modal (but not in selection mode)
  // We need to set label/value elements for the modal to work
  pairState.value.labelElement = pair.keyElement
  pairState.value.valueElement = pair.valueElement
  
  // Open modal
  showPairModal.value = true
}

function handleUndoLastPair() {
  const success = undoLastPair(pairState.value.currentGroup)
  if (success) {
    toast.success('Last pair removed')
    refreshSavedHighlights()
  } else {
    toast.error('No pairs to undo')
  }
}

function cancelPairMode() {
  cleanupPairHighlights()
  // Clean up any hover highlights
  if (hoveredElement) {
    hoveredElement.classList.remove('selectflow-highlight-hover', 'selectflow-highlight-hover-label', 'selectflow-highlight-hover-value')
    hoveredElement = null
  }
  selectionMode.value = 'single'
  pairState.value.step = 'idle'
  pairState.value.labelElement = null
  pairState.value.valueElement = null
  showPairModal.value = false
  toast.info('Returned to Single Field mode')
}

function handlePairDone() {
  const count = pairCount.value
  cancelPairMode()
  if (count > 0) {
    toast.success(`Added ${count} ${count === 1 ? 'pair' : 'pairs'} to "${pairState.value.currentGroup}"`)
  }
}

function closePairModal() {
  showPairModal.value = false
  // Clean up the selected elements' highlights
  cleanupPairHighlights()
  // Reset elements for next pair
  pairState.value.labelElement = null
  pairState.value.valueElement = null
  // Keep step as awaiting_label so user can continue adding pairs
  pairState.value.step = 'awaiting_label'
}

// Keyboard Shortcuts
function handleKeyPress(event: KeyboardEvent) {
  // Don't interfere if modal is open or user is typing
  if (showModal.value || showPairModal.value) return
  if (event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement) return
  
  if (event.key === 'k' || event.key === 'K') {
    event.preventDefault()
    toggleSelectionMode()
  } else if (event.key === 'Escape' && selectionMode.value === 'pair') {
    event.preventDefault()
    if (pairState.value.step !== 'idle') {
      handlePairDone()
    } else {
      cancelPairMode()
    }
  }
}

// Backdrop click handler - ignore clicks on SelectFlow UI elements
function handleBackdropClick(event: MouseEvent) {
  const target = event.target as HTMLElement
  
  // Check if clicking on a SelectFlow UI component (dropdown, etc)
  if (target.closest('[data-selectflow-ui]')) {
    return  // Don't close modal if clicking on UI elements
  }
  
  // Close the appropriate modal
  if (showModal.value) {
    closeModal()
  } else if (showPairModal.value) {
    closePairModal()
  }
}
</script>

<template>
  <div class="selector-app">
    <Toolbar 
      :field-count="allFields.length" 
      @copy="handleCopyJSON" 
      @clear="handleClearAll"
      @preview="handlePreview" 
    />
    
    <!-- Mode Toggle -->
    <PairModeToggle 
      :mode="selectionMode"
      @toggle="toggleSelectionMode"
      data-selectflow-ui="toggle"
      class="fixed top-4 left-1/2 transform -translate-x-1/2 z-[450]"
    />
    
    <!-- Pair Status Indicator -->
    <PairStatusIndicator
      :step="pairState.step"
      :label-text="labelPreviewText"
      :pair-count="pairCount"
      @cancel="handlePairDone"
      @done="handlePairDone"
      @undo="handleUndoLastPair"
      data-selectflow-ui="status"
    />
    
    <FieldBadge
      v-for="field in fields"
      :key="field.id"
      :field="field"
      @edit="openEditModal"
      @remove="removeField"
    />

    <!-- Pair Badges -->
    <template v-for="group in keyValueGroups" :key="group.id">
      <PairBadge
        v-for="pair in group.pairs"
        :key="pair.id"
        :pair="pair"
        :group="group"
        @edit="handleEditPair"
        @remove="handleRemovePair"
      />
    </template>
    
    
    <!-- Overlay for Popover focus -->
    <div 
      v-if="showModal || showPairModal"
      data-selectflow-ui="overlay"
      class="fixed inset-0 bg-black/20 backdrop-blur-[2px] z-[400] transition-opacity pointer-events-auto"
      @click="handleBackdropClick"
    />
    
    <!-- Unified Field Modal (Popover) -->
    <FieldModal
      v-model:open="showModal"
      :mode="modalMode"
      :element="selectedElement"
      :field="editingField"
      :click-position="clickPosition"
      @save="handleSave"
      @cancel="closeModal"
    />
    
    <!-- Pair Config Modal -->
    <PairConfigModal
      v-model:open="showPairModal"
      :label-element="pairState.labelElement!"
      :value-element="pairState.valueElement!"
      :click-position="clickPosition"
      @save="handlePairSave"
      @add-another="handleAddAnotherPair"
      @cancel="closePairModal"
    />
    
    <!-- Data Preview Drawer -->
    <DataPreview
      v-model:open="showPreview"
      :fields="allFields"
    />
    
    <!-- Toast Notifications -->
    <Toaster />
    
    <!-- Sample Content for Testing -->

    <!-- Sample Content for Testing -->
<!--    <SampleContent />-->
  </div>
</template>

<style>


/* Pair Mode Highlights */
.selectflow-highlight-label {
  outline: 3px solid #3b82f6 !important;
  outline-offset: 2px;
  background-color: rgba(59, 130, 246, 0.1) !important;
  position: relative;
  z-index: 100;
}

.selectflow-highlight-value {
  outline: 3px solid #10b981 !important;
  outline-offset: 2px;
  background-color: rgba(16, 185, 129, 0.1) !important;
  position: relative;
  z-index: 100;
}

/* Pair Mode Hover States */
.selectflow-highlight-hover-label {
  outline: 2px dashed #3b82f6 !important;
  outline-offset: 2px;
  background-color: rgba(59, 130, 246, 0.05) !important;
  cursor: crosshair;
}

.selectflow-highlight-hover-value {
  outline: 2px dashed #10b981 !important;
  outline-offset: 2px;
  background-color: rgba(16, 185, 129, 0.05) !important;
  cursor: crosshair;
}
</style>
