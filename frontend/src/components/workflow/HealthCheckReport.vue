<script setup lang="ts">
import { computed } from 'vue'
import { CheckCircle2, XCircle, AlertTriangle, Clock, Camera } from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
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
      return { icon: CheckCircle2, variant: 'default', color: 'text-green-600', label: 'Healthy' }
    case 'degraded':
      return { icon: AlertTriangle, variant: 'secondary', color: 'text-yellow-600', label: 'Degraded' }
    case 'failed':
      return { icon: XCircle, variant: 'destructive', color: 'text-red-600', label: 'Failed' }
    default:
      return { icon: Clock, variant: 'outline', color: 'text-blue-600', label: 'Running' }
  }
})

const nodeStatusConfig = (status: string) => {
  switch (status) {
    case 'pass':
      return { icon: CheckCircle2, color: 'text-green-600', label: 'Pass' }
    case 'fail':
      return { icon: XCircle, color: 'text-red-600', label: 'Fail' }
    case 'warning':
      return { icon: AlertTriangle, color: 'text-yellow-600', label: 'Warning' }
    default:
      return { icon: Clock, color: 'text-gray-600', label: 'Skip' }
  }
}

const formatDuration = (ms?: number) => {
  if (!ms) return 'N/A'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

const formatDate = (dateStr: string) => {
  return new Date(dateStr).toLocaleString()
}
</script>

<template>
  <Card>
    <CardHeader>
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <component :is="statusConfig.icon" :class="['h-6 w-6', statusConfig.color]" />
          <div>
            <CardTitle>Health Check Report</CardTitle>
            <CardDescription>{{ formatDate(report.started_at) }}</CardDescription>
          </div>
        </div>
        <Badge :variant="statusConfig.variant as any">{{ statusConfig.label }}</Badge>
      </div>
    </CardHeader>

    <CardContent class="space-y-4">
      <!-- Summary -->
      <div v-if="report.summary" class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div class="text-center p-3 bg-gray-50 rounded-lg">
          <div class="text-2xl font-bold">{{ report.summary.total_nodes }}</div>
          <div class="text-sm text-gray-600">Total Nodes</div>
        </div>
        <div class="text-center p-3 bg-green-50 rounded-lg">
          <div class="text-2xl font-bold text-green-600">{{ report.summary.passed_nodes }}</div>
          <div class="text-sm text-gray-600">Passed</div>
        </div>
        <div class="text-center p-3 bg-yellow-50 rounded-lg">
          <div class="text-2xl font-bold text-yellow-600">{{ report.summary.warning_nodes }}</div>
          <div class="text-sm text-gray-600">Warnings</div>
        </div>
        <div class="text-center p-3 bg-red-50 rounded-lg">
          <div class="text-2xl font-bold text-red-600">{{ report.summary.failed_nodes }}</div>
          <div class="text-sm text-gray-600">Failed</div>
        </div>
      </div>

      <!-- Duration -->
      <div class="text-sm text-gray-600">
        Duration: <span class="font-medium">{{ formatDuration(report.duration_ms) }}</span>
      </div>

      <!-- Critical Issues -->
      <div v-if="report.summary?.critical_issues && report.summary.critical_issues.length > 0" class="space-y-2">
        <h4 class="font-semibold text-sm">Critical Issues</h4>
        <Alert v-for="(issue, idx) in report.summary.critical_issues.slice(0, 5)" :key="idx" variant="destructive">
          <AlertDescription>
            <div class="font-medium">{{ issue.code }}: {{ issue.message }}</div>
            <div v-if="issue.selector" class="text-sm mt-1">Selector: <code class="bg-red-100 px-1 rounded">{{ issue.selector }}</code></div>
            <div v-if="issue.suggestion" class="text-sm mt-1 italic">💡 {{ issue.suggestion }}</div>
          </AlertDescription>
        </Alert>
      </div>

      <!-- Phase Results -->
      <div v-if="report.phase_results" class="space-y-3">
        <h4 class="font-semibold text-sm">Phase Results</h4>
        <div v-for="(phaseResult, phaseId) in report.phase_results" :key="phaseId" class="border rounded-lg p-3">
          <div class="font-medium mb-2">{{ phaseResult.phase_name || phaseId }}</div>
          
          <div class="space-y-2">
            <div v-for="nodeResult in phaseResult.node_results" :key="nodeResult.node_id" class="flex items-start gap-2 text-sm">
              <component 
                :is="nodeStatusConfig(nodeResult.status).icon" 
                :class="['h-4 w-4 mt-0.5 flex-shrink-0', nodeStatusConfig(nodeResult.status).color]" 
              />
              <div class="flex-1">
                <div class="flex items-center justify-between">
                  <div class="font-medium">{{ nodeResult.node_name || nodeResult.node_id }}</div>
                  
                  <!-- Camera icon for failed/warning nodes with snapshots -->
                  <button
                    v-if="(nodeResult.status === 'fail' || nodeResult.status === 'warning') && hasSnapshot && hasSnapshot(nodeResult.node_id)"
                    @click.stop="openSnapshot && openSnapshot(getSnapshotId!(nodeResult.node_id))"
                    class="snapshot-button"
                    title="View diagnostic snapshot"
                  >
                    <Camera class="w-4 h-4" />
                  </button>
                </div>
                <div class="text-gray-600 text-xs">{{ nodeResult.node_type }} • {{ formatDuration(nodeResult.duration_ms) }}</div>
                
                <!-- Node Issues -->
                <div v-if="nodeResult.issues && nodeResult.issues.length > 0" class="mt-1 space-y-1">
                  <div v-for="(issue, idx) in nodeResult.issues" :key="idx" class="text-xs p-2 bg-red-50 rounded">
                    <div class="font-medium">{{ issue.code }}: {{ issue.message }}</div>
                    <div v-if="issue.suggestion" class="text-gray-600 mt-0.5">💡 {{ issue.suggestion }}</div>
                  </div>
                </div>

                <!-- Node Metrics -->
                <div v-if="Object.keys(nodeResult.metrics).length > 0" class="mt-1 text-xs text-gray-500">
                  <span v-for="(value, key) in nodeResult.metrics" :key="key" class="mr-3">
                    {{ key }}: <span class="font-medium">{{ typeof value === 'object' ? JSON.stringify(value).slice(0, 50) : value }}</span>
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </CardContent>
  </Card>
</template>

<style scoped>
.snapshot-button {
  @apply p-1.5 rounded-lg transition-all;
  @apply text-gray-500 hover:text-blue-600;
  @apply hover:bg-blue-50 dark:hover:bg-blue-900/20;
}

.snapshot-button:hover {
  @apply scale-110;
}
</style>
