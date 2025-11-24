<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { Activity, RefreshCw, Clock, TrendingUp } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import HealthCheckButton from '@/components/workflow/HealthCheckButton.vue'
import HealthCheckReport from '@/components/workflow/HealthCheckReport.vue'
import { useHealthCheckStore } from '@/stores/healthcheck'
import { useWorkflowsStore } from '@/stores/workflows'
import type { HealthCheckReport as HealthCheckReportType } from '@/types'

const route = useRoute()
const workflowId = ref(route.params.id as string)

const healthCheckStore = useHealthCheckStore()
const workflowStore = useWorkflowsStore()

const selectedReport = ref<HealthCheckReportType | null>(null)
const refreshing = ref(false)

// Computed
const workflow = computed(() => workflowStore.workflows.find((w: any) => w.id === workflowId.value))

const sortedReports = computed(() => {
  return [...healthCheckStore.reports].sort((a, b) => 
    new Date(b.started_at).getTime() - new Date(a.started_at).getTime()
  )
})

const statusStats = computed(() => {
  const stats = {
    total: healthCheckStore.reports.length,
    healthy: 0,
    degraded: 0,
    failed: 0,
    running: 0
  }
  
  healthCheckStore.reports.forEach(report => {
    if (report.status === 'healthy') stats.healthy++
    else if (report.status === 'degraded') stats.degraded++
    else if (report.status === 'failed') stats.failed++
    else if (report.status === 'running') stats.running++
  })
  
  return stats
})

// Methods
const fetchData = async () => {
  await Promise.all([
    workflowStore.fetchWorkflows(),
    healthCheckStore.fetchHealthChecks(workflowId.value, 20)
  ])
  
  if (selectedReport.value) {
    await healthCheckStore.fetchHealthCheckReport(selectedReport.value.id)
    selectedReport.value = healthCheckStore.currentReport
  }
}

const refresh = async () => {
  refreshing.value = true
  try {
    await fetchData()
  } finally {
    refreshing.value = false
  }
}

const selectReport = async (report: HealthCheckReportType) => {
  if (report.phase_results) {
    // Already has full data
    selectedReport.value = report
  } else {
    // Need to fetch full report
    await healthCheckStore.fetchHealthCheckReport(report.id)
    selectedReport.value = healthCheckStore.currentReport
  }
}

const getStatusVariant = (status: string) => {
  switch (status) {
    case 'healthy': return 'default'
    case 'degraded': return 'secondary'
    case 'failed': return 'destructive'
    default: return 'outline'
  }
}

