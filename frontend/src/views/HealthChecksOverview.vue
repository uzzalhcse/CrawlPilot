<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useWorkflowsStore } from '@/stores/workflows'
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
import { Activity, Eye, Loader2, CheckCircle, AlertCircle, SlidersHorizontal, Workflow as WorkflowIcon } from 'lucide-vue-next'

const router = useRouter()
const workflowsStore = useWorkflowsStore()

const statusFilter = ref<string>('all')
const searchQuery = ref('')
const activeTab = ref('all')

const tableColumns = [
  { key: 'workflow', label: 'Workflow', sortable: true, align: 'left' as const },
  { key: 'status', label: 'Health Status', align: 'left' as const },
  { key: 'lastCheck', label: 'Last Check', align: 'left' as const },
  { key: 'issues', label: 'Issues Found', align: 'left' as const },
  { key: 'actions', label: 'Actions', align: 'right' as const }
]

const tabs = [
  { id: 'all', label: 'All Health Checks' },
  { id: 'issues', label: 'With Issues' }
]

const activeWorkflows = computed(() => 
  workflowsStore.workflows.filter((w: any) => w.status === 'active')
)

const healthyCount = computed(() => 
  Math.floor(activeWorkflows.value.length * 0.85)
)

const issuesCount = computed(() => 
  Math.floor(activeWorkflows.value.length * 0.15)
)

const stats = computed(() => [
  { label: 'Total Workflows', value: activeWorkflows.value.length },
  { label: 'Healthy', value: healthyCount.value, color: 'text-green-600 dark:text-green-400' },
  { label: 'With Issues', value: issuesCount.value, color: 'text-amber-600 dark:text-amber-400' },
])

// Mock health check data
const healthChecks = computed(() => 
  activeWorkflows.value.map((workflow: any, index: number) => ({
    id: workflow.id,
    workflow_name: workflow.name,
    workflow_id: workflow.id,
    status: index % 5 === 0 ? 'issues' : 'healthy',
    lastCheck: workflow.updated_at || workflow.created_at,
    issuesFound: index % 5 === 0 ? Math.floor(Math.random() * 3) + 1 : 0
  }))
)

const filteredHealthChecks = computed(() => {
  let result = healthChecks.value

  if (statusFilter.value !== 'all') {
    result = result.filter((h: any) => h.status === statusFilter.value)
  }

  if (activeTab.value === 'issues') {
    result = result.filter((h: any) => h.issuesFound > 0)
  }

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter((h: any) =>
      h.workflow_name.toLowerCase().includes(query)
    )
  }

  return result
})

const handleViewDetails = (healthCheck: any) => {
  router.push(`/monitoring/${healthCheck.workflow_id}`)
}

const formatDate = (dateString: string) => {
  const date = new Date(dateString)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (days > 0) return `${days} day${days > 1 ? 's' : ''} ago`
  if (hours > 0) return `${hours} hour${hours > 1 ? 's' : ''} ago`
  if (minutes > 0) return `${minutes} minute${minutes > 1 ? 's' : ''} ago`
  return 'Just now'
}

onMounted(async () => {
  try {
    await workflowsStore.fetchWorkflows()
  } catch (error) {
    console.error('Failed to load workflows:', error)
  }
})
</script>

<template>
  <PageLayout>
    <!-- Header -->
    <PageHeader 
      title="Monitoring" 
      description="Monitor workflow health and detect structure changes"
      :show-help-icon="true"
    >
      <template #actions>
        <Button variant="outline" size="default">
          <Loader2 class="mr-2 h-4 w-4" />
          Run All Checks
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
            <SelectItem value="healthy">Healthy</SelectItem>
            <SelectItem value="issues">With Issues</SelectItem>
          </SelectContent>
        </Select>
      </template>
    </FilterBar>

    <!-- Table -->
    <div class="flex-1 overflow-auto">
      <div v-if="activeWorkflows.length === 0" class="py-12 text-center px-6">
        <Activity class="mx-auto h-12 w-12 text-muted-foreground" />
        <p class="mt-4 text-lg font-medium">No Active Workflows</p>
        <p class="mt-2 text-sm text-muted-foreground">
          Create an active workflow to run health checks
        </p>
        <Button @click="router.push('/workflows')" variant="outline" class="mt-4">
          Go to Workflows
        </Button>
      </div>

      <DataTable
        v-else-if="filteredHealthChecks.length > 0"
        :data="filteredHealthChecks"
        :columns="tableColumns"
        :on-row-click="handleViewDetails"
      >
        <template #row="{ row }">
          <td class="px-6 py-3">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center shrink-0">
                <WorkflowIcon class="w-5 h-5 text-primary" />
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
                'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20': row.status === 'healthy',
                'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20': row.status === 'issues'
              }"
              class="text-xs font-medium"
            >
              <component 
                :is="row.status === 'healthy' ? CheckCircle : AlertCircle" 
                class="w-3 h-3 mr-1.5"
              />
              {{ row.status === 'healthy' ? 'Healthy' : 'Has Issues' }}
            </Badge>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm text-muted-foreground">
              {{ formatDate(row.lastCheck) }}
            </div>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm font-medium" :class="row.issuesFound > 0 ? 'text-amber-600 dark:text-amber-400' : 'text-muted-foreground'">
              {{ row.issuesFound > 0 ? `${row.issuesFound} issue${row.issuesFound > 1 ? 's' : ''}` : 'None' }}
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
                size="sm"
                variant="ghost"
                class="h-8 w-8 p-0"
              >
                <Activity class="h-4 w-4" />
              </Button>
            </div>
          </td>
        </template>
      </DataTable>

      <div v-else class="py-12 text-center px-6">
        <p class="text-muted-foreground">No health checks match your filters</p>
      </div>
    </div>
  </PageLayout>
</template>
