<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useWorkflowsStore } from '@/stores/workflows'
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
import { Plus, Play, Pencil, Trash2, MoreVertical, Loader2 } from 'lucide-vue-next'
import WorkflowDialog from '@/components/workflows/WorkflowDialog.vue'
import DeleteDialog from '@/components/workflows/DeleteDialog.vue'

const router = useRouter()
const workflowsStore = useWorkflowsStore()

const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const showDeleteDialog = ref(false)
const selectedWorkflow = ref<any>(null)
const statusFilter = ref<string>('all')

const filteredWorkflows = computed(() => {
  if (statusFilter.value === 'all') {
    return workflowsStore.workflows
  }
  return workflowsStore.workflows.filter(w => w.status === statusFilter.value)
})

const getStatusVariant = (status: string) => {
  switch (status) {
    case 'active':
      return 'success'
    case 'draft':
      return 'secondary'
    case 'inactive':
      return 'outline'
    default:
      return 'default'
  }
}

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
  }
}

const handleViewDetails = (id: string) => {
  router.push(`/workflows/${id}`)
}

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString()
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
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">Workflows</h2>
        <p class="text-muted-foreground">
          Manage your web crawling workflows
        </p>
      </div>
      <Button @click="handleCreateWorkflow">
        <Plus class="mr-2 h-4 w-4" />
        Create Workflow
      </Button>
    </div>

    <!-- Stats Cards -->
    <div class="grid gap-4 md:grid-cols-3">
      <Card class="p-6">
        <div class="flex flex-col space-y-2">
          <span class="text-sm font-medium text-muted-foreground">Total Workflows</span>
          <span class="text-3xl font-bold">{{ workflowsStore.workflows.length }}</span>
        </div>
      </Card>
      <Card class="p-6">
        <div class="flex flex-col space-y-2">
          <span class="text-sm font-medium text-muted-foreground">Active</span>
          <span class="text-3xl font-bold text-green-600">{{ workflowsStore.activeWorkflows.length }}</span>
        </div>
      </Card>
      <Card class="p-6">
        <div class="flex flex-col space-y-2">
          <span class="text-sm font-medium text-muted-foreground">Draft</span>
          <span class="text-3xl font-bold text-gray-600">{{ workflowsStore.draftWorkflows.length }}</span>
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
            <SelectItem value="active">Active</SelectItem>
            <SelectItem value="draft">Draft</SelectItem>
            <SelectItem value="inactive">Inactive</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>

    <!-- Workflows Table -->
    <Card>
      <div class="p-6">
        <div v-if="workflowsStore.loading" class="flex items-center justify-center py-12">
          <Loader2 class="h-8 w-8 animate-spin text-primary" />
        </div>

        <div v-else-if="workflowsStore.error" class="py-12 text-center">
          <p class="text-destructive">{{ workflowsStore.error }}</p>
          <Button @click="workflowsStore.fetchWorkflows()" variant="outline" class="mt-4">
            Retry
          </Button>
        </div>

        <div v-else-if="filteredWorkflows.length === 0" class="py-12 text-center">
          <p class="text-muted-foreground">No workflows found</p>
          <Button @click="handleCreateWorkflow" variant="outline" class="mt-4">
            <Plus class="mr-2 h-4 w-4" />
            Create Your First Workflow
          </Button>
        </div>

        <Table v-else>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Description</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Created</TableHead>
              <TableHead class="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow 
              v-for="workflow in filteredWorkflows" 
              :key="workflow.id"
              class="cursor-pointer hover:bg-muted/50"
            >
              <TableCell 
                @click="handleViewDetails(workflow.id)"
                class="font-medium"
              >
                {{ workflow.name }}
              </TableCell>
              <TableCell 
                @click="handleViewDetails(workflow.id)"
                class="max-w-md truncate"
              >
                {{ workflow.description }}
              </TableCell>
              <TableCell @click="handleViewDetails(workflow.id)">
                <Badge :variant="getStatusVariant(workflow.status)">
                  {{ workflow.status }}
                </Badge>
              </TableCell>
              <TableCell 
                @click="handleViewDetails(workflow.id)"
                class="text-muted-foreground"
              >
                {{ formatDate(workflow.created_at) }}
              </TableCell>
              <TableCell class="text-right">
                <div class="flex items-center justify-end gap-2">
                  <Button 
                    @click="handleExecuteWorkflow(workflow.id)"
                    size="sm"
                    variant="outline"
                    :disabled="workflow.status !== 'active'"
                  >
                    <Play class="h-4 w-4" />
                  </Button>
                  <Button 
                    @click="handleEditWorkflow(workflow)"
                    size="sm"
                    variant="outline"
                  >
                    <Pencil class="h-4 w-4" />
                  </Button>
                  <Button 
                    @click="handleDeleteWorkflow(workflow)"
                    size="sm"
                    variant="outline"
                  >
                    <Trash2 class="h-4 w-4 text-destructive" />
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
    </Card>

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
  </div>
</template>

