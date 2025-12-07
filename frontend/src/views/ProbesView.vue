<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { probesApi, type ProbeResult } from '@/api/probes'
import { workflowsApi } from '@/api/workflows'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import DataTable from '@/components/ui/data-table.vue'
import PageLayout from '@/components/layout/PageLayout.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import StatsBar from '@/components/layout/StatsBar.vue'
import FilterBar from '@/components/layout/FilterBar.vue'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { 
  Activity,
  CheckCircle2, 
  AlertTriangle,
  XCircle,
  Clock,
  Eye, 
  Loader2, 
  PlayCircle,
  RefreshCw,
  SlidersHorizontal
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const router = useRouter()

const loading = ref(true)
const probeResults = ref<(ProbeResult & { workflow_name?: string })[]>([])
const workflows = ref<Record<string, string>>({})
const allWorkflows = ref<{ id: string; name: string }[]>([])
const statusFilter = ref<string>('all')
const searchQuery = ref('')
const selectedWorkflowForProbe = ref<string>('')
const runningWorkflows = ref<Set<string>>(new Set())

const tableColumns = [
  { key: 'workflow', label: 'Workflow', sortable: true, align: 'left' as const },
  { key: 'status', label: 'Status', align: 'left' as const },
  { key: 'phases', label: 'Phases', align: 'left' as const },
  { key: 'duration', label: 'Duration', align: 'left' as const },
  { key: 'created_at', label: 'Last Check', align: 'left' as const },
  { key: 'actions', label: 'Actions', align: 'right' as const }
]

const stats = computed(() => {
  const healthy = probeResults.value.filter(p => p.status === 'healthy').length
  const degraded = probeResults.value.filter(p => p.status === 'degraded').length
  const broken = probeResults.value.filter(p => p.status === 'broken').length
  
  return [
    { label: 'Total', value: probeResults.value.length },
    { label: 'Healthy', value: healthy, color: 'text-green-600 dark:text-green-400' },
    { label: 'Degraded', value: degraded, color: 'text-amber-600 dark:text-amber-400' },
    { label: 'Broken', value: broken, color: 'text-red-600 dark:text-red-400' }
  ]
})

const filteredProbes = computed(() => {
  let result = probeResults.value
  
  if (statusFilter.value !== 'all') {
    result = result.filter(p => p.status === statusFilter.value)
  }
  
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(p => 
      p.workflow_name?.toLowerCase().includes(query) ||
      p.workflow_id.toLowerCase().includes(query)
    )
  }
  
  return result
})

const fetchData = async () => {
  loading.value = true
  try {
    // Fetch workflows
    const workflowsRes = await workflowsApi.list({ limit: 100 })
    allWorkflows.value = workflowsRes.data.workflows.map(w => ({ id: w.id, name: w.name }))
    workflowsRes.data.workflows.forEach(w => {
      workflows.value[w.id] = w.name
    })
    
    // Use efficient /probes/recent API (single request)
    const probesRes = await probesApi.getRecentProbes(50)
    
    // Add workflow names to results
    probeResults.value = probesRes.data.probes.map(probe => ({
      ...probe,
      workflow_name: workflows.value[probe.workflow_id] || probe.workflow_id
    }))
  } catch (error) {
    console.error('Failed to fetch probes:', error)
    toast.error('Failed to load probe results')
  } finally {
    loading.value = false
  }
}

const handleViewDetails = (probe: ProbeResult) => {
  router.push(`/probes/${probe.workflow_id}/${probe.id}`)
}

const handleRunProbe = async (probe: ProbeResult) => {
  if (runningWorkflows.value.has(probe.workflow_id)) return
  
  runningWorkflows.value.add(probe.workflow_id)
  try {
    await probesApi.runProbe(probe.workflow_id)
    toast.success('Probe check started')
    // Wait a bit then refresh
    setTimeout(fetchData, 3000)
  } catch (error) {
    console.error('Failed to run probe:', error)
    toast.error('Failed to start probe check')
  } finally {
    runningWorkflows.value.delete(probe.workflow_id)
  }
}

const runProbeForWorkflow = async (workflowId: string) => {
  if (!workflowId || runningWorkflows.value.has(workflowId)) return
  
  runningWorkflows.value.add(workflowId)
  try {
    await probesApi.runProbe(workflowId)
    toast.success(`Probe started for ${workflows.value[workflowId] || workflowId}`)
    selectedWorkflowForProbe.value = ''
    setTimeout(fetchData, 3000)
  } catch (error) {
    console.error('Failed to run probe:', error)
    toast.error('Failed to start probe')
  } finally {
    runningWorkflows.value.delete(workflowId)
  }
}

const getStatusColor = (status: string) => {
  switch(status) {
    case 'healthy': return 'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20'
    case 'degraded': return 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20'
    case 'broken': return 'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20'
    default: return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
  }
}

const getStatusIcon = (status: string) => {
  switch(status) {
    case 'healthy': return CheckCircle2
    case 'degraded': return AlertTriangle
    case 'broken': return XCircle
    default: return Clock
  }
}

