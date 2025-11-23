<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useWorkflowsStore } from '@/stores/workflows'
import type { WorkflowNode, WorkflowEdge, WorkflowConfig } from '@/types'
import WorkflowBuilder from '@/components/workflow-builder/WorkflowBuilder.vue'
import { Loader2 } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const route = useRoute()
const router = useRouter()
const workflowsStore = useWorkflowsStore()

const workflowId = route.params.id as string
const loading = ref(true)
const error = ref<string | null>(null)

onMounted(async () => {
  try {
    await workflowsStore.fetchWorkflowById(workflowId)
  } catch (e: any) {
    error.value = e.message || 'Failed to load workflow'
  } finally {
    loading.value = false
  }
})

import { convertNodesToWorkflowConfig } from '@/lib/workflow-utils'

async function handleSave(data: {
  name: string
  description: string
  status: 'draft' | 'active'
  nodes: WorkflowNode[]
  edges: WorkflowEdge[]
  config?: WorkflowConfig
}) {
  try {
    if (!data.name.trim()) {
      toast.error('Workflow name required', {
        description: 'Please enter a name for your workflow'
      })
      return
    }

    // Use the config passed from the builder (if available, e.g. from JSON mode)
    // or convert nodes/edges to config using our utility
    let config = data.config
    
    if (!config) {
      config = convertNodesToWorkflowConfig(data.nodes, data.edges, {
        // Preserve existing config
        start_urls: workflowsStore.currentWorkflow?.config.start_urls,
        max_depth: workflowsStore.currentWorkflow?.config.max_depth,
        rate_limit_delay: workflowsStore.currentWorkflow?.config.rate_limit_delay,
        storage: workflowsStore.currentWorkflow?.config.storage,
      })
    }

    await workflowsStore.updateWorkflow(workflowId, {
      name: data.name,
      description: data.description,
      status: data.status,
      config
    })

    // Dismiss loading toast and show success
    toast.dismiss('save-workflow')
    toast.success('Workflow saved successfully', {
      description: `${data.name} has been updated`
    })
  } catch (e: any) {
    const errorMessage = e.response?.data?.error || e.message || 'Unknown error'
    // Dismiss loading toast and show error
    toast.dismiss('save-workflow')
    toast.error('Failed to save workflow', {
      description: errorMessage
    })
    console.error('Save error:', e)
  }
}

async function handleExecute() {
  try {
    const result = await workflowsStore.executeWorkflow(workflowId)
    router.push(`/executions/${result.execution_id}`)
  } catch (e: any) {
    alert('Failed to execute workflow: ' + (e.message || 'Unknown error'))
  }
}


</script>

<template>
  <div class="h-screen">
    <!-- Loading State -->
    <div v-if="loading" class="flex items-center justify-center h-full">
      <Loader2 class="w-8 h-8 animate-spin text-primary" />
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="flex items-center justify-center h-full">
      <div class="text-center">
        <p class="text-destructive mb-4">{{ error }}</p>
        <button @click="router.push('/workflows')" class="text-primary underline">
          Back to Workflows
        </button>
      </div>
    </div>

    <!-- Workflow Builder -->
    <WorkflowBuilder
      v-else
      :workflow="workflowsStore.currentWorkflow"
      @save="handleSave"
      @execute="handleExecute"
    />
  </div>
</template>
