<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { probesApi, type ProbeResult, type AutoFix } from '@/api/probes'
import { workflowsApi } from '@/api/workflows'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
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
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
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
  SlidersHorizontal,
  Wand2,
  Check,
  X,
  Bot,
  Search
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const router = useRouter()

// State
const loading = ref(true)
const workflows = ref<Record<string, string>>({})
const allWorkflows = ref<{ id: string; name: string }[]>([])

// Probes State
const probeResults = ref<(ProbeResult & { workflow_name?: string })[]>([])
const probeStatusFilter = ref<string>('all')
const probeSearchQuery = ref('')
const selectedWorkflowForProbe = ref<string>('')
const runningWorkflows = ref<Set<string>>(new Set())

// Auto-Fix Review State
const processingFixes = ref<Set<string>>(new Set())
const selectedFix = ref<AutoFix | null>(null)
const showFixDetailDialog = ref(false)
const editedSelector = ref('')
const previewLoading = ref(false)
const previewResult = ref<{ match_count: number; matches: string[]; error?: string } | null>(null)

// Columns
const probeColumns = [
  { key: 'workflow', label: 'Workflow', sortable: true, align: 'left' as const },
  { key: 'status', label: 'Status', align: 'left' as const },
  { key: 'phases', label: 'Phases', align: 'left' as const },
  { key: 'duration', label: 'Duration', align: 'left' as const },
  { key: 'created_at', label: 'Last Check', align: 'left' as const },
  { key: 'actions', label: 'Actions', align: 'right' as const }
]