const formatDate = (dateString?: string) => {
  if (!dateString) return 'N/A'
  const date = new Date(dateString)
  
  // Check for invalid date
  if (isNaN(date.getTime())) return 'Invalid date'
  
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / (1000 * 60))
  const hours = Math.floor(minutes / 60)
  
  if (minutes < 1) return 'Just now'
  if (minutes < 60) return `${minutes}m ago`
  if (hours < 24) return `${hours}h ago`
  
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}d ago`
  
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

const formatDuration = (ms?: number) => {
  if (!ms) return 'N/A'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

const getPhasesSummary = (phases?: any[]) => {
  if (!phases || phases.length === 0) return { passed: 0, failed: 0, total: 0 }
  const passed = phases.filter(p => p.status === 'passed').length
  const failed = phases.filter(p => p.status === 'failed').length
  return { passed, failed, total: phases.length }
}

onMounted(fetchData)
</script>

<template>
  <PageLayout>
    <!-- Header -->
    <PageHeader 
      title="Probe Health Monitor" 
      description="Monitor workflow probes and track selector health across all phases"
      :show-help-icon="true"
    >
      <template #actions>
        <div class="flex items-center gap-2">
          <Select v-model="selectedWorkflowForProbe" @update:model-value="(val: any) => runProbeForWorkflow(val as string)">
            <SelectTrigger class="w-[200px] h-9">
              <div class="flex items-center gap-2">
                <PlayCircle class="w-4 h-4" />
                <SelectValue placeholder="Run Probe..." />
              </div>
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="wf in allWorkflows" :key="wf.id" :value="wf.id">
                {{ wf.name }}
              </SelectItem>
            </SelectContent>
          </Select>
          <Button @click="fetchData" variant="outline" size="sm" :disabled="loading">
            <RefreshCw class="w-4 h-4 mr-2" :class="{ 'animate-spin': loading }" />
            Refresh
          </Button>
        </div>
      </template>
    </PageHeader>

    <!-- Stats -->
    <StatsBar :stats="stats" />

    <!-- Filters -->
    <FilterBar 
      search-placeholder="Search by workflow name..." 
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
            <SelectItem value="healthy">Healthy</SelectItem>
            <SelectItem value="degraded">Degraded</SelectItem>
            <SelectItem value="broken">Broken</SelectItem>
          </SelectContent>
        </Select>
      </template>
    </FilterBar>

    <!-- Table -->
    <div class="flex-1 overflow-auto">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <div v-else-if="filteredProbes.length === 0" class="py-12 text-center px-6">
        <Activity class="h-12 w-12 text-muted-foreground mx-auto mb-3" />
        <p class="text-muted-foreground">No probe results found</p>
        <p class="text-sm text-muted-foreground mt-1">Run a probe check on your workflows to see results.</p>
      </div>

      <DataTable
        v-else
        :data="filteredProbes"
        :columns="tableColumns"
        :on-row-click="handleViewDetails"
      >
        <template #row="{ row }">
          <td class="px-6 py-3">
            <div class="flex items-center gap-3">
              <div 
                class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0"
                :class="{
                  'bg-green-500/10': row.status === 'healthy',
                  'bg-amber-500/10': row.status === 'degraded',
                  'bg-red-500/10': row.status === 'broken'
                }"
              >
                <component 
                  :is="getStatusIcon(row.status)" 
                  class="w-5 h-5" 
                  :class="{
                    'text-green-500': row.status === 'healthy',
                    'text-amber-500': row.status === 'degraded',
                    'text-red-500': row.status === 'broken'
                  }"
                />
              </div>
              <div class="min-w-0">
                <div class="font-medium text-sm truncate">{{ row.workflow_name || row.workflow_id }}</div>
                <div class="text-xs text-muted-foreground">ID: {{ row.workflow_id.substring(0, 8) }}...</div>
              </div>
            </div>
          </td>
          <td class="px-6 py-3">
            <Badge 
              variant="outline"
              :class="getStatusColor(row.status)"
              class="text-xs font-medium capitalize"
            >
              <div class="w-1.5 h-1.5 rounded-full mr-1.5" :class="{
                'bg-green-500': row.status === 'healthy',
                'bg-amber-500': row.status === 'degraded',
                'bg-red-500': row.status === 'broken'
              }"></div>
              {{ row.status }}
            </Badge>
          </td>
          <td class="px-6 py-3">
            <div class="flex flex-wrap items-center gap-2">
              <template v-for="phase in (row.phases || [])" :key="phase.phase_id">
                <div class="flex items-center gap-1 text-xs">
                  <span class="text-muted-foreground">{{ phase.phase_name || phase.phase_id }}</span>
                  <CheckCircle2 v-if="phase.status === 'passed'" class="w-3.5 h-3.5 text-green-500" />
                  <XCircle v-else-if="phase.status === 'failed'" class="w-3.5 h-3.5 text-red-500" />
                  <Clock v-else class="w-3.5 h-3.5 text-gray-400" />
                </div>
              </template>
              <span v-if="!row.phases?.length" class="text-xs text-muted-foreground">No phases</span>
            </div>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm text-muted-foreground">
              {{ formatDuration(row.duration_ms) }}
            </div>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm text-muted-foreground">
              {{ formatDate(row.created_at) }}
            </div>
          </td>
          <td class="px-6 py-3 text-right" @click.stop>
            <div class="flex items-center justify-end gap-1">
              <Button 
                @click="handleViewDetails(row)"
                size="sm"
                variant="ghost"
                class="h-8 w-8 p-0"
                title="View Details"
              >
                <Eye class="h-4 w-4" />
              </Button>
              <Button 
                @click="handleRunProbe(row)"
                size="sm"
                variant="ghost"
                class="h-8 w-8 p-0"
                :disabled="runningWorkflows.has(row.workflow_id)"
                title="Run Probe"
              >
                <Loader2 v-if="runningWorkflows.has(row.workflow_id)" class="h-4 w-4 animate-spin" />
                <PlayCircle v-else class="h-4 w-4" />
              </Button>
            </div>
          </td>
        </template>
      </DataTable>
    </div>
  </PageLayout>
</template>
