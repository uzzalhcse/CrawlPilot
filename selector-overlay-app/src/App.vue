<template>
  <div id="crawlify-selector-overlay" class="absolute inset-0 pointer-events-none z-[999999] font-sans"
       :style="{ minHeight: '100vh', width: '100%', top: 0, left: 0 }">
    <!-- Floating Tooltip for Click-First Workflow -->
    <FloatingTooltip
        v-if="tooltipState"
        :visible="tooltipState.visible"
        :position="tooltipState.position"
        :suggested-name="tooltipState.suggestedName"
        :suggested-selector="tooltipState.suggestedSelector"
        :match-count="tooltipState.matchCount"
        @quick-add="handleQuickAdd"
        @customize="handleCustomize"
        @dismiss="handleDismissTooltip"
    />

    <!-- Highlights overlay (disabled when dialogs are open) -->
    <HighlightOverlay
        v-if="!isDialogOpen && !tooltipState"
        :hovered-element="hoveredElement"
        :locked-element="lockedElement"
        :selected-fields="selectedFields"
        :test-results="testResults"
        :current-field-type="currentFieldType"
        :current-field-attribute="currentFieldAttribute"
        :hovered-element-count="hoveredElementCount"
        @navigate="handleNavigate"
        @edit-field="handleEditField"
    />

    <!-- Control Panel -->
    <ControlPanel
        ref="controlPanelRef"
        :field-name="currentFieldName"
        :field-type="currentFieldType"
        :field-attribute="currentFieldAttribute"
        :hovered-element-count="hoveredElementCount"
        :live-preview-samples="livePreviewSamples"
        :selector-analysis="selectorAnalysis"
        :selected-fields="selectedFields"
        :test-results="testResults"
        :detailed-view-field="detailedViewField"
        :mode="currentMode"
        @update:field-name="currentFieldName = $event"
        @update:field-type="currentFieldType = $event as FieldType"
        @update:field-attribute="currentFieldAttribute = $event"
        @update:mode="currentMode = $event"
        @add-field="(transforms) => addField(transforms)"
        @update-field="updateField"
        @remove-field="removeField"
        @add-key-value-field="addKeyValueField"
        @update-k-v-field="updateKVField"
        @load-field-for-edit="loadFieldForEdit"
        @load-k-v-field-for-edit="loadKVFieldForEdit"
        @open-detailed-view="openDetailedView"
        @close-detailed-view="closeDetailedView"
        @switch-tab="switchTab"
        @enable-edit-mode="enableEditMode"
        @save-edit="saveEdit"
        @cancel-edit="cancelEditField"
        @test-selector="(field) => testSelectorInline(field.selector, field)"
        @scroll-to-result="scrollToTestResult"
        @use-alternative-selector="useAlternativeSelector"
        @dialog-state-change="(open) => isDialogOpen = open"
        @form-state-change="(open) => showFieldForm = open"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch, provide, nextTick } from 'vue'
import ControlPanel from './components/ControlPanel.vue'
import HighlightOverlay from './components/HighlightOverlay.vue'
import FloatingTooltip from './components/FloatingTooltip.vue'
import { useElementSelection } from './composables/useElementSelection'
import { useNavigationPrevention } from './composables/useNavigationPrevention'
import { useKeyValueSelection } from './composables/useKeyValueSelection'
import { generateSelector } from './utils/selectorGenerator'
import { suggestFieldName, makeUniqueFieldName } from './utils/fieldNameSuggester'
import type { FieldType, SelectedField } from './types'

const controlPanelRef = ref<InstanceType<typeof ControlPanel> | null>(null)
const isDialogOpen = ref(false)
const showFieldForm = ref(false) // Track if add/edit form is open

// Tooltip state for click-first workflow
const tooltipState = ref<{
  visible: boolean
  element: Element
  position: { x: number, y: number }
  suggestedName: string
  suggestedSelector: string
  matchCount: number
} | null>(null)

const {
  hoveredElement,
  hoveredElementCount,
  hoveredElementValidation,
  lockedElement,
  selectedFields,
  currentFieldName,
  currentFieldType,
  currentFieldAttribute,
  currentMode,
  detailedViewField,
  detailedViewTab,
  editMode,
  testResults,
  livePreviewSamples,
  selectorAnalysis,
  addField,
  quickAddField,
  addKeyValueField,
  removeField,
  openDetailedView,
  closeDetailedView,
  switchTab,
  enableEditMode,
  saveEdit,
  cancelEdit,
  testSelectorInline,
  scrollToTestResult,
  navigateToElement,
  getSelections,
  updateLivePreview,
  updateSelectorAnalysis,
  useAlternativeSelector
} = useElementSelection(isDialogOpen)

// Update live preview and selector analysis when field type, attribute, or locked element changes
watch([currentFieldType, currentFieldAttribute, lockedElement], () => {
  updateLivePreview()
  updateSelectorAnalysis()
}, { immediate: true })

const kvSelection = useKeyValueSelection()

// Provide kvSelection to child components
provide('kvSelection', kvSelection)

