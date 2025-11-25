<script setup lang="ts">
import { ref, watch, markRaw, onMounted } from 'vue'
import { VueFlow, useVueFlow, type Edge, type Node, Panel } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import ExecutionNode from './ExecutionNode.vue'
import NodeDetailPanel from './NodeDetailPanel.vue'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { WorkflowConfig, WorkflowNode, WorkflowEdge, NodeType } from '@/types'

// Import Vue Flow styles
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'

const props = defineProps<{
  workflowConfig?: WorkflowConfig
  activeNodes?: Set<string>
  nodeStatuses?: Map<string, 'pending' | 'running' | 'completed' | 'failed'>
  executionTree?: any[]
}>()

console.log('[GRAPH] Component mounted. executionTree:', props.executionTree) // DEBUG
console.log('[GRAPH] workflowConfig:', props.workflowConfig) // DEBUG

const { fitView } = useVueFlow()


const nodes = ref<Node[]>([])
const edges = ref<Edge[]>([])

// Selected node for detail panel
const selectedNode = ref<Node | null>(null)
const showDetailPanel = ref(false)

// Configurable max nodes per level
const maxNodesPerLevel = ref(20)

// Custom node types
const nodeTypes: any = {
  custom: markRaw(ExecutionNode)
}

// Initialize graph from config
const initGraph = () => {
  if (!props.workflowConfig) return

  // Reuse logic from WorkflowBuilder to layout nodes
  // We need to reconstruct the graph from the config
  // This is a simplified version of what's in WorkflowBuilder
  
  const loadedNodes: WorkflowNode[] = []
  const loadedEdges: WorkflowEdge[] = []
  
  // Helper to expand nodes (same as builder)
  const expandNode = (node: any, phaseId: string, level: number = 0, parentId?: string): any[] => {
    const expandedNodes: any[] = []
    expandedNodes.push({ ...node, phaseId, parentId, level })
    
    if (node.type === 'sequence' && node.params?.steps) {
      node.params.steps.forEach((step: any, index: number) => {
        const childId = step.id || `${node.id}_step_${index}`
        const childNode = {
          id: childId,
          type: step.type,
          name: step.name || `${step.type} (${index + 1})`,
          params: step.params || {},
          optional: step.optional,
          dependencies: index === 0 ? [] : [`${node.id}_step_${index - 1}`]
        }
        expandedNodes.push(...expandNode(childNode, phaseId, level + 1, node.id))
      })
    }
    
    // Handle other nested types (conditional, loop) similarly if needed
    // For now, basic sequence support covers most cases
    
    return expandedNodes
  }

  let allNodes: any[] = []
  let phaseNodeIds: string[][] = []

  if (props.workflowConfig.phases) {
    props.workflowConfig.phases.forEach((phase: any, phaseIndex: number) => {
      if (phase.nodes) {
        phaseNodeIds.push([])
        phase.nodes.forEach((node: any) => {
          const expanded = expandNode(node, phase.id, 0)
          allNodes = [...allNodes, ...expanded]
          phaseNodeIds[phaseIndex].push(...expanded.map((n: any) => n.id))
        })
      }
    })
  } else {
    // Legacy support
    allNodes = [
      ...(props.workflowConfig.url_discovery || []).map((n: any) => ({ ...n, phaseId: 'discovery' })),
      ...(props.workflowConfig.data_extraction || []).map((n: any) => ({ ...n, phaseId: 'extraction' }))
    ]
  }

  // Layout logic (simplified)
  const nodeWidth = 280
  const horizontalGap = 100
  const levelVerticalGap = 200
  
  const nodesByLevel = new Map<number, any[]>()
  allNodes.forEach(node => {
    const level = node.level || 0
    if (!nodesByLevel.has(level)) nodesByLevel.set(level, [])
    nodesByLevel.get(level)!.push(node)
  })

  let currentY = 100
  const nodePositions = new Map<string, { x: number, y: number }>()

  // Position nodes
  const maxLevel = Math.max(...Array.from(nodesByLevel.keys()), 0)
  for (let level = 0; level <= maxLevel; level++) {
    const levelNodes = nodesByLevel.get(level) || []
    if (levelNodes.length === 0) continue

    const totalWidth = (levelNodes.length * nodeWidth) + ((levelNodes.length - 1) * horizontalGap)
    const startX = Math.max(100, (1200 - totalWidth) / 2)

    levelNodes.forEach((node, index) => {
      nodePositions.set(node.id, {
        x: startX + (index * (nodeWidth + horizontalGap)),
        y: currentY
      })
    })
    currentY += levelVerticalGap
  }

  // Create nodes
  allNodes.forEach(node => {
    const position = nodePositions.get(node.id) || { x: 0, y: 0 }
    
    loadedNodes.push({
      id: node.id,
      type: 'custom',
      position,
      data: {
        label: node.name,
        nodeType: node.type,
        params: node.params,
        status: 'pending' // Initial status
      }
    })

    // Create edges
    if (node.dependencies) {
      node.dependencies.forEach((depId: string) => {
        loadedEdges.push({
          id: `${depId}-${node.id}`,
          source: depId,
          target: node.id,
          animated: true,
          style: { stroke: '#94a3b8' }
        })
      })
    }
    
    if (node.parentId) {
      loadedEdges.push({
        id: `parent_${node.parentId}-${node.id}`,
        source: node.parentId,
        target: node.id,
        animated: false,
        style: { strokeDasharray: '5,5', stroke: '#cbd5e1' }
      })
    }
  })

  // Phase connections
  if (phaseNodeIds.length > 1) {
    for (let i = 0; i < phaseNodeIds.length - 1; i++) {
      const current = phaseNodeIds[i]
      const next = phaseNodeIds[i+1]
      if (current.length && next.length) {
        const source = current[current.length - 1]
        const target = next[0]
        if (!loadedEdges.some(e => e.source === source && e.target === target)) {
          loadedEdges.push({
            id: `phase_${i}_${i+1}`,
            source,
            target,
            animated: true,
            style: { stroke: '#94a3b8' }
          })
        }
      }
    }
  }

  nodes.value = loadedNodes
  edges.value = loadedEdges
  
  setTimeout(() => fitView(), 100)
}

