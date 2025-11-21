import { ref, computed } from 'vue'
import type { FieldType, KeyValuePair } from '../types'
import { generateSelector } from '../utils/selectorGenerator'

type SelectionState = 'idle' | 'selecting-keys' | 'selecting-values'

export function useKeyValueSelection() {
  const selectionState = ref<SelectionState>('idle')
  const keySelector = ref('')
  const valueSelector = ref('')
  const keyType = ref<FieldType>('text')
  const valueType = ref<FieldType>('text')
  const keyAttribute = ref('')
  const valueAttribute = ref('')
  const keyMatches = ref<string[]>([])
  const valueMatches = ref<string[]>([])
  const keyElements = ref<Element[]>([])
  const valueElements = ref<Element[]>([])
  const applyTrim = ref(true)

  const isSelectingKeys = computed(() => selectionState.value === 'selecting-keys')
  const isSelectingValues = computed(() => selectionState.value === 'selecting-values')
  
  const keyCount = computed(() => keyMatches.value.length)
  const valueCount = computed(() => valueMatches.value.length)
  
  const hasCountMismatch = computed(() => {
    return keyCount.value > 0 && valueCount.value > 0 && keyCount.value !== valueCount.value
  })

  const pairs = computed<KeyValuePair[]>(() => {
    if (keyMatches.value.length === 0 || valueMatches.value.length === 0) {
      return []
    }
    
    const minLength = Math.min(keyMatches.value.length, valueMatches.value.length)
    const result: KeyValuePair[] = []
    
    for (let i = 0; i < minLength; i++) {
      result.push({
        key: keyMatches.value[i] || '',
        value: valueMatches.value[i] || '',
        index: i + 1
      })
    }
    
    return result
  })

  function startKeySelection() {
    selectionState.value = 'selecting-keys'
  }

  function startValueSelection() {
    selectionState.value = 'selecting-values'
  }

  function cancelSelection() {
    selectionState.value = 'idle'
  }

  function extractContent(element: Element, type: FieldType, attribute?: string): string {
    if (type === 'attribute' && attribute) {
      return element.getAttribute(attribute) || ''
    } else if (type === 'html') {
      return element.innerHTML
    } else {
      return element.textContent?.trim() || ''
    }
  }

  function selectKeyElement(element: Element) {
    const selector = generateSelector(element)
    keySelector.value = selector
    
    // Find all matching elements
    const elements = Array.from(document.querySelectorAll(selector))
    keyElements.value = elements
    
    // Extract content from all elements
    keyMatches.value = elements.map(el => extractContent(el, keyType.value, keyAttribute.value))
    
    selectionState.value = 'idle'
    
    // Try to suggest value selector
    suggestValueSelector(element)
  }

  function selectValueElement(element: Element) {
    const selector = generateSelector(element)
    valueSelector.value = selector
    
    // Find all matching elements
    const elements = Array.from(document.querySelectorAll(selector))
    valueElements.value = elements
    
    // Extract content from all elements
    valueMatches.value = elements.map(el => extractContent(el, valueType.value, valueAttribute.value))
    
    selectionState.value = 'idle'
  }

  function suggestValueSelector(keyElement: Element) {
    // Try to detect common patterns
    const tagName = keyElement.tagName.toLowerCase()
    
    // dt -> dd pattern
    if (tagName === 'dt') {
      const parent = keyElement.parentElement
      if (parent && parent.tagName.toLowerCase() === 'dl') {
        const dds = parent.querySelectorAll('dd')
        if (dds.length > 0) {
          const selector = generateSelector(dds[0])
          return { selector, confidence: 'high' }
        }
      }
    }
    
    // th -> td pattern
    if (tagName === 'th') {
      const row = keyElement.closest('tr')
      if (row) {
        const td = row.querySelector('td')
        if (td) {
          const selector = generateSelector(td)
          return { selector, confidence: 'high' }
        }
      }
    }
    
    // Sibling with similar class pattern
    const nextSibling = keyElement.nextElementSibling
    if (nextSibling) {
      const selector = generateSelector(nextSibling)
      return { selector, confidence: 'medium' }
    }
    
    return null
  }

  function updateKeyType(type: FieldType) {
    keyType.value = type
    // Re-extract content if we have elements
    if (keyElements.value.length > 0) {
      keyMatches.value = keyElements.value.map(el => extractContent(el, type, keyAttribute.value))
    }
  }

  function updateValueType(type: FieldType) {
    valueType.value = type
    // Re-extract content if we have elements
    if (valueElements.value.length > 0) {
      valueMatches.value = valueElements.value.map(el => extractContent(el, type, valueAttribute.value))
    }
  }

  function updateKeyAttribute(attr: string) {
    keyAttribute.value = attr
    // Re-extract content if we have elements and type is attribute
    if (keyElements.value.length > 0 && keyType.value === 'attribute') {
      keyMatches.value = keyElements.value.map(el => extractContent(el, keyType.value, attr))
    }
  }

  function updateValueAttribute(attr: string) {
    valueAttribute.value = attr
    // Re-extract content if we have elements and type is attribute
    if (valueElements.value.length > 0 && valueType.value === 'attribute') {
      valueMatches.value = valueElements.value.map(el => extractContent(el, valueType.value, attr))
    }
  }

  function reset() {
    selectionState.value = 'idle'
    keySelector.value = ''
    valueSelector.value = ''
    keyType.value = 'text'
    valueType.value = 'text'
    keyAttribute.value = ''
    valueAttribute.value = ''
    keyMatches.value = []
    valueMatches.value = []
    keyElements.value = []
    valueElements.value = []
    applyTrim.value = true
  }

  function getExtractionData() {
    return {
      key_selector: keySelector.value,
      value_selector: valueSelector.value,
      key_type: keyType.value,
      value_type: valueType.value,
      key_attribute: keyAttribute.value || undefined,
      value_attribute: valueAttribute.value || undefined,
      transform: applyTrim.value ? 'trim' : undefined
    }
  }

  return {
    selectionState,
    isSelectingKeys,
    isSelectingValues,
    keySelector,
    valueSelector,
    keyType,
    valueType,
    keyAttribute,
    valueAttribute,
    keyMatches,
    valueMatches,
    keyElements,
    valueElements,
    keyCount,
    valueCount,
    hasCountMismatch,
    pairs,
    applyTrim,
    startKeySelection,
    startValueSelection,
    cancelSelection,
    selectKeyElement,
    selectValueElement,
    updateKeyType,
    updateValueType,
    updateKeyAttribute,
    updateValueAttribute,
    reset,
    getExtractionData
  }
}
