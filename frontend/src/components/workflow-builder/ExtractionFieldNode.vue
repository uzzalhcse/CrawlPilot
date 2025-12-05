<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import { Database, FileText, Code, Link, List, Box, Zap } from 'lucide-vue-next'

interface Props {
  data: {
    label: string // The field key
    field: {
      selector: string
      type: string
      attribute?: string
      actions?: any[]
      multiple?: boolean
    }
    parentId: string
  }
  selected?: boolean
}

const props = defineProps<Props>()

const icon = computed(() => {
  switch (props.data.field.type) {
    case 'text': return FileText
    case 'html': return Code
    case 'attribute': return Link
    case 'list': return List
    case 'nested': return Box
    default: return Database
  }
})

// Count actions
const actionsCount = computed(() => {
  if (!props.data.field.actions) return 0
  if (Array.isArray(props.data.field.actions)) return props.data.field.actions.length
  return 0
})

// Check if field is array type
const isMultiple = computed(() => props.data.field.multiple === true)
</script>

<template>
  <div 
    class="min-w-[140px] max-w-[180px] bg-card border rounded-md shadow-sm transition-all hover:shadow-md"
    :class="{ 'ring-2 ring-primary': selected }"
  >
    <Handle type="target" :position="Position.Left" class="!w-2 !h-2 !bg-muted-foreground" />
    
    <!-- Header with label and badges -->
    <div class="p-1.5 flex items-center gap-1.5 border-b bg-muted/20 rounded-t-md">
      <component :is="icon" class="w-3 h-3 text-muted-foreground shrink-0" />
      <span class="font-mono text-[10px] font-semibold truncate flex-1" :title="data.label">
        {{ data.label }}
      </span>
      
      <!-- Actions Badge -->
      <div 
        v-if="actionsCount > 0"
        class="flex items-center gap-0.5 px-1 py-0.5 rounded-full bg-amber-500/20 text-amber-600 dark:text-amber-400"
        :title="`${actionsCount} pre-extraction action(s)`"
      >
        <Zap class="w-2.5 h-2.5" />
        <span class="text-[8px] font-bold">{{ actionsCount }}</span>
      </div>
      
      <!-- Multiple Badge -->
      <div 
        v-if="isMultiple"
        class="px-1 py-0.5 rounded-full bg-purple-500/20 text-purple-600 dark:text-purple-400 text-[8px] font-bold"
        title="Extracts multiple values (array)"
      >
        []
      </div>
    </div>
    
    <!-- Selector -->
    <div class="p-1.5 text-[10px] space-y-0.5">
      <div class="text-muted-foreground truncate" :title="data.field.selector">
        {{ data.field.selector || 'No selector' }}
      </div>
      <div v-if="data.field.type === 'attribute'" class="flex items-center gap-1 text-[10px] bg-muted inline-flex px-1 rounded">
        <span class="opacity-50">@</span>
        {{ data.field.attribute }}
      </div>
    </div>

    <Handle type="source" :position="Position.Right" class="!w-2 !h-2 !bg-muted-foreground" />
  </div>
</template>
