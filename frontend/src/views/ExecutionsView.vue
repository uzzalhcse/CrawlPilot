<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useWorkflowsStore } from '@/stores/workflows'
import { useExecutionsStore } from '@/stores/executions'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import DataTable from '@/components/ui/data-table.vue'
import PageLayout from '@/components/layout/PageLayout.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import StatsBar from '@/components/layout/StatsBar.vue'
import TabBar from '@/components/layout/TabBar.vue'
import FilterBar from '@/components/layout/FilterBar.vue'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { Eye, StopCircle, Loader2, CheckCircle, XCircle, AlertCircle, SlidersHorizontal, Activity } from 'lucide-vue-next'

const router = useRouter()
const workflowsStore = useWorkflowsStore()
const executionsStore = useExecutionsStore()

const statusFilter = ref<string>('all')
const workflowFilter = ref<string>('all')
const searchQuery = ref('')
const activeTab = ref('all')
const refreshInterval = ref<number | null>(null)

const tableColumns = [
  { key: 'workflow', label: 'Workflow', sortable: true, align: 'left' as const },
  { key: 'status', label: 'Status', align: 'left' as const },
  { key: 'started', label: 'Started', align: 'left' as const },
  { key: 'duration', label: 'Duration', align: 'left' as const },
  { key: 'progress', label: 'Progress', align: 'left' as const },
  { key: 'items', label: 'Items', align: 'left' as const },
  { key: 'actions', label: 'Actions', align: 'right' as const }
]

const tabs = [
  { id: 'all', label: 'All Executions' },
  { id: 'recent', label: 'Recent' }
]

const executions = computed(() => executionsStore.executions)

const runningCount = computed(() => 
  executions.value.filter((e: any) => e.status === 'running').length
)

const completedCount = computed(() => 
  executions.value.filter((e: any) => e.status === 'completed').length
)

const failedCount = computed(() => 
  executions.value.filter((e: any) => e.status === 'failed').length
)

const stats = computed(() => [
  { label: 'Total', value: executions.value.length },
  { label: 'Running', value: runningCount.value, color: 'text-blue-600 dark:text-blue-400' },
  { label: 'Completed', value: completedCount.value, color: 'text-green-600 dark:text-green-400' },
  { label: 'Failed', value: failedCount.value, color: 'text-red-600 dark:text-red-400' }
])

const filteredExecutions = computed(() => {
  let result = executions.value

  if (statusFilter.value !== 'all') {
    result = result.filter((e: any) => e.status === statusFilter.value)
  }

  if (workflowFilter.value !== 'all') {
    result = result.filter((e: any) => e.workflow_id === workflowFilter.value)
  }

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter((e: any) =>
      e.workflow_name.toLowerCase().includes(query)
    )
  }

  return result
})

const getStatusIcon = (status: string) => {
  switch (status) {
    case 'running':
      return Loader2
    case 'completed':
      return CheckCircle
    case 'failed':
      return XCircle
    case 'stopped':
      return StopCircle
    default:
      return AlertCircle
  }
}

const handleViewDetails = (execution: any) => {
  router.push(`/executions/${execution.id}`)
}

const handleStopExecution = async (id: string) => {
  try {
    await executionsStore.stopExecution(id)
  } catch (error) {
    console.error('Failed to stop execution:', error)
  }
}

const formatDate = (dateString: string) => {
  const date = new Date(dateString)
  return date.toLocaleDateString('en-US', { month: '2-digit', day: '2-digit', year: 'numeric' }) + ', ' + date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false })
}

