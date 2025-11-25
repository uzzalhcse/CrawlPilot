<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Activity, RefreshCw, Calendar, GitCompare } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import PageLayout from '@/components/layout/PageLayout.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import StatsBar from '@/components/layout/StatsBar.vue'
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
const statusFilter = ref<'all' | 'healthy' | 'degraded' | 'failed'>('all')
const showScheduleDialog = ref(false)

const workflow = computed(() => workflowStore.workflows.find((w: any) => w.id === workflowId.value))

const sortedReports = computed(() => {
  return [...healthCheckStore.reports].sort((a, b) => 
    new Date(b.started_at).getTime() - new Date(a.started_at).getTime()
  )
})

const filteredReports = computed(() => {
  if (statusFilter.value === 'all') return sortedReports.value
  return sortedReports.value.filter(r => r.status === statusFilter.value)
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

const stats = computed(() => [
  { label: 'Total Checks', value: statusStats.value.total },
  { label: 'Healthy', value: statusStats.value.healthy, color: 'text-green-600 dark:text-green-400' },
  { label: 'Degraded', value: statusStats.value.degraded, color: 'text-amber-600 dark:text-amber-400' },
  { label: 'Failed', value: statusStats.value.failed, color: 'text-red-600 dark:text-red-400' }
])

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
    selectedReport.value = report
  } else {
    await healthCheckStore.fetchHealthCheckReport(report.id)
    selectedReport.value = healthCheckStore.currentReport
  }
}

