<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useExecutionsStore } from '@/stores/executions'
import { useWorkflowsStore } from '@/stores/workflows'
import { 
  PlayCircle, 
  Workflow, 
  CheckCircle2, 
  Activity, 
  Plus, 
  Search, 
  FileText, 
  Server,
  Clock,
  AlertCircle
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'

const router = useRouter()
const executionsStore = useExecutionsStore()
const workflowsStore = useWorkflowsStore()

const loading = ref(true)

// Stats
const totalWorkflows = computed(() => workflowsStore.workflows.length)
const activeExecutions = computed(() => executionsStore.runningExecutions.length)
const recentExecutions = computed(() => executionsStore.executions.slice(0, 5))

const successRate = computed(() => {
  const total = executionsStore.executions.length
  if (total === 0) return 0
  const successful = executionsStore.completedExecutions.length
  return Math.round((successful / total) * 100)
})

const systemHealth = ref('Healthy') // Placeholder for now

// Quick Actions
const quickActions = [
  { label: 'New Workflow', icon: Plus, route: '/workflows/create', color: 'text-blue-500' },
  { label: 'Browse Plugins', icon: Search, route: '/plugins', color: 'text-purple-500' },
  { label: 'View Executions', icon: PlayCircle, route: '/executions', color: 'text-green-500' },
  { label: 'System Status', icon: Activity, route: '/monitoring', color: 'text-orange-500' }
]

const navigateTo = (route: string) => {
  router.push(route)
}

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString()
}

const getStatusColor = (status: string) => {
  switch (status) {
    case 'completed': return 'text-green-500'
    case 'failed': return 'text-red-500'
    case 'running': return 'text-blue-500'
    default: return 'text-gray-500'
  }
}

const getStatusIcon = (status: string) => {
  switch (status) {
    case 'completed': return CheckCircle2
    case 'failed': return AlertCircle
    case 'running': return Activity
    default: return Clock
  }
}

