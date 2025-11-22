<template>
  <Teleport to="#crawlify-selector-overlay">
    <Transition name="fade">
      <div
        v-if="visible"
        class="floating-tooltip fixed pointer-events-none z-[1000001]"
        :style="{ left: `${position.x}px`, top: `${position.y}px` }"
      >
        <div
          class="pointer-events-auto bg-white rounded-lg shadow-lg border border-gray-300 p-3 min-w-[280px] max-w-[320px]"
          @click.stop
        >
          <!-- Suggested Field Name -->
          <div class="mb-3">
            <label class="text-xs text-gray-500 mb-1 block">Suggested field name:</label>
            <input
              v-model="editableName"
              type="text"
              class="w-full px-2 py-1.5 text-sm font-semibold text-gray-900 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-gray-400 focus:border-transparent"
              @click.stop
              @keydown.enter="$emit('quick-add', editableName, editableSelector)"
            />
          </div>

          <!-- CSS Selector (Expandable) -->
          <div class="mb-3">
            <button
              @click="showSelector = !showSelector"
              class="text-xs text-gray-500 hover:text-gray-700 flex items-center gap-1 mb-1"
            >
              <svg
                class="h-3 w-3 transition-transform"
                :class="{ 'rotate-90': showSelector }"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
              </svg>
              CSS Selector {{ showSelector ? '(click to hide)' : '(click to edit)' }}
            </button>
            <div v-if="showSelector" class="animate-in slide-in-from-top-2 duration-150">
              <input
                v-model="editableSelector"
                type="text"
                class="w-full px-2 py-1.5 text-xs font-mono text-gray-700 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-gray-400 focus:border-transparent bg-gray-50"
                @click.stop
                placeholder="e.g., .product-name"
              />
            </div>
          </div>

          <!-- Match Count Preview -->
          <div class="flex items-center gap-1.5 mb-3">
            <div :class="[
              'h-2 w-2 rounded-full',
              matchCount === 1 ? 'bg-green-500' : 'bg-blue-500'
            ]"></div>
            <span class="text-xs text-gray-600">
              {{ matchCount }} {{ matchCount === 1 ? 'match' : 'matches' }} found
            </span>
          </div>

          <!-- Action Buttons -->
          <div class="flex gap-2">
            <button
              @click="$emit('quick-add', editableName, editableSelector)"
              class="flex-1 px-3 py-2 bg-gray-900 text-white text-sm font-medium rounded hover:bg-gray-800 transition-colors"
            >
              Quick Add
            </button>
            <button
              @click="$emit('customize', editableName, editableSelector)"
              class="flex-1 px-3 py-2 bg-white text-gray-900 text-sm font-medium rounded border border-gray-300 hover:bg-gray-50 transition-colors"
            >
              Customize
            </button>
          </div>

          <!-- Dismiss hint -->
          <div class="mt-2 text-[10px] text-gray-400 text-center">
            Press ESC or click outside to dismiss
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'

interface Props {
  visible: boolean
  position: { x: number, y: number }
  suggestedName: string
  suggestedSelector: string
  matchCount: number
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'quick-add': [fieldName: string, selector: string]
  'customize': [fieldName: string, selector: string]
  'dismiss': []
}>()

// Editable name and selector
const editableName = ref(props.suggestedName)
const editableSelector = ref(props.suggestedSelector)
const showSelector = ref(false)

// Update editable values when suggestions change
watch(() => props.suggestedName, (newName) => {
  editableName.value = newName
})

watch(() => props.suggestedSelector, (newSelector) => {
  editableSelector.value = newSelector
})

// Handle ESC key to dismiss
const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.visible) {
    emit('dismiss')
  }
}

// Handle click outside to dismiss
const handleClickOutside = (e: MouseEvent) => {
  if (!props.visible) return
  
  const target = e.target as Element
  // Check if click is outside the tooltip
  if (target && !target.closest('.floating-tooltip')) {
    emit('dismiss')
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
  // Use mousedown instead of click to catch before other handlers
  window.addEventListener('mousedown', handleClickOutside, true)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('mousedown', handleClickOutside, true)
})
</script>