const formatDate = (dateStr: string) => {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  
  // If time is in future or invalid, show full date
  if (diff < 0 || isNaN(diff)) {
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }
  
  if (diff < 3600000) {
    const minutes = Math.floor(diff / 60000)
    return minutes === 0 ? 'Just now' : `${minutes}m ago`
  }
  
  if (diff < 86400000) {
    const hours = Math.floor(diff / 3600000)
    return `${hours}h ago`
  }
  
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

const formatDuration = (ms?: number) => {
  if (!ms) return 'N/A'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

const snapshots = ref<HealthCheckSnapshot[]>([])
const selectedSnapshotId = ref<string | null>(null)
const showSnapshotViewer = ref(false)

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

function hasSnapshot(nodeId: string): boolean {
  return snapshots.value.some(s => s.node_id === nodeId)
}

function getSnapshotId(nodeId: string): string | null {
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
  if (sortedReports.value.length > 0) {
    await selectReport(sortedReports.value[0])
  }
})
</script>

<template>
  <PageLayout>
    <PageHeader
      :title="workflow?.name || 'Loading...'"
      description="Monitor workflow validation and detect structure changes"
    >
      <template #breadcrumb>
        <div class="flex items-center text-sm text-muted-foreground">
          <router-link to="/monitoring" class="hover:text-foreground transition-colors">Monitoring</router-link>
          <span class="mx-2">/</span>
          <span class="text-foreground">{{ workflow?.name || 'Loading...' }}</span>
        </div>
      </template>
      <template #actions>
        <Button @click="refresh" :disabled="refreshing" variant="outline" size="sm">
          <RefreshCw :class="['h-4 w-4 mr-2', refreshing && 'animate-spin']" />
          Refresh
        </Button>
        
        <Dialog v-model:open="showScheduleDialog">
          <DialogTrigger as-child>
            <Button variant="outline" size="sm">
              <Calendar class="h-4 w-4 mr-2" />
              Schedule
            </Button>
          </DialogTrigger>
          <DialogContent class="max-w-2xl max-h-[85vh] overflow-hidden flex flex-col">
            <DialogHeader>
              <DialogTitle>Schedule Settings</DialogTitle>
            </DialogHeader>
            <div class="overflow-y-auto flex-1 -mx-6 px-6">
              <ScheduleSettings 
                v-if="workflow"
                :workflow-id="workflowId"
              />
            </div>
          </DialogContent>
        </Dialog>
        
        <HealthCheckButton v-if="workflow" :workflow-id="workflowId" @click="refresh" />
      </template>
    </PageHeader>

    <StatsBar :stats="stats" />

    <div class="flex-1 overflow-hidden">
      <div class="h-full grid grid-cols-1 lg:grid-cols-3 gap-4 p-4">
        <!-- Recent Checks Sidebar (1/3 width) -->
        <Card class="lg:col-span-1 overflow-hidden flex flex-col">
          <div class="p-3 border-b flex-shrink-0">
            <div class="flex items-center justify-between mb-2">
              <h3 class="text-sm font-semibold">Recent Checks</h3>
              <Badge variant="secondary" class="text-xs">{{ sortedReports.length }}</Badge>
            </div>
            
            <!-- Filter Tabs -->
            <div class="flex gap-1">
              <button
                @click="statusFilter = 'all'"
                :class="[
                  'flex-1 px-2 py-1 text-xs rounded transition-colors',
                  statusFilter === 'all' 
                    ? 'bg-primary text-primary-foreground' 
                    : 'bg-muted hover:bg-muted/80'
                ]"
              >
                All
                <span class="ml-1 opacity-75">({{ sortedReports.length }})</span>
              </button>
              <button
                @click="statusFilter = 'healthy'"
                :class="[
                  'flex-1 px-2 py-1 text-xs rounded transition-colors',
                  statusFilter === 'healthy' 
                    ? 'bg-green-600 dark:bg-green-700 text-white' 
                    : 'bg-muted hover:bg-muted/80'
                ]"
              >
                ✓
                <span class="ml-1 opacity-75">({{ statusStats.healthy }})</span>
              </button>
              <button
                @click="statusFilter = 'degraded'"
                :class="[
                  'flex-1 px-2 py-1 text-xs rounded transition-colors',
                  statusFilter === 'degraded' 
                    ? 'bg-amber-600 dark:bg-amber-700 text-white' 
                    : 'bg-muted hover:bg-muted/80'
                ]"
              >
                ⚠
                <span class="ml-1 opacity-75">({{ statusStats.degraded }})</span>
              </button>
              <button
                @click="statusFilter = 'failed'"
                :class="[
                  'flex-1 px-2 py-1 text-xs rounded transition-colors',
                  statusFilter === 'failed' 
                    ? 'bg-red-600 dark:bg-red-700 text-white' 
                    : 'bg-muted hover:bg-muted/80'
                ]"
              >
                ✗
                <span class="ml-1 opacity-75">({{ statusStats.failed }})</span>
              </button>
            </div>
          </div>
          
          <ScrollArea class="flex-1">
            <div v-if="filteredReports.length === 0" class="p-8 text-center">
              <Activity class="h-8 w-8 mx-auto mb-2 text-muted-foreground opacity-20" />
              <p class="text-xs text-muted-foreground">
                {{ statusFilter === 'all' ? 'No checks yet' : `No ${statusFilter} checks` }}
              </p>
            </div>
            
            <div v-else class="p-2 space-y-1">
              <button
                v-for="report in filteredReports.slice(0, 20)"
                :key="report.id"
                :class="[
                  'w-full text-left p-2 rounded-md transition-colors text-xs',
                  selectedReport?.id === report.id 
                    ? 'bg-accent border border-primary' 
                    : 'hover:bg-accent/50 border border-transparent'
                ]"
                @click="selectReport(report)"
              >
                <div class="flex items-center justify-between gap-2 mb-1">
                  <Badge 
                    variant="outline"
                    :class="{
                      'bg-green-50 dark:bg-green-950 text-green-700 dark:text-green-400 border-green-200 dark:border-green-800': report.status === 'healthy',
                      'bg-amber-50 dark:bg-amber-950 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-800': report.status === 'degraded',
                      'bg-red-50 dark:bg-red-950 text-red-700 dark:text-red-400 border-red-200 dark:border-red-800': report.status === 'failed',
                    }"
                    class="text-[10px] px-1.5 py-0"
                  >
                    {{ report.status }}
                  </Badge>
                  <span class="text-[10px] text-muted-foreground">{{ formatDate(report.started_at) }}</span>
                </div>
                <div class="text-[10px] text-muted-foreground">
                  {{ formatDuration(report.duration_ms) }}
                </div>
              </button>
            </div>
          </ScrollArea>
        </Card>

        <!-- Main Content (2/3 width) -->
        <div class="lg:col-span-2 overflow-hidden flex flex-col gap-4">
          <!-- Latest Check Report -->
          <Card v-if="selectedReport" class="overflow-hidden flex-1 flex flex-col">
            <div class="p-3 border-b flex items-center justify-between flex-shrink-0">
              <div>
                <h2 class="text-sm font-semibold">Latest Check Report</h2>
                <p class="text-[10px] text-muted-foreground">
                  {{ formatDate(selectedReport.started_at) }} • {{ formatDuration(selectedReport.duration_ms) }}
                </p>
              </div>
            </div>
            
            <ScrollArea class="flex-1">
              <CardContent class="p-3">
                <HealthCheckReport 
                  :report="selectedReport" 
                  :has-snapshot="hasSnapshot"
                  :get-snapshot-id="getSnapshotId"
                  :open-snapshot="openSnapshot"
                />
              </CardContent>
            </ScrollArea>
          </Card>

          <!-- Empty State -->
          <Card v-else class="flex-1 flex items-center justify-center">
            <CardContent class="text-center py-8">
              <Activity class="h-10 w-10 mx-auto mb-2 text-muted-foreground opacity-20" />
              <p class="text-sm font-medium">No checks yet</p>
              <p class="text-xs text-muted-foreground mt-1">Run your first check to start monitoring</p>
            </CardContent>
          </Card>

          <!-- Collapsible Sections -->
          <Accordion type="multiple" class="space-y-2">
            <!-- Baseline Comparison -->
            <AccordionItem value="baseline" class="border rounded-md">
              <AccordionTrigger class="hover:no-underline px-3 py-2 text-xs font-medium">
                <div class="flex items-center gap-2">
                  <GitCompare class="h-3.5 w-3.5 text-blue-600 dark:text-blue-400" />
                  <span>Baseline Comparison</span>
                </div>
              </AccordionTrigger>
              <AccordionContent class="px-3 pb-2">
                <BaselineComparison 
                  v-if="selectedReport"
                  :report-id="selectedReport.id"
                  :workflow-id="workflowId"
                />
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </div>
      </div>
    </div>
  </PageLayout>

  <SnapshotViewer
    :snapshot-id="selectedSnapshotId"
    :open="showSnapshotViewer"
    @close="closeSnapshotViewer"
  />
</template>
