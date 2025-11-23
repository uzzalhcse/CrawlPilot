<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useExecutionStream } from '@/composables/useExecutionStream'
import PhaseStepper from './PhaseStepper.vue'
import LiveLogViewer from './LiveLogViewer.vue'
import ActiveExecutionGraph from './ActiveExecutionGraph.vue'
import { Card } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Wifi, WifiOff } from 'lucide-vue-next'

const props = defineProps<{
  executionId: string
  workflowConfig?: any
}>()

const { connect, disconnect, isConnected, logs, currentPhase, activeNodes, nodeStatuses } = useExecutionStream(props.executionId)

onMounted(() => {
  connect()
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
      />
    </Card>

    <!-- Live Logs -->
    <LiveLogViewer :logs="logs" />
  </div>
</template>
