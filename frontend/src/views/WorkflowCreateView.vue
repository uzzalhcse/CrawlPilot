<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useWorkflowsStore } from '@/stores/workflows'
import type { WorkflowNode, WorkflowEdge } from '@/types'
import WorkflowBuilder from '@/components/workflow-builder/WorkflowBuilder.vue'
import { toast } from 'vue-sonner'

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
      toast.error('Workflow name required', {
        description: 'Please enter a name for your workflow'
      })
      return
    }

    if (data.nodes.length === 0) {
      toast.error('No nodes added', {
        description: 'Please add at least one node to the workflow'
      })
      return
    }

    // Convert nodes and edges to phase-based workflow config
    // Group nodes into phases based on their type and dependencies
    
    const phases: any[] = []
    
    // Phase 1: URL Discovery - nodes that discover/navigate URLs
    const discoveryNodes = data.nodes.filter(node => {
      const type = node.data.nodeType
      return ['fetch', 'extract_links', 'filter_urls', 'navigate', 'paginate'].includes(type)
    })
    
    if (discoveryNodes.length > 0) {
      phases.push({
        id: 'discovery_phase',
        type: 'discovery',
        name: 'URL Discovery',
        nodes: discoveryNodes.map(node => convertToBackendNode(node, data.edges)),
        url_filter: {
          depth: 0 // Start URLs
        }
      })
    }
    
    // Phase 2: Extraction - nodes that extract data from product pages
    const extractionNodes = data.nodes.filter(node => {
      const type = node.data.nodeType
      return ['extract', 'extract_text', 'extract_attr', 'extract_json', 'sequence', 
              'click', 'scroll', 'hover', 'type', 'wait'].includes(type)
    })
    
    if (extractionNodes.length > 0) {
      phases.push({
        id: 'extraction_phase',
        type: 'extraction',
        name: 'Data Extraction',
        nodes: extractionNodes.map(node => convertToBackendNode(node, data.edges)),
        url_filter: {
          markers: ['product'] // Default marker
        }
      })
    }
    
    // Phase 3: Processing - transformation and validation nodes
    const processingNodes = data.nodes.filter(node => {
      const type = node.data.nodeType
      return ['transform', 'filter', 'map', 'validate'].includes(type)
    })
    
    if (processingNodes.length > 0) {
      phases.push({
        id: 'processing_phase',
        type: 'processing',
        name: 'Data Processing',
        nodes: processingNodes.map(node => convertToBackendNode(node, data.edges))
      })
    }

    const config = {
      start_urls: [],
      phases,
      max_depth: 3,
      rate_limit_delay: 1000,
      storage: {
        type: 'database' as const
      }
    }

    const workflow = await workflowsStore.createWorkflow({
      name: data.name,
      description: data.description,
      config
    })

    // Dismiss loading toast and show success
    toast.dismiss('save-workflow')
    toast.success('Workflow created successfully', {
      description: `${data.name} has been created with ${data.nodes.length} node(s) in ${phases.length} phase(s)`
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
