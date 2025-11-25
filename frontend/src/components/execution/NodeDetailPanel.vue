<script setup lang="ts">
import { computed } from 'vue'
import { 
  X, 
  Clock, 
  CheckCircle, 
  XCircle, 
  Loader2, 
  Circle,
  FileJson,
  Terminal,
  ArrowRight
} from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ScrollArea } from '@/components/ui/scroll-area'

const props = defineProps<{
  node: any
  open: boolean
}>()

const emit = defineEmits(['close'])

const statusColor = computed(() => {
  switch (props.node?.data?.status) {
    case 'running': return 'text-blue-500'
    case 'completed': return 'text-green-500'
    case 'failed': return 'text-red-500'
    default: return 'text-muted-foreground'
  }
})

const statusIcon = computed(() => {
  switch (props.node?.data?.status) {
    case 'running': return Loader2
    case 'completed': return CheckCircle
    case 'failed': return XCircle
    default: return Circle
  }
})

const duration = computed(() => {
  if (!props.node?.data?.startTime) return null
  const start = new Date(props.node.data.startTime).getTime()
  const end = props.node.data.endTime 
    ? new Date(props.node.data.endTime).getTime() 
    : Date.now()
  return end - start
})

const formatJSON = (data: any) => {
  try {
    return JSON.stringify(data, null, 2)
  } catch (e) {
    return String(data)
  }
}
</script>

<template>
  <div 
    v-if="open"
    class="fixed inset-y-0 right-0 w-[400px] bg-background border-l shadow-xl transform transition-transform duration-300 ease-in-out z-50 flex flex-col"
    :class="open ? 'translate-x-0' : 'translate-x-full'"
  >
    <!-- Header -->
    <div class="p-4 border-b flex items-center justify-between bg-muted/10">
      <div class="flex items-center gap-3">
        <div class="p-2 rounded-md bg-muted">
          <component :is="statusIcon" class="w-5 h-5" :class="[statusColor, { 'animate-spin': node?.data?.status === 'running' }]" />
        </div>
        <div>
          <h3 class="font-semibold text-sm">{{ node?.data?.label || 'Select a Node' }}</h3>
          <div class="text-xs text-muted-foreground capitalize">{{ node?.data?.nodeType?.replace('_', ' ') }}</div>
        </div>
      </div>
      <Button variant="ghost" size="icon" @click="emit('close')">
        <X class="w-4 h-4" />
      </Button>
    </div>

    <!-- Content -->
    <div v-if="node" class="flex-1 overflow-hidden flex flex-col">
      <!-- Status Banner -->
      <div class="p-4 bg-muted/30 border-b space-y-2">
        <div class="flex items-center justify-between text-sm">
          <span class="text-muted-foreground">Status</span>
          <Badge :variant="node.data.status === 'failed' ? 'destructive' : 'outline'" class="capitalize">
            {{ node.data.status || 'Pending' }}
          </Badge>
        </div>
        <div v-if="duration" class="flex items-center justify-between text-sm">
          <span class="text-muted-foreground">Duration</span>
          <span class="font-mono">{{ duration }}ms</span>
        </div>
        <div v-if="node.data.error" class="p-2 rounded bg-destructive/10 text-destructive text-xs border border-destructive/20">
          {{ node.data.error }}
        </div>
      </div>

      <!-- Tabs -->
      <Tabs defaultValue="details" class="flex-1 flex flex-col">
        <div class="px-4 pt-2">
          <TabsList class="w-full">
            <TabsTrigger value="details" class="flex-1">Details</TabsTrigger>
            <TabsTrigger value="io" class="flex-1">Input/Output</TabsTrigger>
            <TabsTrigger value="logs" class="flex-1">Logs</TabsTrigger>
          </TabsList>
        </div>

        <div class="flex-1 overflow-hidden">
          <TabsContent value="details" class="h-full p-0 m-0">
            <ScrollArea class="h-full p-4">
              <div class="space-y-4">
                <div>
                  <h4 class="text-xs font-medium text-muted-foreground mb-2 uppercase tracking-wider">Configuration</h4>
                  <div class="rounded-md border bg-muted/30 p-3">
                    <pre class="text-xs font-mono whitespace-pre-wrap">{{ formatJSON(node.data.params) }}</pre>
                  </div>
                </div>
                
                <div v-if="node.data.retry">
                  <h4 class="text-xs font-medium text-muted-foreground mb-2 uppercase tracking-wider">Retry Policy</h4>
                  <div class="grid grid-cols-2 gap-2 text-sm">
                    <div class="p-2 border rounded bg-card">
                      <div class="text-muted-foreground text-xs">Max Retries</div>
                      <div>{{ node.data.retry.max_retries }}</div>
                    </div>
                    <div class="p-2 border rounded bg-card">
                      <div class="text-muted-foreground text-xs">Delay</div>
                      <div>{{ node.data.retry.delay }}ms</div>
                    </div>
                  </div>
                </div>
              </div>
            </ScrollArea>
          </TabsContent>

          <TabsContent value="io" class="h-full p-0 m-0">
            <ScrollArea class="h-full p-4">
              <div class="space-y-6">
                <!-- Input -->
                <div>
                  <div class="flex items-center gap-2 mb-2">
                    <ArrowRight class="w-4 h-4 text-muted-foreground" />
                    <h4 class="text-xs font-medium text-muted-foreground uppercase tracking-wider">Input</h4>
                  </div>
                  <div class="rounded-md border bg-muted/30 p-3">
                    <pre class="text-xs font-mono whitespace-pre-wrap">{{ formatJSON(node.data.params) }}</pre>
                  </div>
                </div>

                <!-- Output -->
                <div>
                  <div class="flex items-center gap-2 mb-2">
                    <FileJson class="w-4 h-4 text-muted-foreground" />
                    <h4 class="text-xs font-medium text-muted-foreground uppercase tracking-wider">Output</h4>
                  </div>
                  <div v-if="node.data.result" class="rounded-md border bg-muted/30 p-3">
                    <pre class="text-xs font-mono whitespace-pre-wrap">{{ formatJSON(node.data.result) }}</pre>
                  </div>
                  <div v-else class="text-sm text-muted-foreground italic p-2">
                    No output captured yet
                  </div>
                </div>
              </div>
            </ScrollArea>
          </TabsContent>

          <TabsContent value="logs" class="h-full p-0 m-0">
            <ScrollArea class="h-full p-4">
              <div v-if="node.data.logs && node.data.logs.length > 0" class="space-y-1">
                <div 
                  v-for="(log, i) in node.data.logs" 
                  :key="i"
                  class="text-xs font-mono p-1 border-b border-border/50 last:border-0"
                >
                  {{ log }}
                </div>
              </div>
              <div v-else class="flex flex-col items-center justify-center h-40 text-muted-foreground">
                <Terminal class="w-8 h-8 mb-2 opacity-50" />
                <span class="text-sm">No logs for this node</span>
              </div>
            </ScrollArea>
          </TabsContent>
        </div>
      </Tabs>
    </div>
  </div>
</template>
