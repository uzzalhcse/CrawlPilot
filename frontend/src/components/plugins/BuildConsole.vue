<template>
  <div class="build-console h-full flex flex-col">
    <!-- Build Header -->
    <div class="flex items-center justify-between px-4 py-3 bg-muted/30 border-b">
      <div class="flex items-center gap-3">
        <div :class="['flex items-center gap-2', getStatusColor()]">
          <component :is="getStatusIcon()" class="h-5 w-5" />
          <span class="font-semibold capitalize">{{ buildStatus }}</span>
        </div>
        <span v-if="buildJob" class="text-sm text-muted-foreground">
          Build ID: {{ buildJob.id.slice(0, 8) }}
        </span>
      </div>
    </div>

    <!-- Build Log -->
    <ScrollArea class="flex-1 bg-black/90 text-green-400 font-mono text-sm">
      <div class="p-4">
        <template v-if="buildLog">
          <div v-for="(line, index) in buildLogLines" :key="index" class="leading-relaxed">
            {{ line }}
          </div>
        </template>
        <template v-else-if="building">
          <div class="flex items-center gap-2">
            <Loader2 class="h-4 w-4 animate-spin" />
            <span>Building...</span>
          </div>
        </template>
        <template v-else>
          <span class="text-muted-foreground">No build logs yet. Click "Build Plugin" to start.</span>
        </template>
      </div>
    </ScrollArea>

    <!-- Build Actions -->
    <div class="flex items-center justify-between px-4 py-3 bg-muted/30 border-t">
      <div class="text-xs text-muted-foreground">
        <template v-if="buildJob?.created_at">
          Started: {{ formatDate(buildJob.created_at) }}
        </template>
        <template v-if="buildJob?.completed_at">
          • Completed: {{ formatDate(buildJob.completed_at) }}
        </template>
      </div>
      <Button
        size="sm"
        @click="$emit('build')"
        :disabled="building"
      >
        <Hammer class="h-4 w-4 mr-2" v-if="!building" />
        <Loader2 class="h-4 w-4 mr-2 animate-spin" v-else />
        {{ building ? 'Building...' : 'Build Plugin' }}
      </Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { 
  Loader2, 
  CheckCircle2, 
  XCircle, 
  Clock,
  Hammer
} from 'lucide-vue-next'

interface BuildJob {
  id: string
  plugin_slug: string
  status: 'queued' | 'building' | 'success' | 'failed'
  log: string
  artifact?: string
  created_at: string
  completed_at?: string
}

interface Props {
  buildJob: BuildJob | null
}

interface Emits {
  (e: 'build'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const buildStatus = computed(() => props.buildJob?.status || 'idle')
const buildLog = computed(() => props.buildJob?.log || '')
const buildLogLines = computed(() => buildLog.value.split('\n').filter(line => line.trim()))
const building = computed(() => buildStatus.value === 'queued' || buildStatus.value === 'building')

function getStatusIcon() {
  switch (buildStatus.value) {
    case 'success': return CheckCircle2
    case 'failed': return XCircle
    case 'building': return Loader2
    case 'queued': return Clock
    default: return Hammer
  }
}

function getStatusColor(): string {
  switch (buildStatus.value) {
    case 'success': return 'text-green-600 dark:text-green-400'
    case 'failed': return 'text-red-600 dark:text-red-400'
    case 'building': return 'text-blue-600 dark:text-blue-400'
    case 'queued': return 'text-yellow-600 dark:text-yellow-400'
    default: return 'text-muted-foreground'
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString()
}
</script>

<style scoped>
.build-console {
  min-height: 400px;
}
</style>
