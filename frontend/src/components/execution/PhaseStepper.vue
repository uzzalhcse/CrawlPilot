<script setup lang="ts">
import { computed } from 'vue'
import { CheckCircle2, Circle, Loader2, AlertCircle } from 'lucide-vue-next'

interface Phase {
  id: string
  name: string
  status: 'pending' | 'running' | 'completed' | 'failed'
}

const props = defineProps<{
  phases: Phase[]
  currentPhaseId?: string | null
}>()

const displayPhases = computed(() => {
  // If we have explicit phases from props, use them
  if (props.phases && props.phases.length > 0) {
    return props.phases
  }
  
  // Default phases if none provided (fallback)
  return [
    { id: 'init', name: 'Initialization', status: 'completed' },
    { id: 'discovery', name: 'Discovery', status: 'running' },
    { id: 'extraction', name: 'Extraction', status: 'pending' },
    { id: 'completion', name: 'Completion', status: 'pending' }
  ] as Phase[]
})
</script>

<template>
  <div class="w-full py-4">
    <div class="relative flex items-center justify-between w-full">
      <!-- Connecting Line -->
      <div class="absolute left-0 top-1/2 h-0.5 w-full -translate-y-1/2 bg-muted z-0"></div>
      
      <!-- Steps -->
      <div 
        v-for="(phase, index) in displayPhases" 
        :key="phase.id"
        class="relative z-10 flex flex-col items-center bg-background px-2"
      >
        <div 
          class="flex h-8 w-8 items-center justify-center rounded-full border-2 transition-colors duration-300"
          :class="{
            'border-primary bg-primary text-primary-foreground': phase.status === 'completed',
            'border-primary bg-background text-primary': phase.status === 'running',
            'border-muted-foreground bg-background text-muted-foreground': phase.status === 'pending',
            'border-destructive bg-destructive text-destructive-foreground': phase.status === 'failed'
          }"
        >
          <CheckCircle2 v-if="phase.status === 'completed'" class="h-5 w-5" />
          <Loader2 v-else-if="phase.status === 'running'" class="h-5 w-5 animate-spin" />
          <AlertCircle v-else-if="phase.status === 'failed'" class="h-5 w-5" />
          <span v-else class="text-xs font-bold">{{ index + 1 }}</span>
        </div>
        <span 
          class="mt-2 text-xs font-medium transition-colors duration-300"
          :class="{
            'text-primary': phase.status === 'running' || phase.status === 'completed',
            'text-muted-foreground': phase.status === 'pending',
            'text-destructive': phase.status === 'failed'
          }"
        >
          {{ phase.name }}
        </span>
      </div>
    </div>
  </div>
</template>
