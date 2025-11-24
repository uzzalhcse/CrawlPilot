<script setup lang="ts">
import { ref } from 'vue'
import { Activity, Loader2 } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { useHealthCheckStore } from '@/stores/healthcheck'

const props = defineProps<{
  workflowId: string
  variant?: 'default' | 'outline' | 'ghost'
  size?: 'default' | 'sm' | 'lg'
}>()

const healthCheckStore = useHealthCheckStore()
const running = ref(false)

const runHealthCheck = async () => {
  running.value = true
  try {
    await healthCheckStore.runHealthCheck(props.workflowId)
    console.log('Health check started successfully')
    
    // Optionally refresh health checks list after a delay
    setTimeout(() => {
      healthCheckStore.fetchHealthChecks(props.workflowId, 5)
    }, 2000)
  } catch (error: any) {
    console.error('Failed to start health check:', error.message || error)
    alert('Failed to start health check: ' + (error.message || 'An error occurred'))
  } finally {
    running.value = false
  }
}
</script>

<template>
  <Button 
    @click="runHealthCheck" 
    :disabled="running" 
    :variant="variant || 'outline'" 
    :size="size || 'sm'"
  >
    <Loader2 v-if="running" class="mr-2 h-4 w-4 animate-spin" />
    <Activity v-else class="mr-2 h-4 w-4" />
    Run Health Check
  </Button>
</template>
