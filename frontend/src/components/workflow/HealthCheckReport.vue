<script setup lang="ts">
import { computed } from 'vue'
import { CheckCircle2, XCircle, AlertTriangle, Clock, Camera } from 'lucide-vue-next'
import { Card } from '@/components/ui/card'

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import type { HealthCheckReport } from '@/types'

const props = defineProps<{
  report: HealthCheckReport
  hasSnapshot?: (nodeId: string) => boolean
  getSnapshotId?: (nodeId: string) => string | null
  openSnapshot?: (snapshotId: string | null) => void
}>()

const statusConfig = computed(() => {
  switch (props.report.status) {
    case 'healthy':
      return { icon: CheckCircle2, color: 'text-green-600 dark:text-green-400', label: 'Healthy' }
    case 'degraded':
      return { icon: AlertTriangle, color: 'text-amber-600 dark:text-amber-400', label: 'Degraded' }
    case 'failed':
      return { icon: XCircle, color: 'text-red-600 dark:text-red-400', label: 'Failed' }
    default:
      return { icon: Clock, color: 'text-blue-600 dark:text-blue-400', label: 'Running' }
  }
})

const nodeStatusConfig = (status: string) => {
  switch (status) {
    case 'pass':
      return { icon: CheckCircle2, color: 'text-green-600 dark:text-green-400', label: 'Pass' }
    case 'fail':
      return { icon: XCircle, color: 'text-red-600 dark:text-red-400', label: 'Fail' }
    case 'warning':
      return { icon: AlertTriangle, color: 'text-amber-600 dark:text-amber-400', label: 'Warning' }
    default:
      return { icon: Clock, color: 'text-muted-foreground', label: 'Skip' }
  }
}

const formatDuration = (ms?: number) => {
  if (!ms) return 'N/A'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}
</script>

<template>
  <div class="space-y-4">
    <!-- Critical Issues -->
    <div v-if="report.summary?.critical_issues && report.summary.critical_issues.length > 0" class="space-y-2">
      <h3 class="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Critical Issues</h3>
      <Card
        v-for="(issue, idx) in report.summary.critical_issues.slice(0, 5)"
        :key="idx"
        class="p-3 border-l-2 border-l-red-500 bg-red-50 dark:bg-red-950 border-red-200 dark:border-red-800"
      >
        <div class="space-y-1.5">
          <div class="text-sm font-medium text-red-900 dark:text-red-100">{{ issue.code }}: {{ issue.message }}</div>
          <div v-if="issue.selector" class="text-xs">
            <span class="text-muted-foreground">Selector:</span>
            <code class="ml-1 bg-red-100 dark:bg-red-900 px-1.5 py-0.5 rounded text-xs">{{ issue.selector }}</code>
          </div>
          <div v-if="issue.suggestion" class="text-xs text-muted-foreground italic">💡 {{ issue.suggestion }}</div>
        </div>
      </Card>
    </div>

    <!-- Phase Results -->
    <div v-if="report.phase_results" class="space-y-3">
      <h3 class="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Phase Results</h3>
      
      <Accordion type="multiple" class="space-y-2" default-value="phase-0">
        <AccordionItem 
          v-for="(phaseResult, phaseId, index) in report.phase_results" 
          :key="phaseId"
          :value="`phase-${index}`"
          class="border rounded-lg"
        >
          <AccordionTrigger class="px-4 py-2.5 hover:no-underline text-sm font-medium">
            <div class="flex items-center justify-between w-full pr-2">
              <span>{{ phaseResult.phase_name || phaseId }}</span>
              <div class="flex items-center gap-2 text-xs">
                <div v-if="phaseResult.node_results" class="flex items-center gap-3">
                  <span class="text-green-600 dark:text-green-400">
                    ✓ {{ phaseResult.node_results.filter(n => n.status === 'pass').length }}
                  </span>
                  <span v-if="phaseResult.node_results.filter(n => n.status === 'warning').length > 0" class="text-amber-600 dark:text-amber-400">
                    ⚠ {{ phaseResult.node_results.filter(n => n.status === 'warning').length }}
                  </span>
                  <span v-if="phaseResult.node_results.filter(n => n.status === 'fail').length > 0" class="text-red-600 dark:text-red-400">
                    ✗ {{ phaseResult.node_results.filter(n => n.status === 'fail').length }}
                  </span>
                </div>
              </div>
            </div>
          </AccordionTrigger>
          
          <AccordionContent class="px-4 pb-3 pt-1">
            <div class="space-y-2">
              <div 
                v-for="nodeResult in phaseResult.node_results" 
                :key="nodeResult.node_id"
                class="flex items-start gap-2 p-2.5 rounded-md border bg-card hover:bg-accent/50 transition-colors"
              >
                <component 
                  :is="nodeStatusConfig(nodeResult.status).icon" 
                  :class="['h-4 w-4 mt-0.5 flex-shrink-0', nodeStatusConfig(nodeResult.status).color]" 
                />
                <div class="flex-1 min-w-0 space-y-1">
                  <div class="flex items-center justify-between gap-2">
                    <span class="text-sm font-medium truncate">{{ nodeResult.node_name || nodeResult.node_id }}</span>
                    
                    <!-- Snapshot button -->
                    <button
                      v-if="(nodeResult.status === 'fail' || nodeResult.status === 'warning') && hasSnapshot && hasSnapshot(nodeResult.node_id)"
                      @click.stop="openSnapshot && openSnapshot(getSnapshotId!(nodeResult.node_id))"
                      class="flex-shrink-0 p-1 rounded hover:bg-background transition-colors text-muted-foreground hover:text-primary"
                      title="View diagnostic snapshot"
                    >
                      <Camera class="w-3.5 h-3.5" />
                    </button>
                  </div>
                  
                  <div class="text-xs text-muted-foreground">
                    {{ nodeResult.node_type }} • {{ formatDuration(nodeResult.duration_ms) }}
                  </div>
                  
                  <!-- Node Issues -->
                  <div v-if="nodeResult.issues && nodeResult.issues.length > 0" class="space-y-1 mt-1.5">
                    <div 
                      v-for="(issue, idx) in nodeResult.issues" 
                      :key="idx"
                      class="text-xs p-2 bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 rounded"
                    >
                      <div class="font-medium">{{ issue.code }}: {{ issue.message }}</div>
                      <div v-if="issue.suggestion" class="text-muted-foreground mt-0.5">💡 {{ issue.suggestion }}</div>
                    </div>
                  </div>

                  <!-- Node Metrics -->
                  <div v-if="Object.keys(nodeResult.metrics).length > 0" class="mt-1 text-xs text-muted-foreground">
                    <span v-for="(value, key) in nodeResult.metrics" :key="key" class="mr-3">
                      {{ key }}: <span class="font-medium">{{ typeof value === 'object' ? JSON.stringify(value).slice(0, 50) : value }}</span>
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </AccordionContent>
        </AccordionItem>
      </Accordion>
    </div>
  </div>
</template>
