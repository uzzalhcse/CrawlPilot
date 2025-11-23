<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useWorkflowsStore } from '@/stores/workflows'
import type { WorkflowNode, WorkflowEdge, WorkflowConfig } from '@/types'
import WorkflowBuilder from '@/components/workflow-builder/WorkflowBuilder.vue'
import { toast } from 'vue-sonner'
import { convertNodesToWorkflowConfig } from '@/lib/workflow-utils'

const router = useRouter()
const workflowsStore = useWorkflowsStore()

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

    if (data.nodes.length === 0 && !data.config) {
      toast.error('No nodes added', {
        description: 'Please add at least one node to the workflow'
      })
      return
    }

    // Use the config passed from the builder (if available, e.g. from JSON mode)
    // or convert nodes/edges to config using our utility
    let config = data.config
    
    if (!config) {
      config = convertNodesToWorkflowConfig(data.nodes, data.edges)
    }

    const workflow = await workflowsStore.createWorkflow({
      name: data.name,
      description: data.description,
      status: data.status,
      config
    })

    // Dismiss loading toast and show success
    toast.dismiss('save-workflow')
    toast.success('Workflow created successfully', {
      description: `${data.name} has been created`
    })
    router.push(`/workflows/${workflow.id}`)
  } catch (e: any) {
    const errorMessage = e.response?.data?.error || e.message || 'Unknown error'
    // Dismiss loading toast and show error
    toast.dismiss('save-workflow')
    toast.error('Failed to create workflow', {
      description: errorMessage
    })
    console.error('Create error:', e)
  }
}

function handleExecute() {
  toast.info('Save workflow first', {
    description: 'Please save the workflow before executing it'
  })
}
</script>

<template>
  <div class="h-screen">
    <WorkflowBuilder
      :workflow="null"
      @save="handleSave"
      @execute="handleExecute"
    />
  </div>
</template>
