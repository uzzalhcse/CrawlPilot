<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { Activity, TrendingUp, AlertTriangle, Clock } from 'lucide-vue-next'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useWorkflowsStore } from '@/stores/workflows'

const router = useRouter()
const workflowsStore = useWorkflowsStore()

onMounted(async () => {
  await workflowsStore.fetchWorkflows()
})

const activeWorkflows = computed(() => 
  workflowsStore.workflows.filter((w: any) => w.status === 'active')
)

const viewHealthChecks = (workflowId: string) => {
  router.push(`/health-checks/${workflowId}`)
}

const formatDate = (dateStr: string) => {
  return new Date(dateStr).toLocaleDateString()
}
</script>

<template>
  <div class="container mx-auto py-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold tracking-tight">Health Checks</h1>
        <p class="text-muted-foreground">
          Monitor workflow health and detect structure changes
        </p>
      </div>
    </div>

    <!-- Info Card -->
    <Card>
      <CardHeader>
        <CardTitle class="flex items-center gap-2">
          <Activity class="h-5 w-5" />
          What are Health Checks?
        </CardTitle>
      </CardHeader>
      <CardContent class="space-y-3">
        <p class="text-sm text-muted-foreground">
          Health checks validate your workflows by running test executions without performing full crawls. They help you:
        </p>
        <ul class="list-disc list-inside space-y-1 text-sm text-muted-foreground">
          <li>Detect website structure changes automatically</li>
          <li>Validate selectors and navigation flows</li>
          <li>Test pagination and interactions</li>
          <li>Get specific error messages with fix suggestions</li>
        </ul>
      </CardContent>
    </Card>

    <!-- Workflows List -->
    <div class="space-y-4">
      <h2 class="text-xl font-semibold">Active Workflows</h2>
      
      <div v-if="activeWorkflows.length === 0" class="text-center py-12 text-muted-foreground">
        <Activity class="h-16 w-16 mx-auto mb-4 opacity-50" />
        <p class="text-lg">No active workflows</p>
        <p class="text-sm">Create an active workflow to run health checks</p>
      </div>

      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <Card 
          v-for="workflow in activeWorkflows" 
          :key="workflow.id"
          class="hover:shadow-lg transition-shadow cursor-pointer"
          @click="viewHealthChecks(workflow.id)"
        >
          <CardHeader>
            <div class="flex items-center justify-between">
              <CardTitle class="text-lg">{{ workflow.name }}</CardTitle>
              <Badge variant="default">Active</Badge>
            </div>
            <CardDescription class="line-clamp-2">
              {{ workflow.description || 'No description' }}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div class="space-y-2">
              <div class="text-sm text-muted-foreground">
                Created: {{ formatDate(workflow.created_at) }}
              </div>
              <Button 
                @click.stop="viewHealthChecks(workflow.id)" 
                variant="outline" 
                size="sm" 
                class="w-full"
              >
                <Activity class="h-4 w-4 mr-2" />
                View Health Checks
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>

    <!-- Quick Stats (Future Enhancement) -->
    <Card v-if="false">
      <CardHeader>
        <CardTitle>Overall Health Status</CardTitle>
        <CardDescription>Aggregate health check statistics</CardDescription>
      </CardHeader>
      <CardContent>
        <div class="grid gap-4 md:grid-cols-3">
          <div class="text-center p-4 bg-green-50 rounded-lg">
            <TrendingUp class="h-8 w-8 mx-auto mb-2 text-green-600" />
            <div class="text-2xl font-bold text-green-600">85%</div>
            <div class="text-sm text-gray-600">Healthy Workflows</div>
          </div>
          <div class="text-center p-4 bg-yellow-50 rounded-lg">
            <AlertTriangle class="h-8 w-8 mx-auto mb-2 text-yellow-600" />
            <div class="text-2xl font-bold text-yellow-600">12%</div>
            <div class="text-sm text-gray-600">With Warnings</div>
          </div>
          <div class="text-center p-4 bg-gray-50 rounded-lg">
            <Clock class="h-8 w-8 mx-auto mb-2 text-gray-600" />
            <div class="text-2xl font-bold text-gray-600">45s</div>
            <div class="text-sm text-gray-600">Avg Check Time</div>
          </div>
        </div>
      </CardContent>
    </Card>
  </div>
</template>
