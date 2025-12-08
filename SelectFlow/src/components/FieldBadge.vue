<script setup lang="ts">
import { computed } from 'vue'
import type { Field } from '../types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Pencil, Trash2 } from 'lucide-vue-next'

const props = defineProps<{
  field: Field
}>()

const emit = defineEmits<{
  (e: 'edit', field: Field): void
  (e: 'remove', id: string): void
}>()

// Calculate position (top-left of element)
const position = computed(() => {
  if (props.field.fieldType !== 'single') return { top: 0, left: 0 }
  
  const rect = props.field.element.getBoundingClientRect()
  const scrollTop = window.scrollY || document.documentElement.scrollTop
  const scrollLeft = window.scrollX || document.documentElement.scrollLeft
  
  return {
    top: rect.top + scrollTop - 28, // Position above the element
    left: rect.left + scrollLeft
  }
})

// Calculate match count
const matchCount = computed(() => {
  if (props.field.fieldType !== 'single') return 0
  try {
    return document.querySelectorAll(props.field.config.selector).length
  } catch {
    return 0
  }
})
</script>

<template>
  <div 
    data-selectflow-ui="badge"
    class="absolute z-[100] pointer-events-auto transition-all duration-200 ease-in-out"
    :style="{ top: position.top + 'px', left: position.left + 'px' }"
  >
    <Badge class="gap-1 pr-0.5 shadow-md hover:bg-primary text-xs font-medium">
      {{ field.name }}
      <span class="ml-1 px-1.5 py-0.5 rounded-full bg-primary-foreground/20 text-[10px] leading-none">
        {{ matchCount }}
      </span>
      <div class="flex items-center gap-0.5 ml-1 border-l border-primary-foreground/20 pl-0.5">
        <Button 
          variant="ghost" 
          size="icon" 
          class="h-5 w-5 hover:bg-primary-foreground/20 text-primary-foreground rounded-full" 
          @click.stop="emit('edit', field)"
          title="Edit field"
        >
          <Pencil class="h-3 w-3" />
        </Button>
        <Button 
          variant="ghost" 
          size="icon" 
          class="h-5 w-5 hover:bg-primary-foreground/20 text-primary-foreground rounded-full" 
          @click.stop="emit('remove', field.id)"
          title="Remove field"
        >
          <Trash2 class="h-3 w-3" />
        </Button>
      </div>
    </Badge>
  </div>
</template>
