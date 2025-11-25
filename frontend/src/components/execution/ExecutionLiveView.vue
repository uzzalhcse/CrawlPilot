<script setup lang="ts">
import { onMounted, computed, ref } from 'vue'
import { useExecutionStream } from '@/composables/useExecutionStream'
import { useExecutionsStore } from '@/stores/executions'
import PhaseStepper from './PhaseStepper.vue'
import LiveLogViewer from './LiveLogViewer.vue'
import ActiveExecutionGraph from './ActiveExecutionGraph.vue'
import { Card } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Wifi, WifiOff } from 'lucide-vue-next'

const props = defineProps<{
  executionId: string
  workflowConfig?: any
  executionStatus?: string
}>()

const executionsStore = useExecutionsStore()
const { connect, disconnect, isConnected, logs, currentPhase, activeNodes, nodeStatuses, executionTree } = useExecutionStream(props.executionId)

// For completed executions, load the tree from API
const apiExecutionTree = ref<any[]>([])

onMounted(async () => {
  connect()
  
  console.log('[ExecutionLiveView] Mounted. Status:', props.executionStatus) // DEBUG
  
  // Check if execution is completed and load tree from API
  if (props.executionStatus && (props.executionStatus === 'completed' || props.executionStatus === 'failed')) {
    console.log('[ExecutionLiveView] Fetching node tree for completed execution') // DEBUG
    await executionsStore.fetchNodeTree(props.executionId)
    console.log('[ExecutionLiveView] Node tree response:', executionsStore.nodeTree) // DEBUG
    if (executionsStore.nodeTree) {
      // Convert API tree format to executionTree format
      apiExecutionTree.value = convertNodeTreeToExecutionTree(executionsStore.nodeTree.tree || [])
      console.log('[ExecutionLiveView] Converted tree:', apiExecutionTree.value) // DEBUG
    }
  }
})

// Convert NodeTree API format to executionTree format
function convertNodeTreeToExecutionTree(nodes: any[]): any[] {
  const result = nodes.map(node => {
    // Skip nodes without proper IDs
    if (!node.id) {
      console.warn('[ExecutionLiveView] Skipping node with missing ID:', node) // DEBUG
      return null
    }
    
    return {
      id: node.id, // This is the node_execution_id from the database
      node_id: node.node_id || node.id,
      node_execution_id: node.id, // API returns 'id' which is the node_execution_id
      parent_node_execution_id: node.parent_node_execution_id,
      node_type: node.node_type || '',
      status: node.status || 'completed',
      error: node.error_message,
      result: node.result,
      children: node.children ? convertNodeTreeToExecutionTree(node.children) : []
    }
  }).filter(n => n !== null) // Remove null entries
  
  console.log('[ExecutionLiveView] Converted nodes:', result.length, result) // DEBUG
  return result
}

// Use API tree if execution is completed, otherwise use live SSE tree
const effectiveExecutionTree = computed(() => {
  if (apiExecutionTree.value.length > 0) {
    return apiExecutionTree.value
  }
  return executionTree.value
})

// Compute phases based on workflow config + current state
const phases = computed(() => {
  const configPhases = props.workflowConfig?.phases || []
  
  if (configPhases.length === 0) {
    // Fallback if no phases defined
    return [
      { id: 'init', name: 'Initialization', status: 'completed' },
      { id: 'processing', name: 'Processing', status: 'running' },
      { id: 'completion', name: 'Completion', status: 'pending' }
    ]
  }

  return configPhases.map((p: any) => {
    let status = 'pending'
    
    // Simple logic: 
    // If currentPhase matches this phase, it's running.
    // If we passed it (how do we know?), it's completed.
    // This is tricky without full history. 
    // For now, let's just highlight the current one.
    
    if (currentPhase.value === p.id) {
      status = 'running'
    } else if (currentPhase.value && configPhases.findIndex((cp: any) => cp.id === currentPhase.value) > configPhases.findIndex((cp: any) => cp.id === p.id)) {
      status = 'completed'
    }
    
    return {
      id: p.id,
      name: p.name || p.id,
      status
    }
  })
})

</script>

<template>
  <div class="space-y-6">
    <!-- Connection Status -->
    <div class="flex items-center justify-end">
      <Badge :variant="isConnected ? 'outline' : 'destructive'" class="gap-1 transition-all duration-300">
        <Wifi v-if="isConnected" class="h-3 w-3 text-green-500" />
        <WifiOff v-else class="h-3 w-3" />
        {{ isConnected ? 'Live Stream Connected' : 'Disconnected' }}
      </Badge>
    </div>

    <!-- Phase Stepper -->
    <Card class="p-6">
      <h3 class="text-lg font-semibold mb-4">Execution Progress</h3>
      <PhaseStepper :phases="phases" :current-phase-id="currentPhase" />
    </Card>

    <!-- Active Execution Graph -->
    <Card class="p-0 overflow-hidden border shadow-sm">
      <div class="p-4 border-b bg-muted/30 flex items-center justify-between">
        <h3 class="font-semibold">Live Workflow Graph</h3>
        <div class="text-xs text-muted-foreground">
          {{ activeNodes.size }} active nodes
        </div>
      </div>
      <ActiveExecutionGraph 
        :workflow-config="workflowConfig" 
        :active-nodes="activeNodes"
        :node-statuses="nodeStatuses"
        :execution-tree="effectiveExecutionTree"
      />
    </Card>

    <!-- Live Logs -->
    <LiveLogViewer :logs="logs" />
  </div>
</template>
