<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useWorkflowsStore } from '@/stores/workflows'
import type { WorkflowNode, WorkflowEdge } from '@/types'
import WorkflowBuilder from '@/components/workflow-builder/WorkflowBuilder.vue'

const router = useRouter()
const workflowsStore = useWorkflowsStore()

async function handleSave(data: {
  name: string
  description: string
  nodes: WorkflowNode[]
  edges: WorkflowEdge[]
}) {
  try {
    if (!data.name.trim()) {
      alert('Please enter a workflow name')
      return
    }

    if (data.nodes.length === 0) {
      alert('Please add at least one node to the workflow')
      return
    }

    // Convert nodes and edges to workflow config
    const urlDiscoveryNodes = data.nodes
      .filter(n => ['fetch', 'extract_links', 'filter_urls', 'navigate', 'paginate'].includes(n.data.nodeType))
      .map(n => convertToBackendNode(n, data.edges))

    const dataExtractionNodes = data.nodes
      .filter(n => n.data.nodeType.startsWith('extract'))
      .map(n => convertToBackendNode(n, data.edges))

    // Check for duplicate node IDs
    const allNodes = [...urlDiscoveryNodes, ...dataExtractionNodes]
    const nodeIds = allNodes.map(n => n.id)
    const duplicates = nodeIds.filter((id, index) => nodeIds.indexOf(id) !== index)

    if (duplicates.length > 0) {
      alert(`Duplicate node IDs found: ${duplicates.join(', ')}. This is a bug - please refresh and try again.`)
      console.error('Duplicate node IDs:', duplicates)
      return
    }

    const config = {
      start_urls: [],
      max_depth: 3,
      rate_limit_delay: 1000,
      url_discovery: urlDiscoveryNodes,
      data_extraction: dataExtractionNodes,
      storage: {
        type: 'database' as const
      }
    }

    const workflow = await workflowsStore.createWorkflow({
      name: data.name,
      description: data.description,
      config
    })

    alert('Workflow created successfully!')
    router.push(`/workflows/${workflow.id}`)
  } catch (e: any) {
    const errorMessage = e.response?.data?.error || e.message || 'Unknown error'
    alert(`Failed to create workflow: ${errorMessage}`)
    console.error('Create error:', e)
  }
}

function convertToBackendNode(node: WorkflowNode, edges: WorkflowEdge[]) {
  // Find dependencies from edges
  const dependencies = edges
    .filter(e => e.target === node.id)
    .map(e => e.source)

  return {
    id: node.id,
    type: node.data.nodeType,
    name: node.data.label,
    params: node.data.params,
    dependencies: dependencies.length > 0 ? dependencies : undefined,
    output_key: node.data.outputKey,
    optional: node.data.optional,
    retry: node.data.retry
  }
}

function handleExecute() {
  alert('Please save the workflow first before executing')
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
