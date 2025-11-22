<template>
  <div id="crawlify-selector-overlay" class="absolute inset-0 pointer-events-none z-[999999] font-sans"
       :style="{ minHeight: '100vh', width: '100%', top: 0, left: 0 }">
    <!-- Highlights overlay (disabled when dialogs are open) -->
    <HighlightOverlay
        v-if="!isDialogOpen"
        :hovered-element="hoveredElement"
        :locked-element="lockedElement"
        :selected-fields="selectedFields"
        :test-results="testResults"
        :current-field-type="currentFieldType"
        :current-field-attribute="currentFieldAttribute"
        :hovered-element-count="hoveredElementCount"
        @navigate="handleNavigate"
    />

    <!-- Control Panel -->
    <ControlPanel
        ref="controlPanelRef"
        v-model:field-name="currentFieldName"
        v-model:field-type="currentFieldType"
        v-model:field-attribute="currentFieldAttribute"
        v-model:mode="currentMode"
        :selected-fields="selectedFields"
        :hovered-element-count="hoveredElementCount"
        :hovered-element-validation="hoveredElementValidation"
        :live-preview-samples="livePreviewSamples"
        :selector-analysis="selectorAnalysis"
        :detailed-view-field="detailedViewField"
        :detailed-view-tab="detailedViewTab"
        :edit-mode="editMode"
        :test-results="testResults"
        @add-field="(transforms) => addField(transforms)"
        @update-field="updateField"
        @update-k-v-field="updateKVField"
        @load-field-for-edit="loadFieldForEdit"
        @load-k-v-field-for-edit="loadKVFieldForEdit"
        @add-key-value-field="addKeyValueField"
        @remove-field="removeField"
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
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch, provide, nextTick } from 'vue'
import ControlPanel from './components/ControlPanel.vue'
import HighlightOverlay from './components/HighlightOverlay.vue'
import { useElementSelection } from './composables/useElementSelection'
import { useNavigationPrevention } from './composables/useNavigationPrevention'
import { useKeyValueSelection } from './composables/useKeyValueSelection'
import { generateSelector } from './utils/selectorGenerator'
import type { FieldType } from './types'

const controlPanelRef = ref<InstanceType<typeof ControlPanel> | null>(null)
const isDialogOpen = ref(false)

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

// Initialize on mount
onMounted(() => {
  initNavigationPrevention()
  document.addEventListener('click', handleKeyValueClick, true)
})

// Cleanup on unmount
onBeforeUnmount(() => {
  cleanupNavigationPrevention()
  document.removeEventListener('click', handleKeyValueClick, true)
})

// Expose getSelections to parent
defineExpose({
  getSelections
})
</script>
