<template>
  <div class="recovery-event-card">
    <Card class="overflow-hidden transition-all hover:shadow-md">
      <div class="p-4">
        <!-- Header -->
        <div class="flex items-start justify-between gap-3 mb-3">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 mb-1">
              <Badge :variant="statusVariant" class="font-mono text-[10px]">
                {{ event.error_type.split('.').pop() }}
              </Badge>
              <Badge v-if="event.status_code" variant="outline" class="text-[10px]">
                HTTP {{ event.status_code }}
              </Badge>
            </div>
            <p class="text-xs text-muted-foreground">
              {{ formatTime(event.detected_at) }}
              <span v-if="event.time_to_recovery_ms" class="text-muted-foreground/60">
                • Resolved in {{ formatDuration(event.time_to_recovery_ms) }}
              </span>
            </p>
          </div>
          
          <Badge :variant="outcomeVariant" class="shrink-0">
            <component :is="outcomeIcon" class="mr-1 h-3 w-3" />
            {{ outcomeText }}
          </Badge>
        </div>

        <!-- URL -->
        <div class="mb-3">
          <p class="text-xs text-muted-foreground mb-1">URL</p>
          <p class="text-sm font-mono truncate" :title="event.url">{{ event.url }}</p>
        </div>

        <!-- Error Message -->
        <div v-if="event.error_message" class="mb-3">
          <p class="text-xs text-muted-foreground mb-1">Error</p>
          <p class="text-sm text-destructive">{{ event.error_message }}</p>
        </div>

        <Separator class="my-3" />

        <!-- Recovery Details -->
        <div class="space-y-2">
          <!-- Rule -->
          <div v-if="event.rule_name" class="flex items-center justify-between text-sm">
            <span class="text-muted-foreground">Rule Applied</span>
            <div class="flex items-center gap-2">
              <span class="font-medium">{{ event.rule_name }}</span>
              <Badge v-if="event.confidence" variant="secondary" class="text-[10px]">
                {{ Math.round(event.confidence * 100) }}%
              </Badge>
            </div>
          </div>

          <!-- Pattern -->
          <div v-if="event.pattern_detected" class="flex items-center justify-between text-sm">
            <span class="text-muted-foreground">Pattern</span>
            <span class="font-medium">{{ event.pattern_type || 'Detected' }}</span>
          </div>

          <!-- Domain -->
          <div class="flex items-center justify-between text-sm">
            <span class="text-muted-foreground">Domain</span>
            <span class="font-medium font-mono text-xs">{{ event.domain }}</span>
          </div>

          <!-- Retry Count -->
          <div class="flex items-center justify-between text-sm">
            <span class="text-muted-foreground">Retry Count</span>
            <Badge variant="outline" class="text-[10px]">{{ event.retry_count }}</Badge>
          </div>
        </div>

        <!-- Actions -->
        <div v-if="event.actions_applied && event.actions_applied.length > 0" class="mt-3">
          <p class="text-xs text-muted-foreground mb-2">Actions Applied</p>
          <div class="flex flex-wrap gap-1.5">
            <Badge 
              v-for="(action, index) in event.actions_applied" 
              :key="index" 
              variant="secondary"
              class="text-[10px]"
            >
              <component :is="getActionIcon(action.type)" class="mr-1 h-3 w-3" />
              {{ formatActionName(action.type) }}
              <span v-if="hasParams(action)" class="ml-1 opacity-70">
                ({{ formatActionParams(action.parameters) }})
              </span>
            </Badge>
          </div>
        </div>

        <!-- Expandable Details -->
        <div v-if="showDetails && event.request_context" class="mt-3">
          <Button
            variant="ghost"
            size="sm"
            @click="expanded = !expanded"
            class="h-7 text-xs w-full justify-between"
          >
            <span>{{ expanded ? 'Less' : 'More' }} Details</span>
            <component :is="expanded ? ChevronUp : ChevronDown" class="h-3 w-3" />
          </Button>

          <div v-if="expanded" class="mt-3 p-3 rounded-md bg-muted/50">
            <div class="grid grid-cols-2 gap-3 text-xs">
              <div v-if="event.activation_reason">
                <p class="text-muted-foreground mb-1">Activation Reason</p>
                <p class="font-medium">{{ event.activation_reason }}</p>
              </div>
              <div v-if="event.error_rate">
                <p class="text-muted-foreground mb-1">Error Rate</p>
                <p class="font-medium">{{ (event.error_rate * 100).toFixed(2) }}%</p>
              </div>
              <div>
                <p class="text-muted-foreground mb-1">Solution Type</p>
                <p class="font-medium capitalize">{{ event.solution_type }}</p>
              </div>
              <div v-if="event.phase_id">
                <p class="text-muted-foreground mb-1">Phase</p>
                <p class="font-medium font-mono">{{ event.phase_id }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { RecoveryHistoryRecord } from '@/services/recoveryHistoryService'
import { Card } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { 
  CheckCircle, 
  XCircle, 
  AlertCircle,
  Clock,
  Shield,
  Zap,
  RotateCw,
  Timer,
  Users,
  Pause,
  Play,
  ChevronDown,
  ChevronUp
} from 'lucide-vue-next'

interface Props {
  event: RecoveryHistoryRecord
  showDetails?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showDetails: true
})

const expanded = ref(false)

const statusVariant = computed(() => {
  if (props.event.recovery_successful) return 'default'
  if (props.event.recovery_attempted) return 'destructive'
  return 'secondary'
})

const outcomeVariant = computed(() => {
  if (props.event.recovery_successful) return 'default'
  if (props.event.recovery_attempted) return 'destructive'
  return 'secondary'
})

const outcomeIcon = computed(() => {
  if (props.event.recovery_successful) return CheckCircle
  if (props.event.recovery_attempted) return XCircle
  return AlertCircle
})

const outcomeText = computed(() => {
  if (props.event.recovery_successful) return 'Recovered'
  if (props.event.recovery_attempted) return 'Failed'
  return 'No Recovery'
})

const formatTime = (timestamp: string) => {
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  
  if (diffMins < 1) return 'just now'
  if (diffMins < 60) return `${diffMins} minute${diffMins > 1 ? 's' : ''} ago`
  
  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`
  
  const diffDays = Math.floor(diffHours / 24)
  return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`
}

const formatDuration = (ms: number) => {
  if (ms < 1000) return `${ms}ms`
  const seconds = Math.round(ms / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  return `${minutes}m ${remainingSeconds}s`
}

const getActionIcon = (type: string) => {
  const icons: Record<string, any> = {
    wait: Clock,
    enable_stealth: Shield,
    rotate_proxy: RotateCw,
    adjust_timeout: Timer,
    reduce_workers: Users,
    add_delay: Clock,
    pause_execution: Pause,
    resume_execution: Play
  }
  return icons[type] || Zap
}

const formatActionName = (type: string) => {
  return type.split('_').map(word => 
    word.charAt(0).toUpperCase() + word.slice(1)
  ).join(' ')
}

const hasParams = (action: any) => {
  return action.parameters && Object.keys(action.parameters).length > 0
}

const formatActionParams = (params: Record<string, any>) => {
  const entries = Object.entries(params)
  if (entries.length === 0) return ''
  
  return entries.map(([key, value]) => {
    if (key === 'duration' && typeof value === 'number') {
      return value >= 1000 ? `${value / 1000}s` : `${value}ms`
    }
    return value
  }).join(', ')
}
</script>

<style scoped>
.recovery-event-card {
  @apply w-full;
}
</style>
