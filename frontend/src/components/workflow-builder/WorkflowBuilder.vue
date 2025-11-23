<script setup lang="ts">
import { ref, watch, markRaw } from 'vue'
import { VueFlow, useVueFlow, Panel } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import { useWorkflowsStore } from '@/stores/workflows'
import type { WorkflowNode, WorkflowEdge, NodeTemplate, Workflow, WorkflowConfig } from '@/types'
import CustomNode from './CustomNode.vue'
import NodePalette from './NodePalette.vue'
import NodeConfigPanel from './NodeConfigPanel.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import MonacoEditor from '@/components/ui/MonacoEditor.vue'
import { Save, Play, Sparkles, Code, Layout } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { convertNodesToWorkflowConfig } from '@/lib/workflow-utils'

// Import Vue Flow styles
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'

interface Props {
  workflow?: Workflow | null
}

interface Emits {
  (e: 'save', data: { name: string; description: string; status: 'draft' | 'active'; nodes: WorkflowNode[]; edges: WorkflowEdge[]; config?: WorkflowConfig }): void
  (e: 'execute'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const workflowsStore = useWorkflowsStore()

// Workflow metadata
const workflowName = ref('')
const workflowDescription = ref('')
const workflowStatus = ref<'draft' | 'active'>('draft')
const workflowConfig = ref<Partial<WorkflowConfig>>({})

// Mode state
const mode = ref<'builder' | 'json'>('builder')
const jsonContent = ref('')
const jsonError = ref('')

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

// Toggle Mode
function toggleMode() {
  if (mode.value === 'builder') {
    // Switch to JSON
    try {
      const config = convertNodesToWorkflowConfig(nodes.value, edges.value, {
        // Preserve existing config if available, or defaults
        start_urls: workflowConfig.value.start_urls || props.workflow?.config?.start_urls,
        max_depth: workflowConfig.value.max_depth || props.workflow?.config?.max_depth,
        rate_limit_delay: workflowConfig.value.rate_limit_delay || props.workflow?.config?.rate_limit_delay,
        storage: workflowConfig.value.storage || props.workflow?.config?.storage,
      })
      jsonContent.value = JSON.stringify(config, null, 2)
      mode.value = 'json'
      jsonError.value = ''
    } catch (e: any) {
      toast.error('Failed to generate JSON', { description: e.message })
    }
  } else {
    // Switch to Builder
    try {
      let parsed = JSON.parse(jsonContent.value)
      let config = parsed

      // Handle case where user pastes full workflow object
      if (parsed.config && typeof parsed.config === 'object') {
        config = parsed.config
        if (parsed.name) workflowName.value = parsed.name
        if (parsed.description) workflowDescription.value = parsed.description
        if (parsed.status) workflowStatus.value = parsed.status
      }

      // Create a temporary workflow object to load
      // We need to mock the Workflow structure as loadWorkflow expects it
      const tempWorkflow = {
        ...props.workflow,
        name: workflowName.value,
        description: workflowDescription.value,
        config
      } as Workflow
      
      loadWorkflow(tempWorkflow)
      mode.value = 'builder'
      jsonError.value = ''
    } catch (e: any) {
      jsonError.value = e.message
      toast.error('Invalid JSON', { description: e.message })
    }
  }
}

// Load workflow into the builder
function loadWorkflow(workflow: Workflow) {
  workflowName.value = workflow.name
  workflowDescription.value = workflow.description
  workflowStatus.value = (workflow.status as 'draft' | 'active') || 'draft'
  
  // Store the full config to preserve non-node settings (start_urls, etc.)
  workflowConfig.value = { ...workflow.config }

  // Convert workflow config to nodes and edges
  const loadedNodes: WorkflowNode[] = []
  const loadedEdges: WorkflowEdge[] = []

  let allNodes: any[] = []
  let phaseNodeIds: string[][] = [] // Track which nodes belong to which phase
  
  // Support both phase-based and legacy formats
  if (workflow.config.phases && workflow.config.phases.length > 0) {
    // NEW: Phase-based format
    workflow.config.phases.forEach((phase: any) => {
      if (phase.nodes) {
        const phaseNodes = phase.nodes.map((n: any) => {
          // Add phaseId to node data so we can preserve it
          return { ...n, phaseId: phase.id }
        })
        phaseNodeIds.push(phaseNodes.map((n: any) => n.id))
        allNodes = [...allNodes, ...phaseNodes]
      }
    })
  } else {
    // LEGACY: Old url_discovery/data_extraction format
    allNodes = [
      ...(workflow.config.url_discovery || []).map(n => ({ ...n, phaseId: 'discovery_phase' })),
      ...(workflow.config.data_extraction || []).map(n => ({ ...n, phaseId: 'extraction_phase' }))
    ]
  }

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
        retry: (node as any).retry,
        phaseId: node.phaseId // Preserve phase ID
      }
    })

    // Create edges based on dependencies (if they exist)
    if (node.dependencies && node.dependencies.length > 0) {
      node.dependencies.forEach((depId: string) => {
        loadedEdges.push({
          id: `${depId}-${node.id}`,
          source: depId,
          target: node.id,
          animated: true
        })
      })
    }
  })

  // For phase-based workflows: Create visual connections between phases
  if (phaseNodeIds.length > 1) {
    for (let i = 0; i < phaseNodeIds.length - 1; i++) {
      const currentPhaseNodes = phaseNodeIds[i]
      const nextPhaseNodes = phaseNodeIds[i + 1]
      
      // Connect last node of current phase to first node of next phase
      if (currentPhaseNodes.length > 0 && nextPhaseNodes.length > 0) {
        const sourceId = currentPhaseNodes[currentPhaseNodes.length - 1]
        const targetId = nextPhaseNodes[0]
        
        // Only add if not already connected via dependencies
        const edgeExists = loadedEdges.some(e => e.source === sourceId && e.target === targetId)
        if (!edgeExists) {
          loadedEdges.push({
            id: `phase_${i}_to_${i + 1}`,
            source: sourceId,
            target: targetId,
            animated: true
          })
        }
      }
    }
  }

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
      // phaseId will be assigned by heuristic on save/convert if not set
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

  let config: WorkflowConfig | undefined
  
  if (mode.value === 'json') {
    try {
      const parsed = JSON.parse(jsonContent.value)
      // Handle case where user pastes full workflow object
      if (parsed.config && typeof parsed.config === 'object') {
        config = parsed.config
        // Note: We use the name/description from the inputs, not the JSON, 
        // unless the user switched modes which would have synced them.
      } else {
        config = parsed
      }
    } catch (e) {
      toast.error('Invalid JSON', { description: 'Please fix JSON errors before saving' })
      return
    }
  } else {
    config = convertNodesToWorkflowConfig(nodes.value, edges.value, {
      start_urls: workflowConfig.value.start_urls || props.workflow?.config?.start_urls,
      max_depth: workflowConfig.value.max_depth || props.workflow?.config?.max_depth,
      rate_limit_delay: workflowConfig.value.rate_limit_delay || props.workflow?.config?.rate_limit_delay,
      storage: workflowConfig.value.storage || props.workflow?.config?.storage,
    })
  }

  emit('save', {
    name: workflowName.value,
    description: workflowDescription.value,
    status: workflowStatus.value,
    nodes: nodes.value,
    edges: edges.value,
    config // Pass the generated config
  })
}