// Handle key-value mode clicks
watch(currentMode, (newMode) => {
  if (newMode !== 'key-value-pairs') {
    kvSelection.reset()
  }
})

// Handle clicks for key-value selection
const handleKeyValueClick = (e: MouseEvent) => {
  const target = e.target as Element
  
  // Check if the KeyValuePairSelector is in selection mode (works for both add and edit modes)
  if (kvSelection.isSelectingKeys.value || kvSelection.isSelectingValues.value) {
    // Don't handle clicks inside the control panel itself
    if (target && !target.closest('.fixed.top-5.right-5')) {
      e.preventDefault()
      e.stopPropagation()
      
      if (kvSelection.isSelectingKeys.value) {
        kvSelection.selectKeyElement(target)
      } else if (kvSelection.isSelectingValues.value) {
        kvSelection.selectValueElement(target)
      }
    }
  }
}

const handleNavigate = (element: Element) => {
  navigateToElement(element)
}

// Handler for clicking on highlighted field badges
const handleEditField = (field: SelectedField) => {
  // Trigger edit mode in control panel
  if (controlPanelRef.value) {
    (controlPanelRef.value as any).openEditFieldForm?.(field)
  }
}

// Handler for quick-add from floating tooltip
const handleQuickAdd = (editedName: string, editedSelector: string) => {
  if (!tooltipState.value) return
  
  const { element } = tooltipState.value
  
  // Make field name unique if conflicts exist
  const existingNames = selectedFields.value.map(f => f.name)
  const uniqueName = makeUniqueFieldName(editedName, existingNames)
  
  // Quick add field with custom selector if provided
  quickAddField(element, uniqueName)
  
  // Dismiss tooltip
  tooltipState.value = null
}

// Handler for customize from floating tooltip  
const handleCustomize = (editedName: string, editedSelector: string) => {
  if (!tooltipState.value) return
  
  const { element } = tooltipState.value
  
  // Make field name unique
  const existingNames = selectedFields.value.map(f => f.name)
  const uniqueName = makeUniqueFieldName(editedName, existingNames)
  
  // Lock element first
  lockedElement.value = element
  updateLivePreview()
  updateSelectorAnalysis()
  
  // Dismiss tooltip
  tooltipState.value = null
  
  // Trigger the add field form, then set the field name
  nextTick(() => {
    if (controlPanelRef.value) {
      (controlPanelRef.value as any).openAddFieldForm?.()
      // Set field name AFTER form opens (since openAddFieldForm resets it)
      nextTick(() => {
        currentFieldName.value = uniqueName
      })
    }
  })
}

// Handler to dismiss tooltip
const handleDismissTooltip = () => {
  tooltipState.value = null
}

// Update existing field
const updateField = (data: { id: string; transforms: any }) => {
  console.log('🔄 [Update Field] Starting update for field ID:', data.id)
  console.log('  - Transforms:', data.transforms)
  
  const fieldIndex = selectedFields.value.findIndex(f => f.id === data.id)
  if (fieldIndex !== -1) {
    const field = selectedFields.value[fieldIndex]
    console.log('  - Found field at index:', fieldIndex)
    console.log('  - Current field:', field)
    console.log('  - Locked element:', lockedElement.value)
    
    // Generate new selector from locked element
    let newSelector = field.selector
    let newSampleValue = field.sampleValue
    let newMatchCount = field.matchCount
    
    if (lockedElement.value) {
      newSelector = generateSelector(lockedElement.value)
      newSampleValue = getSampleValue(lockedElement.value, currentFieldType.value, currentFieldAttribute.value)
      
      try {
        newMatchCount = document.querySelectorAll(newSelector).length
      } catch {
        newMatchCount = 0
      }
      
      console.log('  - New selector generated:', newSelector)
      console.log('  - New sample value:', newSampleValue)
      console.log('  - New match count:', newMatchCount)
    } else {
      console.log('  - No locked element, keeping existing selector')
    }
    
    selectedFields.value[fieldIndex] = {
      ...field,
      name: currentFieldName.value.trim(),
      selector: newSelector,
      type: currentFieldType.value,
      attribute: currentFieldType.value === 'attribute' ? currentFieldAttribute.value : undefined,
      sampleValue: newSampleValue,
      matchCount: newMatchCount,
      transforms: Object.keys(data.transforms).length > 0 ? data.transforms : undefined,
      mode: currentMode.value
    }
    
    console.log('  - ✅ Updated field:', selectedFields.value[fieldIndex])
  } else {
    console.log('  - ❌ Field not found with ID:', data.id)
  }
}

// Helper function to get sample value
const getSampleValue = (element: Element, type: FieldType, attribute?: string): string => {
  switch (type) {
    case 'text':
      return element.textContent?.trim() || ''
    case 'attribute':
      return attribute ? element.getAttribute(attribute) || '' : ''
    case 'html':
      return element.innerHTML
    default:
      return ''
  }
}

