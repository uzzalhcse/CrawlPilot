<template>
  <div class="execution-node-tree">
    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="text-sm text-muted-foreground">Loading execution tree...</div>
    </div>

    <div v-else-if="error" class="p-4 border border-red-200 bg-red-50 rounded-md">
      <p class="text-sm text-red-600">{{ error }}</p>
    </div>

    <div v-else-if="nodeTree" class="space-y-4">
      <!-- Stats Summary -->
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div class="p-3 border rounded-md bg-muted/30">
          <div class="text-xs text-muted-foreground">Total Nodes</div>
          <div class="text-xl font-semibold">{{ nodeTree.stats.total_nodes }}</div>
        </div>
        <div class="p-3 border rounded-md bg-green-50">
          <div class="text-xs text-muted-foreground">Completed</div>
          <div class="text-xl font-semibold text-green-600">{{ nodeTree.stats.completed_nodes }}</div>
        </div>
        <div class="p-3 border rounded-md bg-red-50">
          <div class="text-xs text-muted-foreground">Failed</div>
          <div class="text-xl font-semibold text-red-600">{{ nodeTree.stats.failed_nodes }}</div>
        </div>
        <div class="p-3 border rounded-md bg-muted/30">
          <div class="text-xs text-muted-foreground">Max Depth</div>
          <div class="text-xl font-semibold">{{ nodeTree.stats.max_depth }}</div>
        </div>
      </div>

      <!-- Tree Visualization -->
      <div class="border rounded-md p-4 bg-muted/10">
        <div class="text-sm font-semibold mb-3">Execution Flow</div>
        <div class="node-tree-container">
          <NodeTreeItem
            v-for="node in nodeTree.tree"
            :key="node.id"
            :node="node"
            :depth="0"
            @select="selectNode"
          />
        </div>
      </div>

      <!-- Node Details Panel -->
      <div v-if="selectedNode" class="border rounded-md p-4 bg-muted/10">
        <div class="flex items-center justify-between mb-3">
          <div class="text-sm font-semibold">Node Details</div>
          <Button variant="ghost" size="sm" @click="selectedNode = null">
            <X class="h-4 w-4" />
          </Button>
        </div>
        <div class="space-y-2 text-sm">
          <div class="grid grid-cols-2 gap-2">
            <div><span class="text-muted-foreground">Node ID:</span> {{ selectedNode.node_id }}</div>
            <div><span class="text-muted-foreground">Type:</span> {{ selectedNode.node_type }}</div>
            <div><span class="text-muted-foreground">Status:</span> 
              <Badge :variant="getStatusVariant(selectedNode.status)">{{ selectedNode.status }}</Badge>
            </div>
            <div v-if="selectedNode.duration_ms">
              <span class="text-muted-foreground">Duration:</span> {{ selectedNode.duration_ms }}ms
            </div>
            <div><span class="text-muted-foreground">URLs Found:</span> {{ selectedNode.urls_discovered }}</div>
            <div><span class="text-muted-foreground">Items Extracted:</span> {{ selectedNode.items_extracted }}</div>
          </div>
          <div v-if="selectedNode.error_message" class="text-xs text-red-600 p-2 bg-red-50 rounded">
            {{ selectedNode.error_message }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useExecutionsStore } from '@/stores/executions'
import type { NodeTreeNode } from '@/api/executions'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { X } from 'lucide-vue-next'
import NodeTreeItem from './NodeTreeItem.vue'

const route = useRoute()
const executionsStore = useExecutionsStore()

const nodeTree = ref(executionsStore.nodeTree)
const loading = ref(false)
const error = ref<string | null>(null)
const selectedNode = ref<NodeTreeNode | null>(null)

onMounted(async () => {
  const executionId = route.params.id as string
  loading.value = true
  error.value = null
  
  try {
    const data = await executionsStore.fetchNodeTree(executionId)
    nodeTree.value = data
  } catch (e: any) {
    error.value = e.message || 'Failed to load node tree'
  } finally {
    loading.value = false
  }
})

function selectNode(node: NodeTreeNode) {
  selectedNode.value = node
}

function getStatusVariant(status: string) {
  switch (status) {
    case 'completed':
      return 'default' // Will style with green text
    case 'failed':
      return 'destructive'
    case 'running':
      return 'default'
    default:
      return 'secondary'
  }
}
</script>

<style scoped>
.node-tree-container {
  max-height: 600px;
  overflow-y: auto;
}
</style>
