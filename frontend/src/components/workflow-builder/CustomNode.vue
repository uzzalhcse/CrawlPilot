<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import type { NodeData } from '@/types'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import {
  Globe,
  Database,
  MousePointer,
  RefreshCw,
  GitBranch,
  Link,
  Filter as FilterIcon,
  FileText,
  CheckCircle,
  Sparkles
} from 'lucide-vue-next'

interface Props {
  data: NodeData
  selected?: boolean
}

const props = defineProps<Props>()

// Category configuration with dark mode support
const categoryConfig = {
  'URL Discovery': {
    light: 'bg-blue-50 border-blue-200 text-blue-900',
    dark: 'dark:bg-blue-950/50 dark:border-blue-800 dark:text-blue-100',
    badge: 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300',
    icon: 'text-blue-600 dark:text-blue-400'
  },
  'Extraction': {
    light: 'bg-emerald-50 border-emerald-200 text-emerald-900',
    dark: 'dark:bg-emerald-950/50 dark:border-emerald-800 dark:text-emerald-100',
    badge: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300',
    icon: 'text-emerald-600 dark:text-emerald-400'
  },
  'Interaction': {
    light: 'bg-purple-50 border-purple-200 text-purple-900',
    dark: 'dark:bg-purple-950/50 dark:border-purple-800 dark:text-purple-100',
    badge: 'bg-purple-100 text-purple-700 dark:bg-purple-900/50 dark:text-purple-300',
    icon: 'text-purple-600 dark:text-purple-400'
  },
  'Transformation': {
    light: 'bg-amber-50 border-amber-200 text-amber-900',
    dark: 'dark:bg-amber-950/50 dark:border-amber-800 dark:text-amber-100',
    badge: 'bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300',
    icon: 'text-amber-600 dark:text-amber-400'
  },
  'Control Flow': {
    light: 'bg-pink-50 border-pink-200 text-pink-900',
    dark: 'dark:bg-pink-950/50 dark:border-pink-800 dark:text-pink-100',
    badge: 'bg-pink-100 text-pink-700 dark:bg-pink-900/50 dark:text-pink-300',
    icon: 'text-pink-600 dark:text-pink-400'
  }
}

const nodeIcon = computed(() => {
  const type = props.data.nodeType

  // URL Discovery
  if (['fetch', 'navigate'].includes(type)) return Globe
  if (['extract_links'].includes(type)) return Link
  if (['filter_urls', 'paginate'].includes(type)) return FilterIcon

  // Extraction
  if (type.startsWith('extract')) return Database

  // Interaction
  if (['click', 'scroll', 'hover', 'type'].includes(type)) return MousePointer
  if (['wait', 'screenshot'].includes(type)) return RefreshCw

  // Transformation
  if (['transform', 'filter', 'map'].includes(type)) return Sparkles
  if (['validate'].includes(type)) return CheckCircle

  // Control Flow
  if (['conditional', 'loop', 'parallel'].includes(type)) return GitBranch

  return FileText
})

const category = computed(() => getCategoryByNodeType(props.data.nodeType))

const config = computed(() => {
  return categoryConfig[category.value as keyof typeof categoryConfig] || {
    light: 'bg-gray-50 border-gray-200 text-gray-900',
    dark: 'dark:bg-gray-950/50 dark:border-gray-800 dark:text-gray-100',
    badge: 'bg-gray-100 text-gray-700 dark:bg-gray-900/50 dark:text-gray-300',
    icon: 'text-gray-600 dark:text-gray-400'
  }
})

const nodeClasses = computed(() => [
  'rounded-lg border-2 shadow-md hover:shadow-xl min-w-[240px] max-w-[280px] transition-all duration-200',
  'backdrop-blur-sm',
  config.value.light,
  config.value.dark,
  props.selected ? 'ring-2 ring-primary ring-offset-2 ring-offset-background shadow-xl scale-105' : 'hover:scale-102'
])

function getCategoryByNodeType(type: string): string {
  if (['fetch', 'extract_links', 'filter_urls', 'navigate', 'paginate'].includes(type)) {
    return 'URL Discovery'
  }
  if (type.startsWith('extract')) {
    return 'Extraction'
  }
  if (['click', 'scroll', 'type', 'hover', 'wait', 'screenshot'].includes(type)) {
    return 'Interaction'
  }
  if (['transform', 'filter', 'map', 'validate'].includes(type)) {
    return 'Transformation'
  }
  if (['conditional', 'loop', 'parallel'].includes(type)) {
    return 'Control Flow'
  }
  return 'Other'
}

const displayParams = computed(() => {
  const params = props.data.params
  const entries = Object.entries(params).slice(0, 3) // Show max 3 params
  return entries.map(([key, value]) => ({
    key,
    value: typeof value === 'object'
      ? JSON.stringify(value).slice(0, 40) + '...'
      : String(value).slice(0, 40)
  }))
})
</script>

<template>
  <div :class="nodeClasses">
    <!-- Input Handle -->
    <Handle
      type="target"
      :position="Position.Left"
      class="!w-3 !h-3 !bg-primary !border-2 !border-background transition-transform hover:scale-125"
    />

    <!-- Node Header -->
    <div class="p-3 pb-2">
      <div class="flex items-start gap-2">
        <div :class="['p-1.5 rounded-md', config.badge]">
          <component :is="nodeIcon" :class="['w-4 h-4', config.icon]" />
        </div>
        <div class="flex-1 min-w-0">
          <div class="font-semibold text-sm leading-tight mb-1">
            {{ data.label }}
          </div>
          <Badge variant="outline" class="text-xs font-mono">
            {{ data.nodeType }}
          </Badge>
        </div>
      </div>
    </div>

    <!-- Parameters Section -->
    <div v-if="displayParams.length > 0" class="px-3 pb-2">
      <Separator class="mb-2" />
      <div class="space-y-1">
        <div
          v-for="param in displayParams"
          :key="param.key"
          class="text-xs flex items-start gap-1.5"
        >
          <span class="font-medium opacity-70 shrink-0">{{ param.key }}:</span>
          <span class="opacity-60 truncate font-mono">{{ param.value }}</span>
        </div>
        <div v-if="Object.keys(data.params).length > 3" class="text-xs opacity-50 italic">
          +{{ Object.keys(data.params).length - 3 }} more...
        </div>
      </div>
    </div>

    <!-- Footer Badges -->
    <div v-if="data.optional || data.retry" class="px-3 pb-3 flex gap-1.5 flex-wrap">
      <Separator class="mb-1" />
      <Badge v-if="data.optional" variant="secondary" class="text-xs">
        Optional
      </Badge>
      <Badge v-if="data.retry && data.retry.max_retries > 0" variant="secondary" class="text-xs">
        Retry: {{ data.retry.max_retries }}x
      </Badge>
    </div>

    <!-- Output Handle -->
    <Handle
      type="source"
      :position="Position.Right"
      class="!w-3 !h-3 !bg-primary !border-2 !border-background transition-transform hover:scale-125"
    />
  </div>
</template>

<style scoped>
/* Smooth scaling animation */
.scale-102 {
  transform: scale(1.02);
}
</style>