// Execute workflow
function handleExecute() {
  emit('execute')
}

// Handle status toggle
async function handleToggleStatus() {
  const newStatus = workflowStatus.value === 'active' ? 'draft' : 'active'
  
  // If it's a new workflow (no ID), just toggle local state
  if (!props.workflow?.id) {
    workflowStatus.value = newStatus
    return
  }

  // If existing workflow, save and update status
  try {
    toast.loading(`Updating status to ${newStatus}...`, { id: 'update-status' })
    
    // Generate config
    const config = convertNodesToWorkflowConfig(nodes.value, edges.value, {
      start_urls: workflowConfig.value.start_urls || props.workflow.config.start_urls,
      max_depth: workflowConfig.value.max_depth || props.workflow.config.max_depth,
      rate_limit_delay: workflowConfig.value.rate_limit_delay || props.workflow.config.rate_limit_delay,
      storage: workflowConfig.value.storage || props.workflow.config.storage,
    })

    await workflowsStore.updateWorkflow(props.workflow.id, {
      name: workflowName.value,
      description: workflowDescription.value,
      status: newStatus,
      config
    })

    workflowStatus.value = newStatus
    toast.dismiss('update-status')
    toast.success(`Workflow ${newStatus === 'active' ? 'published' : 'set to draft'}`)
  } catch (e: any) {
    toast.dismiss('update-status')
    toast.error('Failed to update status', {
      description: e.message || 'Unknown error'
    })
  }
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
          <!-- Mode Toggle -->
          <div class="flex items-center bg-muted rounded-lg p-1 mr-2">
            <Button 
              variant="ghost" 
              size="sm" 
              :class="{ 'bg-background shadow-sm': mode === 'builder' }"
              @click="mode !== 'builder' && toggleMode()"
            >
              <Layout class="w-4 h-4 mr-2" />
              Builder
            </Button>
            <Button 
              variant="ghost" 
              size="sm" 
              :class="{ 'bg-background shadow-sm': mode === 'json' }"
              @click="mode !== 'json' && toggleMode()"
            >
              <Code class="w-4 h-4 mr-2" />
              JSON
            </Button>
          </div>

          <!-- Status Toggle -->
          <Button 
            variant="outline" 
            size="default"
            :class="workflowStatus === 'active' 
              ? 'bg-green-100 text-green-900 border-green-300 hover:bg-green-200 hover:text-green-950' 
              : 'bg-amber-100 text-amber-900 border-amber-300 hover:bg-amber-200 hover:text-amber-950'"
            @click="handleToggleStatus"
          >
            <div class="w-2 h-2 rounded-full mr-2" :class="workflowStatus === 'active' ? 'bg-green-600' : 'bg-amber-600'"></div>
            {{ workflowStatus === 'active' ? 'Published' : 'Draft' }}
          </Button>

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
      
      <!-- BUILDER MODE -->
      <template v-if="mode === 'builder'">
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
      </template>

      <!-- JSON MODE -->
      <template v-else>
        <div class="flex-1 p-4 bg-muted/10 flex flex-col">
          <div class="flex-1 relative">
            <MonacoEditor
              v-model="jsonContent"
              language="json"
              theme="vs-dark"
              class="w-full h-full border border-border rounded-lg"
            />
            <div v-if="jsonError" class="absolute bottom-4 left-4 right-4 bg-destructive/10 text-destructive p-3 rounded border border-destructive/20 text-sm z-10">
              {{ jsonError }}
            </div>
          </div>
        </div>
      </template>

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
