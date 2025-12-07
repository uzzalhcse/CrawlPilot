<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { probesApi, type AutoFix } from '@/api/probes'
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
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { 
  Wand2,
  CheckCircle2, 
  XCircle,
  Clock,
  Eye, 
  Loader2, 
  RefreshCw,
  SlidersHorizontal,
  Check,
  X,
  Sparkles,
  Bot
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const loading = ref(true)
const fixes = ref<(AutoFix & { workflow_name?: string })[]>([])
const workflows = ref<Record<string, string>>({})
const statusFilter = ref<string>('all')
const searchQuery = ref('')
const processingFixes = ref<Set<string>>(new Set())

// Detail dialog
const selectedFix = ref<AutoFix | null>(null)
const showDetailDialog = ref(false)

const tableColumns = [
  { key: 'workflow', label: 'Workflow', align: 'left' as const },
  { key: 'node', label: 'Node', align: 'left' as const },
  { key: 'fix_type', label: 'Fix Type', align: 'left' as const },
  { key: 'confidence', label: 'Confidence', align: 'left' as const },
  { key: 'status', label: 'Status', align: 'left' as const },
  { key: 'created_at', label: 'Created', align: 'left' as const },
  { key: 'actions', label: 'Actions', align: 'right' as const }
]

const stats = computed(() => {
  const applied = fixes.value.filter(f => f.status === 'applied').length
  const pending = fixes.value.filter(f => f.status === 'pending').length
  const rejected = fixes.value.filter(f => f.status === 'rejected').length
  const autoApplied = fixes.value.filter(f => f.auto_applied).length
  
  return [
    { label: 'Total', value: fixes.value.length },
    { label: 'Applied', value: applied, color: 'text-green-600 dark:text-green-400' },
    { label: 'Pending', value: pending, color: 'text-amber-600 dark:text-amber-400' },
    { label: 'Auto-Applied', value: autoApplied, color: 'text-blue-600 dark:text-blue-400' }
  ]
})

const filteredFixes = computed(() => {
  let result = fixes.value
  
  if (statusFilter.value !== 'all') {
    result = result.filter(f => f.status === statusFilter.value)
  }
  
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(f => 
      f.workflow_name?.toLowerCase().includes(query) ||
      f.node_id.toLowerCase().includes(query) ||
      f.reasoning.toLowerCase().includes(query)
    )
  }
  
  return result
})

const fetchData = async () => {
  loading.value = true
  try {
    // Fetch workflows for names
    const workflowsRes = await workflowsApi.list({ limit: 100 })
    workflowsRes.data.workflows?.forEach(w => {
      workflows.value[w.id] = w.name
    })
    
    // Fetch auto-fixes
    const fixesRes = await probesApi.getAutoFixes({ limit: 100 })
    const fixesList = fixesRes.data.fixes || []
    fixes.value = fixesList.map(fix => ({
      ...fix,
      workflow_name: workflows.value[fix.workflow_id] || fix.workflow_id
    }))
  } catch (error) {
    console.error('Failed to fetch auto-fixes:', error)
    fixes.value = [] // Reset to empty array on error
  } finally {
    loading.value = false
  }
}

const handleApprove = async (fix: AutoFix) => {
  if (processingFixes.value.has(fix.id)) return
  
  processingFixes.value.add(fix.id)
  try {
    await probesApi.approveAutoFix(fix.id)
    toast.success('Auto-fix approved')
    // Update local state
    const idx = fixes.value.findIndex(f => f.id === fix.id)
    if (idx !== -1) {
      fixes.value[idx].status = 'applied'
    }
    showDetailDialog.value = false
  } catch (error) {
    console.error('Failed to approve fix:', error)
    toast.error('Failed to approve auto-fix')
  } finally {
    processingFixes.value.delete(fix.id)
  }
}

const handleReject = async (fix: AutoFix) => {
  if (processingFixes.value.has(fix.id)) return
  
  processingFixes.value.add(fix.id)
  try {
    await probesApi.rejectAutoFix(fix.id)
    toast.success('Auto-fix rejected')
    // Update local state
    const idx = fixes.value.findIndex(f => f.id === fix.id)
    if (idx !== -1) {
      fixes.value[idx].status = 'rejected'
    }
    showDetailDialog.value = false
  } catch (error) {
    console.error('Failed to reject fix:', error)
    toast.error('Failed to reject auto-fix')
  } finally {
    processingFixes.value.delete(fix.id)
  }
}

const viewDetails = (fix: AutoFix) => {
  selectedFix.value = fix
  showDetailDialog.value = true
}

const getStatusColor = (status: string) => {
  switch(status) {
    case 'applied': return 'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20'
    case 'pending': return 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20'
    case 'rejected': return 'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20'
    default: return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
  }
}

