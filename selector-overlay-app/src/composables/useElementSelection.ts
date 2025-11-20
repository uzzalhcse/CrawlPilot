import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import type { SelectedField, SelectionMode, FieldType, ElementInfo, TestResult } from '../types'
import { generateSelector } from '../utils/selectorGenerator'

export function useElementSelection() {
  // State
  const hoveredElement = ref<Element | null>(null)
  const lockedElement = ref<Element | null>(null)
  const selectedFields = ref<SelectedField[]>([])
  const mode = ref<SelectionMode>('single')
  const currentFieldName = ref('')
  const currentFieldType = ref<FieldType>('text')
  const currentFieldAttribute = ref('')
  const detailedViewField = ref<SelectedField | null>(null)
  const detailedViewTab = ref<'preview' | 'edit'>('preview')
  const editMode = ref(false)
  const testResults = ref<TestResult[]>([])

  // Computed
  const hoveredElementSelector = computed(() => {
    if (!hoveredElement.value) return ''
    return generateSelector(hoveredElement.value)
  })

  const hoveredElementCount = computed(() => {
    if (!hoveredElementSelector.value) return 0
    try {
      return document.querySelectorAll(hoveredElementSelector.value).length
    } catch {
      return 0
    }
  })

  const hoveredElementValidation = computed(() => {
    const count = hoveredElementCount.value
    if (count === 0) {
      return { isValid: false, isUnique: false, message: 'Selector matches no elements' }
    }
    if (count === 1) {
      return { isValid: true, isUnique: true, message: 'Unique selector' }
    }
    if (mode.value === 'list') {
      return { isValid: true, isUnique: false, message: `List selector (${count} items)` }
    }
    return { isValid: true, isUnique: false, message: `Multiple matches (${count})` }
  })

  const lockedElementSelector = computed(() => {
    if (!lockedElement.value) return ''
    return generateSelector(lockedElement.value)
  })

  // Methods
  const handleMouseMove = (e: MouseEvent) => {
    if (lockedElement.value) return
    
    const target = e.target as Element
    if (target && !target.closest('#crawlify-selector-overlay')) {
      hoveredElement.value = target
    }
  }

  const handleClick = (e: MouseEvent) => {
    const target = e.target as Element
    // Don't intercept clicks on the control panel or if detailed view is open
    if (target && !target.closest('#crawlify-selector-overlay') && !detailedViewField.value) {
      e.preventDefault()
      e.stopPropagation()
      lockedElement.value = target
    }
  }

  const addField = () => {
    if (!lockedElement.value || !currentFieldName.value.trim()) return

    const selector = generateSelector(lockedElement.value)
    const sampleValue = getSampleValue(lockedElement.value, currentFieldType.value, currentFieldAttribute.value)
    
    const field: SelectedField = {
      id: `field-${Date.now()}`,
      name: currentFieldName.value.trim(),
      selector,
      type: currentFieldType.value,
      attribute: currentFieldType.value === 'attribute' ? currentFieldAttribute.value : undefined,
      timestamp: Date.now(),
      sampleValue,
      matchCount: hoveredElementCount.value
    }

    selectedFields.value.push(field)
    
    // Reset form and clear locked element
    currentFieldName.value = ''
    currentFieldAttribute.value = ''
    lockedElement.value = null
    hoveredElement.value = null
  }

  const removeField = (fieldId: string) => {
    selectedFields.value = selectedFields.value.filter(f => f.id !== fieldId)
  }

  const openDetailedView = (field: SelectedField) => {
    detailedViewField.value = field
    detailedViewTab.value = 'preview'
    editMode.value = false
  }

  const closeDetailedView = () => {
    detailedViewField.value = null
    editMode.value = false
    testResults.value = []
    detailedViewTab.value = 'preview'
  }

  const switchTab = (tab: 'preview' | 'edit') => {
    detailedViewTab.value = tab
    if (tab === 'edit') {
      editMode.value = true
    }
  }

  const enableEditMode = () => {
    editMode.value = true
  }

  const saveEdit = (updatedField: Partial<SelectedField>) => {
    if (!detailedViewField.value) return
    
    const index = selectedFields.value.findIndex(f => f.id === detailedViewField.value!.id)
    if (index !== -1) {
      selectedFields.value[index] = {
        ...selectedFields.value[index],
        ...updatedField
      }
      detailedViewField.value = selectedFields.value[index]
    }
    editMode.value = false
    detailedViewTab.value = 'preview'
  }

  const cancelEdit = () => {
    editMode.value = false
    detailedViewTab.value = 'preview'
  }

  const testSelectorInline = (selector: string) => {
    try {
      const elements = document.querySelectorAll(selector)
      testResults.value = Array.from(elements).slice(0, 10).map((el, index) => ({
        element: el as Element,
        index,
        value: el.textContent?.trim() || ''
      }))
    } catch (error) {
      testResults.value = []
      console.error('Invalid selector:', error)
    }
  }

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

  const getSelections = () => {
    return selectedFields.value
  }

  // Keyboard handler
  const handleKeyDown = (e: KeyboardEvent) => {
    // ESC to unlock element or close detailed view
    if (e.key === 'Escape') {
      if (detailedViewField.value) {
        closeDetailedView()
      } else if (lockedElement.value) {
        lockedElement.value = null
        hoveredElement.value = null
      }
    }
    
    // Enter to add field when element is locked
    if (e.key === 'Enter' && lockedElement.value && currentFieldName.value.trim()) {
      addField()
    }
  }

  // Lifecycle
  onMounted(() => {
    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('click', handleClick, true)
    document.addEventListener('keydown', handleKeyDown)
  })

  onBeforeUnmount(() => {
    document.removeEventListener('mousemove', handleMouseMove)
    document.removeEventListener('click', handleClick, true)
    document.removeEventListener('keydown', handleKeyDown)
  })

  return {
    hoveredElement,
    hoveredElementSelector,
    hoveredElementCount,
    hoveredElementValidation,
    lockedElement,
    lockedElementSelector,
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
    getSelections
  }
}
