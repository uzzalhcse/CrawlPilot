<script setup lang="ts">
import { ref, markRaw } from 'vue'
import { VueFlow, useVueFlow, Panel, type Node, type Edge, type Connection } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import type { WorkflowNode, WorkflowEdge } from '@/types'
import CustomNode from './CustomNode.vue'
import PhaseLabelNode from './PhaseLabelNode.vue'
import ExtractionFieldNode from './ExtractionFieldNode.vue'
import { Sparkles, Scissors, Plus, Trash2, X } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { Button } from '@/components/ui/button'

// Import Vue Flow styles
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'

interface Props {
  nodes: WorkflowNode[]
  edges: WorkflowEdge[]
}

interface Emits {
  (e: 'update:nodes', nodes: WorkflowNode[]): void
  (e: 'update:edges', edges: WorkflowEdge[]): void
  (e: 'node-click', node: WorkflowNode): void
  (e: 'connect', connection: Connection): void
  (e: 'pane-click'): void
  (e: 'drop', data: { template: any; position: { x: number; y: number } }): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// Custom node types
const nodeTypes = {
  custom: markRaw(CustomNode),
  phaseLabel: markRaw(PhaseLabelNode),
  extractField: markRaw(ExtractionFieldNode)
}

const { onConnect, onPaneClick, onNodeClick, onEdgeClick, project } = useVueFlow()

// Edge context menu state
const showEdgeMenu = ref(false)
const edgeMenuPosition = ref({ x: 0, y: 0 })
const selectedEdge = ref<WorkflowEdge | null>(null)

// Handle interactions
onConnect((params) => {
  emit('connect', params)
})

onPaneClick(() => {
  emit('pane-click')
  closeEdgeMenu()
})

onNodeClick(({ node }) => {
  emit('node-click', node as WorkflowNode)
  closeEdgeMenu()
})

// Handle edge click - show context menu
onEdgeClick(({ edge, event }) => {
  event.stopPropagation()
  selectedEdge.value = edge as WorkflowEdge
  edgeMenuPosition.value = {
    x: event.clientX,
    y: event.clientY
  }
  showEdgeMenu.value = true
})

function closeEdgeMenu() {
  showEdgeMenu.value = false
  selectedEdge.value = null
}

// Disconnect (remove) the selected edge
function disconnectEdge() {
  if (!selectedEdge.value) return
  
  const newEdges = props.edges.filter(e => e.id !== selectedEdge.value!.id)
  emit('update:edges', newEdges)
  toast.success('Connection removed')
  closeEdgeMenu()
}

// Insert a node in the middle of the edge
function insertNodeOnEdge() {
  if (!selectedEdge.value) return
  
  const edge = selectedEdge.value
  const sourceNode = props.nodes.find(n => n.id === edge.source)
  const targetNode = props.nodes.find(n => n.id === edge.target)
  
  if (!sourceNode || !targetNode) return
  
  // Calculate midpoint position
  const midX = (sourceNode.position.x + targetNode.position.x) / 2
  const midY = (sourceNode.position.y + targetNode.position.y) / 2
  
  // Create a new "wait" node as a placeholder (user can change type later)
  const newNodeId = `node_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  const newNode: WorkflowNode = {
    id: newNodeId,
    type: 'custom',
    position: { x: midX, y: midY },
    data: {
      label: 'Wait',
      nodeType: 'wait',
      params: { duration: 1000 }
    }
  }
  
  // Remove old edge, add two new edges
  const newEdges = props.edges.filter(e => e.id !== edge.id)
  
  // Edge from source to new node
  newEdges.push({
    id: `${edge.source}-${newNodeId}`,
    source: edge.source,
    target: newNodeId,
    animated: edge.animated,
    style: edge.style
  })
  
  // Edge from new node to target
  newEdges.push({
    id: `${newNodeId}-${edge.target}`,
    source: newNodeId,
    target: edge.target,
    animated: edge.animated,
    style: edge.style
  })
  
  // Update nodes and edges
  emit('update:nodes', [...props.nodes, newNode])
  emit('update:edges', newEdges)
  
  toast.success('Node inserted', { description: 'Click the new node to configure it' })
  closeEdgeMenu()
}

function onDragOver(event: DragEvent) {
  event.preventDefault()
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'move'
  }
}

function onDrop(event: DragEvent) {
  const data = event.dataTransfer?.getData('application/vueflow')
  if (!data) return

  const template = JSON.parse(data)
  const bounds = (event.target as Element).getBoundingClientRect()
  
  const position = project({
    x: event.clientX - bounds.left,
    y: event.clientY - bounds.top,
  })

  emit('drop', { template, position })
}

// Expose fitView for parent component to call
const { fitView } = useVueFlow()
defineExpose({ fitView })
</script>

<template>
  <div 
    class="h-full w-full bg-muted/30 relative"
    @dragover="onDragOver"
    @drop="onDrop"
  >
    <VueFlow
      :nodes="nodes"
      :edges="edges"
      :node-types="nodeTypes"
      :default-viewport="{ zoom: 1 }"
      :min-zoom="0.2"
      :max-zoom="4"
      :edges-updatable="true"
      fit-view-on-init
      @update:nodes="emit('update:nodes', $event as WorkflowNode[])"
      @update:edges="emit('update:edges', $event as WorkflowEdge[])"
    >
      <Background
        pattern-color="hsl(var(--border))"
        :gap="20"
        :size="1"
        variant="dots"
      />
      
      <Controls
        class="!bg-card !border-border !shadow-lg"
        :show-zoom="true"
        :show-fit-view="true"
        :show-interactive="false"
      />
      
      <MiniMap
        class="!bg-card !border-border !shadow-lg"
        :node-color="() => 'hsl(var(--primary))'"
        :mask-color="'hsl(var(--muted))'"
      />

      <!-- Canvas Stats Panel -->
      <Panel position="top-right" class="space-y-2">
        <div class="bg-card/95 backdrop-blur-sm border-2 border-border/50 rounded-lg px-4 py-2.5 shadow-xl">
          <div class="flex items-center gap-4 text-sm">
            <div class="flex items-center gap-2">
              <span class="text-muted-foreground">Nodes:</span>
              <span class="font-bold text-lg text-primary">{{ nodes.length }}</span>
            </div>
            <div class="w-px h-4 bg-border"></div>
            <div class="flex items-center gap-2">
              <span class="text-muted-foreground">Edges:</span>
              <span class="font-bold text-lg text-primary">{{ edges.length }}</span>
            </div>
          </div>
        </div>
        
        <!-- Edge Help Tip -->
        <div class="bg-card/90 backdrop-blur-sm border border-border/50 rounded-lg px-3 py-2 shadow-lg text-xs text-muted-foreground">
          💡 Click on a connection line to disconnect or insert a node
        </div>
      </Panel>
    </VueFlow>

    <!-- Edge Context Menu -->
    <Teleport to="body">
      <div 
        v-if="showEdgeMenu"
        class="fixed z-[9999] bg-popover border border-border rounded-lg shadow-2xl overflow-hidden min-w-[180px]"
        :style="{ left: `${edgeMenuPosition.x}px`, top: `${edgeMenuPosition.y}px` }"
      >
        <div class="p-2 border-b bg-muted/30">
          <div class="text-xs font-medium text-muted-foreground">Connection Options</div>
        </div>
        <div class="p-1">
          <button
            class="w-full flex items-center gap-2 px-3 py-2 text-sm rounded-md hover:bg-muted transition-colors text-left"
            @click="insertNodeOnEdge"
          >
            <Plus class="w-4 h-4 text-green-500" />
            Insert Node Here
          </button>
          <button
            class="w-full flex items-center gap-2 px-3 py-2 text-sm rounded-md hover:bg-destructive/10 hover:text-destructive transition-colors text-left"
            @click="disconnectEdge"
          >
            <Scissors class="w-4 h-4" />
            Disconnect
          </button>
        </div>
        <div class="p-1 border-t">
          <button
            class="w-full flex items-center gap-2 px-3 py-2 text-xs rounded-md hover:bg-muted transition-colors text-muted-foreground"
            @click="closeEdgeMenu"
          >
            <X class="w-3 h-3" />
            Cancel
          </button>
        </div>
      </div>
    </Teleport>

    <!-- Empty State -->
    <div
      v-if="nodes.length === 0"
      class="absolute inset-0 flex items-center justify-center pointer-events-none z-10"
    >
      <div class="bg-card border border-border p-10 rounded-xl shadow-2xl text-center max-w-lg">
        <div class="w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center mx-auto mb-4">
          <Sparkles class="w-8 h-8 text-primary" />
        </div>
        <h3 class="text-2xl font-bold mb-3">Start Building Your Workflow</h3>
        <p class="text-muted-foreground">
          Drag and drop nodes from the palette on the left to get started.
        </p>
      </div>
    </div>
  </div>
</template>
