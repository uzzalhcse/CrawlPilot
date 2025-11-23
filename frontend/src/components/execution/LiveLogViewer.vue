<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { Badge } from '@/components/ui/badge'
import type { LogEntry } from '@/composables/useExecutionStream'

const props = defineProps<{
  logs: LogEntry[]
}>()

const scrollViewport = ref<HTMLElement | null>(null)
const autoScroll = ref(true)

watch(() => props.logs.length, () => {
  if (autoScroll.value) {
    nextTick(() => {
      scrollToBottom()
    })
  }
})

function scrollToBottom() {
  if (scrollViewport.value) {
    scrollViewport.value.scrollTop = scrollViewport.value.scrollHeight
  }
}

function getLevelColor(level: string) {
  switch (level) {
    case 'info': return 'text-blue-500'
    case 'warn': return 'text-yellow-500'
    case 'error': return 'text-red-500'
    case 'debug': return 'text-gray-500'
    default: return 'text-foreground'
  }
}

function formatTime(isoString: string) {
  const date = new Date(isoString)
  return `${date.toLocaleTimeString([], { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })}.${date.getMilliseconds().toString().padStart(3, '0')}`
}
</script>

<template>
  <div class="flex flex-col h-full border rounded-md bg-black font-mono text-xs">
    <div class="flex items-center justify-between px-4 py-2 border-b bg-muted/10">
      <span class="font-semibold text-muted-foreground">Live Logs</span>
      <div class="flex items-center gap-2">
        <label class="flex items-center gap-2 text-xs cursor-pointer select-none">
          <input type="checkbox" v-model="autoScroll" class="rounded border-gray-600 bg-transparent" />
          Auto-scroll
        </label>
      </div>
    </div>
    
    <div class="flex-1 p-4 h-[300px] overflow-y-auto" ref="scrollViewport">
      <div v-if="logs.length === 0" class="text-muted-foreground italic text-center mt-10">
        Waiting for logs...
      </div>
      <div v-else class="space-y-1">
        <div v-for="log in logs" :key="log.id" class="flex gap-2 hover:bg-white/5 p-0.5 rounded">
          <span class="text-gray-500 shrink-0 select-none">[{{ formatTime(log.timestamp) }}]</span>
          <span :class="['font-bold uppercase shrink-0 w-12', getLevelColor(log.level)]">{{ log.level }}</span>
          <span class="text-gray-300 break-all">{{ log.message }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
