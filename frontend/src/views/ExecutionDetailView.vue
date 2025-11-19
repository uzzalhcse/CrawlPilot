<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useExecutionsStore } from '@/stores/executions'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@/components/ui/table'
import { 
  ArrowLeft, 
  Loader2, 
  CheckCircle, 
  XCircle, 
  Clock,
  StopCircle,
  Download,
  RefreshCw
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const executionsStore = useExecutionsStore()
const executionId = route.params.id as string

const refreshInterval = ref<number | null>(null)
const activeTab = ref('overview')

// Real execution data from store
const execution = computed(() => executionsStore.currentExecution)
const timeline = computed(() => executionsStore.timeline)
const extractedData = computed(() => executionsStore.extractedData)
const performance = computed(() => executionsStore.performance)

const progressPercentage = computed(() => {
  const stats = executionsStore.executionStats
  if (!stats || stats.total_urls === 0) return 0
  return Math.round((stats.completed / stats.total_urls) * 100)
})

const getStatusVariant = (status: string) => {
  switch (status) {
    case 'running':
      return 'default'
    case 'completed':
      return 'success'
    case 'failed':
      return 'destructive'
    default:
      return 'secondary'
  }
}

const getStatusIcon = (status: string) => {
  switch (status) {
    case 'running':
      return Loader2
    case 'completed':
      return CheckCircle
    case 'failed':
      return XCircle
    default:
      return Clock
  }
}

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString()
}

