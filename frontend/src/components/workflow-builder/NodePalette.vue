<script setup lang="ts">
import { ref, computed } from 'vue'
import { nodeCategories } from '@/config/nodeTemplates'
import type { NodeTemplate } from '@/types'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger
} from '@/components/ui/accordion'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Globe, Database, MousePointer, RefreshCw, GitBranch, Search, Sparkles } from 'lucide-vue-next'

interface Emits {
  (e: 'add-node', template: NodeTemplate): void
}

const emit = defineEmits<Emits>()

const searchQuery = ref('')

const categoryIcons: Record<string, any> = {
  'URL Discovery': Globe,
  'Extraction': Database,
  'Interaction': MousePointer,
  'Transformation': Sparkles,
  'Control Flow': GitBranch
}

const categoryColors: Record<string, string> = {
  'URL Discovery': 'text-blue-600 dark:text-blue-400',
  'Extraction': 'text-emerald-600 dark:text-emerald-400',
  'Interaction': 'text-purple-600 dark:text-purple-400',
  'Transformation': 'text-amber-600 dark:text-amber-400',
  'Control Flow': 'text-pink-600 dark:text-pink-400'
}

const filteredCategories = computed(() => {
  if (!searchQuery.value.trim()) {
    return nodeCategories
  }

  const query = searchQuery.value.toLowerCase()
  return nodeCategories.map(category => ({
    ...category,
    nodes: category.nodes.filter(node =>
      node.label.toLowerCase().includes(query) ||
      node.description.toLowerCase().includes(query) ||
      node.type.toLowerCase().includes(query)
    )
  })).filter(category => category.nodes.length > 0)
})

const totalNodes = computed(() => {
  return filteredCategories.value.reduce((sum, cat) => sum + cat.nodes.length, 0)
})

function handleAddNode(template: NodeTemplate) {
  emit('add-node', template)
}
</script>

<template>
  <div class="w-72 bg-background border-r border-border flex flex-col h-full">
    <!-- Header -->
    <div class="p-4 border-b border-border">
      <div class="flex items-center gap-2 mb-2">
        <Sparkles class="w-5 h-5 text-primary" />
        <h3 class="font-semibold text-lg">Node Palette</h3>
      </div>
      <p class="text-xs text-muted-foreground">
        {{ totalNodes }} nodes available
      </p>
    </div>

    <!-- Search -->
    <div class="p-3 border-b border-border">
      <div class="relative">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <Input
          v-model="searchQuery"
          placeholder="Search nodes..."
          class="pl-9 h-9"
        />
      </div>
    </div>

    <!-- Categories -->
    <div class="flex-1 overflow-y-auto">
      <div v-if="filteredCategories.length === 0" class="p-8 text-center">
        <p class="text-muted-foreground text-sm">No nodes found</p>
      </div>

      <Accordion
        v-else
        type="multiple"
        class="w-full px-2 py-2"
        :default-value="['URL Discovery', 'Extraction']"
      >
        <AccordionItem
          v-for="category in filteredCategories"
          :key="category.name"
          :value="category.name"
          class="border-0"
        >
          <AccordionTrigger class="px-3 py-2.5 hover:bg-accent rounded-md hover:no-underline">
            <div class="flex items-center gap-2.5 w-full">
              <component
                :is="categoryIcons[category.name]"
                :class="['w-4 h-4', categoryColors[category.name]]"
              />
              <span class="text-sm font-medium flex-1 text-left">{{ category.name }}</span>
              <Badge variant="secondary" class="text-xs">
                {{ category.nodes.length }}
              </Badge>
            </div>
          </AccordionTrigger>

          <AccordionContent class="pt-1 pb-2">
            <div class="space-y-1 px-1">
              <button
                v-for="node in category.nodes"
                :key="node.type"
                @click="handleAddNode(node)"
                class="w-full text-left p-3 rounded-md border border-border bg-card hover:bg-accent hover:border-primary/50 transition-all duration-200 group"
              >
                <div class="flex items-start justify-between gap-2 mb-1">
                  <span class="font-medium text-sm group-hover:text-primary transition-colors">
                    {{ node.label }}
                  </span>
                  <Badge variant="outline" class="text-[10px] font-mono shrink-0">
                    {{ node.type }}
                  </Badge>
                </div>
                <p class="text-xs text-muted-foreground leading-relaxed">
                  {{ node.description }}
                </p>
              </button>
            </div>
          </AccordionContent>
        </AccordionItem>
      </Accordion>
    </div>

    <!-- Footer -->
    <div class="p-3 border-t border-border bg-muted/30">
      <div class="text-xs text-muted-foreground text-center">
        Click any node to add to canvas
      </div>
    </div>
  </div>
</template>
