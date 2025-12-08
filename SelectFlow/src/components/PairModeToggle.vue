<script setup lang="ts">
import { Button } from '@/components/ui/button'

defineProps<{
  mode: 'single' | 'pair'
}>()

const emit = defineEmits<{
  (e: 'toggle'): void
}>()
</script>

<template>
  <div class="mode-toggle" data-selectflow-ui="toggle">
    <Button 
      :variant="mode === 'single' ? 'default' : 'outline'"
      size="sm"
      @click="emit('toggle')"
      class="h-8 px-3"
    >
      <span class="text-xs">Single</span>
    </Button>
    <Button 
      :variant="mode === 'pair' ? 'default' : 'outline'"
      size="sm"
      @click="emit('toggle')"
      class="h-8 px-3"
    >
      <span class="text-xs">Pair</span>
    </Button>
  </div>
</template>

<style scoped>
.mode-toggle {
  display: flex;
  gap: 2px;
  background: white;
  border-radius: 8px;
  padding: 2px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  pointer-events: auto;
}
</style>
