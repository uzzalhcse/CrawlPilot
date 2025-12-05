<script setup lang="ts">
import { computed, ref } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import { 
  Database, FileText, Code, Link, List, Box, 
  Zap, Eye, Clock, MousePointer, ArrowDown, Move
} from 'lucide-vue-next'

interface ActionNode {
  type: string
  name?: string
}

interface Props {
  data: {
    label: string
    field: {
      selector: string
      type: string
      attribute?: string
      actions?: ActionNode[]
      multiple?: boolean
      transform?: string
      default?: string
    }
    parentId: string
  }
  selected?: boolean
}

const props = defineProps<Props>()

// Tooltip state
const showTooltip = ref(false)
const tooltipTimeout = ref<number | null>(null)
const tooltipPosition = ref({ x: 0, y: 0 })
const nodeRef = ref<HTMLElement | null>(null)

function handleMouseEnter(event: MouseEvent) {
  const target = event.currentTarget as HTMLElement
  nodeRef.value = target
  
  tooltipTimeout.value = window.setTimeout(() => {
    if (nodeRef.value) {
      const rect = nodeRef.value.getBoundingClientRect()
      // Position tooltip to the left of the node
      tooltipPosition.value = {
        x: rect.left - 10,
        y: rect.top + rect.height / 2
      }
    }
    showTooltip.value = true
  }, 400)
}

function handleMouseLeave() {
  if (tooltipTimeout.value) {
    clearTimeout(tooltipTimeout.value)
  }
  showTooltip.value = false
}

const icon = computed(() => {
  switch (props.data.field.type) {
    case 'text': return FileText
    case 'html': return Code
    case 'attribute': 
    case 'attr': return Link
    case 'list': return List
    case 'nested': return Box
    default: return Database
  }
})

// Get action icons
const actionIcons = computed(() => {
  if (!props.data.field.actions || !Array.isArray(props.data.field.actions)) return []
  
  const iconMap: Record<string, { icon: any; color: string; label: string }> = {
    'wait_for': { icon: Eye, color: 'text-blue-500', label: 'Wait For' },
    'wait': { icon: Clock, color: 'text-purple-500', label: 'Wait' },
    'click': { icon: MousePointer, color: 'text-green-500', label: 'Click' },
    'scroll': { icon: ArrowDown, color: 'text-orange-500', label: 'Scroll' },
    'hover': { icon: Move, color: 'text-pink-500', label: 'Hover' }
  }
  
  return props.data.field.actions.map(action => {
    const mapping = iconMap[action.type] || { icon: Zap, color: 'text-gray-500', label: action.type }
    return {
      ...mapping,
      name: action.name || action.type
    }
  })
})

const actionsCount = computed(() => actionIcons.value.length)
const isMultiple = computed(() => props.data.field.multiple === true)
</script>