onMounted(async () => {
  try {
    await Promise.all([
      workflowsStore.fetchWorkflows(),
      executionsStore.fetchAllExecutions({ limit: 20 })
    ])
  } catch (error) {
    console.error('Failed to load dashboard data:', error)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="p-8 space-y-8 max-w-[1600px] mx-auto">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold tracking-tight text-white">Dashboard</h1>
        <p class="text-muted-foreground mt-1">
          Overview of your crawling automation platform.
        </p>
      </div>
      <div class="flex items-center gap-2">
        <div class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-green-500/10 text-green-500 text-sm font-medium border border-green-500/20">
          <div class="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
          System Operational
        </div>
      </div>
    </div>

    <!-- Stats Grid -->
    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      <Card class="bg-[#1a1d21] border-gray-800">
        <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle class="text-sm font-medium text-gray-400">Total Workflows</CardTitle>
          <Workflow class="h-4 w-4 text-blue-500" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold text-white">{{ totalWorkflows }}</div>
          <p class="text-xs text-gray-500 mt-1">Defined automations</p>
        </CardContent>
      </Card>

      <Card class="bg-[#1a1d21] border-gray-800">
        <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle class="text-sm font-medium text-gray-400">Active Executions</CardTitle>
          <Activity class="h-4 w-4 text-green-500" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold text-white">{{ activeExecutions }}</div>
          <p class="text-xs text-gray-500 mt-1">Currently running jobs</p>
        </CardContent>
      </Card>

      <Card class="bg-[#1a1d21] border-gray-800">
        <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle class="text-sm font-medium text-gray-400">Success Rate</CardTitle>
          <CheckCircle2 class="h-4 w-4 text-purple-500" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold text-white">{{ successRate }}%</div>
          <p class="text-xs text-gray-500 mt-1">Based on recent runs</p>
        </CardContent>
      </Card>

      <Card class="bg-[#1a1d21] border-gray-800">
        <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle class="text-sm font-medium text-gray-400">System Health</CardTitle>
          <Server class="h-4 w-4 text-orange-500" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold text-white">{{ systemHealth }}</div>
          <p class="text-xs text-gray-500 mt-1">All systems normal</p>
        </CardContent>
      </Card>
    </div>

    <!-- Quick Actions -->
    <div>
      <h2 class="text-lg font-semibold text-white mb-4">Quick Actions</h2>
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <button
          v-for="action in quickActions"
          :key="action.label"
          class="flex items-center gap-4 p-4 rounded-xl bg-[#1a1d21] border border-gray-800 hover:bg-[#222529] hover:border-gray-700 transition-all group text-left"
          @click="navigateTo(action.route)"
        >
          <div class="p-3 rounded-lg bg-gray-800/50 group-hover:bg-gray-800 transition-colors">
            <component :is="action.icon" :class="['w-6 h-6', action.color]" />
          </div>
          <div>
            <div class="font-medium text-white group-hover:text-blue-400 transition-colors">{{ action.label }}</div>
            <div class="text-xs text-gray-500">Jump to section</div>
          </div>
        </button>
      </div>
    </div>

    <!-- Main Content Split -->
    <div class="grid gap-8 lg:grid-cols-3">
      <!-- Recent Activity -->
      <div class="lg:col-span-2 space-y-4">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-white">Recent Activity</h2>
          <Button variant="ghost" size="sm" class="text-gray-400 hover:text-white" @click="navigateTo('/executions')">
            View all
          </Button>
        </div>

        <Card class="bg-[#1a1d21] border-gray-800">
          <CardContent class="p-0">
            <div v-if="loading" class="p-8 text-center text-gray-500">
              Loading activity...
            </div>
            <div v-else-if="recentExecutions.length === 0" class="p-8 text-center text-gray-500">
              No recent activity found.
            </div>
            <div v-else class="divide-y divide-gray-800">
              <div
                v-for="execution in recentExecutions"
                :key="execution.id"
                class="flex items-center justify-between p-4 hover:bg-white/5 transition-colors cursor-pointer"
                @click="navigateTo(`/executions/${execution.id}`)"
              >
                <div class="flex items-center gap-4">
                  <div :class="['p-2 rounded-full bg-gray-800/50', getStatusColor(execution.status)]">
                    <component :is="getStatusIcon(execution.status)" class="w-4 h-4" />
                  </div>
                  <div>
                    <div class="font-medium text-white">{{ execution.workflow_name || 'Untitled Workflow' }}</div>
                    <div class="text-xs text-gray-500 flex items-center gap-2">
                      <span>{{ formatDate(execution.created_at) }}</span>
                      <span>•</span>
                      <span class="capitalize">{{ execution.status }}</span>
                    </div>
                  </div>
                </div>
                <div class="text-right">
                  <div class="text-sm font-medium text-gray-300">
                    {{ execution.stats?.items_extracted || 0 }} items
                  </div>
                  <div class="text-xs text-gray-500">extracted</div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <!-- System Status / Resources -->
      <div class="space-y-4">
        <h2 class="text-lg font-semibold text-white">System Status</h2>
        
        <Card class="bg-[#1a1d21] border-gray-800">
          <CardContent class="p-4 space-y-6">
            <!-- Node Status -->
            <div class="space-y-3">
              <div class="flex items-center justify-between text-sm">
                <span class="text-gray-400">Active Nodes</span>
                <span class="text-white font-medium">1/1</span>
              </div>
              <div class="h-2 bg-gray-800 rounded-full overflow-hidden">
                <div class="h-full bg-blue-500 w-full" />
              </div>
            </div>

            <!-- Proxy Status -->
            <div class="space-y-3">
              <div class="flex items-center justify-between text-sm">
                <span class="text-gray-400">Proxy Health</span>
                <span class="text-white font-medium">98%</span>
              </div>
              <div class="h-2 bg-gray-800 rounded-full overflow-hidden">
                <div class="h-full bg-green-500 w-[98%]" />
              </div>
            </div>

            <!-- Storage -->
            <div class="space-y-3">
              <div class="flex items-center justify-between text-sm">
                <span class="text-gray-400">Storage Usage</span>
                <span class="text-white font-medium">45%</span>
              </div>
              <div class="h-2 bg-gray-800 rounded-full overflow-hidden">
                <div class="h-full bg-purple-500 w-[45%]" />
              </div>
            </div>
          </CardContent>
        </Card>

        <!-- Documentation Link -->
        <Card class="bg-gradient-to-br from-blue-900/20 to-purple-900/20 border-blue-500/20">
          <CardContent class="p-4">
            <h3 class="font-semibold text-blue-400 mb-1">Need Help?</h3>
            <p class="text-sm text-gray-400 mb-3">Check out our documentation to learn more about creating workflows and plugins.</p>
            <Button variant="outline" size="sm" class="w-full border-blue-500/30 hover:bg-blue-500/10 text-blue-400">
              View Documentation
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  </div>
</template>
