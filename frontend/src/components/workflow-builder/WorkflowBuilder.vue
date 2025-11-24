<script setup lang="ts">
import { ref, watch } from 'vue'
import { useWorkflowsStore } from '@/stores/workflows'
import type { WorkflowNode, WorkflowEdge, NodeTemplate, Workflow, WorkflowConfig } from '@/types'
import NodePalette from './NodePalette.vue'
import WorkflowCanvas from './WorkflowCanvas.vue'
import PropertiesPanel from './PropertiesPanel.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import MonacoEditor from '@/components/ui/MonacoEditor.vue'
import { Save, Play, Layout, Code } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { convertNodesToWorkflowConfig } from '@/lib/workflow-utils'

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

// State
const nodes = ref<WorkflowNode[]>([])
const edges = ref<WorkflowEdge[]>([])
const selectedNode = ref<WorkflowNode | null>(null)

// Generate unique node ID
function generateNodeId(): string {
  return `node_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
}

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
        start_urls: workflowConfig.value.start_urls || props.workflow?.config?.start_urls,
        max_depth: workflowConfig.value.max_depth || props.workflow?.config?.max_depth,
        rate_limit_delay: workflowConfig.value.rate_limit_delay || props.workflow?.config?.rate_limit_delay,
        storage: workflowConfig.value.storage || props.workflow?.config?.storage,
      })
      jsonContent.value = JSON.stringify(config, null, 2)
      mode.value = 'json'
    } catch (e: any) {
      toast.error('Failed to generate JSON', { description: e.message })
    }
  } else {
    // Switch to Builder
    try {
      let parsed = JSON.parse(jsonContent.value)
      let config = parsed

      if (parsed.config && typeof parsed.config === 'object') {
        config = parsed.config
        if (parsed.name) workflowName.value = parsed.name
        if (parsed.description) workflowDescription.value = parsed.description
        if (parsed.status) workflowStatus.value = parsed.status
      }

      const tempWorkflow = {
        ...props.workflow,
        name: workflowName.value,
        description: workflowDescription.value,
        config
      } as Workflow
      
      loadWorkflow(tempWorkflow)
      mode.value = 'builder'
    } catch (e: any) {
      toast.error('Invalid JSON', { description: e.message })
    }
  }
}

// Helper function to recursively expand nodes with nested steps
function expandNode(node: any, phaseId: string, level: number = 0, parentId?: string): any[] {
  const expandedNodes: any[] = []
  
  // Add the current node
  expandedNodes.push({ 
    ...node, 
    phaseId, 
    parentId,
    level // Track nesting level for visual positioning
  })
  
  // Expand nested steps for sequence nodes
  if (node.type === 'sequence' && node.params?.steps) {
    node.params.steps.forEach((step: any, index: number) => {
      const childId = step.id || `${node.id}_step_${index}`
      const childNode = {
        id: childId,
        type: step.type,
        name: step.name || `${step.type} (${index + 1})`,
        params: step.params || {},
        optional: step.optional,
        dependencies: index === 0 ? [] : [`${node.id}_step_${index - 1}`] // Chain steps sequentially
      }
      // Recursively expand if child also has nested steps
      expandedNodes.push(...expandNode(childNode, phaseId, level + 1, node.id))
    })
  }
  
  // Expand conditional branches
  if (node.type === 'conditional') {
    let branchIndex = 0
    
    // Expand if_true branch
    if (node.params?.if_true) {
      node.params.if_true.forEach((step: any, index: number) => {
        const childId = step.id || `${node.id}_true_${index}`
        const childNode = {
          id: childId,
          type: step.type,
          name: step.name || `${step.type} (✓ true)`,
          params: step.params || {},
          branch: 'true'
        }
        expandedNodes.push(...expandNode(childNode, phaseId, level + 1, node.id))
        branchIndex++
      })
    }
    
    // Expand if_false branch
    if (node.params?.if_false) {
      node.params.if_false.forEach((step: any, index: number) => {
        const childId = step.id || `${node.id}_false_${index}`
        const childNode = {
          id: childId,
          type: step.type,
          name: step.name || `${step.type} (✗ false)`,
          params: step.params || {},
          branch: 'false'
        }
        expandedNodes.push(...expandNode(childNode, phaseId, level + 1, node.id))
        branchIndex++
      })
    }
    
    // Expand then branch (legacy support)
    if (node.params?.then) {
      const childId = node.params.then.id || `${node.id}_then`
      const childNode = {
        id: childId,
        type: node.params.then.type,
        name: node.params.then.name || node.params.then.type,
        params: node.params.then.params || {}
      }
      expandedNodes.push(...expandNode(childNode, phaseId, level + 1, node.id))
    }
  }
  
  return expandedNodes
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
    // NEW: Phase-based format with nested node expansion
    workflow.config.phases.forEach((phase: any, phaseIndex: number) => {
      if (phase.nodes) {
        phaseNodeIds.push([]) // Initialize array for this phase
        
        phase.nodes.forEach((node: any) => {
          // Recursively expand nodes (handles sequence, conditional, etc.)
          const expanded = expandNode(node, phase.id, 0)
          
          // NEW: Explode extract nodes into field nodes
          const finalNodes: any[] = []
          expanded.forEach(n => {
            finalNodes.push(n)
            
            if (n.type === 'extract' && n.params?.fields) {
              const fieldKeys = Object.keys(n.params.fields)
              fieldKeys.forEach((key, fieldIndex) => {
                const field = n.params.fields[key]
                const fieldNodeId = `${n.id}_field_${key}`
                
                finalNodes.push({
                  id: fieldNodeId,
                  type: 'extractField',
                  name: key,
                  params: { ...field }, // Copy field params
                  parentId: n.id,
                  phaseId: phase.id,
                  level: (n.level || 0) + 1,
                  isVirtual: true, // Mark as virtual so we know to implode it later
                  fieldKey: key
                })
              })
            }
          })

          allNodes = [...allNodes, ...finalNodes]
          
          // Track all node IDs in this phase
          phaseNodeIds[phaseIndex].push(...finalNodes.map((n: any) => n.id))
        })
      }
    })
  } else {
    // LEGACY: Old url_discovery/data_extraction format
    allNodes = [
      ...(workflow.config.url_discovery || []).map(n => ({ ...n, phaseId: 'discovery_phase', level: 0 })),
      ...(workflow.config.data_extraction || []).map(n => ({ ...n, phaseId: 'extraction_phase', level: 0 }))
    ]
  }

  // Create nodes with horizontal grid-based positions for cleaner hierarchy
  const nodePositions = new Map<string, { x: number; y: number }>()
  const nodeWidth = 320
  const horizontalGap = 100 // More spacing between nodes in same row
  const verticalGap = 120 // Tighter vertical spacing for better density
  const phaseGap = 250 // Optimized gap between phases
  const phaseLabelHeight = 60 // Reserved space for phase labels
  
  let currentPhaseX = 100

  // Helper to layout a set of nodes (a phase)
  const layoutPhase = (phaseNodes: any[], startX: number) => {
    // Group by level within this phase
    const phaseNodesByLevel = new Map<number, any[]>()
    let maxLevel = 0
    let maxNodesInRow = 0

    phaseNodes.forEach(node => {
      const level = node.level || 0
      if (!phaseNodesByLevel.has(level)) {
        phaseNodesByLevel.set(level, [])
      }
      phaseNodesByLevel.get(level)!.push(node)
      maxLevel = Math.max(maxLevel, level)
    })

    // Calculate width of this phase
    for (let level = 0; level <= maxLevel; level++) {
      const count = phaseNodesByLevel.get(level)?.length || 0
      maxNodesInRow = Math.max(maxNodesInRow, count)
    }
    
    const phaseWidth = Math.max(nodeWidth, (maxNodesInRow * nodeWidth) + ((maxNodesInRow - 1) * horizontalGap))
    
    // Position nodes below phase label
    let currentY = 100 + phaseLabelHeight
    
    for (let level = 0; level <= maxLevel; level++) {
      const nodesAtLevel = phaseNodesByLevel.get(level) || []
      if (nodesAtLevel.length === 0) continue

      // Center nodes in the phase width
      const rowWidth = (nodesAtLevel.length * nodeWidth) + ((nodesAtLevel.length - 1) * horizontalGap)
      const rowStartX = startX + (phaseWidth - rowWidth) / 2

      nodesAtLevel.forEach((node, index) => {
        const x = rowStartX + (index * (nodeWidth + horizontalGap))
        const y = currentY
        nodePositions.set(node.id, { x, y })
      })

      currentY += verticalGap
    }

    return phaseWidth
  }

  // Layout each phase
  if (phaseNodeIds.length > 0) {
    phaseNodeIds.forEach((ids, index) => {
      const phaseNodes = allNodes.filter(n => ids.includes(n.id))
      
      // Add Phase Label Node
      const phaseName = props.workflow?.config?.phases?.[index]?.name || `Phase ${index + 1}`
      const labelId = `phase_label_${index}`
      
      // Calculate phase width first
      const phaseWidth = layoutPhase(phaseNodes, currentPhaseX)
      
      // Position label centered above the phase with better spacing
      loadedNodes.push({
        id: labelId,
        type: 'phaseLabel',
        position: { x: currentPhaseX + (phaseWidth / 2) - 150, y: 20 }, // Better vertical position
        data: { label: phaseName, index, phaseWidth },
        draggable: false,
        selectable: false
      })

      currentPhaseX += phaseWidth + phaseGap
    })
  } else {
    // Fallback for legacy/flat lists
    layoutPhase(allNodes, currentPhaseX)
  }

  // Create WorkflowNode objects with calculated positions
  allNodes.forEach((node, index) => {
    const position = nodePositions.get(node.id) || { x: 100, y: 100 + index * 200 }

    loadedNodes.push({
      id: node.id,
      type: ['extractField', 'phaseLabel'].includes(node.type) ? node.type : 'custom',
      position,
        data: {
          label: node.name,
          nodeType: node.type as any,
          params: node.params,
          dependencies: node.dependencies,
          outputKey: (node as any).output_key,
          optional: (node as any).optional,
          retry: (node as any).retry,
          phaseId: node.phaseId,
          parentId: node.parentId,
          level: node.level,
          branch: node.branch,
          // For field nodes
          field: node.type === 'extractField' ? node.params : undefined,
          isVirtual: (node as any).isVirtual
        }
      })

      // Create edges for virtual field nodes
      if (node.type === 'extractField' && node.parentId) {
        loadedEdges.push({
          id: `${node.parentId}-${node.id}`,
          source: node.parentId,
          target: node.id,
          type: 'smoothstep',
          animated: false,
          style: { strokeDasharray: '5,5', stroke: '#94a3b8' }
        })
      }

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
    
    // Create parent-child edge for nested nodes
    if (node.parentId) {
      const edgeId = `parent_${node.parentId}-${node.id}`
      // Only add if not already added by dependencies
      if (!loadedEdges.some(e => e.id === edgeId)) {
        loadedEdges.push({
          id: edgeId,
          source: node.parentId,
          target: node.id,
          animated: false,
          style: { 
            strokeDasharray: '5,5', // Dashed line for parent-child
            stroke: node.branch === 'true' ? '#10b981' : node.branch === 'false' ? '#ef4444' : '#8b5cf6',
            strokeWidth: 2
          }
        })
      }
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
    }
  }

  nodes.value.push(newNode)
  // Auto-select the new node
  selectedNode.value = newNode
}

// Handle node selection
function handleNodeClick(node: WorkflowNode) {
  selectedNode.value = node
}

// Handle node updates from panel
function handleNodeUpdate(updatedNode: WorkflowNode) {
  console.log('🔄 [WorkflowBuilder] handleNodeUpdate called for node:', updatedNode.id, updatedNode.data.nodeType)
  
  const index = nodes.value.findIndex(n => n.id === updatedNode.id)
  if (index !== -1) {
    // Special handling for extract nodes - regenerate extractField nodes
    if (updatedNode.data.nodeType === 'extract') {
      console.log('🔄 [Extract Node] Updating extract node and regenerating field nodes:', updatedNode.id)
      
      const newFields = updatedNode.data.params?.fields
      
      // Always regenerate extractField nodes to ensure they are in sync
      // First, capture positions of existing field nodes to preserve them
      const existingFieldPositions = new Map<string, { x: number, y: number }>()
      nodes.value.forEach(n => {
        if (n.data.nodeType === 'extractField' && 
            (n.data.parentId === updatedNode.id || n.data.params?.parentId === updatedNode.id)) {
          existingFieldPositions.set(n.data.params?.fieldKey || n.data.label, n.position)
        }
      })
      
      // Remove old extractField nodes for this extract node
      nodes.value = nodes.value.filter(n => {
        const isExtractFieldNode = n.data.nodeType === 'extractField' || n.type === 'extractField'
        if (!isExtractFieldNode) return true
        
        const nodeParentId = n.data.parentId || n.data.params?.parentId
        return nodeParentId !== updatedNode.id
      })

      // Also remove old edges connected to these field nodes
      // We'll regenerate them to ensure everything is clean
      // Note: We only remove edges where the source is the parent extract node and target is a field node
      // This might be too aggressive if users manually connected things, but for virtual nodes it's safer
      edges.value = edges.value.filter(e => e.source !== updatedNode.id || !e.target.includes('_field_'))
      
      // Generate new extractField nodes and edges
      const newExtractFieldNodes: WorkflowNode[] = []
      const newEdges: any[] = []

      if (newFields) {
        const fieldKeys = Object.keys(newFields)
        fieldKeys.forEach((key, index) => {
          const field = newFields[key]
          const fieldNodeId = `${updatedNode.id}_field_${key}`
          
          // Use existing position if available, otherwise default layout
          // Default layout: stack them vertically to the right of the parent
          // Improved layout: Stagger them slightly to avoid complete overlap if many
          const defaultX = (updatedNode.position?.x || 0) + 350
          const defaultY = (updatedNode.position?.y || 0) + (index * 120) - ((fieldKeys.length * 120) / 2) + 60
          
          const position = existingFieldPositions.get(key) || { x: defaultX, y: defaultY }
          
          newExtractFieldNodes.push({
            id: fieldNodeId,
            type: 'extractField',
            position: position,
            data: {
              label: key,
              nodeType: 'extractField',
              field: {
                selector: field.selector || '',
                type: field.type || 'text',
                attribute: field.attribute || '',
                multiple: field.multiple || false,
                transform: field.transform || 'none',
                default_value: field.default_value || ''
              },
              params: {
                ...field,
                parentId: updatedNode.id,
                fieldKey: key
              },
              parentId: updatedNode.id,
              isVirtual: true
            }
          })

          // Create edge from parent to field node
          newEdges.push({
            id: `${updatedNode.id}-${fieldNodeId}`,
            source: updatedNode.id,
            target: fieldNodeId,
            animated: true,
            style: { stroke: '#a855f7' } // Purple edge to match extract theme
          })
        })
      }
      
      console.log(`  Regenerated ${newExtractFieldNodes.length} extractField nodes and edges`)
      
      // Update the extract node and add new extractField nodes
      nodes.value = [
        ...nodes.value.slice(0, index),
        updatedNode,
        ...nodes.value.slice(index + 1),
        ...newExtractFieldNodes
      ]

      // Add new edges
      edges.value = [...edges.value, ...newEdges]

    } else {
      // Regular node update
      nodes.value = [
        ...nodes.value.slice(0, index),
        updatedNode,
        ...nodes.value.slice(index + 1)
      ]
    }
    
    selectedNode.value = updatedNode
  }
}


// Delete node
function handleNodeDelete() {
  if (selectedNode.value) {
    // Remove the node from the array
    nodes.value = nodes.value.filter(n => n.id !== selectedNode.value!.id)
    // Also remove any connected edges
    edges.value = edges.value.filter(e => e.source !== selectedNode.value!.id && e.target !== selectedNode.value!.id)
    
    selectedNode.value = null
  }
}

// Handle edge connection
function handleConnect(params: any) {
  // Prevent self-connections
  if (params.source === params.target) return

  // Check for cycles (simple DFS)
  const wouldCreateCycle = (source: string, target: string) => {
    const visited = new Set<string>()
    const dfs = (nodeId: string): boolean => {
      if (nodeId === source) return true
      if (visited.has(nodeId)) return false
      visited.add(nodeId)
      const outgoing = edges.value.filter(e => e.source === nodeId)
      return outgoing.some(e => dfs(e.target))
    }
    return dfs(target)
  }

  if (wouldCreateCycle(params.source, params.target)) {
    toast.error('Cannot create cycle', { description: 'Workflows cannot have circular dependencies' })
    return
  }

  edges.value.push({
    id: `${params.source}-${params.target}`,
    source: params.source!,
    target: params.target!,
    animated: true
  })
}

// Save workflow
function handleSave() {
  if (!workflowName.value.trim()) {
    toast.error('Workflow name required')
    return
  }

  toast.loading('Saving workflow...', { id: 'save-workflow' })

  let config: WorkflowConfig | undefined
  
  if (mode.value === 'json') {
    try {
      const parsed = JSON.parse(jsonContent.value)
      if (parsed.config && typeof parsed.config === 'object') {
        config = parsed.config
      } else {
        config = parsed
      }
    } catch (e) {
      toast.error('Invalid JSON')
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
    config
  })
}

// Execute workflow
function handleExecute() {
  emit('execute')
}

// Handle status toggle
async function handleToggleStatus() {
  const newStatus = workflowStatus.value === 'active' ? 'draft' : 'active'
  
  if (!props.workflow?.id) {
    workflowStatus.value = newStatus
    return
  }

  try {
    toast.loading(`Updating status to ${newStatus}...`, { id: 'update-status' })
    
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
    toast.error('Failed to update status')
  }
}

// Dismiss the saving toast (called from parent after save completes)
function dismissSavingToast() {
  toast.dismiss('save-workflow')
}

defineExpose({
  dismissSavingToast
})
</script>

<template>
  <div class="flex flex-col h-screen bg-background">
    <!-- Top Toolbar -->
    <div class="bg-card border-b border-border p-4 space-y-3 shrink-0 z-30">
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
        <div class="flex-1 relative h-full">
          <WorkflowCanvas
            :nodes="nodes"
            :edges="edges"
            @update:nodes="nodes = $event"
            @update:edges="edges = $event"
            @node-click="handleNodeClick"
            @connect="handleConnect"
            @pane-click="selectedNode = null"
          />
        </div>

        <!-- Properties Panel -->
        <PropertiesPanel
          :node="selectedNode"
          @update="handleNodeUpdate"
          @delete="handleNodeDelete"
          @close="selectedNode = null"
        />
      </template>

      <!-- JSON MODE -->
      <template v-else>
        <div class="flex-1 p-6 bg-muted/30">
          <div class="h-full border rounded-lg overflow-hidden bg-card shadow-sm">
            <monaco-editor
              v-model="jsonContent"
              language="json"
              theme="vs-dark"
            />
          </div>
        </div>
      </template>

    </div>
  </div>
</template>
