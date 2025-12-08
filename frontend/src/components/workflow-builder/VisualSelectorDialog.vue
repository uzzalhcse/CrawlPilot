<script setup lang="ts">
import { ref, watch, computed, onUnmounted } from 'vue'
import { visualSelectorApi } from '@/api/visual-selector'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Loader2, Target, AlertCircle, CheckCircle2 } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

interface Props {
  open: boolean
  workflowId?: string
  existingFields?: Record<string, any> // Existing extract fields to pre-populate
}

interface Emits {
  (e: 'update:open', value: boolean): void
  (e: 'fields-selected', fields: Record<string, any>): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const url = ref('')
const sessionId = ref<string | null>(null)
const status = ref<'idle' | 'launching' | 'running' | 'completed' | 'failed' | 'timeout'>('idle')
const fields = ref<Record<string, any> | null>(null)
const error = ref<string | null>(null)
const pollInterval = ref<number | null>(null)
const pollCount = ref(0)
const MAX_POLL_COUNT = 300 * 6 // 30 minutes at 1 second intervals

const isOpen = computed({
  get: () => props.open,
  set: (value) => emit('update:open', value)
})

const canStart = computed(() => {
  return url.value.trim() !== '' && status.value === 'idle'
})

const statusMessage = computed(() => {
  switch (status.value) {
    case 'launching':
      return 'Launching browser...'
    case 'running':
      return 'Browser session active. Select elements and click "Done" when finished.'
    case 'completed':
      return `${Object.keys(fields.value || {}).length} fields collected!`
    case 'failed':
      return error.value || 'Session failed'
    case 'timeout':
      return 'Session timed out. Please try again.'
    default:
      return ''
  }
})

watch(() => props.open, (newOpen) => {
  if (!newOpen) {
    // Reset state when dialog closes
    stopPolling()
    if (sessionId.value && status.value === 'running') {
      // Close the browser session if still running
      visualSelectorApi.closeSession(sessionId.value).catch(() => {})
    }
    resetState()
  }
})

onUnmounted(() => {
  stopPolling()
})

function resetState() {
  url.value = ''
  sessionId.value = null
  status.value = 'idle'
  fields.value = null
  error.value = null
  pollCount.value = 0
}

async function startSession() {
  if (!canStart.value) return

  status.value = 'launching'
  error.value = null

  try {
    const response = await visualSelectorApi.startSession({
      url: url.value,
      workflow_id: props.workflowId,
      existing_fields: props.existingFields
    })

    sessionId.value = response.data.session_id
    status.value = 'running'
    
    // Start polling for results
    startPolling()
    
    toast.success('Browser launched! Select elements on the page.')
  } catch (err: any) {
    status.value = 'failed'
    error.value = err.response?.data?.error || err.message || 'Failed to start session'
    toast.error('Failed to start visual selector', { description: error.value || undefined })
  }
}

function startPolling() {
  stopPolling() // Clear any existing interval
  pollCount.value = 0
  
  pollInterval.value = window.setInterval(async () => {
    if (!sessionId.value) {
      stopPolling()
      return
    }

    pollCount.value++
    
    // Check for timeout
    if (pollCount.value >= MAX_POLL_COUNT) {
      status.value = 'timeout'
      stopPolling()
      return
    }

    try {
      const response = await visualSelectorApi.getResult(sessionId.value)
      const session = response.data

      if (session.status === 'completed') {
        status.value = 'completed'
        fields.value = session.fields || {}
        stopPolling()
        toast.success('Fields collected successfully!')
      } else if (session.status === 'failed') {
        status.value = 'failed'
        error.value = session.error || 'Session failed'
        stopPolling()
      }
    } catch (err: any) {
      // Ignore polling errors, just continue
      console.warn('Polling error:', err)
    }
  }, 1000) // Poll every second
}

function stopPolling() {
  if (pollInterval.value) {
    clearInterval(pollInterval.value)
    pollInterval.value = null
  }
}

function applyFields() {
  if (fields.value) {
    emit('fields-selected', fields.value)
    isOpen.value = false
    toast.success('Fields applied to workflow!')
  }
}

function closeDialog() {
  isOpen.value = false
}
</script>

<template>
  <Dialog v-model:open="isOpen">
    <DialogContent class="sm:max-w-[500px]">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2">
          <Target class="w-5 h-5 text-primary" />
          Visual Selector
        </DialogTitle>
        <DialogDescription>
          Enter a product page URL to visually select data extraction fields using point-and-click.
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-4">
        <div class="space-y-2">
          <Label for="url">Page URL</Label>
          <Input
            id="url"
            v-model="url"
            placeholder="https://example.com/product/123"
            :disabled="status !== 'idle'"
          />
        </div>

        <!-- Status Display -->
        <div v-if="status !== 'idle'" class="space-y-2">
          <Alert :variant="status === 'failed' || status === 'timeout' ? 'destructive' : 'default'">
            <component 
              :is="status === 'completed' ? CheckCircle2 : status === 'failed' || status === 'timeout' ? AlertCircle : Loader2" 
              :class="['w-4 h-4', { 'animate-spin': status === 'launching' || status === 'running' }]"
            />
            <AlertDescription>
              {{ statusMessage }}
            </AlertDescription>
          </Alert>
          
          <!-- Polling progress -->
          <div v-if="status === 'running'" class="text-xs text-muted-foreground text-center">
            Waiting for completion... ({{ Math.floor((MAX_POLL_COUNT - pollCount) / 60) }}m {{ (MAX_POLL_COUNT - pollCount) % 60 }}s remaining)
          </div>
        </div>

        <!-- Fields Preview -->
        <div v-if="status === 'completed' && fields" class="space-y-2">
          <Label>Collected Fields</Label>
          <div class="bg-muted rounded-md p-3 max-h-[200px] overflow-auto">
            <pre class="text-xs">{{ JSON.stringify(fields, null, 2) }}</pre>
          </div>
        </div>
      </div>

      <DialogFooter class="gap-2">
        <Button variant="outline" @click="closeDialog">
          {{ status === 'completed' ? 'Cancel' : 'Close' }}
        </Button>
        
        <Button 
          v-if="status === 'idle'" 
          @click="startSession" 
          :disabled="!canStart"
        >
          <Target class="w-4 h-4 mr-2" />
          Launch Browser
        </Button>
        
        <Button 
          v-if="status === 'completed'" 
          @click="applyFields"
          variant="default"
        >
          <CheckCircle2 class="w-4 h-4 mr-2" />
          Apply to Workflow
        </Button>
        
        <Button 
          v-if="status === 'failed' || status === 'timeout'" 
          @click="resetState"
          variant="outline"
        >
          Try Again
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
