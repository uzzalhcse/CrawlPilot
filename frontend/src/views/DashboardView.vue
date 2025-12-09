<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useExecutionsStore } from '@/stores/executions'
import { useWorkflowsStore } from '@/stores/workflows'
import { schedulesApi, type Schedule } from '@/api/schedules'
import type { Workflow } from '@/types'
import { 
  PlayCircle, 
  Workflow as WorkflowIcon, 
  CheckCircle2, 
  Activity, 
  Plus,
  Clock,
  Calendar,
  ArrowRight,
  Zap,
  Timer,
  ChevronRight,
  Loader2,
  Globe
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

const router = useRouter()
const executionsStore = useExecutionsStore()
const workflowsStore = useWorkflowsStore()

const loading = ref(true)
const schedules = ref<Schedule[]>([])
const activeTab = ref<'recent' | 'scheduled'>('recent')

// Stats
const totalWorkflows = computed(() => workflowsStore.workflows.length)
const activeExecutions = computed(() => executionsStore.runningExecutions.length)
const recentExecutions = computed(() => executionsStore.executions.slice(0, 6))
const recentWorkflows = computed(() => workflowsStore.workflows.slice(0, 3))

const successRate = computed(() => {
  const total = executionsStore.executions.length
  if (total === 0) return 0
  const successful = executionsStore.completedExecutions.length
  return Math.round((successful / total) * 100)
})

const upcomingSchedules = computed(() => {
  return schedules.value
    .filter(s => s.is_enabled && s.next_run_at)
    .sort((a, b) => new Date(a.next_run_at!).getTime() - new Date(b.next_run_at!).getTime())
    .slice(0, 6)
})

const navigateTo = (route: string) => router.push(route)

// Count nodes in a workflow - nodes are inside phases
const getNodeCount = (workflow: Workflow) => {
  const phases = workflow.config?.phases || []
  return phases.reduce((total, phase) => total + (phase.nodes?.length || 0), 0)
}

// Get phase count
const getPhaseCount = (workflow: Workflow) => {
  return workflow.config?.phases?.length || 0
}

const formatTimeAgo = (dateString?: string) => {
  if (!dateString) return 'Unknown'
  const date = new Date(dateString)
  if (isNaN(date.getTime())) return 'Unknown'
  
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / (1000 * 60))
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  
  if (minutes < 1) return 'Just now'
  if (minutes < 60) return `${minutes}m ago`
  if (hours < 24) return `${hours}h ago`
  if (days < 7) return `${days}d ago`
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

const formatNextRun = (dateString?: string) => {
  if (!dateString) return 'Not scheduled'
  const date = new Date(dateString)
  if (isNaN(date.getTime())) return 'N/A'
  
  const now = new Date()
  const diff = date.getTime() - now.getTime()
  const minutes = Math.floor(diff / (1000 * 60))
  const hours = Math.floor(minutes / 60)
  
  if (minutes < 1) return 'Now'
  if (minutes < 60) return `in ${minutes}m`
  if (hours < 24) return `in ${hours}h`
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

const getStatusColor = (status: string) => {
  switch (status) {
    case 'completed': return 'bg-emerald-500'
    case 'failed': return 'bg-red-500'
    case 'running': return 'bg-blue-500'
    default: return 'bg-gray-500'
  }
}

const getStatusBadgeClass = (status: string) => {
  switch (status) {
    case 'completed': return 'bg-emerald-500/10 text-emerald-500 border-emerald-500/20'
    case 'failed': return 'bg-red-500/10 text-red-500 border-red-500/20'
    case 'running': return 'bg-blue-500/10 text-blue-500 border-blue-500/20'
    default: return 'bg-gray-500/10 text-gray-400 border-gray-500/20'
  }
}

onMounted(async () => {
  try {
    const [, , schedulesRes] = await Promise.all([
      workflowsStore.fetchWorkflows(),
      executionsStore.fetchAllExecutions({ limit: 20 }),
      schedulesApi.list({ limit: 10 })
    ])
    schedules.value = schedulesRes.data.schedules || []
  } catch (error) {
    console.error('Failed to load dashboard data:', error)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="min-h-screen">
    <div class="p-6 space-y-6">
      
      <!-- Welcome Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-semibold">Welcome back</h1>
          <p class="text-muted-foreground text-sm mt-0.5">Here's what's happening with your workflows</p>
        </div>
        <Button @click="navigateTo('/workflows/create')" size="sm">
          <Plus class="w-4 h-4 mr-2" />
          New Workflow
        </Button>
      </div>

      <!-- Stats Row -->
      <div class="grid grid-cols-4 gap-4">
        <div 
          @click="navigateTo('/workflows')"
          class="group rounded-xl border bg-card p-4 cursor-pointer hover:border-primary/40 transition-all"
        >
          <div class="flex items-center justify-between">
            <div>
              <p class="text-xs text-muted-foreground font-medium">Workflows</p>
              <p class="text-2xl font-bold mt-0.5">{{ totalWorkflows }}</p>
            </div>
            <div class="p-2 rounded-lg bg-blue-500/10">
              <WorkflowIcon class="w-4 h-4 text-blue-500" />
            </div>
          </div>
        </div>

        <div 
          @click="navigateTo('/executions?status=running')"
          class="group rounded-xl border bg-card p-4 cursor-pointer hover:border-primary/40 transition-all"
        >
          <div class="flex items-center justify-between">
            <div>
              <p class="text-xs text-muted-foreground font-medium">Running</p>
              <p class="text-2xl font-bold mt-0.5">{{ activeExecutions }}</p>
            </div>
            <div class="p-2 rounded-lg bg-emerald-500/10">
              <Activity class="w-4 h-4 text-emerald-500" :class="{ 'animate-pulse': activeExecutions > 0 }" />
            </div>
          </div>
        </div>

        <div 
          @click="navigateTo('/executions')"
          class="group rounded-xl border bg-card p-4 cursor-pointer hover:border-primary/40 transition-all"
        >
          <div class="flex items-center justify-between">
            <div>
              <p class="text-xs text-muted-foreground font-medium">Success Rate</p>
              <p class="text-2xl font-bold mt-0.5">{{ successRate }}%</p>
            </div>
            <div class="p-2 rounded-lg bg-purple-500/10">
              <CheckCircle2 class="w-4 h-4 text-purple-500" />
            </div>
          </div>
        </div>

        <div 
          @click="navigateTo('/schedules')"
          class="group rounded-xl border bg-card p-4 cursor-pointer hover:border-primary/40 transition-all"
        >
          <div class="flex items-center justify-between">
            <div>
              <p class="text-xs text-muted-foreground font-medium">Schedules</p>
              <p class="text-2xl font-bold mt-0.5">{{ schedules.filter(s => s.is_enabled).length }}</p>
            </div>
            <div class="p-2 rounded-lg bg-amber-500/10">
              <Calendar class="w-4 h-4 text-amber-500" />
            </div>
          </div>
        </div>
      </div>

      <!-- Your Workflows -->
      <div>
        <div class="flex items-center justify-between mb-3">
          <h2 class="font-semibold">Your Workflows</h2>
          <Button variant="link" size="sm" @click="navigateTo('/workflows')" class="text-muted-foreground hover:text-foreground h-auto p-0">
            View all <ArrowRight class="w-3 h-3 ml-1" />
          </Button>
        </div>
        
        <div v-if="loading" class="grid grid-cols-4 gap-4">
          <div v-for="i in 4" :key="i" class="h-24 rounded-xl bg-muted/30 animate-pulse" />
        </div>
        
        <div v-else class="grid grid-cols-4 gap-4">
          <div
            v-for="workflow in recentWorkflows"
            :key="workflow.id"
            @click="navigateTo(`/workflows/${workflow.id}`)"
            class="group rounded-xl border bg-card p-4 cursor-pointer hover:border-primary/40 transition-all"
          >
            <div class="flex items-start gap-3">
              <div class="p-2 rounded-lg bg-primary/10 shrink-0">
                <Globe class="w-4 h-4 text-primary" />
              </div>
              <div class="min-w-0 flex-1">
                <h3 class="font-medium text-sm truncate">{{ workflow.name }}</h3>
                <p class="text-xs text-muted-foreground mt-0.5 truncate">
                  {{ workflow.config?.start_urls?.[0] || 'No target URL' }}
                </p>
              </div>
            </div>
            <div class="flex items-center gap-2 mt-3 pt-2 border-t border-border/50">
              <span class="text-xs text-muted-foreground">{{ getPhaseCount(workflow) }} phases</span>
              <span class="text-muted-foreground/50">•</span>
              <span class="text-xs text-muted-foreground">{{ getNodeCount(workflow) }} nodes</span>
            </div>
          </div>

          <!-- Create New -->
          <div
            @click="navigateTo('/workflows/create')"
            class="rounded-xl border-2 border-dashed border-muted-foreground/20 p-4 cursor-pointer hover:border-primary/40 hover:bg-muted/20 transition-all flex flex-col items-center justify-center text-center min-h-[96px]"
          >
            <Plus class="w-5 h-5 text-muted-foreground mb-1" />
            <span class="text-sm text-muted-foreground">Create New</span>
          </div>
        </div>
      </div>

      <!-- Workflow Runs -->
      <div>
        <div class="flex items-center justify-between mb-3">
          <h2 class="font-semibold">Workflow Runs</h2>
          <Button variant="link" size="sm" @click="navigateTo('/executions')" class="text-muted-foreground hover:text-foreground h-auto p-0">
            View all runs <ArrowRight class="w-3 h-3 ml-1" />
          </Button>
        </div>

        <!-- Custom Tabs -->
        <div class="flex items-center gap-1 mb-4 bg-muted/50 p-1 rounded-lg w-fit">
          <button 
            @click="activeTab = 'recent'"
            :class="[
              'px-4 py-1.5 text-sm font-medium rounded-md transition-all flex items-center gap-2',
              activeTab === 'recent' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
            ]"
          >
            <Clock class="w-3.5 h-3.5" /> Recent
          </button>
          <button 
            @click="activeTab = 'scheduled'"
            :class="[
              'px-4 py-1.5 text-sm font-medium rounded-md transition-all flex items-center gap-2',
              activeTab === 'scheduled' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
            ]"
          >
            <Calendar class="w-3.5 h-3.5" /> Scheduled
          </button>
        </div>

        <!-- Recent Runs -->
        <div v-if="activeTab === 'recent'">
          <div v-if="loading" class="flex items-center justify-center py-12">
            <Loader2 class="w-6 h-6 animate-spin text-muted-foreground" />
          </div>
          
          <div v-else-if="recentExecutions.length === 0" class="text-center py-12 border rounded-xl bg-card/50">
            <PlayCircle class="w-10 h-10 text-muted-foreground mx-auto mb-2" />
            <p class="text-muted-foreground text-sm">No recent runs</p>
          </div>

          <div v-else class="border rounded-xl overflow-hidden">
            <div class="divide-y divide-border">
              <div
                v-for="execution in recentExecutions"
                :key="execution.id"
                @click="navigateTo(`/executions/${execution.id}`)"
                class="flex items-center justify-between px-4 py-3 hover:bg-muted/30 cursor-pointer transition-colors"
              >
                <div class="flex items-center gap-3">
                  <div :class="['w-2 h-2 rounded-full shrink-0', getStatusColor(execution.status)]" />
                  <div>
                    <p class="font-medium text-sm">{{ execution.workflow_name || 'Untitled' }}</p>
                    <p class="text-xs text-muted-foreground">{{ formatTimeAgo(execution.started_at) }}</p>
                  </div>
                </div>
                <div class="flex items-center gap-3">
                  <div class="text-right">
                    <p class="text-sm font-medium">{{ execution.stats?.items_extracted || 0 }} items</p>
                    <p class="text-xs text-muted-foreground">extracted</p>
                  </div>
                  <Badge :class="getStatusBadgeClass(execution.status)" class="capitalize text-xs">
                    {{ execution.status }}
                  </Badge>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Scheduled Runs -->
        <div v-if="activeTab === 'scheduled'">
          <div v-if="loading" class="flex items-center justify-center py-12">
            <Loader2 class="w-6 h-6 animate-spin text-muted-foreground" />
          </div>

          <div v-else-if="upcomingSchedules.length === 0" class="text-center py-12 border rounded-xl bg-card/50">
            <Calendar class="w-10 h-10 text-muted-foreground mx-auto mb-2" />
            <p class="text-muted-foreground text-sm">No scheduled runs</p>
            <Button @click="navigateTo('/schedules')" size="sm" variant="outline" class="mt-3">
              <Plus class="w-3 h-3 mr-1" /> Create Schedule
            </Button>
          </div>

          <div v-else class="border rounded-xl overflow-hidden">
            <div class="divide-y divide-border">
              <div
                v-for="schedule in upcomingSchedules"
                :key="schedule.id"
                @click="navigateTo('/schedules')"
                class="flex items-center justify-between px-4 py-3 hover:bg-muted/30 cursor-pointer transition-colors"
              >
                <div class="flex items-center gap-3">
                  <div class="p-1.5 rounded-lg bg-amber-500/10">
                    <Timer class="w-3.5 h-3.5 text-amber-500" />
                  </div>
                  <div>
                    <p class="font-medium text-sm">{{ schedule.name }}</p>
                    <p class="text-xs text-muted-foreground">{{ schedule.workflow_name }}</p>
                  </div>
                </div>
                <div class="flex items-center gap-3">
                  <div class="text-right">
                    <p class="text-sm font-medium text-amber-500">{{ formatNextRun(schedule.next_run_at) }}</p>
                    <code class="text-xs text-muted-foreground">{{ schedule.cron_expression }}</code>
                  </div>
                  <Badge class="bg-emerald-500/10 text-emerald-500 border-emerald-500/20 text-xs">
                    <Zap class="w-2.5 h-2.5 mr-1" /> Active
                  </Badge>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

    </div>
  </div>
</template>
