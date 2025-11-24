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
import { Sparkles } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

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
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// Custom node types
const nodeTypes = {
  custom: markRaw(CustomNode),
  phaseLabel: markRaw(PhaseLabelNode),
  extractField: markRaw(ExtractionFieldNode)
}

const { onConnect, addEdges, onPaneClick, onNodeClick } = useVueFlow()

// Handle interactions
onConnect((params) => {
  emit('connect', params)
})

onPaneClick(() => {
  emit('pane-click')
})

onNodeClick(({ node }) => {
  emit('node-click', node as WorkflowNode)
})

</script>

<template>
  <div class="h-full w-full bg-muted/30 relative">
    <VueFlow
      :nodes="nodes"
      :edges="edges"
      :node-types="nodeTypes"
      :default-viewport="{ zoom: 1 }"
      :min-zoom="0.2"
      :max-zoom="4"
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
      </Panel>
    </VueFlow>

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