// Build graph from execution tree (dynamic, live data)
const buildGraphFromExecutionTree = () => {
  console.log('[GRAPH] buildGraphFromExecutionTree called with tree:', props.executionTree) // DEBUG
  if (!props.executionTree || props.executionTree.length === 0) {
    console.warn('[GRAPH] No execution tree data, skipping') // DEBUG
    return
  }

  const loadedNodes: WorkflowNode[] = []
  const loadedEdges: WorkflowEdge[] = []
  const nodePositions = new Map<string, { x: number, y: number }>()
  const nodesByLevel = new Map<number, any[]>()

  // Flatten tree and assign depths
  const flattenTree = (nodes: any[], depth: number = 0) => {
    nodes.forEach(node => {
      const nodeWithDepth = { ...node, depth }
      
      if (!nodesByLevel.has(depth)) {
        nodesByLevel.set(depth, [])
      }
      nodesByLevel.get(depth)!.push(nodeWithDepth)
      
      if (node.children && node.children.length > 0) {
        flattenTree(node.children, depth + 1)
      }
    })
  }

  flattenTree(props.executionTree)
  console.log('[GRAPH] Building dynamic graph from', props.executionTree.length, 'root nodes')
  
  // Calculate positions by level (centered)
  const nodeWidth = 250
  const nodeHeight = 150
  const horizontalGap = 20
  
  nodesByLevel.forEach((nodes, depth) => {
    const nodesToShow = nodes.slice(0, maxNodesPerLevel.value)
    const hiddenCount = nodes.length - maxNodesPerLevel.value
    
    // Calculate total width needed for this level
    const totalNodes = nodesToShow.length + (hiddenCount > 0 ? 1 : 0) // +1 for overflow node
    const totalWidth = (totalNodes * nodeWidth) + ((totalNodes - 1) * horizontalGap)
    
    // Start from center
    const startX = -totalWidth / 2
    
    nodesToShow.forEach((execution, index) => {
      const x = startX + (index * (nodeWidth + horizontalGap)) + (nodeWidth / 2)
      const y = depth * nodeHeight
      nodePositions.set(execution.node_execution_id, { x, y })
    })
    
    // If there are hidden nodes, add overflow summary node position
    if (hiddenCount > 0) {
      const x = startX + (nodesToShow.length * (nodeWidth + horizontalGap)) + (nodeWidth / 2)
      const y = depth * nodeHeight
      nodePositions.set(`overflow-level-${depth}`, { x, y })
    }
  })

  // Create Vue Flow nodes (limited per level)
  nodesByLevel.forEach((nodes, depth) => {
    const nodesToShow = nodes.slice(0, maxNodesPerLevel.value)
    const hiddenCount = nodes.length - maxNodesPerLevel.value
    
    nodesToShow.forEach(execution => {
      const position = nodePositions.get(execution.node_execution_id) || { x: 0, y: 0 }
      
      loadedNodes.push({
        id: execution.node_execution_id,
        type: 'custom',
        position,
        data: {
          label: execution.node_id,
          nodeType: execution.node_type || 'unknown',
          params: {},
          status: execution.status,
          error: execution.error,
          result: execution.result
        },
        draggable: false,
        connectable: false,
        selectable: true
      })

      // Create edge from parent (only if parent is shown)
      if (execution.parent_node_execution_id) {
        const parentDepth = execution.depth - 1
        const parentLevelNodes = nodesByLevel.get(parentDepth) || []
        const parentIndex = parentLevelNodes.findIndex((n: any) => n.node_execution_id === execution.parent_node_execution_id)
        
        // Only create edge if parent is in the visible nodes (first 20)
        if (parentIndex < maxNodesPerLevel.value) {
          loadedEdges.push({
            id: `${execution.parent_node_execution_id}-${execution.node_execution_id}`,
            source: execution.parent_node_execution_id,
            target: execution.node_execution_id,
            animated: execution.status === 'running',
            style: { 
              stroke: execution.status === 'completed' ? '#22c55e' : 
                      execution.status === 'failed' ? '#ef4444' : '#94a3b8' 
            }
          })
        }
      }
    })
    
    // Add overflow summary node if needed
    if (hiddenCount > 0) {
      const position = nodePositions.get(`overflow-level-${depth}`)!
      const hiddenNodes = nodes.slice(maxNodesPerLevel.value)
      const completedCount = hiddenNodes.filter((n: any) => n.status === 'completed').length
      const failedCount = hiddenNodes.filter((n: any) => n.status === 'failed').length
      
      loadedNodes.push({
        id: `overflow-level-${depth}`,
        type: 'custom',
        position,
        data: {
          label: `... and ${hiddenCount} more`,
          nodeType: 'filter', // Use a valid type
          params: {},
          status: 'completed', // Use valid status
          result: {
            total: hiddenCount,
            completed: completedCount,
            failed: failedCount,
            hiddenNodes: hiddenNodes // Store for potential expansion
          }
        },
        draggable: false,
        connectable: false,
        selectable: true
      })
    }
  })

  nodes.value = loadedNodes
  edges.value = loadedEdges
  
  console.log('[GRAPH] Created', loadedNodes.length, 'nodes and', loadedEdges.length, 'edges')
  setTimeout(() => fitView(), 100)
}

