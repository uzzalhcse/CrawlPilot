<template>
  <div class="node-tree-item" :style="{ marginLeft: `${depth * 20}px` }">
    <div 
      class="node-item flex items-center gap-2 p-2 rounded hover:bg-muted/50 cursor-pointer transition-colors"
      @click="$emit('select', node)"
    >
      <!-- Expand/Collapse Icon -->
      <button 
        v-if="node.children && node.children.length > 0"
        @click.stop="toggleExpanded"
        class="flex-shrink-0"
      >
        <ChevronRight v-if="!expanded" class="h-4 w-4 text-muted-foreground" />
        <ChevronDown v-else class="h-4 w-4 text-muted-foreground" />
      </button>
      <div v-else class="w-4"></div>

      <!-- Status Icon -->
      <div class="flex-shrink-0">
        <CheckCircle v-if="node.status === 'completed'" class="h-4 w-4 text-green-600" />
        <XCircle v-else-if="node.status === 'failed'" class="h-4 w-4 text-red-600" />
        <Clock v-else-if="node.status === 'running'" class="h-4 w-4 text-blue-600 animate-spin" />
        <Circle v-else class="h-4 w-4 text-muted-foreground" />
      </div>

      <!-- Node Info -->
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2">
          <span class="text-sm font-medium truncate">{{ node.node_id }}</span>
          <Badge variant="outline" class="text-[10px] px-1 py-0">{{ node.node_type }}</Badge>
        </div>
        <div class="text-xs text-muted-foreground">
          <span v-if="node.duration_ms">{{ node.duration_ms }}ms</span>
          <span v-if="node.urls_discovered > 0" class="ml-2">→ {{ node.urls_discovered }} URLs</span>
          <span v-if="node.items_extracted > 0" class="ml-2">📦 {{ node.items_extracted }} items</span>
        </div>
      </div>
    </div>

    <!-- Children (recursive) -->
    <div v-if="expanded &&  node.children && node.children.length > 0" class="mt-1">
      <NodeTreeItem
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :depth="depth + 1"
        @select="$emit('select', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { NodeTreeNode } from '@/api/executions'
import { Badge } from '@/components/ui/badge'
import { ChevronRight, ChevronDown, CheckCircle, XCircle, Clock, Circle } from 'lucide-vue-next'

defineProps<{
  node: NodeTreeNode
  depth: number
}>()

defineEmits<{
  select: [node: NodeTreeNode]
}>()

const expanded = ref(true)

function toggleExpanded() {
  expanded.value = !expanded.value
}
</script>

<style scoped>
.node-tree-item {
  position: relative;
}

.node-item {
  border-left: 2px solid transparent;
}

.node-item:hover {
  border-left-color: hsl(var(--primary));
}
</style>
