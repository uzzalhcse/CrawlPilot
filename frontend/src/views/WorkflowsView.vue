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
import { Plus, Play, Pencil, Trash2, Loader2, SlidersHorizontal, Bookmark, Workflow as WorkflowIcon } from 'lucide-vue-next'
import WorkflowDialog from '@/components/workflows/WorkflowDialog.vue'
import DeleteDialog from '@/components/workflows/DeleteDialog.vue'

import { toast } from 'vue-sonner'

const router = useRouter()
const workflowsStore = useWorkflowsStore()

const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const showDeleteDialog = ref(false)
const selectedWorkflow = ref<any>(null)
const statusFilter = ref<string>('all')
const searchQuery = ref('')
const activeTab = ref('recent')

const tableColumns = [
  { key: 'name', label: 'Name', sortable: true, align: 'left' as const },
  { key: 'description', label: 'Description', align: 'left' as const },
  { key: 'status', label: 'Status', align: 'left' as const },
  { key: 'created', label: 'Created', align: 'left' as const },
  { key: 'actions', label: 'Actions', align: 'right' as const }
]

const tabs = [
  { id: 'recent', label: 'Recent & Bookmarked' },
  { id: 'issues', label: 'Issues' }
]

const stats = computed(() => [
  { label: 'Total Workflows', value: workflowsStore.workflows.length },
  { label: 'Active', value: workflowsStore.activeWorkflows.length, color: 'text-green-600 dark:text-green-400' },
  { label: 'Draft', value: workflowsStore.draftWorkflows.length }
])

const filteredWorkflows = computed(() => {
  let result = workflowsStore.workflows
  
  if (statusFilter.value !== 'all') {
    result = result.filter((w: any) => w.status === statusFilter.value)
  }
  
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter((w: any) => 
      w.name.toLowerCase().includes(query) || 
      w.description?.toLowerCase().includes(query)
    )
  }
  
  return result
})

const handleCreateWorkflow = () => {
  router.push('/workflows/create')
}

const handleEditWorkflow = (workflow: any) => {
  selectedWorkflow.value = workflow
  showEditDialog.value = true
}

const handleDeleteWorkflow = (workflow: any) => {
  selectedWorkflow.value = workflow
  showDeleteDialog.value = true
}

const handleExecuteWorkflow = async (id: string) => {
  try {
    const result = await workflowsStore.executeWorkflow(id)
    router.push(`/executions/${result.execution_id}`)
  } catch (error) {
    console.error('Failed to execute workflow:', error)
    toast.error('Failed to execute workflow')
  }
}

const handleViewDetails = (workflow: any) => {
  router.push(`/workflows/${workflow.id}`)
}

const formatDate = (dateString: string) => {
  const date = new Date(dateString)
  return date.toLocaleDateString('en-US', { month: '2-digit', day: '2-digit', year: 'numeric' }) + ', ' + date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false })
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
      title="Workflows" 
      description="Manage your web crawling workflows"
      :show-help-icon="true"
    >
      <template #actions>
        <Button variant="outline" size="default">Go to Store</Button>
        <Button variant="outline" size="default">Develop new</Button>
        <Button @click="handleCreateWorkflow" variant="default" class="bg-primary hover:bg-primary/90">Create Workflow</Button>
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
          <SelectTrigger class="w-[160px] h-9">
            <div class="flex items-center gap-2">
              <SlidersHorizontal class="w-4 h-4" />
              <SelectValue placeholder="Last run status" />
            </div>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All status</SelectItem>
            <SelectItem value="active">Active</SelectItem>
            <SelectItem value="draft">Draft</SelectItem>
          </SelectContent>
        </Select>

        <Button variant="outline" size="sm" class="h-9 gap-2">
          <Bookmark class="w-4 h-4" />
          Bookmarked
        </Button>

        <Button variant="outline" size="sm" class="h-9">Pricing model</Button>
      </template>
    </FilterBar>

    <!-- Table -->
    <div class="flex-1 overflow-auto">
      <div v-if="workflowsStore.loading" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <div v-else-if="filteredWorkflows.length === 0" class="py-12 text-center px-6">
        <p class="text-muted-foreground">No workflows found</p>
      </div>

      <DataTable
        v-else
        :data="filteredWorkflows"
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
                <div class="font-medium text-sm truncate">{{ row.name }}</div>
                <div class="text-xs text-muted-foreground truncate">{{ row.id }}</div>
              </div>
            </div>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm text-muted-foreground max-w-md truncate">
              {{ row.description }}
            </div>
          </td>
          <td class="px-6 py-3" @click.stop>
            <Badge 
              variant="outline"
              :class="{
                'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20': row.status === 'active',
                'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20': row.status === 'draft'
              }"
              class="text-xs font-medium"
            >
              <div class="w-1.5 h-1.5 rounded-full mr-1.5" :class="{
                'bg-green-500': row.status === 'active',
                'bg-amber-500': row.status === 'draft'
              }"></div>
              {{ row.status }}
            </Badge>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm text-muted-foreground">
              {{ formatDate(row.created_at) }}
            </div>
          </td>
          <td class="px-6 py-3 text-right" @click.stop>
            <div class="flex items-center justify-end gap-1">
              <Button 
                @click="handleExecuteWorkflow(row.id)"
                size="sm"
                variant="ghost"
                class="h-8 w-8 p-0"
                :disabled="row.status !== 'active'"
              >
                <Play class="h-4 w-4" />
              </Button>
              <Button 
                @click="handleEditWorkflow(row)"
                size="sm"
                variant="ghost"
                class="h-8 w-8 p-0"
              >
                <Pencil class="h-4 w-4" />
              </Button>
              <Button 
                @click="handleDeleteWorkflow(row)"
                size="sm"
                variant="ghost"
                class="h-8 w-8 p-0 text-destructive hover:text-destructive"
              >
                <Trash2 class="h-4 w-4" />
              </Button>
            </div>
          </td>
        </template>
      </DataTable>
    </div>

    <!-- Dialogs -->
    <WorkflowDialog 
      v-model:open="showCreateDialog"
      mode="create"
    />
    
    <WorkflowDialog 
      v-model:open="showEditDialog"
      mode="edit"
      :workflow="selectedWorkflow"
    />
    
    <DeleteDialog 
      v-model:open="showDeleteDialog"
      :workflow="selectedWorkflow"
    />
  </PageLayout>
</template>
