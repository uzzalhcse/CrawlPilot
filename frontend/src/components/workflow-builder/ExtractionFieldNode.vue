<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import { Database, FileText, Code, Link, List, Box } from 'lucide-vue-next'

interface Props {
  data: {
    label: string // The field key
    field: {
      selector: string
      type: string
      attribute?: string
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
</script>

<template>
  <div 
    class="min-w-[200px] max-w-[250px] bg-card border rounded-md shadow-sm transition-all hover:shadow-md"
    :class="{ 'ring-2 ring-primary': selected }"
  >
    <Handle type="target" :position="Position.Left" class="!w-2 !h-2 !bg-muted-foreground" />
    
    <div class="p-2 flex items-center gap-2 border-b bg-muted/20 rounded-t-md">
      <component :is="icon" class="w-4 h-4 text-muted-foreground" />
      <span class="font-mono text-xs font-semibold truncate" :title="data.label">
        {{ data.label }}
      </span>
    </div>
    
    <div class="p-2 text-xs space-y-1">
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
