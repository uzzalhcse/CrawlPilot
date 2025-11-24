<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Activity, RefreshCw, Clock, TrendingUp, Settings, GitCompare } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import HealthCheckButton from '@/components/workflow/HealthCheckButton.vue'
import HealthCheckReport from '@/components/workflow/HealthCheckReport.vue'
import ScheduleSettings from '@/components/healthcheck/ScheduleSettings.vue'
import BaselineComparison from '@/components/healthcheck/BaselineComparison.vue'
import SnapshotViewer from '@/components/healthcheck/SnapshotViewer.vue'
import { useHealthCheckStore } from '@/stores/healthcheck'
import { useWorkflowsStore } from '@/stores/workflows'
import { workflowsApi } from '@/api/workflows'
import type { HealthCheckReport as HealthCheckReportType, HealthCheckSnapshot } from '@/types'

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

const activeTab = ref('report')
const baselineComparison = ref<any>(null)
const settingBaseline = ref(false)

const setBaseline = async () => {
  if (!selectedReport.value) return
  
  settingBaseline.value = true
  try {
    await workflowsApi.setBaseline(selectedReport.value.id)
    alert('Baseline set successfully!')
    // Reload comparison if on baseline tab
    if (activeTab.value === 'baseline' && baselineComparison.value) {
      baselineComparison.value.loadComparison()
    }
  } catch (error) {
    alert('Failed to set baseline')
  } finally {
    settingBaseline.value = false
  }
}

// Snapshot viewer state
const snapshots = ref<HealthCheckSnapshot[]>([])
const selectedSnapshotId = ref<string | null>(null)
const showSnapshotViewer = ref(false)

// Load snapshots when report changes
watch(() => selectedReport.value?.id, async (reportId) => {
  if (!reportId) {
    snapshots.value = []
    return
  }
  
  try {
    const response = await workflowsApi.getSnapshotsByReport(reportId)
    snapshots.value = response.data.snapshots
  } catch (err) {
    console.error('Failed to load snapshots:', err)
    snapshots.value = []
  }
})

// Snapshot methods
function hasSnapshot(nodeId: string): boolean {
  if (!snapshots.value || snapshots.value.length === 0) return false
  return snapshots.value.some(s => s.node_id === nodeId)
}

function getSnapshotId(nodeId: string): string | null {
  if (!snapshots.value || snapshots.value.length === 0) return null
  const snapshot = snapshots.value.find(s => s.node_id === nodeId)
  return snapshot?.id || null
}

function openSnapshot(snapshotId: string | null) {
  if (!snapshotId) return
  selectedSnapshotId.value = snapshotId
  showSnapshotViewer.value = true
}

function closeSnapshotViewer() {
  showSnapshotViewer.value = false
  selectedSnapshotId.value = null
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
    <div class="page-header">
      <div>
        <div class="breadcrumb">
          <router-link to="/health-checks" class="breadcrumb-link">Health Checks</router-link>
          <span class="breadcrumb-separator">/</span>
          <span class="breadcrumb-current">{{ workflow?.name || 'Loading...' }}</span>
        </div>
        <h1 class="text-3xl font-bold tracking-tight mt-2">{{ workflow?.name }}</h1>
        <p class="text-muted-foreground mt-1">
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

      <!-- Report Detail with Tabs -->
      <div class="md:col-span-2">
        <Tabs v-model="activeTab" class="w-full">
          <Card>
            <CardHeader>
              <div class="flex items-center justify-between">
                <div>
                  <CardTitle>Health Check Details</CardTitle>
                  <CardDescription>
                    <span v-if="selectedReport">{{ formatDate(selectedReport.started_at) }}</span>
                    <span v-else>Select a health check to view details</span>
                  </CardDescription>
                </div>
                <TabsList class="grid w-auto grid-cols-3">
                  <TabsTrigger value="report">
                    <Activity class="h-4 w-4 mr-2" />
                    Report
                  </TabsTrigger>
                  <TabsTrigger value="baseline">
                    <GitCompare class="h-4 w-4 mr-2" />
                    Baseline
                  </TabsTrigger>
                  <TabsTrigger value="schedule">
                    <Settings class="h-4 w-4 mr-2" />
                    Schedule
                  </TabsTrigger>
                </TabsList>
              </div>
            </CardHeader>

            <CardContent>
              <!-- Report Tab -->
              <TabsContent value="report" class="mt-0">
                <div v-if="!selectedReport" class="text-center py-12 text-muted-foreground">
                  <Activity class="h-16 w-16 mx-auto mb-4 opacity-50" />
                  <p class="text-lg">No report selected</p>
                  <p class="text-sm">Click on a health check from the history to view details</p>
                </div>

                <div v-else-if="healthCheckStore.loading" class="text-center py-12">
                  <RefreshCw class="h-8 w-8 mx-auto mb-2 animate-spin text-muted-foreground" />
                  <p class="text-muted-foreground">Loading report...</p>
                </div>

                <div v-else>
                  <!-- Baseline Control -->
                  <div class="mb-4 p-4 bg-muted rounded-lg flex items-center justify-between">
                    <div>
                      <p class="font-medium">Use as Baseline</p>
                      <p class="text-sm text-muted-foreground">Set this health check as the baseline for future comparisons</p>
                    </div>
                    <Button @click="setBaseline" :disabled="settingBaseline" variant="outline">
                      {{ settingBaseline ? 'Setting...' : 'Set as Baseline' }}
                    </Button>
                  </div>

                  <HealthCheckReport 
                    :report="selectedReport" 
                    :has-snapshot="hasSnapshot"
                    :get-snapshot-id="getSnapshotId"
                    :open-snapshot="openSnapshot"
                  />
                </div>
              </TabsContent>

              <!-- Baseline Tab -->
              <TabsContent value="baseline" class="mt-0">
                <BaselineComparison 
                  v-if="selectedReport" 
                  :report-id="selectedReport.id"
                  @set-baseline="setBaseline"
                  ref="baselineComparison"
                />
                <div v-else class="text-center py-12 text-muted-foreground">
                  <GitCompare class="h-16 w-16 mx-auto mb-4 opacity-50" />
                  <p class="text-lg">No report selected</p>
                  <p class="text-sm">Select a health check to compare with baseline</p>
                </div>
              </TabsContent>

              <!-- Schedule Tab -->
              <TabsContent value="schedule" class="mt-0">
                <ScheduleSettings :workflow-id="workflowId" />
              </TabsContent>
            </CardContent>
          </Card>
        </Tabs>
      </div>
    </div>
  </div>

  <!-- Snapshot Viewer Dialog -->
  <SnapshotViewer
    :snapshot-id="selectedSnapshotId"
    :open="showSnapshotViewer"
    @close="closeSnapshotViewer"
  />
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 1.5rem;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  color: hsl(var(--muted-foreground));
}

.breadcrumb-link {
  color: hsl(var(--primary));
  text-decoration: none;
  transition: color 0.2s;
}

.breadcrumb-link:hover {
  color: hsl(var(--primary) / 0.8);
  text-decoration: underline;
}

.breadcrumb-separator {
  color: hsl(var(--muted-foreground) / 0.5);
}

.breadcrumb-current {
  font-weight: 500;
  color: hsl(var(--foreground));
}
</style>
