<template>
  <div>
    <!-- Hover highlight -->
    <div
      v-if="hoveredRect && !props.lockedElement"
      class="absolute pointer-events-none border-2 z-[999998] transition-all duration-150 ease-out"
      :class="getHoverHighlightClass()"
      :style="highlightStyle(hoveredRect)"
    />
    
    <!-- Locked element highlight with navigation buttons -->
    <div
      v-if="lockedRect"
      class="absolute pointer-events-none border-2 z-[999998] shadow-lg"
      :class="getLockedHighlightClass()"
      :style="highlightStyle(lockedRect)"
    >
      <div class="absolute -top-7 left-0 text-white text-xs px-2 py-1 rounded shadow-md"
           :class="getLockedBadgeClass()">
        Selected ✓
      </div>
      
      <!-- Navigation Buttons with improved design -->
      <div class="absolute -right-2 top-1/2 transform -translate-y-1/2 flex flex-col gap-2 pointer-events-auto">
        <button
          v-if="canNavigateToParent"
          @click.stop="navigateToParent"
          class="bg-gradient-to-br from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 text-white text-xs px-3 py-2 rounded-lg shadow-xl transition-all hover:scale-110 font-bold border-2 border-blue-400"
          title="Select Parent Element (Alt+↑)"
        >
          ↑ Parent
        </button>
        <button
          v-if="canNavigateToChild"
          @click.stop="navigateToFirstChild"
          class="bg-gradient-to-br from-green-500 to-green-600 hover:from-green-600 hover:to-green-700 text-white text-xs px-3 py-2 rounded-lg shadow-xl transition-all hover:scale-110 font-bold border-2 border-green-400"
          title="Select First Child (Alt+↓)"
        >
          ↓ Child
        </button>
        <button
          v-if="canNavigateToPrevSibling"
          @click.stop="navigateToPrevSibling"
          class="bg-gradient-to-br from-purple-500 to-purple-600 hover:from-purple-600 hover:to-purple-700 text-white text-xs px-3 py-2 rounded-lg shadow-xl transition-all hover:scale-110 font-bold border-2 border-purple-400"
          title="Select Previous Sibling (Alt+←)"
        >
          ← Prev
        </button>
        <button
          v-if="canNavigateToNextSibling"
          @click.stop="navigateToNextSibling"
          class="bg-gradient-to-br from-purple-500 to-purple-600 hover:from-purple-600 hover:to-purple-700 text-white text-xs px-3 py-2 rounded-lg shadow-xl transition-all hover:scale-110 font-bold border-2 border-purple-400"
          title="Select Next Sibling (Alt+→)"
        >
          → Next
        </button>
      </div>
    </div>

    <!-- Selected fields highlights (show all matching elements) -->
    <template v-for="field in props.selectedFields" :key="field.id">
      <div
        v-for="(rect, index) in getFieldRects(field)"
        :key="`${field.id}-${index}`"
        class="absolute pointer-events-none border-2 z-[999996]"
        :class="getFieldHighlightClass(field)"
        :style="highlightStyle(rect)"
      >
        <div 
          class="absolute -top-7 left-0 text-white text-xs px-2 py-1 rounded shadow-md font-medium whitespace-nowrap cursor-pointer hover:scale-105 transition-transform pointer-events-auto"
          :class="getFieldBadgeClass(field)"
          @click.stop="emit('edit-field', field)"
          :title="`Click to edit '${field.name}'`"
        >
          {{ field.name }}<span v-if="field.type === 'attribute' && field.attribute" class="opacity-90 ml-1">@{{ field.attribute }}</span><span v-if="getFieldRects(field).length > 1" class="opacity-75 ml-1">#{{ index + 1 }}</span>
        </div>
      </div>
    </template>

    <!-- Test results highlights -->
    <div
      v-for="(result, index) in props.testResults"
      :key="index"
      class="absolute pointer-events-none border-2 border-purple-500 bg-purple-500/10 z-[999997]"
      :style="highlightStyle(result.element.getBoundingClientRect())"
    >
      <div class="absolute -top-6 left-0 bg-purple-600 text-white text-xs px-2 py-1 rounded shadow-md">
        {{ index + 1 }}
      </div>
    </div>

    <!-- Key-Value Selection Highlights -->
    <template v-if="kvSelection">
      <!-- Key elements -->
      <div
        v-for="(element, index) in kvSelection.keyElements.value"
        :key="`key-${index}`"
        class="absolute pointer-events-none border-3 border-green-500 bg-green-500/15 z-[999997] shadow-lg"
        :style="highlightStyle(element.getBoundingClientRect())"
      >
        <div class="absolute -top-7 left-0 bg-green-600 text-white text-xs px-2 py-1 rounded shadow-md font-bold">
          🔑 Key {{ index + 1 }}
        </div>
      </div>

      <!-- Value elements -->
      <div
        v-for="(element, index) in kvSelection.valueElements.value"
        :key="`value-${index}`"
        class="absolute pointer-events-none border-3 border-blue-500 bg-blue-500/15 z-[999997] shadow-lg"
        :style="highlightStyle(element.getBoundingClientRect())"
      >
        <div class="absolute -top-7 left-0 bg-blue-600 text-white text-xs px-2 py-1 rounded shadow-md font-bold">
          💎 Value {{ index + 1 }}
        </div>
      </div>
    </template>

    <!-- Element tag label with improved design -->
    <div
      v-if="tagLabelStyle"
      class="absolute bg-gradient-to-r from-slate-800 to-slate-900 text-slate-100 px-3 py-2 rounded-lg text-xs font-mono z-[1000001] pointer-events-none shadow-2xl max-w-[500px] overflow-hidden text-ellipsis whitespace-nowrap border-2 border-slate-600"
      :style="tagLabelStyle"
    >
      <span class="font-bold">{{ tagLabel }}</span>
      <span 
        v-if="props.hoveredElementCount && props.hoveredElementCount > 0"
        class="ml-2 px-2 py-0.5 rounded-full text-[10px] font-bold"
        :class="getCountBadgeClass()"
      >
        {{ props.hoveredElementCount }} {{ props.hoveredElementCount === 1 ? 'match' : 'matches' }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, onBeforeUnmount, inject } from 'vue'