const formatDate = (dateStr: string) => {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  
  // Less than 1 hour
  if (diff < 3600000) {
    const minutes = Math.floor(diff / 60000)
    return minutes === 0 ? 'Just now' : `${minutes}m ago`
  }
  
  // Less than 24 hours
  if (diff < 86400000) {
    const hours = Math.floor(diff / 3600000)
    return `${hours}h ago`
  }
  
  // Otherwise show date
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

const formatDuration = (ms?: number) => {
  if (!ms) return 'N/A'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

onMounted(async () => {
  await fetchData()
  
  // Auto-select latest report if available
  if (sortedReports.value.length > 0) {
    await selectReport(sortedReports.value[0])
  }
})
</script>

<template>
  <div class="container mx-auto py-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold tracking-tight">Health Checks</h1>
        <p class="text-muted-foreground">
          Monitor workflow validation and detect structure changes
        </p>
      </div>
      <div class="flex gap-2">
        <Button @click="refresh" :disabled="refreshing" variant="outline" size="sm">
          <RefreshCw :class="['h-4 w-4 mr-2', refreshing && 'animate-spin']" />
          Refresh
        </Button>
        <HealthCheckButton v-if="workflow" :workflow-id="workflowId" @click="refresh" />
      </div>
    </div>

    <!-- Stats Cards -->
    <div class="grid gap-4 md:grid-cols-4">
      <Card>
        <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle class="text-sm font-medium">Total Checks</CardTitle>
          <Activity class="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold">{{ statusStats.total }}</div>
          <p class="text-xs text-muted-foreground">All time health checks</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle class="text-sm font-medium">Healthy</CardTitle>
          <TrendingUp class="h-4 w-4 text-green-600" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold text-green-600">{{ statusStats.healthy }}</div>
          <p class="text-xs text-muted-foreground">Passed validation</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle class="text-sm font-medium">Degraded</CardTitle>
          <Activity class="h-4 w-4 text-yellow-600" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold text-yellow-600">{{ statusStats.degraded }}</div>
          <p class="text-xs text-muted-foreground">With warnings</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle class="text-sm font-medium">Failed</CardTitle>
          <Activity class="h-4 w-4 text-red-600" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold text-red-600">{{ statusStats.failed }}</div>
          <p class="text-xs text-muted-foreground">Critical issues</p>
        </CardContent>
      </Card>
    </div>

    <!-- Main Content -->
    <div class="grid gap-6 md:grid-cols-3">
      <!-- Health Check History -->
      <Card class="md:col-span-1">
        <CardHeader>
          <CardTitle>History</CardTitle>
          <CardDescription>Recent health check executions</CardDescription>
        </CardHeader>
        <CardContent class="p-0">
          <div class="max-h-[600px] overflow-y-auto">
            <div v-if="sortedReports.length === 0" class="p-6 text-center text-muted-foreground">
              <Activity class="h-12 w-12 mx-auto mb-2 opacity-50" />
              <p>No health checks yet</p>
              <p class="text-sm">Run your first check to get started</p>
            </div>
            
            <div 
              v-for="report in sortedReports" 
              :key="report.id"
              :class="[
                'p-4 border-b cursor-pointer hover:bg-accent transition-colors',
                selectedReport?.id === report.id && 'bg-accent'
              ]"
              @click="selectReport(report)"
            >
              <div class="flex items-center justify-between mb-2">
                <Badge :variant="getStatusVariant(report.status) as any">
                  {{ report.status }}
                </Badge>
                <span class="text-xs text-muted-foreground">
                  {{ formatDate(report.started_at) }}
                </span>
              </div>
              
              <div v-if="report.summary" class="grid grid-cols-3 gap-2 text-xs">
                <div class="text-center p-1 bg-green-50 rounded">
                  <div class="font-bold text-green-600">{{ report.summary.passed_nodes }}</div>
                  <div class="text-gray-600">Pass</div>
                </div>
                <div class="text-center p-1 bg-yellow-50 rounded">
                  <div class="font-bold text-yellow-600">{{ report.summary.warning_nodes }}</div>
                  <div class="text-gray-600">Warn</div>
                </div>
                <div class="text-center p-1 bg-red-50 rounded">
                  <div class="font-bold text-red-600">{{ report.summary.failed_nodes }}</div>
                  <div class="text-gray-600">Fail</div>
                </div>
              </div>

              <div class="text-xs text-muted-foreground mt-2">
                <Clock class="h-3 w-3 inline mr-1" />
                {{ formatDuration(report.duration_ms) }}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- Report Detail -->
      <Card class="md:col-span-2">
        <CardHeader>
          <CardTitle>Report Details</CardTitle>
          <CardDescription>
            <span v-if="selectedReport">{{ formatDate(selectedReport.started_at) }}</span>
            <span v-else>Select a health check to view details</span>
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div v-if="!selectedReport" class="text-center py-12 text-muted-foreground">
            <Activity class="h-16 w-16 mx-auto mb-4 opacity-50" />
            <p class="text-lg">No report selected</p>
            <p class="text-sm">Click on a health check from the history to view details</p>
          </div>

          <div v-else-if="healthCheckStore.loading" class="text-center py-12">
            <RefreshCw class="h-8 w-8 mx-auto mb-2 animate-spin text-muted-foreground" />
            <p class="text-muted-foreground">Loading report...</p>
          </div>

          <HealthCheckReport v-else :report="selectedReport" />
        </CardContent>
      </Card>
    </div>
  </div>
</template>
