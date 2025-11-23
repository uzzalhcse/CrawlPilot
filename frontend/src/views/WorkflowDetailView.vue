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
      start_urls: workflowsStore.currentWorkflow?.config.start_urls || [],
      phases,
      max_depth: workflowsStore.currentWorkflow?.config.max_depth || 3,
      rate_limit_delay: workflowsStore.currentWorkflow?.config.rate_limit_delay || 1000,
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
      description: `${data.name} has been updated with ${data.nodes.length} node(s) in ${phases.length} phase(s)`
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