import type { SelectedField, TestResult, FieldType } from '../types'
import { getElementColor } from '../utils/elementColors'
import { generateSelector } from '../utils/selectorGenerator'

// Inject key-value selection state
const kvSelection = inject<any>('kvSelection', null)

interface Props {
  hoveredElement: Element | null
  lockedElement: Element | null
  selectedFields: SelectedField[]
  testResults: TestResult[]
  currentFieldType: FieldType
  currentFieldAttribute?: string
  hoveredElementCount?: number
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'navigate': [element: Element]
  'edit-field': [field: SelectedField]
}>()

// Store rects for selected fields (all matching elements, not just first)
const fieldRects = ref<Map<string, DOMRect[]>>(new Map())

const hoveredRect = ref<DOMRect | null>(null)
const lockedRect = ref<DOMRect | null>(null)

const updateRects = () => {
  if (props.hoveredElement) {
    hoveredRect.value = props.hoveredElement.getBoundingClientRect()
  } else {
    hoveredRect.value = null
  }

  if (props.lockedElement) {
    lockedRect.value = props.lockedElement.getBoundingClientRect()
  } else {
    lockedRect.value = null
  }

  // Update rects for all selected fields (store all matching elements)
  fieldRects.value.clear()
  props.selectedFields.forEach(field => {
    // Skip fields without selectors (e.g., key-value-pairs mode)
    if (!field.selector) return
    
    try {
      const elements = document.querySelectorAll(field.selector)
      if (elements.length > 0) {
        const rects = Array.from(elements).map(el => el.getBoundingClientRect())
        fieldRects.value.set(field.id, rects)
      }
    } catch (error) {
      // Silently skip invalid selectors
    }
  })
}

// Update rectangles when elements change
watch([() => props.hoveredElement, () => props.lockedElement, () => props.selectedFields], updateRects, { deep: true })

// Update on scroll/resize/mousemove
let rafId: number | null = null
const scheduleUpdate = () => {
  if (rafId === null) {
    rafId = requestAnimationFrame(() => {
      updateRects()
      rafId = null
    })
  }
}

// Continuous update for smooth tracking
let continuousUpdateInterval: number | null = null
const startContinuousUpdate = () => {
  if (continuousUpdateInterval === null) {
    continuousUpdateInterval = window.setInterval(() => {
      if (props.hoveredElement || props.lockedElement || props.selectedFields.length > 0) {
        updateRects()
      }
    }, 16) // ~60fps
  }
}

const stopContinuousUpdate = () => {
  if (continuousUpdateInterval !== null) {
    clearInterval(continuousUpdateInterval)
    continuousUpdateInterval = null
  }
}

onMounted(() => {
  window.addEventListener('scroll', scheduleUpdate, true)
  window.addEventListener('resize', scheduleUpdate)
  window.addEventListener('mousemove', scheduleUpdate)
  updateRects()
  startContinuousUpdate()
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', scheduleUpdate, true)
  window.removeEventListener('resize', scheduleUpdate)
  window.removeEventListener('mousemove', scheduleUpdate)
  stopContinuousUpdate()
  if (rafId !== null) {
    cancelAnimationFrame(rafId)
  }
})

const highlightStyle = (rect: DOMRect) => ({
  top: `${rect.top + window.scrollY}px`,
  left: `${rect.left + window.scrollX}px`,
  width: `${rect.width}px`,
  height: `${rect.height}px`,
  boxShadow: '0 0 0 1px rgba(59, 130, 246, 0.3), 0 4px 12px rgba(59, 130, 246, 0.2)'
})

