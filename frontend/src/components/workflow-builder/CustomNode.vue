<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import type { NodeData } from '@/types'
import { Badge } from '@/components/ui/badge'
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
  Sparkles,
  Clock,
  Target,
  Hash
} from 'lucide-vue-next'

interface Props {
  data: NodeData
  selected?: boolean
}

const props = defineProps<Props>()

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
  if (['wait', 'wait_for', 'screenshot'].includes(type)) return RefreshCw
  if (['sequence'].includes(type)) return GitBranch

  // Transformation
  if (['transform', 'filter', 'map'].includes(type)) return Sparkles
  if (['validate'].includes(type)) return CheckCircle

  // Control Flow
  if (['conditional', 'loop', 'parallel'].includes(type)) return GitBranch

  return FileText
})

const category = computed(() => getCategoryByNodeType(props.data.nodeType))

const nodeClasses = computed(() => [
  'rounded-lg border-2 shadow-md hover:shadow-xl min-w-[260px] max-w-[300px] transition-all duration-200 overflow-hidden',
  props.selected ? 'ring-2 ring-primary ring-offset-2 ring-offset-background shadow-xl scale-105' : 'hover:scale-102'
])

function getCategoryByNodeType(type: string): string {
  if (['fetch', 'extract_links', 'filter_urls', 'navigate', 'paginate'].includes(type)) {
    return 'URL Discovery'
  }
  if (type.startsWith('extract')) {
    return 'Extraction'
  }
  if (['click', 'scroll', 'type', 'hover', 'wait', 'wait_for', 'screenshot'].includes(type)) {
    return 'Interaction'
  }
  if (['transform', 'filter', 'map', 'validate'].includes(type)) {
    return 'Transformation'
  }
  if (['sequence', 'conditional', 'loop', 'parallel'].includes(type)) {
    return 'Control Flow'
  }
  return 'Other'
}

function getHeaderClass() {
  switch (category.value) {
    case 'URL Discovery':
      return 'bg-blue-200 text-blue-900 dark:bg-blue-700 dark:text-blue-50'
    case 'Extraction':
      return 'bg-purple-200 text-purple-900 dark:bg-purple-700 dark:text-purple-50'
    case 'Interaction':
      return 'bg-green-200 text-green-900 dark:bg-green-700 dark:text-green-50'
    case 'Transformation':
      return 'bg-amber-200 text-amber-900 dark:bg-amber-700 dark:text-amber-50'
    case 'Control Flow':
      return 'bg-pink-200 text-pink-900 dark:bg-pink-700 dark:text-pink-50'
    default:
      return 'bg-gray-200 text-gray-900 dark:bg-gray-700 dark:text-gray-50'
  }
}

const keyParams = computed(() => {
  const params = props.data.params
  const result: Array<{ key: string, label: string, value: string, icon: any }> = []
  
  if (params.selector) {
    result.push({
      key: 'selector',
      label: 'Selector',
      value: String(params.selector).slice(0, 35),
      icon: Target
    })
  }
  
  if (params.timeout || params.duration) {
    const time = params.timeout || params.duration
    result.push({
      key: 'time',
      label: 'Time',
      value: typeof time === 'number' ? `${time / 1000}s` : String(time),
      icon: Clock
    })
  }
  
  if (params.schema) {
    result.push({
      key: 'schema',
      label: 'Source',
      value: String(params.schema),
      icon: Database
    })
  }
  
  if (params.marker) {
    result.push({
      key: 'marker',
      label: 'Marker',
      value: String(params.marker),
      icon: Hash
    })
  }

  if (params.max_pages) {
    result.push({
      key: 'max_pages',
      label: 'Pages',
      value: String(params.max_pages),
      icon: FileText
    })
  }
  
  return result.slice(0, 3)
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

    <!-- Level Indicator -->
    <div v-if="data.level && data.level > 0" class="absolute -top-6 left-0 text-xs font-medium opacity-60">
      Level {{ data.level }}
    </div>

    <!-- Node Header - Prominent -->
    <div :class="['p-3 rounded-t-lg font-semibold text-sm', getHeaderClass()]">
      <div class="flex items-center gap-2">
        <component :is="nodeIcon" class="w-5 h-5" />
        <span>{{ data.nodeType }}</span>
      </div>
    </div>

    <!-- Node Content -->
    <div class="p-4 bg-white dark:bg-gray-900 border-t-2 border-gray-100 dark:border-gray-800">
      <div class="font-medium text-base mb-3 text-gray-900 dark:text-gray-100">
        {{ data.label }}
      </div>

      <!-- Key Parameters -->
      <div v-if="keyParams.length > 0" class="space-y-2">
        <div
          v-for="param in keyParams"
          :key="param.key"
          class="flex items-center gap-2 text-xs bg-gray-50 dark:bg-gray-800 p-2 rounded text-gray-700 dark:text-gray-300"
        >
          <component :is="param.icon" class="w-4 h-4 opacity-60 shrink-0" />
          <span class="font-medium shrink-0">{{ param.label }}:</span>
          <span class="truncate">{{ param.value }}</span>
        </div>
      </div>

      <!-- Branch Indicator -->
      <div v-if="data.branch" class="mt-3">
        <Badge :class="data.branch === 'true' ? 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300' : 'bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300'">
          {{ data.branch === 'true' ? '✓ True Branch' : '✗ False Branch' }}
        </Badge>
      </div>

      <!-- Badges -->
      <div v-if="data.optional || (data.retry && data.retry.max_retries > 0)" class="mt-3 flex gap-2 flex-wrap">
        <Badge v-if="data.optional" variant="secondary" class="text-xs">
          Optional
        </Badge>
        <Badge v-if="data.retry && data.retry.max_retries > 0" variant="secondary" class="text-xs">
          Retry: {{ data.retry.max_retries }}x
        </Badge>
      </div>
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
