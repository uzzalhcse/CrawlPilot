<script setup lang="ts">
import { ref, watch, markRaw } from 'vue'
import { VueFlow, useVueFlow, Panel } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import type { WorkflowNode, WorkflowEdge, NodeTemplate, Workflow } from '@/types'
import CustomNode from './CustomNode.vue'
import NodePalette from './NodePalette.vue'
import NodeConfigPanel from './NodeConfigPanel.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Save, Play, Sparkles } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

// Import Vue Flow styles
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'

interface Props {
  workflow?: Workflow | null
}

interface Emits {
  (e: 'save', data: { name: string; description: string; nodes: WorkflowNode[]; edges: WorkflowEdge[] }): void
  (e: 'execute'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// Workflow metadata
const workflowName = ref('')
const workflowDescription = ref('')

// Vue Flow
const nodes = ref<WorkflowNode[]>([])
const edges = ref<WorkflowEdge[]>([])
const selectedNode = ref<WorkflowNode | null>(null)
const showConfigPanel = ref(false)

// Generate unique node ID
function generateNodeId(): string {
  return `node_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
}

const { onConnect, addEdges, removeNodes, findNode } = useVueFlow()

// Watch for workflow changes
watch(
  () => props.workflow,
  (newWorkflow) => {
    if (newWorkflow) {
      loadWorkflow(newWorkflow)
    }
  },
  { immediate: true }
)

// Load workflow into the builder
function loadWorkflow(workflow: Workflow) {
  workflowName.value = workflow.name
  workflowDescription.value = workflow.description

  // Convert workflow config to nodes and edges
  const loadedNodes: WorkflowNode[] = []
  const loadedEdges: WorkflowEdge[] = []

  // Combine url_discovery and data_extraction nodes
  const allNodes = [
    ...(workflow.config.url_discovery || []),
    ...(workflow.config.data_extraction || [])
  ]

  // Create nodes with positions
  allNodes.forEach((node, index) => {
    const row = Math.floor(index / 3)
    const col = index % 3

    loadedNodes.push({
      id: node.id,
      type: 'custom',
      position: { x: 100 + col * 300, y: 100 + row * 200 },
      data: {
        label: node.name,
        nodeType: node.type as any,
        params: node.params,
        dependencies: node.dependencies,
        outputKey: (node as any).output_key,
        optional: (node as any).optional,
        retry: (node as any).retry
      }
    })

    // Create edges based on dependencies
    if (node.dependencies && node.dependencies.length > 0) {
      node.dependencies.forEach(depId => {
        loadedEdges.push({
          id: `${depId}-${node.id}`,
          source: depId,
          target: node.id,
          animated: true
        })
      })
    }
  })

  nodes.value = loadedNodes
  edges.value = loadedEdges
}

// Add node from palette
function handleAddNode(template: NodeTemplate) {
  const id = generateNodeId()

  // Calculate position - center of viewport or offset from last node
  const lastNode = nodes.value[nodes.value.length - 1]
  const position = lastNode
    ? { x: lastNode.position.x + 50, y: lastNode.position.y + 50 }
    : { x: 250, y: 100 }

  const newNode: WorkflowNode = {
    id,
    type: 'custom',
    position,
    data: {
      label: template.label,
      nodeType: template.type,
      params: { ...template.defaultParams }
    }
  }

  nodes.value.push(newNode)
}

// Handle node selection
function handleNodeClick(event: any) {
  const node = findNode(event.node.id)
  if (node) {
    selectedNode.value = node as WorkflowNode
    showConfigPanel.value = true
  }
}

// Update node configuration
function handleNodeUpdate(updatedNode: WorkflowNode) {
  const index = nodes.value.findIndex(n => n.id === updatedNode.id)
  if (index !== -1) {
    // Create a new array to trigger reactivity
    nodes.value = [
      ...nodes.value.slice(0, index),
      updatedNode,
      ...nodes.value.slice(index + 1)
    ]
    selectedNode.value = updatedNode
  }
}

// Delete node
function handleNodeDelete() {
  if (selectedNode.value) {
    removeNodes([selectedNode.value.id])
    selectedNode.value = null
    showConfigPanel.value = false
  }
}

// Handle edge connection
onConnect((params) => {
  // Prevent self-connections
  if (params.source === params.target) return

  // Check for cycles
  if (wouldCreateCycle(params.source, params.target)) {
    toast.error('Cannot create cycle', {
      description: 'Workflows cannot have circular dependencies'
    })
    return
  }

  addEdges([{
    id: `${params.source}-${params.target}`,
    source: params.source!,
    target: params.target!,
    animated: true
  }])
})

// Check if adding an edge would create a cycle
function wouldCreateCycle(sourceId: string, targetId: string): boolean {
  const visited = new Set<string>()

  function dfs(nodeId: string): boolean {
    if (nodeId === sourceId) return true
    if (visited.has(nodeId)) return false

    visited.add(nodeId)

    const outgoingEdges = edges.value.filter(e => e.source === nodeId)
    for (const edge of outgoingEdges) {
      if (dfs(edge.target)) return true
    }

    return false
  }

  return dfs(targetId)
}

// Dismiss the saving toast (called from parent after save completes)
function dismissSavingToast() {
  toast.dismiss('save-workflow')
}

// Expose for parent components
defineExpose({
  dismissSavingToast
})

// Save workflow
function handleSave() {
  if (!workflowName.value.trim()) {
    toast.error('Workflow name required', {
      description: 'Please enter a name for your workflow'
    })
    return
  }

  // Show a brief "saving" toast
  toast.loading('Saving workflow...', { id: 'save-workflow' })

  emit('save', {
    name: workflowName.value,
    description: workflowDescription.value,
    nodes: nodes.value,
    edges: edges.value
  })
}

// Execute workflow
function handleExecute() {
  emit('execute')
}

// Custom node types - use markRaw to prevent Vue from making this reactive
const nodeTypes = {
  custom: markRaw(CustomNode)
}
</script>

<template>
  <div class="flex flex-col h-screen bg-background">
    <!-- Top Toolbar -->
    <div class="bg-card border-b border-border p-4 space-y-3">
      <div class="flex items-center gap-4">
        <div class="flex-1 space-y-1">
          <Input
            v-model="workflowName"
            placeholder="Untitled Workflow"
            class="text-lg font-semibold border-none shadow-none px-0 focus-visible:ring-0 h-auto"
          />
        </div>
        <div class="flex gap-2">
          <Button @click="handleSave" variant="default" size="default">
            <Save class="w-4 h-4 mr-2" />
            Save Workflow
          </Button>
          <Button @click="handleExecute" variant="outline" size="default" :disabled="!workflow">
            <Play class="w-4 h-4 mr-2" />
            Execute
          </Button>
        </div>
      </div>
      <div>
        <Textarea
          v-model="workflowDescription"
          placeholder="Add a description for this workflow..."
          rows="2"
          class="text-sm resize-none"
        />
      </div>
    </div>

    <!-- Main Content -->
    <div class="flex flex-1 overflow-hidden">
      <!-- Node Palette -->
      <NodePalette @add-node="handleAddNode" />

      <!-- Canvas -->
      <div class="flex-1 relative">
        <VueFlow
          v-model:nodes="nodes"
          v-model:edges="edges"
          :node-types="nodeTypes"
          @node-click="handleNodeClick"
          :default-viewport="{ zoom: 1 }"
          :min-zoom="0.2"
          :max-zoom="4"
          fit-view-on-init
          class="bg-muted/30"
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

          <Panel position="top-right" class="space-y-2">
            <div class="bg-card border border-border rounded-lg p-3 shadow-lg">
              <div class="text-xs font-medium mb-1">Canvas Stats</div>
              <div class="flex gap-4 text-xs text-muted-foreground">
                <div>Nodes: <span class="font-medium text-foreground">{{ nodes.length }}</span></div>
                <div>Edges: <span class="font-medium text-foreground">{{ edges.length }}</span></div>
              </div>
            </div>
          </Panel>
        </VueFlow>

        <!-- Instructions overlay when empty -->
        <div
          v-if="nodes.length === 0"
          class="absolute inset-0 flex items-center justify-center pointer-events-none z-10"
        >
          <div class="bg-card border border-border p-10 rounded-xl shadow-2xl text-center max-w-lg">
            <div class="w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center mx-auto mb-4">
              <Sparkles class="w-8 h-8 text-primary" />
            </div>
            <h3 class="text-2xl font-bold mb-3">Start Building Your Workflow</h3>
            <p class="text-muted-foreground leading-relaxed">
              Click on nodes from the left palette to add them to the canvas.
              Connect nodes by dragging from one node's output handle (green) to another's input handle (blue).
            </p>
            <div class="mt-6 flex gap-2 justify-center text-xs text-muted-foreground">
              <div class="flex items-center gap-1">
                <div class="w-3 h-3 rounded-full bg-primary"></div>
                <span>Input</span>
              </div>
              <div class="flex items-center gap-1">
                <div class="w-3 h-3 rounded-full bg-primary"></div>
                <span>Output</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Configuration Panel -->
      <div v-if="showConfigPanel" class="w-96">
        <NodeConfigPanel
          :node="selectedNode"
          @update:node="handleNodeUpdate"
          @close="showConfigPanel = false"
          @delete="handleNodeDelete"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
:deep(.vue-flow__node) {
  cursor: pointer;
}

:deep(.vue-flow__edge) {
  cursor: pointer;
  stroke: hsl(var(--primary));
  stroke-width: 2;
}

:deep(.vue-flow__edge-path) {
  stroke: hsl(var(--primary));
}

:deep(.vue-flow__edge.selected .vue-flow__edge-path) {
  stroke: hsl(var(--primary));
  stroke-width: 3;
}

:deep(.vue-flow__handle) {
  width: 12px;
  height: 12px;
}
</style>
