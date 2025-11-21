<template>
  <div class="absolute inset-0 pointer-events-none z-[999999] font-sans" :style="{ minHeight: '100vh', width: '100%', top: 0, left: 0 }">
    <!-- Highlights overlay -->
    <HighlightOverlay
      :hovered-element="hoveredElement"
      :locked-element="lockedElement"
      :selected-fields="selectedFields"
      :test-results="testResults"
    />
    
    <!-- Control Panel -->
    <ControlPanel
      v-model:mode="mode"
      v-model:field-name="currentFieldName"
      v-model:field-type="currentFieldType"
      v-model:field-attribute="currentFieldAttribute"
      :selected-fields="selectedFields"
      :hovered-element-count="hoveredElementCount"
      :hovered-element-validation="hoveredElementValidation"
      :detailed-view-field="detailedViewField"
      :detailed-view-tab="detailedViewTab"
      :edit-mode="editMode"
      :test-results="testResults"
      @add-field="addField"
      @remove-field="removeField"
      @open-detailed-view="openDetailedView"
      @close-detailed-view="closeDetailedView"
      @switch-tab="switchTab"
      @enable-edit-mode="enableEditMode"
      @save-edit="saveEdit"
      @cancel-edit="cancelEdit"
      @test-selector="testSelectorInline"
      @scroll-to-result="scrollToTestResult"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, onBeforeUnmount } from 'vue'
import ControlPanel from './components/ControlPanel.vue'
import HighlightOverlay from './components/HighlightOverlay.vue'
import { useElementSelection } from './composables/useElementSelection'
import { useNavigationPrevention } from './composables/useNavigationPrevention'

const {
  hoveredElement,
  hoveredElementCount,
  hoveredElementValidation,
  lockedElement,
  selectedFields,
  mode,
  currentFieldName,
  currentFieldType,
  currentFieldAttribute,
  detailedViewField,
  detailedViewTab,
  editMode,
  testResults,
  addField,
  removeField,
  openDetailedView,
  closeDetailedView,
  switchTab,
  enableEditMode,
  saveEdit,
  cancelEdit,
  testSelectorInline,
  scrollToTestResult,
  getSelections
} = useElementSelection()

const { initNavigationPrevention, cleanupNavigationPrevention } = useNavigationPrevention()

// Initialize on mount
onMounted(() => {
  initNavigationPrevention()
})

// Cleanup on unmount
onBeforeUnmount(() => {
  cleanupNavigationPrevention()
})

// Expose getSelections to parent
defineExpose({
  getSelections
})
</script>