// Update existing K-V field
const updateKVField = (data: { id: string; fieldName: string; extractions: any[] }) => {
  console.log('🔄 [Update K-V Field] Starting update for field ID:', data.id)
  console.log('  - Field name:', data.fieldName)
  console.log('  - Extractions count:', data.extractions.length)
  console.log('  - Extractions:', data.extractions)
  
  const fieldIndex = selectedFields.value.findIndex(f => f.id === data.id)
  if (fieldIndex !== -1) {
    const field = selectedFields.value[fieldIndex]
    console.log('  - Found field at index:', fieldIndex)
    console.log('  - Current field:', field)
    console.log('  - Current extractions:', field.attributes?.extractions)
    
    selectedFields.value[fieldIndex] = {
      ...selectedFields.value[fieldIndex],
      name: data.fieldName,
      attributes: {
        extractions: data.extractions.map(ext => ({
          key_selector: ext.key_selector,
          value_selector: ext.value_selector,
          key_type: ext.key_type,
          value_type: ext.value_type,
          key_attribute: ext.key_attribute,
          value_attribute: ext.value_attribute,
          transform: ext.transform
        }))
      }
    }
    
    console.log('  - ✅ Updated K-V field:', selectedFields.value[fieldIndex])
    console.log('  - New extractions count:', selectedFields.value[fieldIndex].attributes?.extractions?.length)
  } else {
    console.log('  - ❌ Field not found with ID:', data.id)
  }
}

// Load field for editing
const loadFieldForEdit = (field: any) => {
  // Find elements matching this selector and lock the first one
  try {
    const elements = document.querySelectorAll(field.selector)
    if (elements.length > 0) {
      lockedElement.value = elements[0] as Element
      updateLivePreview()
      updateSelectorAnalysis()
    }
  } catch (error) {
    console.error('Failed to load field for edit:', error)
  }
}

// Load K-V field for editing
const loadKVFieldForEdit = async (field: any) => {
  // Load the extraction pairs into the K-V selector
  if (field.attributes?.extractions && controlPanelRef.value) {
    // Wait for the tab to switch and K-V selector to be mounted
    // Sometimes need multiple ticks for component to fully mount
    await nextTick()
    await nextTick()
    
    // Try accessing with a slight delay if still not available
    let kvSelector = (controlPanelRef.value as any).kvSelectorRef
    
    if (!kvSelector) {
      await new Promise(resolve => setTimeout(resolve, 50))
      kvSelector = (controlPanelRef.value as any).kvSelectorRef
    }
    
    if (kvSelector && kvSelector.loadFieldData) {
      kvSelector.loadFieldData(field.attributes.extractions)
    }
  }
}

// Cancel edit and clear form
const cancelEditField = () => {
  currentFieldName.value = ''
  currentFieldAttribute.value = ''
  lockedElement.value = null
  hoveredElement.value = null
}

const { initNavigationPrevention, cleanupNavigationPrevention } = useNavigationPrevention()

// Handler for page clicks (for floating tooltip workflow)
const handlePageClick = (e: MouseEvent) => {
  // Don't show tooltip if panel or dialog is in the way
  if (isDialogOpen.value) return
  
  const target = e.target as Element
  if (!target || target.closest('#crawlify-selector-overlay')) return
  
  // Skip tooltip if in key-value mode
  if (currentMode.value === 'key-value-pairs') return
  
  // Check if form is open via controlPanelRef
  const isPanelFormOpen = controlPanelRef.value && (controlPanelRef.value as any).$el?.querySelector('.field-form-active')
  
  // If form is open, use old behavior: just lock the element without tooltip
  if (isPanelFormOpen || showFieldForm.value) {
    e.preventDefault()
    e.stopPropagation()
    lockedElement.value = target
    hoveredElement.value = target
    return
  }
  
  // Form is closed: show tooltip with suggestions
  e.preventDefault()
  e.stopPropagation()
  
  const rect = target.getBoundingClientRect()
  const selector = generateSelector(target)
  
  // Count matches
  let matchCount = 0
  try {
    matchCount = document.querySelectorAll(selector).length
  } catch (e) {
    matchCount = 1
  }
  
  // Generate field name suggestion
  const existingNames = selectedFields.value.map(f => f.name)
  const suggestion = suggestFieldName(target, existingNames)
  
  // Position tooltip near the clicked element
  const tooltipX = rect.left + window.scrollX
  const tooltipY = rect.top + window.scrollY - 10
  
  tooltipState.value = {
    visible: true,
    element: target,
    position: { x: tooltipX, y: tooltipY },
    suggestedName: suggestion.name,
    suggestedSelector: selector,
    matchCount
  }
}

// Initialize on mount
onMounted(() => {
  initNavigationPrevention()
  document.addEventListener('click', handlePageClick, true)
  document.addEventListener('click', handleKeyValueClick, true)
})

// Cleanup on unmount
onBeforeUnmount(() => {
  cleanupNavigationPrevention()
  document.removeEventListener('click', handlePageClick, true)
  document.removeEventListener('click', handleKeyValueClick, true)
})

// Expose getSelections to parent
defineExpose({
  getSelections
})
</script>