// Computed Stats
const probeStats = computed(() => {
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

// Filtered Data
const filteredProbes = computed(() => {
  let result = probeResults.value
  
  if (probeStatusFilter.value !== 'all') {
    result = result.filter(p => p.status === probeStatusFilter.value)
  }
  
  if (probeSearchQuery.value) {
    const query = probeSearchQuery.value.toLowerCase()
    result = result.filter(p => 
      p.workflow_name?.toLowerCase().includes(query) ||
      p.workflow_id.toLowerCase().includes(query)
    )
  }
  
  return result
})

// Methods
const fetchData = async () => {
  loading.value = true
  try {
    // Fetch workflows
    const workflowsRes = await workflowsApi.list({ limit: 100 })
    allWorkflows.value = workflowsRes.data.workflows.map(w => ({ id: w.id, name: w.name }))
    workflowsRes.data.workflows.forEach(w => {
      workflows.value[w.id] = w.name
    })
    
    // Fetch Probes (now includes auto_fix data)
    const probesRes = await probesApi.getRecentProbes(50)
    console.log('Raw Probe Response:', probesRes.data)
    probeResults.value = probesRes.data.probes.map(probe => ({
      ...probe,
      workflow_name: workflows.value[probe.workflow_id] || probe.workflow_id
    }))
    console.log('Mapped Probes:', probeResults.value)

  } catch (error) {
    console.error('Failed to fetch data:', error)
    toast.error('Failed to load data')
  } finally {
    loading.value = false
  }
}

// Probe Actions
const handleViewProbeDetails = (probe: ProbeResult) => {
  router.push(`/probes/${probe.workflow_id}/${probe.id}`)
}

const handleRunProbe = async (probe: ProbeResult) => {
  if (runningWorkflows.value.has(probe.workflow_id)) return
  
  runningWorkflows.value.add(probe.workflow_id)
  try {
    await probesApi.runProbe(probe.workflow_id)
    toast.success('Probe check started')
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

// Auto-Fix Actions
const handleReviewFix = (fix: AutoFix) => {
  selectedFix.value = fix
  editedSelector.value = fix.new_selector || ''
  previewResult.value = null // Reset preview
  showFixDetailDialog.value = true
}

const handlePreviewSelector = async () => {
  if (!selectedFix.value || !editedSelector.value) return
  
  previewLoading.value = true
  previewResult.value = null
  
  try {
    const res = await probesApi.previewSelector(
      selectedFix.value.execution_id!, 
      selectedFix.value.node_id, 
      editedSelector.value
    )
    previewResult.value = res.data
  } catch (error) {
    console.error('Preview failed:', error)
    toast.error('Failed to preview selector')
  } finally {
    previewLoading.value = false
  }
}

const handleApproveFix = async (fix: AutoFix) => {
  if (processingFixes.value.has(fix.id)) return
  
  processingFixes.value.add(fix.id)
  try {
    // Check if selector was modified
    const overrides = editedSelector.value !== fix.new_selector 
      ? { new_selector: editedSelector.value } 
      : undefined

    await probesApi.approveAutoFix(fix.id, overrides)
    toast.success('Auto-fix approved')
    
    // Update local state
    fix.status = 'applied'
    if (overrides) {
      fix.new_selector = overrides.new_selector
      fix.auto_applied = false // It was manually modified
    }

    // Also update probe status if it was broken
    const probe = probeResults.value.find(p => p.auto_fix?.id === fix.id)
    if (probe) {
      probe.status = 'fixed' // UI update only, backend handles real update
    }
    
    showFixDetailDialog.value = false
  } catch (error) {
    console.error('Failed to approve fix:', error)
    toast.error('Failed to approve auto-fix')
  } finally {
    processingFixes.value.delete(fix.id)
  }
}

const handleRejectFix = async (fix: AutoFix) => {
  if (processingFixes.value.has(fix.id)) return
  
  processingFixes.value.add(fix.id)
  try {
    await probesApi.rejectAutoFix(fix.id)
    toast.success('Auto-fix rejected')
    
    // Update local state
    fix.status = 'rejected'
    
    showFixDetailDialog.value = false
  } catch (error) {
    console.error('Failed to reject fix:', error)
    toast.error('Failed to reject auto-fix')
  } finally {
    processingFixes.value.delete(fix.id)
  }
}

// Helpers
const getStatusColor = (status: string) => {
  switch(status) {
    case 'healthy': return 'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20'
    case 'degraded': return 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20'
    case 'broken': return 'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20'
    case 'fixed': return 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20'
    case 'applied': return 'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20'
    case 'pending': return 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20'
    case 'rejected': return 'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20'
    default: return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
  }
}

const getStatusIcon = (status: string) => {
  switch(status) {
    case 'healthy': return CheckCircle2
    case 'degraded': return AlertTriangle
    case 'broken': return XCircle
    case 'fixed': return Wand2
    case 'applied': return CheckCircle2
    case 'pending': return Clock
    case 'rejected': return XCircle
    default: return Clock
  }
}

const getFixTypeLabel = (type: string) => {
  switch(type) {
    case 'update_selector': return 'Update Selector'
    case 'update_field_selector': return 'Update Field Selector'
    case 'skip_node': return 'Skip Node'
    default: return type
  }
}

const getConfidenceColor = (confidence: number) => {
  if (confidence >= 0.8) return 'text-green-600 dark:text-green-400'
  if (confidence >= 0.5) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
}

const formatDate = (dateString?: string) => {
  if (!dateString) return 'N/A'
  const date = new Date(dateString)
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

onMounted(fetchData)
</script>

<template>
  <PageLayout>
    <!-- Header -->
    <PageHeader 
      title="Probe Health Monitor" 
      description="Monitor workflow probes and manage AI auto-fixes"
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

    <div class="flex-1 flex flex-col overflow-hidden">
      <!-- Stats -->
      <StatsBar :stats="probeStats" />

      <!-- Filters -->
      <FilterBar 
        search-placeholder="Search by workflow name..." 
        :search-value="probeSearchQuery"
        @update:search-value="probeSearchQuery = $event"
      >
        <template #filters>
          <Select v-model="probeStatusFilter">
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
          :columns="probeColumns"
          :on-row-click="handleViewProbeDetails"
        >
          <template #row="{ row }">
            <td class="px-6 py-3">
              <div class="flex items-center gap-3">
                <div 
                  class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0"
                  :class="{
                    'bg-green-500/10': row.status === 'healthy',
                    'bg-amber-500/10': row.status === 'degraded',
                    'bg-red-500/10': row.status === 'broken',
                    'bg-blue-500/10': row.status === 'fixed'
                  }"
                >
                  <component 
                    :is="getStatusIcon(row.status)" 
                    class="w-5 h-5" 
                    :class="{
                      'text-green-500': row.status === 'healthy',
                      'text-amber-500': row.status === 'degraded',
                      'text-red-500': row.status === 'broken',
                      'text-blue-500': row.status === 'fixed'
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
              <div class="flex flex-col gap-1">
                <Badge 
                  variant="outline"
                  :class="getStatusColor(row.status)"
                  class="text-xs font-medium capitalize w-fit"
                >
                  <div class="w-1.5 h-1.5 rounded-full mr-1.5" :class="{
                    'bg-green-500': row.status === 'healthy',
                    'bg-amber-500': row.status === 'degraded',
                    'bg-red-500': row.status === 'broken',
                    'bg-blue-500': row.status === 'fixed'
                  }"></div>
                  {{ row.status }}
                </Badge>
                
                <!-- Auto-Fix Badge -->
                <Badge 
                  v-if="row.auto_fix && row.auto_fix.status === 'pending'" 
                  variant="secondary" 
                  class="text-[10px] px-1.5 py-0 h-5 w-fit bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300 border-purple-200 dark:border-purple-800"
                >
                  <Wand2 class="w-3 h-3 mr-1" />
                  Fix Available
                </Badge>
                <Badge 
                  v-else-if="row.auto_fix && row.auto_fix.status === 'applied'" 
                  variant="secondary" 
                  class="text-[10px] px-1.5 py-0 h-5 w-fit bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300 border-green-200 dark:border-green-800"
                >
                  <CheckCircle2 class="w-3 h-3 mr-1" />
                  Fixed by AI
                </Badge>
              </div>
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
                  @click="handleViewProbeDetails(row)"
                  size="sm"
                  variant="ghost"
                  class="h-8 w-8 p-0"
                  title="View Details"
                >
                  <Eye class="h-4 w-4" />
                </Button>
                
                <!-- Review Fix Button -->
                <Button 
                  v-if="row.auto_fix"
                  @click.stop="handleReviewFix(row.auto_fix)"
                  size="sm"
                  variant="ghost"
                  class="h-8 w-8 p-0"
                  :class="{
                    'text-purple-600 hover:text-purple-700 hover:bg-purple-50': row.auto_fix.status === 'pending',
                    'text-muted-foreground hover:text-foreground': row.auto_fix.status !== 'pending'
                  }"
                  :title="row.auto_fix.status === 'pending' ? 'Review AI Fix' : 'View Fix Details'"
                >
                  <Wand2 class="h-4 w-4" />
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
    </div>

    <!-- Fix Detail Dialog -->
    <Dialog v-model:open="showFixDetailDialog">
      <DialogContent class="max-w-2xl">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2">
            <Bot class="w-5 h-5 text-primary" />
            AI Auto-Fix Details
          </DialogTitle>
          <DialogDescription>
            Review the AI-suggested fix before approving or rejecting.
          </DialogDescription>
        </DialogHeader>

        <div v-if="selectedFix" class="space-y-4">
          <!-- Fix Type & Status -->
          <div class="flex items-center gap-2">
            <Badge variant="outline">
              <Wand2 class="w-3 h-3 mr-1" />
              {{ getFixTypeLabel(selectedFix.fix_type) }}
            </Badge>
            <Badge :class="getStatusColor(selectedFix.status)" class="capitalize">
              {{ selectedFix.status }}
            </Badge>
            <Badge v-if="selectedFix.auto_applied" variant="secondary">Auto-Applied</Badge>
          </div>

          <!-- Confidence -->
          <Card class="p-3">
            <div class="flex items-center justify-between">
              <span class="text-sm text-muted-foreground">AI Confidence</span>
              <div class="flex items-center gap-2">
                <div class="w-24 h-2 bg-muted rounded-full overflow-hidden">
                  <div 
                    class="h-full rounded-full"
                    :class="{
                      'bg-green-500': selectedFix.confidence >= 0.8,
                      'bg-amber-500': selectedFix.confidence >= 0.5 && selectedFix.confidence < 0.8,
                      'bg-red-500': selectedFix.confidence < 0.5
                    }"
                    :style="{ width: `${selectedFix.confidence * 100}%` }"
                  ></div>
                </div>
                <span class="text-sm font-medium" :class="getConfidenceColor(selectedFix.confidence)">
                  {{ (selectedFix.confidence * 100).toFixed(0) }}%
                </span>
              </div>
            </div>
          </Card>

          <!-- Node and Workflow Info -->
          <Card class="p-3 space-y-2">
            <div class="flex items-start gap-2">
              <span class="text-xs text-muted-foreground font-medium w-16 shrink-0">Node ID:</span>
              <code class="text-xs bg-muted px-2 py-1 rounded break-all">
                {{ selectedFix.node_id || 'N/A' }}
              </code>
            </div>
            <div class="flex items-start gap-2" v-if="selectedFix.execution_id">
              <span class="text-xs text-muted-foreground font-medium w-16 shrink-0">Execution:</span>
              <router-link 
                :to="{ name: 'execution-detail', params: { id: selectedFix.execution_id } }"
                class="text-xs text-primary hover:underline font-mono"
              >
                {{ selectedFix.execution_id }}
              </router-link>
            </div>
            <div class="flex items-start gap-2">
              <span class="text-xs text-muted-foreground font-medium w-16 shrink-0">Workflow:</span>
              <span class="text-xs text-foreground break-all">
                {{ workflows[selectedFix.workflow_id] || selectedFix.workflow_id }}
              </span>
            </div>
          </Card>

          <!-- Selector Change -->
          <div v-if="selectedFix.fix_type === 'update_selector' || selectedFix.fix_type === 'update_field_selector'" class="space-y-2">
            <div class="text-sm font-medium">Selector Change</div>
            <Card class="p-3 space-y-2">
              <div class="flex items-start gap-2">
                <span class="text-xs text-red-500 font-medium w-12 shrink-0">OLD:</span>
                <code class="text-xs bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 px-2 py-1 rounded break-all">
                  {{ selectedFix.old_selector || 'N/A' }}
                </code>
              </div>
              <div class="flex items-start gap-2">
                <span class="text-xs text-green-500 font-medium w-12 shrink-0 pt-2">NEW:</span>
                <div class="flex-1">
                  <div v-if="selectedFix.status === 'pending'">
                    <Label class="sr-only">New Selector</Label>
                    <Textarea 
                      v-model="editedSelector" 
                      class="font-mono text-xs min-h-[60px]"
                      placeholder="Enter CSS selector..."
                    />
                    <p class="text-[10px] text-muted-foreground mt-1">
                      You can manually modify the selector before approving.
                    </p>
                    
                    <!-- Preview Button -->
                    <div class="mt-2">
                      <Button 
                        size="sm" 
                        variant="outline" 
                        class="h-7 text-xs"
                        @click="handlePreviewSelector"
                        :disabled="!editedSelector || previewLoading"
                      >
                        <Loader2 v-if="previewLoading" class="w-3 h-3 mr-1 animate-spin" />
                        <Search v-else class="w-3 h-3 mr-1" />
                        Preview Matches
                      </Button>
                    </div>

                    <!-- Preview Results -->
                    <div v-if="previewResult" class="mt-2 text-xs border rounded-md p-2 bg-muted/50">
                      <div class="flex items-center justify-between mb-1">
                        <span class="font-medium">Found {{ previewResult.match_count }} elements</span>
                      </div>
                      <ul class="list-disc list-inside space-y-0.5 text-muted-foreground">
                        <li v-for="(match, i) in previewResult.matches" :key="i" class="truncate">
                          {{ match }}
                        </li>
                      </ul>
                      <div v-if="previewResult.error" class="text-red-500 mt-1">
                        Error: {{ previewResult.error }}
                      </div>
                    </div>
                  </div>
                  <code v-else class="text-xs bg-green-50 dark:bg-green-900/20 text-green-600 dark:text-green-400 px-2 py-1 rounded break-all block">
                    {{ selectedFix.new_selector || 'N/A' }}
                  </code>
                </div>
              </div>
            </Card>
          </div>

          <!-- Reasoning -->
          <div class="space-y-2">
            <div class="text-sm font-medium">AI Reasoning</div>
            <Card class="p-3">
              <p class="text-sm text-muted-foreground">{{ selectedFix.reasoning }}</p>
            </Card>
          </div>
        </div>

        <DialogFooter v-if="selectedFix?.status === 'pending'" class="gap-2">
          <Button variant="outline" @click="showFixDetailDialog = false">
            Cancel
          </Button>
          <Button 
            variant="destructive" 
            @click="handleRejectFix(selectedFix!)"
            :disabled="processingFixes.has(selectedFix!.id)"
          >
            <X class="w-4 h-4 mr-2" />
            Reject
          </Button>
          <Button 
            @click="handleApproveFix(selectedFix!)"
            :disabled="processingFixes.has(selectedFix!.id)"
          >
            <Loader2 v-if="processingFixes.has(selectedFix!.id)" class="w-4 h-4 mr-2 animate-spin" />
            <Check v-else class="w-4 h-4 mr-2" />
            Approve
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </PageLayout>
</template>
