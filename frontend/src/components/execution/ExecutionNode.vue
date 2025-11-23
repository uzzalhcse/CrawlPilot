<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import { 
  Globe, 
  MousePointer, 
  Type, 
  Clock, 
  Camera, 
  Download, 
  FileText, 
  Code, 
  Filter, 
  List, 
  GitBranch, 
  Repeat, 
  CheckCircle, 
  XCircle, 
  Loader2, 
  Circle,
  AlertCircle
} from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'

const props = defineProps<{
  id: string
  data: {
    label: string
    nodeType: string
    params: any
    status?: 'pending' | 'running' | 'completed' | 'failed'
    error?: string
    startTime?: string
    endTime?: string
    duration?: number
    result?: any
    logs?: string[]
  }
  selected?: boolean
}>()

const statusColor = computed(() => {
  switch (props.data.status) {
    case 'running': return 'border-blue-500 ring-2 ring-blue-500/30 dark:ring-blue-500/50'
    case 'completed': return 'border-green-500'
    case 'failed': return 'border-red-500'
    default: return 'border-border'
  }
})

const statusBg = computed(() => {
  switch (props.data.status) {
    case 'running': return 'bg-blue-500/10 dark:bg-blue-500/20'
    case 'completed': return 'bg-green-500/10 dark:bg-green-500/20'
    case 'failed': return 'bg-red-500/10 dark:bg-red-500/20'
    default: return 'bg-card'
  }
})

const statusIcon = computed(() => {
  switch (props.data.status) {
    case 'running': return Loader2
    case 'completed': return CheckCircle
    case 'failed': return XCircle
    default: return Circle
  }
})

const iconColor = computed(() => {
  switch (props.data.status) {
    case 'running': return 'text-blue-500 animate-spin'
    case 'completed': return 'text-green-500'
    case 'failed': return 'text-red-500'
    default: return 'text-muted-foreground'
  }
})

const getNodeIcon = (type: string) => {
  switch (type) {
    // Discovery
    case 'fetch': return Globe
    case 'extract_links': return List
    case 'filter_urls': return Filter
    case 'navigate': return Globe
    case 'paginate': return Repeat
    // Interaction
    case 'click': return MousePointer
    case 'type': return Type
    case 'wait': return Clock
    case 'screenshot': return Camera
    // Extraction
    case 'extract': return Download
    case 'extract_text': return FileText
    case 'extract_attr': return Code
    case 'extract_json': return Code
    // Logic
    case 'sequence': return List
    case 'conditional': return GitBranch
    case 'loop': return Repeat
    default: return Code
  }
}
</script>

<template>
  <div 
    class="rounded-lg border bg-card text-card-foreground shadow-sm w-[280px] transition-all duration-200"
    :class="[
      statusColor, 
      statusBg,
      selected ? 'ring-2 ring-primary ring-offset-2' : ''
    ]"
  >
    <!-- Header -->
    <div class="p-3 flex items-center gap-3 border-b border-border/50">
      <div 
        class="p-2 rounded-md bg-background border shadow-sm"
        :class="statusColor"
      >
        <component :is="getNodeIcon(data.nodeType)" class="w-4 h-4" />
      </div>
      <div class="flex-1 min-w-0">
        <div class="font-medium text-sm truncate">{{ data.label }}</div>
        <div class="text-xs text-muted-foreground capitalize">{{ data.nodeType.replace('_', ' ') }}</div>
      </div>
      <component :is="statusIcon" class="w-4 h-4" :class="iconColor" />
    </div>

    <!-- Body (Stats) -->
    <div v-if="data.status && data.status !== 'pending'" class="p-2 text-xs space-y-1 bg-background/50 rounded-b-lg">
      <div v-if="data.duration" class="flex justify-between text-muted-foreground">
        <span>Duration:</span>
        <span class="font-mono">{{ data.duration }}ms</span>
      </div>
      <div v-if="data.error" class="flex items-center gap-1 text-destructive font-medium">
        <AlertCircle class="w-3 h-3" />
        <span class="truncate">{{ data.error }}</span>
      </div>
      <div v-if="data.result" class="flex justify-between text-muted-foreground">
        <span>Result:</span>
        <Badge variant="outline" class="text-[10px] h-4 px-1">Captured</Badge>
      </div>
    </div>

    <!-- Handles -->
    <Handle type="target" :position="Position.Top" class="!bg-muted-foreground" />
    <Handle type="source" :position="Position.Bottom" class="!bg-muted-foreground" />
  </div>
</template>
