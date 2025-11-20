<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useWorkflowsStore } from '@/stores/workflows'
import type { WorkflowNode, WorkflowEdge } from '@/types'
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

    // Convert nodes and edges back to workflow config
    // Backend expects nodes in two categories: url_discovery and data_extraction
    // Based on backend NodeType constants in pkg/models/workflow.go
    
    const URL_DISCOVERY_TYPES = new Set([
      'fetch',
      'extract_links',
      'filter_urls',
      'navigate',
      'paginate'
    ])
    
    const DATA_EXTRACTION_TYPES = new Set([
      'extract',
      'extract_text',
      'extract_attr',
      'extract_json'
    ])
    
    const INTERACTION_TYPES = new Set([
      'click',
      'scroll',
      'type',
      'hover',
      'wait',
      'wait_for',
      'screenshot'
    ])
    
    const TRANSFORMATION_TYPES = new Set([
      'transform',
      'filter',
      'map',
      'validate'
    ])
    
    // Categorize nodes based on their type
    const urlDiscoveryNodes: any[] = []
    const dataExtractionNodes: any[] = []
    
    data.nodes.forEach(node => {
      const nodeType = node.data.nodeType
      const backendNode = convertToBackendNode(node, data.edges)
      
      if (URL_DISCOVERY_TYPES.has(nodeType)) {
        urlDiscoveryNodes.push(backendNode)
      } else if (DATA_EXTRACTION_TYPES.has(nodeType)) {
        dataExtractionNodes.push(backendNode)
      } else if (INTERACTION_TYPES.has(nodeType) || TRANSFORMATION_TYPES.has(nodeType)) {
        // Interaction and transformation nodes can be in either category depending on their dependencies
        // If they depend on extraction nodes, they go to data_extraction, otherwise url_discovery
        const deps = backendNode.dependencies || []
        const dependsOnExtraction = deps.some((depId: string) => {
          const depNode = data.nodes.find(n => n.id === depId)
          return depNode && DATA_EXTRACTION_TYPES.has(depNode.data.nodeType)
        })
        
        if (dependsOnExtraction || dataExtractionNodes.length > 0) {
          dataExtractionNodes.push(backendNode)
        } else {
          urlDiscoveryNodes.push(backendNode)
        }
      } else {
        // Unknown node types default to url_discovery
        console.warn(`Unknown node type: ${nodeType}, adding to url_discovery`)
        urlDiscoveryNodes.push(backendNode)
      }
    })

    // Validate: Check for duplicate node IDs (should never happen with proper categorization)
    const allNodes = [...urlDiscoveryNodes, ...dataExtractionNodes]
    const nodeIds = allNodes.map(n => n.id)
    const duplicates = nodeIds.filter((id, index) => nodeIds.indexOf(id) !== index)

    if (duplicates.length > 0) {
      toast.error('Duplicate node IDs found', {
        description: `The following nodes appear multiple times: ${duplicates.join(', ')}. Please refresh and try again.`
      })
      console.error('Duplicate node IDs:', duplicates)
      console.error('URL Discovery nodes:', urlDiscoveryNodes.map(n => ({ id: n.id, type: n.type })))
      console.error('Data Extraction nodes:', dataExtractionNodes.map(n => ({ id: n.id, type: n.type })))
      return
    }
    
    // Validate: Ensure all nodes are categorized
    if (allNodes.length !== data.nodes.length) {
      toast.warning('Node count mismatch', {
        description: `${data.nodes.length} nodes in builder, but only ${allNodes.length} were categorized. Some nodes may be missing.`
      })
      console.warn(`Node count mismatch: ${data.nodes.length} in builder, ${allNodes.length} categorized`)
    }

    const config = {
      start_urls: workflowsStore.currentWorkflow?.config.start_urls || [],
      max_depth: workflowsStore.currentWorkflow?.config.max_depth || 3,
      rate_limit_delay: workflowsStore.currentWorkflow?.config.rate_limit_delay || 1000,
      url_discovery: urlDiscoveryNodes.length > 0 ? urlDiscoveryNodes : undefined,
      data_extraction: dataExtractionNodes.length > 0 ? dataExtractionNodes : undefined,
      storage: workflowsStore.currentWorkflow?.config.storage || {
        type: 'database'
      }
    }

    await workflowsStore.updateWorkflow(workflowId, {
      name: data.name,
      description: data.description,
      config
    })

    // Dismiss loading toast and show success
    toast.dismiss('save-workflow')
    toast.success('Workflow saved successfully', {
      description: `${data.name} has been updated with ${allNodes.length} node(s)`
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
