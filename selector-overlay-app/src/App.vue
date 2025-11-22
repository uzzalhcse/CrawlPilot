<template>
  <div class="absolute inset-0 pointer-events-none z-[999999] font-sans"
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
        @load-field-for-edit="loadFieldForEdit"
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
import { onMounted, onBeforeUnmount, ref, watch, provide } from 'vue'
import ControlPanel from './components/ControlPanel.vue'
import HighlightOverlay from './components/HighlightOverlay.vue'
import { useElementSelection } from './composables/useElementSelection'
import { useNavigationPrevention } from './composables/useNavigationPrevention'
import { useKeyValueSelection } from './composables/useKeyValueSelection'

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
  const fieldIndex = selectedFields.value.findIndex(f => f.id === data.id)
  if (fieldIndex !== -1) {
    const field = selectedFields.value[fieldIndex]
    selectedFields.value[fieldIndex] = {
      ...field,
      name: currentFieldName.value.trim(),
      type: currentFieldType.value,
      attribute: currentFieldType.value === 'attribute' ? currentFieldAttribute.value : undefined,
      transforms: Object.keys(data.transforms).length > 0 ? data.transforms : undefined,
      mode: currentMode.value
    }
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
