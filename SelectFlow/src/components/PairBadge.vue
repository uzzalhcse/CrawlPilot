<script setup lang="ts">
import { computed } from 'vue'
import type { KeyValuePair, KeyValueGroup } from '../types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Pencil, Trash2 } from 'lucide-vue-next'

const props = defineProps<{
  pair: KeyValuePair
  group: KeyValueGroup
}>()

const emit = defineEmits<{
  (e: 'edit', group: KeyValueGroup, pair: KeyValuePair): void
  (e: 'remove', groupId: string, pairId: string): void
}>()

// Calculate position (top-left of key element)
const position = computed(() => {
  const rect = props.pair.keyElement.getBoundingClientRect()
  const scrollTop = window.scrollY || document.documentElement.scrollTop
  const scrollLeft = window.scrollX || document.documentElement.scrollLeft
  
  return {
    top: rect.top + scrollTop - 28, // Position above the element
    left: rect.left + scrollLeft
  }
})

// Calculate match counts
const matchCounts = computed(() => {
  try {
    const keyCount = document.querySelectorAll(props.pair.config.key_selector).length
    const valueCount = document.querySelectorAll(props.pair.config.value_selector).length
    
    if (keyCount === valueCount) {
      return `${keyCount}`
    }
    return `K:${keyCount} V:${valueCount}`
  } catch {
    return '0'
  }
})
</script>

<template>
  <div 
    data-selectflow-ui="badge"
    class="absolute z-[100] pointer-events-auto transition-all duration-200 ease-in-out"
    :style="{ top: position.top + 'px', left: position.left + 'px' }"
  >
    <Badge class="gap-1 pr-0.5 shadow-md hover:bg-primary text-xs font-medium bg-blue-600 hover:bg-blue-700 border-blue-700">
      {{ group.name }}
      <span class="ml-1 px-1.5 py-0.5 rounded-full bg-blue-800/40 text-[10px] leading-none border border-blue-400/20">
        {{ matchCounts }}
      </span>
      <div class="flex items-center gap-0.5 ml-1 border-l border-primary-foreground/20 pl-0.5">
        <Button 
          variant="ghost" 
          size="icon" 
          class="h-5 w-5 hover:bg-primary-foreground/20 text-primary-foreground rounded-full" 
          @click.stop="emit('edit', group, pair)"
          title="Edit pair"
        >
          <Pencil class="h-3 w-3" />
        </Button>
        <Button 
          variant="ghost" 
          size="icon" 
          class="h-5 w-5 hover:bg-primary-foreground/20 text-primary-foreground rounded-full" 
          @click.stop="emit('remove', group.id, pair.id)"
          title="Remove pair"
        >
          <Trash2 class="h-3 w-3" />
        </Button>
      </div>
    </Badge>
  </div>
</template>
