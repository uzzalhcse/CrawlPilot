import { ref, computed, onMounted, onBeforeUnmount, type Ref } from 'vue'
import type { SelectedField, FieldType, ElementInfo, TestResult, SelectionMode } from '../types'
import { generateSelector, analyzeSelectorQuality, type AlternativeSelector, type SelectorQuality } from '../utils/selectorGenerator'

export function useElementSelection(isDialogOpen?: Ref<boolean>) {
  // State
  const hoveredElement = ref<Element | null>(null)
  const lockedElement = ref<Element | null>(null)
  const selectedFields = ref<SelectedField[]>([])
  const currentFieldName = ref('')
  const currentFieldType = ref<FieldType>('text')
  const currentFieldAttribute = ref('')
  const currentMode = ref<SelectionMode>('single')
  const detailedViewField = ref<SelectedField | null>(null)
  const detailedViewTab = ref<'preview' | 'edit'>('preview')
  const editMode = ref(false)
  const testResults = ref<TestResult[]>([])
  const livePreviewSamples = ref<string[]>([])
  const selectorAnalysis = ref<{
    current: SelectorQuality & { matchCount: number }
    alternatives: AlternativeSelector[]
  } | null>(null)

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
    return { isValid: true, isUnique: false, message: `${count} matches` }
  })

  const lockedElementSelector = computed(() => {
    if (!lockedElement.value) return ''
    return generateSelector(lockedElement.value)
  })

  // Methods
  const handleMouseMove = (e: MouseEvent) => {
    // Don't track hover if dialog is open
    if (isDialogOpen?.value) return
    if (lockedElement.value) return

    const target = e.target as Element
    if (target && !target.closest('#crawlify-selector-overlay')) {
      hoveredElement.value = target
    }
  }

  const handleClick = (e: MouseEvent) => {
    // Don't intercept clicks if dialog is open
    if (isDialogOpen?.value) return

    const target = e.target as Element
    // Don't intercept clicks on the control panel or if detailed view is open
    if (target && !target.closest('#crawlify-selector-overlay') && !detailedViewField.value) {
      e.preventDefault()
      e.stopPropagation()

      // Skip if in key-value mode (handled by KeyValuePairSelector)
      if (currentMode.value === 'key-value-pairs') {
        return
      }

      lockedElement.value = target
      // Update hoveredElement to match the locked element to ensure selector consistency
      hoveredElement.value = target
    }
  }

  const addField = (transforms?: Record<string, boolean>) => {
    if (!lockedElement.value || !currentFieldName.value.trim()) return

    // Generate selector from the locked element (this is the source of truth)
    const selector = generateSelector(lockedElement.value)
    const sampleValue = getSampleValue(lockedElement.value, currentFieldType.value, currentFieldAttribute.value)

    // Calculate match count based on the actual selector generated
    let matchCount = 0
    try {
      matchCount = document.querySelectorAll(selector).length
    } catch {
      matchCount = 0
    }

    const field: SelectedField = {
      id: `field-${Date.now()}`,
      name: currentFieldName.value.trim(),
      selector,
      type: currentFieldType.value,
      attribute: currentFieldType.value === 'attribute' ? currentFieldAttribute.value : undefined,
      timestamp: Date.now(),
      sampleValue,
      matchCount,
      mode: currentMode.value,
      transforms: transforms && Object.keys(transforms).length > 0 ? transforms : undefined
    }

    selectedFields.value.push(field)

    // Reset form and clear locked element
    currentFieldName.value = ''
    currentFieldAttribute.value = ''
    lockedElement.value = null
    hoveredElement.value = null
  }

  // Quick add field with auto-suggested name (for click-first workflow)
  const quickAddField = (element: Element, suggestedName: string, fieldType: FieldType = 'text', attribute?: string) => {
    const selector = generateSelector(element)
    const sampleValue = getSampleValue(element, fieldType, attribute)

    let matchCount = 0
    try {
      matchCount = document.querySelectorAll(selector).length
    } catch {
      matchCount = 0
    }

    const field: SelectedField = {
      id: `field-${Date.now()}`,
      name: suggestedName,
      selector,
      type: fieldType,
      attribute: fieldType === 'attribute' ? attribute : undefined,
      timestamp: Date.now(),
      sampleValue,
      matchCount,
      mode: matchCount > 1 ? 'list' : 'single'
    }

    selectedFields.value.push(field)
  }

  const addKeyValueField = (data: {
    fieldName: string
    extractions: any[]
  }) => {
    const field: SelectedField = {
      id: `field-${Date.now()}`,
      name: data.fieldName,
      selector: '', // Not used for key-value pairs
      type: 'text', // Default, not really used
      timestamp: Date.now(),
      mode: 'key-value-pairs',
      attributes: {
        extractions: data.extractions.map(ext => ({
          key_selector: ext.key_selector,
          value_selector: ext.value_selector,
          key_type: ext.key_type as FieldType,
          value_type: ext.value_type as FieldType,
          key_attribute: ext.key_attribute,
          value_attribute: ext.value_attribute,
          transform: ext.transform
        }))
      }
    }

    selectedFields.value.push(field)

    // Reset form
    currentFieldName.value = ''
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

  const testSelectorInline = (selector: string, field?: SelectedField) => {
    try {
      const elements = document.querySelectorAll(selector)
      // Show all matching elements, not just 10
      // Extract value based on field type if provided
      testResults.value = Array.from(elements).map((el, index) => ({
        element: el as Element,
        index,
        value: field ? getSampleValue(el, field.type, field.attribute) : (el.textContent?.trim() || '')
      }))
    } catch (error) {
      testResults.value = []
    }
  }

  const scrollToTestResult = (testResult: TestResult) => {
    const element = testResult.element
    if (!element) return

    // Remove any existing highlight animations
    document.querySelectorAll('.crawlify-pulse-highlight').forEach(el => {
      el.classList.remove('crawlify-pulse-highlight')
    })

    // Scroll element into view with smooth animation
    element.scrollIntoView({
      behavior: 'smooth',
      block: 'center',
      inline: 'center'
    })

    // Add special highlight class with animation
    element.classList.add('crawlify-pulse-highlight')

    // Remove the highlight after animation completes
    setTimeout(() => {
      element.classList.remove('crawlify-pulse-highlight')
    }, 2000)
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

  // Generate live preview samples for the current selection
  const updateLivePreview = () => {
    if (!hoveredElementSelector.value) {
      livePreviewSamples.value = []
      return
    }

    try {
      const elements = document.querySelectorAll(hoveredElementSelector.value)
      const samples: string[] = []
      const maxSamples = 3

      for (let i = 0; i < Math.min(elements.length, maxSamples); i++) {
        const value = getSampleValue(elements[i], currentFieldType.value, currentFieldAttribute.value)
        if (value) {
          samples.push(value)
        }
      }

      livePreviewSamples.value = samples
    } catch (error) {
      livePreviewSamples.value = []
    }
  }

  // Analyze selector quality and generate alternatives
  const updateSelectorAnalysis = () => {
    if (!lockedElement.value || !hoveredElementSelector.value) {
      selectorAnalysis.value = null
      return
    }

    try {
      const analysis = analyzeSelectorQuality(lockedElement.value, hoveredElementSelector.value)
      selectorAnalysis.value = analysis
    } catch (error) {
      selectorAnalysis.value = null
    }
  }

  // Switch to an alternative selector
  const useAlternativeSelector = (alternativeSelector: string) => {
    // This will update the hoveredElementSelector which triggers other updates
    // We need to find elements matching this selector and set the first one as locked
    try {
      const elements = document.querySelectorAll(alternativeSelector)
      if (elements.length > 0) {
        lockedElement.value = elements[0] as Element
        updateSelectorAnalysis()
        updateLivePreview()
      }
    } catch (error) {
      console.error('Failed to use alternative selector:', error)
    }
  }

  const getSelections = () => {
    return selectedFields.value
  }

  const navigateToElement = (element: Element) => {
    // Update locked element to the navigated element
    lockedElement.value = element
    hoveredElement.value = element
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
    // Disabled: new click-first workflow in App.vue handles clicks now
    // document.addEventListener('click', handleClick, true)
    document.addEventListener('keydown', handleKeyDown)
  })

  onBeforeUnmount(() => {
    document.removeEventListener('mousemove', handleMouseMove)
    // Disabled: new click-first workflow in App.vue handles clicks now
    // document.removeEventListener('click', handleClick, true)
    document.removeEventListener('keydown', handleKeyDown)
  })

  // Function to set fields from external source (for editing existing workflows)
  const setFields = (fields: SelectedField[]) => {
    // Ensure all fields have IDs (important for database-loaded fields)
    selectedFields.value = fields.map((field, index) => {
      if (!field.id) {
        return {
          ...field,
          id: `field-${Date.now()}-${index}`
        }
      }
      return field
    })
  }

  // Expose to window for backend to call
  if (typeof window !== 'undefined') {
    (window as any).__crawlifySetFields = setFields
  }

  return {
    hoveredElement,
    hoveredElementSelector,
    hoveredElementCount,
    hoveredElementValidation,
    lockedElement,
    lockedElementSelector,
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
    setFields,
    updateLivePreview,
    updateSelectorAnalysis,
    useAlternativeSelector
  }
}