const formatDuration = (startedAt: string, completedAt?: string) => {
  const start = new Date(startedAt).getTime()
  const end = completedAt ? new Date(completedAt).getTime() : Date.now()
  const diff = end - start
  
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  
  if (hours > 0) {
    return `${hours}h ${minutes % 60}m`
  } else if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`
  } else {
    return `${seconds}s`
  }
}

const loadExecutions = async () => {
  try {
    const params: any = {}
    
    if (statusFilter.value !== 'all') {
      params.status = statusFilter.value
    }
    
    if (workflowFilter.value !== 'all') {
      params.workflow_id = workflowFilter.value
    }
    
    await executionsStore.fetchAllExecutions(params)
  } catch (error) {
    console.error('Failed to load executions:', error)
  }
}

const startAutoRefresh = () => {
  refreshInterval.value = window.setInterval(() => {
    if (runningCount.value > 0) {
      loadExecutions()
    }
  }, 5000)
}

const stopAutoRefresh = () => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
    refreshInterval.value = null
  }
}

watch([statusFilter, workflowFilter], () => {
  loadExecutions()
})

onMounted(async () => {
  try {
    await workflowsStore.fetchWorkflows()
    await loadExecutions()
    startAutoRefresh()
  } catch (error) {
    console.error('Failed to load data:', error)
  }
})

onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<template>
  <PageLayout>
    <!-- Header -->
    <PageHeader 
      title="Executions" 
      description="Monitor your workflow executions in real-time"
      :show-help-icon="true"
    >
      <template #actions>
        <Button @click="loadExecutions" variant="outline" size="default">
          <Loader2 class="mr-2 h-4 w-4" :class="{ 'animate-spin': runningCount > 0 }" />
          Refresh
        </Button>
      </template>
    </PageHeader>

    <!-- Stats -->
    <StatsBar :stats="stats" />

    <!-- Tabs -->
    <TabBar :tabs="tabs" v-model="activeTab" />

    <!-- Filters -->
    <FilterBar 
      search-placeholder="Search by Workflow name" 
      :search-value="searchQuery"
      @update:search-value="searchQuery = $event"
    >
      <template #filters>
        <Select v-model="statusFilter">
          <SelectTrigger class="w-[140px] h-9">
            <div class="flex items-center gap-2">
              <SlidersHorizontal class="w-4 h-4" />
              <SelectValue placeholder="Status" />
            </div>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Status</SelectItem>
            <SelectItem value="running">Running</SelectItem>
            <SelectItem value="completed">Completed</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
            <SelectItem value="stopped">Stopped</SelectItem>
          </SelectContent>
        </Select>

        <Select v-model="workflowFilter">
          <SelectTrigger class="w-[180px] h-9">
            <SelectValue placeholder="All Workflows" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Workflows</SelectItem>
            <SelectItem 
              v-for="workflow in workflowsStore.workflows" 
              :key="workflow.id"
              :value="workflow.id"
            >
              {{ workflow.name }}
            </SelectItem>
          </SelectContent>
        </Select>
      </template>
    </FilterBar>

    <!-- Table -->
    <div class="flex-1 overflow-auto">
      <div v-if="filteredExecutions.length === 0" class="py-12 text-center px-6">
        <AlertCircle class="mx-auto h-12 w-12 text-muted-foreground" />
        <p class="mt-4 text-lg font-medium">No Executions Yet</p>
        <p class="mt-2 text-sm text-muted-foreground">
          Execute a workflow from the Workflows page to see executions here
        </p>
        <Button @click="router.push('/workflows')" variant="outline" class="mt-4">
          Go to Workflows
        </Button>
      </div>

      <DataTable
        v-else
        :data="filteredExecutions"
        :columns="tableColumns"
        :on-row-click="handleViewDetails"
      >
        <template #row="{ row }">
          <td class="px-6 py-3">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center shrink-0">
                <Activity class="w-5 h-5 text-primary" />
              </div>
              <div class="min-w-0">
                <div class="font-medium text-sm truncate">{{ row.workflow_name }}</div>
                <div class="text-xs text-muted-foreground truncate">{{ row.id }}</div>
              </div>
            </div>
          </td>
          <td class="px-6 py-3" @click.stop>
            <Badge 
              variant="outline"
              :class="{
                'bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20': row.status === 'running',
                'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20': row.status === 'completed',
                'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20': row.status === 'failed',
                'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20': row.status === 'stopped'
              }"
              class="text-xs font-medium"
            >
              <component 
                :is="getStatusIcon(row.status)" 
                class="w-3 h-3 mr-1.5"
                :class="{ 'animate-spin': row.status === 'running' }"
              />
              {{ row.status }}
            </Badge>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm text-muted-foreground">
              {{ formatDate(row.started_at) }}
            </div>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm text-muted-foreground">
              {{ formatDuration(row.started_at, row.completed_at) }}
            </div>
          </td>
          <td class="px-6 py-3">
            <div v-if="row.stats && row.stats.total_urls > 0" class="flex items-center gap-2">
              <div class="h-2 w-24 overflow-hidden rounded-full bg-muted">
                <div 
                  class="h-full bg-primary transition-all"
                  :style="{ 
                    width: `${(row.stats.completed / row.stats.total_urls) * 100}%` 
                  }"
                />
              </div>
              <span class="text-xs text-muted-foreground">
                {{ row.stats.completed }}/{{ row.stats.total_urls }}
              </span>
            </div>
            <span v-else class="text-xs text-muted-foreground">-</span>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm font-medium">
              {{ row.stats?.items_extracted || 0 }}
            </div>
          </td>
          <td class="px-6 py-3 text-right" @click.stop>
            <div class="flex items-center justify-end gap-1">
              <Button 
                @click="handleViewDetails(row)"
                size="sm"
                variant="ghost"
                class="h-8 w-8 p-0"
              >
                <Eye class="h-4 w-4" />
              </Button>
              <Button 
                v-if="row.status === 'running'"
                @click="handleStopExecution(row.id)"
                size="sm"
                variant="ghost"
                class="h-8 w-8 p-0 text-destructive hover:text-destructive"
              >
                <StopCircle class="h-4 w-4" />
              </Button>
            </div>
          </td>
        </template>
      </DataTable>
    </div>
  </PageLayout>
</template>
