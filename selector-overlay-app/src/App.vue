<template>
  <div class="absolute inset-0 pointer-events-none z-[999999] font-sans" :style="{ minHeight: '100vh', width: '100%', top: 0, left: 0 }">
    <!-- Highlights overlay -->
    <HighlightOverlay
      :hovered-element="hoveredElement"
      :locked-element="lockedElement"
      :selected-fields="selectedFields"
      :test-results="testResults"
      :current-field-type="currentFieldType"
      :current-field-attribute="currentFieldAttribute"
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
      :detailed-view-field="detailedViewField"
      :detailed-view-tab="detailedViewTab"
      :edit-mode="editMode"
      :test-results="testResults"
      @add-field="addField"
      @add-key-value-field="addKeyValueField"
      @remove-field="removeField"
      @open-detailed-view="openDetailedView"
      @close-detailed-view="closeDetailedView"
      @switch-tab="switchTab"
      @enable-edit-mode="enableEditMode"
      @save-edit="saveEdit"
      @cancel-edit="cancelEdit"
      @test-selector="(field) => testSelectorInline(field.selector, field)"
      @scroll-to-result="scrollToTestResult"
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
  getSelections
} = useElementSelection()

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
        console.log('Selecting key element:', target)
        kvSelection.selectKeyElement(target)
      } else if (kvSelection.isSelectingValues.value) {
        console.log('Selecting value element:', target)
        kvSelection.selectValueElement(target)
      }
    }
  }
}

const handleNavigate = (element: Element) => {
  navigateToElement(element)
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