const getStatusIcon = (status: string) => {
  switch(status) {
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
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const hours = Math.floor(diff / (1000 * 60 * 60))
  
  if (hours < 1) return 'Just now'
  if (hours < 24) return `${hours}h ago`
  
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}d ago`
  
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

onMounted(fetchData)
</script>

<template>
  <PageLayout>
    <!-- Header -->
    <PageHeader 
      title="AI Auto-Fixes" 
      description="Review and manage AI-suggested workflow fixes"
      :show-help-icon="true"
    >
      <template #actions>
        <Button @click="fetchData" variant="outline" size="sm" :disabled="loading">
          <RefreshCw class="w-4 h-4 mr-2" :class="{ 'animate-spin': loading }" />
          Refresh
        </Button>
      </template>
    </PageHeader>

    <!-- Stats -->
    <StatsBar :stats="stats" />

    <!-- Filters -->
    <FilterBar 
      search-placeholder="Search by workflow, node, or reasoning..." 
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
            <SelectItem value="applied">Applied</SelectItem>
            <SelectItem value="pending">Pending</SelectItem>
            <SelectItem value="rejected">Rejected</SelectItem>
          </SelectContent>
        </Select>
      </template>
    </FilterBar>

    <!-- Table -->
    <div class="flex-1 overflow-auto">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <div v-else-if="filteredFixes.length === 0" class="py-12 text-center px-6">
        <Bot class="h-12 w-12 text-muted-foreground mx-auto mb-3" />
        <p class="text-muted-foreground">No auto-fixes found</p>
        <p class="text-sm text-muted-foreground mt-1">AI-suggested fixes will appear here when probe failures occur.</p>
      </div>

      <DataTable
        v-else
        :data="filteredFixes"
        :columns="tableColumns"
        :on-row-click="viewDetails"
      >
        <template #row="{ row }">
          <td class="px-6 py-3">
            <div class="flex items-center gap-3">
              <div class="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center shrink-0">
                <Sparkles class="w-4 h-4 text-primary" />
              </div>
              <div class="min-w-0">
                <div class="font-medium text-sm truncate">{{ row.workflow_name }}</div>
                <div class="text-xs text-muted-foreground">{{ row.workflow_id.substring(0, 8) }}...</div>
              </div>
            </div>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm font-mono truncate max-w-[150px]">{{ row.node_id }}</div>
          </td>
          <td class="px-6 py-3">
            <Badge variant="outline" class="text-xs">
              <Wand2 class="w-3 h-3 mr-1" />
              {{ getFixTypeLabel(row.fix_type) }}
            </Badge>
          </td>
          <td class="px-6 py-3">
            <div class="flex items-center gap-2">
              <div class="w-16 h-2 bg-muted rounded-full overflow-hidden">
                <div 
                  class="h-full rounded-full transition-all"
                  :class="{
                    'bg-green-500': row.confidence >= 0.8,
                    'bg-amber-500': row.confidence >= 0.5 && row.confidence < 0.8,
                    'bg-red-500': row.confidence < 0.5
                  }"
                  :style="{ width: `${row.confidence * 100}%` }"
                ></div>
              </div>
              <span class="text-sm" :class="getConfidenceColor(row.confidence)">
                {{ (row.confidence * 100).toFixed(0) }}%
              </span>
            </div>
          </td>
          <td class="px-6 py-3">
            <Badge :class="getStatusColor(row.status)" class="text-xs capitalize">
              <component :is="getStatusIcon(row.status)" class="w-3 h-3 mr-1" />
              {{ row.status }}
              <span v-if="row.auto_applied && row.status === 'applied'" class="ml-1">(auto)</span>
            </Badge>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm text-muted-foreground">{{ formatDate(row.created_at) }}</div>
          </td>
          <td class="px-6 py-3 text-right" @click.stop>
            <div class="flex items-center justify-end gap-1">
              <Button 
                @click="viewDetails(row)"
                size="sm"
                variant="ghost"
                class="h-8 w-8 p-0"
                title="View Details"
              >
                <Eye class="h-4 w-4" />
              </Button>
              <template v-if="row.status === 'pending'">
                <Button 
                  @click="handleApprove(row)"
                  size="sm"
                  variant="ghost"
                  class="h-8 w-8 p-0 text-green-600 hover:text-green-700 hover:bg-green-50"
                  :disabled="processingFixes.has(row.id)"
                  title="Approve"
                >
                  <Loader2 v-if="processingFixes.has(row.id)" class="h-4 w-4 animate-spin" />
                  <Check v-else class="h-4 w-4" />
                </Button>
                <Button 
                  @click="handleReject(row)"
                  size="sm"
                  variant="ghost"
                  class="h-8 w-8 p-0 text-red-600 hover:text-red-700 hover:bg-red-50"
                  :disabled="processingFixes.has(row.id)"
                  title="Reject"
                >
                  <X class="h-4 w-4" />
                </Button>
              </template>
            </div>
          </td>
        </template>
      </DataTable>
    </div>

    <!-- Detail Dialog -->
    <Dialog v-model:open="showDetailDialog">
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
            <div class="flex items-start gap-2">
              <span class="text-xs text-muted-foreground font-medium w-16 shrink-0">Workflow:</span>
              <span class="text-xs text-foreground break-all">
                {{ workflows[selectedFix.workflow_id] || selectedFix.workflow_id }}
              </span>
            </div>
          </Card>

          <!-- Selector Change (for both update_selector and update_field_selector) -->
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
                <span class="text-xs text-green-500 font-medium w-12 shrink-0">NEW:</span>
                <code class="text-xs bg-green-50 dark:bg-green-900/20 text-green-600 dark:text-green-400 px-2 py-1 rounded break-all">
                  {{ selectedFix.new_selector || 'N/A' }}
                </code>
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
          <Button variant="outline" @click="showDetailDialog = false">
            Cancel
          </Button>
          <Button 
            variant="destructive" 
            @click="handleReject(selectedFix!)"
            :disabled="processingFixes.has(selectedFix!.id)"
          >
            <X class="w-4 h-4 mr-2" />
            Reject
          </Button>
          <Button 
            @click="handleApprove(selectedFix!)"
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