<template>
  <div 
    class="relative min-w-[140px] max-w-[180px] bg-card border rounded-md shadow-sm transition-all hover:shadow-md"
    :class="{ 'ring-2 ring-primary': selected }"
    @mouseenter="handleMouseEnter"
    @mouseleave="handleMouseLeave"
  >
    <Handle type="target" :position="Position.Left" class="!w-2 !h-2 !bg-muted-foreground" />
    
    <!-- Header with label and badges -->
    <div class="p-1.5 flex items-center gap-1 border-b bg-muted/20 rounded-t-md">
      <component :is="icon" class="w-3 h-3 text-muted-foreground shrink-0" />
      <span class="font-mono text-[10px] font-semibold truncate flex-1" :title="data.label">
        {{ data.label }}
      </span>
      
      <!-- Multiple Badge -->
      <div 
        v-if="isMultiple"
        class="px-1 py-0.5 rounded bg-purple-500/20 text-purple-600 dark:text-purple-400 text-[8px] font-bold"
        title="Extracts multiple values (array)"
      >
        []
      </div>
    </div>
    
    <!-- Selector -->
    <div class="p-1.5 text-[10px] space-y-1">
      <div class="text-muted-foreground truncate" :title="data.field.selector">
        {{ data.field.selector || 'No selector' }}
      </div>
      
      <!-- Attribute badge -->
      <div v-if="data.field.type === 'attribute' || data.field.type === 'attr'" 
           class="flex items-center gap-1 text-[10px] bg-muted inline-flex px-1 rounded">
        <span class="opacity-50">@</span>
        {{ data.field.attribute }}
      </div>
      
      <!-- Action Type Icons Row -->
      <div v-if="actionsCount > 0" class="flex items-center gap-1 pt-0.5 border-t border-border/50">
        <Zap class="w-2.5 h-2.5 text-amber-500 shrink-0" />
        <div class="flex items-center gap-0.5 flex-wrap">
          <div 
            v-for="(action, idx) in actionIcons.slice(0, 4)" 
            :key="idx"
            class="flex items-center justify-center w-4 h-4 rounded bg-muted/50"
            :title="action.name"
          >
            <component :is="action.icon" :class="['w-2.5 h-2.5', action.color]" />
          </div>
          <span v-if="actionsCount > 4" class="text-[8px] text-muted-foreground ml-0.5">
            +{{ actionsCount - 4 }}
          </span>
        </div>
      </div>
    </div>

    <!-- Quick Preview Tooltip - positioned to the left -->
    <Teleport to="body">
      <Transition
        enter-active-class="transition-opacity duration-150"
        leave-active-class="transition-opacity duration-100"
        enter-from-class="opacity-0"
        leave-to-class="opacity-0"
      >
        <div 
          v-if="showTooltip"
          class="fixed z-[9999] p-3 bg-popover/95 backdrop-blur-sm border border-border rounded-lg shadow-2xl text-sm min-w-[220px] max-w-[280px]"
          :style="{
            top: `${tooltipPosition.y}px`,
            left: `${tooltipPosition.x}px`,
            transform: 'translate(-100%, -50%)'
          }"
        >
          <!-- Arrow pointing right -->
          <div class="absolute right-0 top-1/2 -translate-y-1/2 translate-x-full">
            <div class="border-8 border-transparent border-l-border"></div>
            <div class="absolute top-0 left-0 border-8 border-transparent border-l-popover -translate-x-[1px]"></div>
          </div>
          
          <!-- Header -->
          <div class="flex items-center gap-2 mb-2 pb-2 border-b">
            <component :is="icon" class="w-4 h-4 text-primary" />
            <span class="font-semibold">{{ data.label }}</span>
            <span v-if="isMultiple" class="px-1.5 py-0.5 rounded bg-purple-500/20 text-purple-600 text-[10px] font-bold">
              Array
            </span>
          </div>
          
          <!-- Details -->
          <div class="space-y-1.5 text-xs">
            <div class="flex items-start gap-2">
              <span class="text-muted-foreground w-16 shrink-0">Selector:</span>
              <code class="bg-muted px-1 py-0.5 rounded text-[10px] break-all">{{ data.field.selector || 'None' }}</code>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-muted-foreground w-16 shrink-0">Type:</span>
              <span class="capitalize">{{ data.field.type || 'text' }}</span>
            </div>
            <div v-if="data.field.transform" class="flex items-center gap-2">
              <span class="text-muted-foreground w-16 shrink-0">Transform:</span>
              <span>{{ data.field.transform }}</span>
            </div>
            <div v-if="data.field.default" class="flex items-center gap-2">
              <span class="text-muted-foreground w-16 shrink-0">Default:</span>
              <span>{{ data.field.default }}</span>
            </div>
          </div>
          
          <!-- Actions -->
          <div v-if="actionsCount > 0" class="mt-2 pt-2 border-t">
            <div class="flex items-center gap-1 mb-1.5 text-xs font-medium text-amber-600 dark:text-amber-400">
              <Zap class="w-3 h-3" />
              Pre-Extraction Actions ({{ actionsCount }})
            </div>
            <div class="space-y-1">
              <div 
                v-for="(action, idx) in actionIcons" 
                :key="idx"
                class="flex items-center gap-2 text-xs"
              >
                <span class="w-4 h-4 flex items-center justify-center rounded bg-muted">
                  <component :is="action.icon" :class="['w-3 h-3', action.color]" />
                </span>
                <span class="truncate">{{ action.name }}</span>
              </div>
            </div>
          </div>
          
          <!-- Tip -->
          <div class="mt-2 pt-2 border-t text-[10px] text-muted-foreground">
            Click to configure • Hover for preview
          </div>
        </div>
      </Transition>
    </Teleport>

    <Handle type="source" :position="Position.Right" class="!w-2 !h-2 !bg-muted-foreground" />
  </div>
</template>
