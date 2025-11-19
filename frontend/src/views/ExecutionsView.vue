<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useWorkflowsStore } from '@/stores/workflows'
import { useExecutionsStore } from '@/stores/executions'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@/components/ui/table'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { Eye, StopCircle, Loader2, Play, Clock, CheckCircle, XCircle, AlertCircle } from 'lucide-vue-next'

const router = useRouter()
const workflowsStore = useWorkflowsStore()
const executionsStore = useExecutionsStore()

const statusFilter = ref<string>('all')
const workflowFilter = ref<string>('all')
const refreshInterval = ref<number | null>(null)

// Use executions from executions store (fetched from API)
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

const getStatusVariant = (status: string) => {
  switch (status) {
    case 'running':
      return 'default'
    case 'completed':
      return 'success'
    case 'failed':
      return 'destructive'
    case 'stopped':
      return 'secondary'
    default:
      return 'outline'
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
    case 'stopped':
      return StopCircle
    default:
      return Clock
  }
}

const handleViewDetails = (id: string) => {
  router.push(`/executions/${id}`)
}

const handleStopExecution = async (id: string) => {
  try {
    await executionsStore.stopExecution(id)
    
    // Update local state
    const index = workflowsStore.recentExecutions.findIndex(e => e.id === id)
    if (index !== -1) {
      workflowsStore.recentExecutions[index].status = 'stopped'
    }
  } catch (error) {
    console.error('Failed to stop execution:', error)
  }
}

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString()
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
  }, 5000) // Refresh every 5 seconds if there are running executions
}

const stopAutoRefresh = () => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
    refreshInterval.value = null
  }
}

// Watch filters and reload when they change
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
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">Executions</h2>
        <p class="text-muted-foreground">
          Monitor your workflow executions in real-time
        </p>
      </div>
      <Button @click="loadExecutions" variant="outline">
        <Loader2 class="mr-2 h-4 w-4" :class="{ 'animate-spin': runningCount > 0 }" />
        Refresh
      </Button>
    </div>

    <!-- Stats Cards -->
    <div class="grid gap-4 md:grid-cols-4">
      <Card class="p-6">
        <div class="flex items-center justify-between">
          <div class="flex flex-col space-y-2">
            <span class="text-sm font-medium text-muted-foreground">Total</span>
            <span class="text-3xl font-bold">{{ executions.length }}</span>
          </div>
          <Play class="h-8 w-8 text-muted-foreground" />
        </div>
      </Card>
      <Card class="p-6">
        <div class="flex items-center justify-between">
          <div class="flex flex-col space-y-2">
            <span class="text-sm font-medium text-muted-foreground">Running</span>
            <span class="text-3xl font-bold text-blue-600">{{ runningCount }}</span>
          </div>
          <Loader2 class="h-8 w-8 text-blue-600" :class="{ 'animate-spin': runningCount > 0 }" />
        </div>
      </Card>
      <Card class="p-6">
        <div class="flex items-center justify-between">
          <div class="flex flex-col space-y-2">
            <span class="text-sm font-medium text-muted-foreground">Completed</span>
            <span class="text-3xl font-bold text-green-600">{{ completedCount }}</span>
          </div>
          <CheckCircle class="h-8 w-8 text-green-600" />
        </div>
      </Card>
      <Card class="p-6">
        <div class="flex items-center justify-between">
          <div class="flex flex-col space-y-2">
            <span class="text-sm font-medium text-muted-foreground">Failed</span>
            <span class="text-3xl font-bold text-red-600">{{ failedCount }}</span>
          </div>
          <XCircle class="h-8 w-8 text-red-600" />
        </div>
      </Card>
    </div>

    <!-- Filters -->
    <div class="flex items-center gap-4">
      <div class="w-48">
        <Select v-model="statusFilter">
          <SelectTrigger>
            <SelectValue placeholder="Filter by status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Status</SelectItem>
            <SelectItem value="running">Running</SelectItem>
            <SelectItem value="completed">Completed</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
            <SelectItem value="stopped">Stopped</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="w-64">
        <Select v-model="workflowFilter">
          <SelectTrigger>
            <SelectValue placeholder="Filter by workflow" />
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
      </div>
    </div>

    <!-- Executions Table -->
    <Card>
      <div class="p-6">
        <div v-if="executions.length === 0" class="py-12 text-center">
          <AlertCircle class="mx-auto h-12 w-12 text-muted-foreground" />
          <p class="mt-4 text-lg font-medium">No Executions Yet</p>
          <p class="mt-2 text-sm text-muted-foreground">
            Execute a workflow from the Workflows page to see executions here
          </p>
          <Button @click="router.push('/workflows')" variant="outline" class="mt-4">
            Go to Workflows
          </Button>
        </div>

        <Table v-else>
          <TableHeader>
            <TableRow>
              <TableHead>Workflow</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Started</TableHead>
              <TableHead>Duration</TableHead>
              <TableHead>Progress</TableHead>
              <TableHead>Items</TableHead>
              <TableHead class="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow 
              v-for="execution in executions" 
              :key="execution.id"
              class="cursor-pointer hover:bg-muted/50"
            >
              <TableCell 
                @click="handleViewDetails(execution.id)"
                class="font-medium"
              >
                {{ execution.workflow_name }}
              </TableCell>
              <TableCell @click="handleViewDetails(execution.id)">
                <Badge :variant="getStatusVariant(execution.status)" class="flex w-fit items-center gap-1">
                  <component 
                    :is="getStatusIcon(execution.status)" 
                    class="h-3 w-3"
                    :class="{ 'animate-spin': execution.status === 'running' }"
                  />
                  {{ execution.status }}
                </Badge>
              </TableCell>
              <TableCell 
                @click="handleViewDetails(execution.id)"
                class="text-muted-foreground"
              >
                {{ formatDate(execution.started_at) }}
              </TableCell>
              <TableCell @click="handleViewDetails(execution.id)">
                {{ formatDuration(execution.started_at, execution.completed_at) }}
              </TableCell>
              <TableCell @click="handleViewDetails(execution.id)">
                <div class="flex items-center gap-2">
                  <div class="h-2 w-24 overflow-hidden rounded-full bg-muted">
                    <div 
                      class="h-full bg-primary transition-all"
                      :style="{ 
                        width: `${(execution.stats.completed / execution.stats.total_urls) * 100}%` 
                      }"
                    />
                  </div>
                  <span class="text-sm text-muted-foreground">
                    {{ execution.stats.completed }}/{{ execution.stats.total_urls }}
                  </span>
                </div>
              </TableCell>
              <TableCell 
                @click="handleViewDetails(execution.id)"
                class="font-medium"
              >
                {{ execution.stats.items_extracted }}
              </TableCell>
              <TableCell class="text-right">
                <div class="flex items-center justify-end gap-2">
                  <Button 
                    @click="handleViewDetails(execution.id)"
                    size="sm"
                    variant="outline"
                  >
                    <Eye class="h-4 w-4" />
                  </Button>
                  <Button 
                    v-if="execution.status === 'running'"
                    @click="handleStopExecution(execution.id)"
                    size="sm"
                    variant="outline"
                  >
                    <StopCircle class="h-4 w-4 text-destructive" />
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
    </Card>
  </div>
</template>

