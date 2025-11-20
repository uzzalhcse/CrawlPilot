<template>
  <div>
    <!-- Hover highlight -->
    <div
      v-if="hoveredRect && !props.lockedElement"
      class="absolute pointer-events-none border-2 z-[999998] transition-all duration-150 ease-out"
      :class="getHoverHighlightClass()"
      :style="highlightStyle(hoveredRect)"
    />
    
    <!-- Locked element highlight -->
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
    </div>

    <!-- Selected fields highlights -->
    <div
      v-for="field in props.selectedFields"
      :key="field.id"
      class="absolute pointer-events-none border-2 z-[999996]"
      :class="getFieldHighlightClass(field)"
      :style="getFieldHighlightStyle(field)"
    >
      <div class="absolute -top-7 left-0 text-white text-xs px-2 py-1 rounded shadow-md font-medium"
           :class="getFieldBadgeClass(field)">
        {{ field.name }}
      </div>
    </div>

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

    <!-- Element tag label -->
    <div
      v-if="tagLabelStyle"
      class="absolute bg-slate-800 text-slate-100 px-2 py-1 rounded text-xs font-mono z-[1000001] pointer-events-none shadow-lg max-w-[300px] overflow-hidden text-ellipsis whitespace-nowrap"
      :style="tagLabelStyle"
    >
      {{ tagLabel }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, onBeforeUnmount } from 'vue'
import type { SelectedField, TestResult, FieldType } from '../types'
import { getElementColor } from '../utils/elementColors'

interface Props {
  hoveredElement: Element | null
  lockedElement: Element | null
  selectedFields: SelectedField[]
  testResults: TestResult[]
}

const props = defineProps<Props>()

// Store rects for selected fields
const fieldRects = ref<Map<string, DOMRect>>(new Map())

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

  // Update rects for all selected fields
  fieldRects.value.clear()
  props.selectedFields.forEach(field => {
    try {
      const elements = document.querySelectorAll(field.selector)
      if (elements.length > 0) {
        fieldRects.value.set(field.id, elements[0].getBoundingClientRect())
      }
    } catch (error) {
      console.warn('Invalid selector:', field.selector)
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
  
  const tag = element.tagName.toLowerCase()
  const id = element.id ? `#${element.id}` : ''
  const classes = element.classList.length > 0 
    ? `.${Array.from(element.classList).slice(0, 2).join('.')}` 
    : ''
  
  return `${tag}${id}${classes}`
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

const getFieldHighlightStyle = (field: SelectedField) => {
  const rect = fieldRects.value.get(field.id)
  if (!rect) return { display: 'none' }
  return highlightStyle(rect)
}
</script>