// Watch for status updates
watch(() => props.nodeStatuses, (newStatuses) => {
  nodes.value = nodes.value.map(node => {
    const status = newStatuses.get(node.id)
    if (status) {
      // Update node data with status info
      return {
        ...node,
        data: {
          ...node.data,
          ...status, // Merge status, result, error, logs
          status: status.status
        }
      }
    }
    return node
  })
  
  // Update selected node if it's open
  if (selectedNode.value && showDetailPanel.value) {
    const updated = nodes.value.find(n => n.id === selectedNode.value.id)
    if (updated) {
      selectedNode.value = updated
    }
  }
}, { deep: true })

const onNodeClick = (event: any) => {
  selectedNode.value = event.node
  showDetailPanel.value = true
}

onMounted(() => {
  console.log('[GRAPH] Component mounted. executionTree:', props.executionTree) // DEBUG
  console.log('[GRAPH] workflowConfig:', props.workflowConfig) // DEBUG
  // Try execution tree first, fallback to workflow config
  if (props.executionTree && props.executionTree.length > 0) {
    console.log('[GRAPH] Using execution tree') // DEBUG
    buildGraphFromExecutionTree()
  } else {
    console.log('[GRAPH] Using static workflow config') // DEBUG
    initGraph()
  }
})

// Watch for execution tree updates (live mode)
watch(() => props.executionTree, (newTree) => {
  console.log('[GRAPH] executionTree changed:', newTree) // DEBUG
  if (newTree && newTree.length > 0) {
    console.log('[GRAPH] Rebuilding graph from updated tree') // DEBUG
    buildGraphFromExecutionTree()
  }
}, { deep: true })