const tagLabel = computed(() => {
  const element = props.lockedElement || props.hoveredElement
  if (!element) return ''
  
  // Show the actual generated selector that will be used
  const selector = generateSelector(element)
  
  // Add extraction type and attribute information
  let extractionInfo = ''
  if (props.currentFieldType === 'text') {
    extractionInfo = ' → text'
  } else if (props.currentFieldType === 'attribute' && props.currentFieldAttribute) {
    extractionInfo = ` → @${props.currentFieldAttribute}`
  } else if (props.currentFieldType === 'html') {
    extractionInfo = ' → html'
  }
  
  return `${selector}${extractionInfo}`
})

const tagLabelStyle = computed(() => {
  const rect = lockedRect.value || hoveredRect.value
  if (!rect) return null

  return {
    top: `${rect.top + window.scrollY - 28}px`,
    left: `${rect.left + window.scrollX}px`
  }
})

// Get element type for color coding
const getElementType = (element: Element | null): string => {
  if (!element) return 'default'
  const tagName = element.tagName.toLowerCase()
  
  if (tagName === 'button' || element.getAttribute('role') === 'button') return 'button'
  if (tagName === 'input' || tagName === 'textarea' || tagName === 'select') return 'input'
  if (tagName === 'a') return 'link'
  if (tagName === 'img') return 'image'
  if (['h1', 'h2', 'h3', 'h4', 'h5', 'h6'].includes(tagName)) return 'heading'
  if (['p', 'span', 'div'].includes(tagName)) return 'text'
  
  return 'default'
}

const getHoverHighlightClass = () => {
  // Special styling for key-value selection mode
  if (kvSelection) {
    if (kvSelection.isSelectingKeys.value) {
      return 'border-green-400 bg-green-400/20'
    }
    if (kvSelection.isSelectingValues.value) {
      return 'border-blue-400 bg-blue-400/20'
    }
  }
  
  const type = getElementType(props.hoveredElement)
  const colors = getElementColor(type)
  return `border-${colors.border} bg-${colors.bg}`
}

const getLockedHighlightClass = () => {
  const type = getElementType(props.lockedElement)
  const colors = getElementColor(type)
  return `border-${colors.border} bg-${colors.bg}`
}

const getLockedBadgeClass = () => {
  const type = getElementType(props.lockedElement)
  const colors = getElementColor(type)
  return `bg-${colors.badge}`
}

const getFieldHighlightClass = (field: SelectedField) => {
  const colors = getElementColor(field.type)
  return `border-${colors.border} bg-${colors.bg}`
}

const getFieldBadgeClass = (field: SelectedField) => {
  const colors = getElementColor(field.type)
  return `bg-${colors.badge}`
}

const getFieldRects = (field: SelectedField): DOMRect[] => {
  return fieldRects.value.get(field.id) || []
}

// Count badge color based on match count
const getCountBadgeClass = () => {
  const count = props.hoveredElementCount || 0
  if (count === 1) return 'bg-green-500 text-white'
  if (count <= 10) return 'bg-blue-500 text-white'
  return 'bg-orange-500 text-white'
}

// Navigation helpers
const canNavigateToParent = computed(() => {
  if (!props.lockedElement) return false
  const parent = props.lockedElement.parentElement
  return parent && parent !== document.body && parent !== document.documentElement
})

const canNavigateToChild = computed(() => {
  if (!props.lockedElement) return false
  return props.lockedElement.children.length > 0
})

const canNavigateToPrevSibling = computed(() => {
  if (!props.lockedElement) return false
  return props.lockedElement.previousElementSibling !== null
})

const canNavigateToNextSibling = computed(() => {
  if (!props.lockedElement) return false
  return props.lockedElement.nextElementSibling !== null
})

// Navigation functions
const navigateToParent = () => {
  if (props.lockedElement?.parentElement) {
    const parent = props.lockedElement.parentElement
    if (parent !== document.body && parent !== document.documentElement) {
      emit('navigate', parent)
    }
  }
}

const navigateToFirstChild = () => {
  if (props.lockedElement?.children[0]) {
    emit('navigate', props.lockedElement.children[0])
  }
}

const navigateToPrevSibling = () => {
  if (props.lockedElement?.previousElementSibling) {
    emit('navigate', props.lockedElement.previousElementSibling)
  }
}

const navigateToNextSibling = () => {
  if (props.lockedElement?.nextElementSibling) {
    emit('navigate', props.lockedElement.nextElementSibling)
  }
}

// Keyboard shortcuts
onMounted(() => {
  const handleKeyDown = (e: KeyboardEvent) => {
    if (!props.lockedElement || !e.altKey) return
    
    switch(e.key) {
      case 'ArrowUp':
        e.preventDefault()
        navigateToParent()
        break
      case 'ArrowDown':
        e.preventDefault()
        navigateToFirstChild()
        break
      case 'ArrowLeft':
        e.preventDefault()
        navigateToPrevSibling()
        break
      case 'ArrowRight':
        e.preventDefault()
        navigateToNextSibling()
        break
    }
  }
  
  window.addEventListener('keydown', handleKeyDown)
  
  // Cleanup added in existing onBeforeUnmount
  const originalUnmount = onBeforeUnmount(() => {
    window.removeEventListener('keydown', handleKeyDown)
  })
})
</script>