const formatDuration = (ms: number) => {
  const seconds = Math.floor(ms / 1000)
  const minutes = Math.floor(seconds / 60)
  
  if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`
  }
  return `${seconds}s`
}

const handleBack = () => {
  router.push('/executions')
}

const handleStop = async () => {
  try {
    await executionsStore.stopExecution(executionId)
    await loadExecutionData()
  } catch (error) {
    console.error('Failed to stop execution:', error)
  }
}

const handleRefresh = async () => {
  await loadExecutionData()
}

const handleDownloadData = () => {
  const dataStr = JSON.stringify(extractedData.value, null, 2)
  const dataUri = 'data:application/json;charset=utf-8,'+ encodeURIComponent(dataStr)
  const exportFileDefaultName = `execution_${executionId}_data.json`
  
  const linkElement = document.createElement('a')
  linkElement.setAttribute('href', dataUri)
  linkElement.setAttribute('download', exportFileDefaultName)
  linkElement.click()
}

const loadExecutionData = async () => {
  try {
    await Promise.all([
      executionsStore.fetchExecutionById(executionId),
      executionsStore.fetchExecutionStats(executionId),
      executionsStore.fetchTimeline(executionId),
      executionsStore.fetchExtractedData(executionId),
      executionsStore.fetchPerformance(executionId)
    ])
  } catch (error) {
    console.error('Failed to load execution data:', error)
  }
}

const startAutoRefresh = () => {
  refreshInterval.value = window.setInterval(() => {
    if (execution.value && execution.value.status === 'running') {
      loadExecutionData()
    }
  }, 5000)
}

const stopAutoRefresh = () => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
    refreshInterval.value = null
  }
}

onMounted(async () => {
  await loadExecutionData()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
  executionsStore.clearCurrentExecution()
})
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-4">
        <Button @click="handleBack" variant="ghost" size="icon">
          <ArrowLeft class="h-5 w-5" />
        </Button>
        <div>
          <h2 class="text-3xl font-bold tracking-tight">Execution Details</h2>
          <p class="text-muted-foreground">{{ execution?.workflow_name || 'Loading...' }}</p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <Button @click="handleRefresh" variant="outline" size="sm">
          <RefreshCw class="mr-2 h-4 w-4" />
          Refresh
        </Button>
        <Button 
          v-if="execution?.status === 'running'"
          @click="handleStop" 
          variant="destructive" 
          size="sm"
        >
          <StopCircle class="mr-2 h-4 w-4" />
          Stop
        </Button>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="executionsStore.loading && !execution" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <template v-else-if="execution">
      <!-- Status Banner -->
      <Card class="p-6">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-4">
            <Badge v-if="execution" :variant="getStatusVariant(execution.status)" class="flex items-center gap-2 px-3 py-1">
              <component 
                :is="getStatusIcon(execution.status)" 
                class="h-4 w-4"
                :class="{ 'animate-spin': execution.status === 'running' }"
              />
              <span class="text-sm font-semibold uppercase">{{ execution.status }}</span>
            </Badge>
            <div v-if="execution" class="text-sm text-muted-foreground">
              Started: {{ formatDate(execution.started_at) }}
            </div>
          </div>
          <div class="text-right">
            <div class="text-2xl font-bold">{{ progressPercentage }}%</div>
            <div class="text-sm text-muted-foreground">Complete</div>
          </div>
        </div>
        <Progress :model-value="progressPercentage" class="mt-4" />
      </Card>

      <!-- Stats Grid -->
      <div class="grid gap-4 md:grid-cols-3 lg:grid-cols-6">
        <Card class="p-4">
          <div class="text-sm text-muted-foreground">Total URLs</div>
          <div class="text-2xl font-bold">{{ executionsStore.executionStats?.total_urls || 0 }}</div>
        </Card>
        <Card class="p-4">
          <div class="text-sm text-muted-foreground">Pending</div>
          <div class="text-2xl font-bold text-gray-600">{{ executionsStore.executionStats?.pending || 0 }}</div>
        </Card>
        <Card class="p-4">
          <div class="text-sm text-muted-foreground">Processing</div>
          <div class="text-2xl font-bold text-blue-600">{{ executionsStore.executionStats?.processing || 0 }}</div>
        </Card>
        <Card class="p-4">
          <div class="text-sm text-muted-foreground">Completed</div>
          <div class="text-2xl font-bold text-green-600">{{ executionsStore.executionStats?.completed || 0 }}</div>
        </Card>
        <Card class="p-4">
          <div class="text-sm text-muted-foreground">Failed</div>
          <div class="text-2xl font-bold text-red-600">{{ executionsStore.executionStats?.failed || 0 }}</div>
        </Card>
        <Card class="p-4">
          <div class="text-sm text-muted-foreground">Items Extracted</div>
          <div class="text-2xl font-bold text-purple-600">{{ executionsStore.executionStats?.items_extracted || 0 }}</div>
        </Card>
      </div>

    <!-- Tabs -->
    <Tabs v-model="activeTab" class="space-y-4">
      <TabsList>
        <TabsTrigger value="overview">Overview</TabsTrigger>
        <TabsTrigger value="timeline">Timeline</TabsTrigger>
        <TabsTrigger value="data">Extracted Data</TabsTrigger>
        <TabsTrigger value="performance">Performance</TabsTrigger>
      </TabsList>

      <!-- Overview Tab -->
      <TabsContent value="overview" class="space-y-4">
        <Card v-if="execution" class="p-6">
          <h3 class="mb-4 text-lg font-semibold">Execution Information</h3>
          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <div class="text-sm text-muted-foreground">Execution ID</div>
              <div class="font-mono text-sm">{{ execution.id }}</div>
            </div>
            <div>
              <div class="text-sm text-muted-foreground">Workflow ID</div>
              <div class="font-mono text-sm">{{ execution.workflow_id }}</div>
            </div>
            <div>
              <div class="text-sm text-muted-foreground">Started At</div>
              <div class="text-sm">{{ formatDate(execution.started_at) }}</div>
            </div>
            <div v-if="execution.completed_at">
              <div class="text-sm text-muted-foreground">Completed At</div>
              <div class="text-sm">{{ formatDate(execution.completed_at) }}</div>
            </div>
            <div v-else>
              <div class="text-sm text-muted-foreground">Status</div>
              <Badge :variant="getStatusVariant(execution.status)">{{ execution.status }}</Badge>
            </div>
          </div>
        </Card>

        <Card class="p-6">
          <h3 class="mb-4 text-lg font-semibold">Queue Statistics</h3>
          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <span class="text-sm text-muted-foreground">Completion Rate</span>
              <span class="text-sm font-medium">{{ progressPercentage }}%</span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-sm text-muted-foreground">Success Rate</span>
              <span class="text-sm font-medium">
                {{ executionsStore.executionStats ? Math.round((executionsStore.executionStats.completed / (executionsStore.executionStats.completed + executionsStore.executionStats.failed || 1)) * 100) : 0 }}%
              </span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-sm text-muted-foreground">Items per URL</span>
              <span class="text-sm font-medium">
                {{ executionsStore.executionStats ? ((executionsStore.executionStats.items_extracted / executionsStore.executionStats.completed) || 0).toFixed(2) : '0.00' }}
              </span>
            </div>
          </div>
        </Card>
      </TabsContent>

      <!-- Timeline Tab -->
      <TabsContent value="timeline">
        <Card class="p-6">
          <div class="mb-4 flex items-center justify-between">
            <h3 class="text-lg font-semibold">Node Execution Timeline</h3>
          </div>
          <div v-if="!timeline || timeline.length === 0" class="py-12 text-center">
            <p class="text-muted-foreground">No timeline data available</p>
          </div>
          <Table v-else>
            <TableHeader>
              <TableRow>
                <TableHead>Timestamp</TableHead>
                <TableHead>Node</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>URLs Discovered</TableHead>
                <TableHead>Items Extracted</TableHead>
                <TableHead>Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="(item, index) in timeline" :key="index">
                <TableCell class="text-sm text-muted-foreground">
                  {{ formatDate(item.timestamp) }}
                </TableCell>
                <TableCell class="font-medium">{{ item.node_name }}</TableCell>
                <TableCell>
                  <Badge variant="outline">{{ item.node_type }}</Badge>
                </TableCell>
                <TableCell>{{ item.urls_discovered || 0 }}</TableCell>
                <TableCell>{{ item.items_extracted || 0 }}</TableCell>
                <TableCell>
                  <Badge :variant="getStatusVariant(item.status)" class="flex w-fit items-center gap-1">
                    <component 
                      :is="getStatusIcon(item.status)" 
                      class="h-3 w-3"
                      :class="{ 'animate-spin': item.status === 'running' }"
                    />
                    {{ item.status }}
                  </Badge>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </Card>
      </TabsContent>

      <!-- Extracted Data Tab -->
      <TabsContent value="data">
        <Card class="p-6">
          <div class="mb-4 flex items-center justify-between">
            <h3 class="text-lg font-semibold">Extracted Data ({{ extractedData.length }} items)</h3>
            <Button @click="handleDownloadData" size="sm" variant="outline">
              <Download class="mr-2 h-4 w-4" />
              Download JSON
            </Button>
          </div>
          <div class="space-y-4">
            <div 
              v-for="item in extractedData" 
              :key="item.id"
              class="rounded-lg border p-4"
            >
              <div class="mb-2 flex items-center justify-between">
                <Badge variant="outline">{{ item.schema }}</Badge>
                <span class="text-xs text-muted-foreground">{{ formatDate(item.created_at) }}</span>
              </div>
              <div class="text-sm text-muted-foreground mb-2">{{ item.url }}</div>
              <pre class="rounded bg-muted p-3 text-xs overflow-x-auto">{{ JSON.stringify(item.data, null, 2) }}</pre>
            </div>
          </div>
        </Card>
      </TabsContent>

      <!-- Performance Tab -->
      <TabsContent value="performance">
        <Card class="p-6">
          <h3 class="mb-4 text-lg font-semibold">Performance Metrics by Node</h3>
          <div v-if="!performance || performance.length === 0" class="py-12 text-center">
            <p class="text-muted-foreground">No performance data available</p>
          </div>
          <Table v-else>
            <TableHeader>
              <TableRow>
                <TableHead>Node Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Executions</TableHead>
                <TableHead>Success Rate</TableHead>
                <TableHead>Avg Duration</TableHead>
                <TableHead>URLs Discovered</TableHead>
                <TableHead>Items Extracted</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="perf in performance" :key="perf.node_name">
                <TableCell class="font-medium">{{ perf.node_name }}</TableCell>
                <TableCell>
                  <Badge variant="outline">{{ perf.node_type }}</Badge>
                </TableCell>
                <TableCell>{{ perf.executions }}</TableCell>
                <TableCell>
                  <div class="flex items-center gap-2">
                    <div class="text-sm font-medium">
                      {{ perf.success_rate }}%
                    </div>
                    <div class="text-xs text-muted-foreground">
                      ({{ perf.executions - perf.failures }}/{{ perf.executions }})
                    </div>
                  </div>
                </TableCell>
                <TableCell>{{ perf.avg_duration_ms ? formatDuration(perf.avg_duration_ms) : '-' }}</TableCell>
                <TableCell>{{ perf.total_urls_discovered || 0 }}</TableCell>
                <TableCell>{{ perf.total_items_extracted || 0 }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </Card>
      </TabsContent>
    </Tabs>
    </template>

    <!-- Error State -->
    <div v-else-if="executionsStore.error" class="py-12 text-center">
      <p class="text-destructive">{{ executionsStore.error }}</p>
      <Button @click="loadExecutionData()" variant="outline" class="mt-4">
        Retry
      </Button>
    </div>
  </div>
</template>

