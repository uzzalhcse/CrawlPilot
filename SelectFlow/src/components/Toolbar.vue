<script setup lang="ts">
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Clipboard, Trash2, Eye, Check } from 'lucide-vue-next'

defineProps<{
  fieldCount: number
  showDoneButton?: boolean // Show Done button when used in visual selector mode
}>()

const emit = defineEmits<{
  (e: 'copy'): void
  (e: 'clear'): void
  (e: 'preview'): void
  (e: 'done'): void
}>()
</script>

<template>
  <Card 
    data-selectflow-ui="toolbar"
    class="fixed top-4 right-4 z-[200] p-2 flex items-center gap-2 shadow-lg bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 pointer-events-auto"
  >
    <span class="text-sm font-medium text-muted-foreground px-2">{{ fieldCount }} fields</span>
    <Button variant="outline" size="sm" @click="emit('preview')" class="h-8">
      <Eye class="w-3.5 h-3.5 mr-2" />
      Preview
    </Button>
    <Button variant="outline" size="sm" @click="emit('copy')" class="h-8">
      <Clipboard class="w-3.5 h-3.5 mr-2" />
      Copy JSON
    </Button>
    <Button variant="destructive" size="sm" @click="emit('clear')" class="h-8">
      <Trash2 class="w-3.5 h-3.5 mr-2" />
      Clear
    </Button>
    <!-- Done button - shown when SelectFlow is used in visual selector mode -->
    <Button 
      v-if="showDoneButton" 
      variant="default" 
      size="sm" 
      @click="emit('done')" 
      class="h-8 bg-green-600 hover:bg-green-700"
    >
      <Check class="w-3.5 h-3.5 mr-2" />
      Done
    </Button>
  </Card>
</template>