// Watch for max nodes per level changes
watch(maxNodesPerLevel, () => {
  console.log('[GRAPH] maxNodesPerLevel changed, rebuilding graph')
  if (props.executionTree && props.executionTree.length > 0) {
    buildGraphFromExecutionTree()
  }
})
</script>

<template>
  <div class="relative w-full h-[600px] border rounded-lg bg-muted/10 overflow-hidden">
    <VueFlow
      v-model:nodes="nodes"
      v-model:edges="edges"
      :node-types="nodeTypes"
      @node-click="onNodeClick"
      :default-viewport="{ zoom: 1 }"
      :min-zoom="0.2"
      :max-zoom="4"
      fit-view-on-init
    >
      <Background pattern-color="#cbd5e1" :gap="20" />
      <Controls />
      <MiniMap />
      
      <Panel position="top-left">
        <div class="bg-card/90 backdrop-blur p-3 rounded border shadow-sm text-sm">
          <label class="text-xs font-medium mb-2 block">Max Nodes Per Level</label>
          <Select v-model="maxNodesPerLevel">
            <SelectTrigger class="w-[140px] h-8 text-xs">
              <SelectValue placeholder="Select limit" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem :value="10">10 nodes</SelectItem>
              <SelectItem :value="20">20 nodes</SelectItem>
              <SelectItem :value="50">50 nodes</SelectItem>
              <SelectItem :value="100">100 nodes</SelectItem>
              <SelectItem :value="999">All nodes</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </Panel>
      
      <Panel position="top-right">
        <div class="bg-card/90 backdrop-blur p-2 rounded border shadow-sm text-xs space-y-1">
          <div class="flex items-center gap-2">
            <div class="w-2 h-2 rounded-full bg-blue-500 animate-pulse"></div>
            <span>Running</span>
          </div>
          <div class="flex items-center gap-2">
            <div class="w-2 h-2 rounded-full bg-green-500"></div>
            <span>Completed</span>
          </div>
          <div class="flex items-center gap-2">
            <div class="w-2 h-2 rounded-full bg-red-500"></div>
            <span>Failed</span>
          </div>
        </div>
      </Panel>
    </VueFlow>

    <NodeDetailPanel 
      :node="selectedNode" 
      :open="showDetailPanel" 
      @close="showDetailPanel = false" 
    />
  </div>
</template>
