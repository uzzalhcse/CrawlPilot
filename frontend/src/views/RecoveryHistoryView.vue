<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { 
  getRecoveryAttempts, 
  getRecoveryAttemptStats,
  type RecoveryAttempt, 
  type RecoveryAttemptStats 
} from '@/api/recovery'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import DataTable from '@/components/ui/data-table.vue'
import PageLayout from '@/components/layout/PageLayout.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import StatsBar from '@/components/layout/StatsBar.vue'
import FilterBar from '@/components/layout/FilterBar.vue'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { 
  Activity, 
  CheckCircle2, 
  Loader2, 
  SlidersHorizontal,
  RefreshCw,
  Bot,
  Cog,
  Zap
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const loading = ref(true)
const attempts = ref<RecoveryAttempt[]>([])
const attemptStats = ref<RecoveryAttemptStats | null>(null)
const totalAttempts = ref(0)
const statusFilter = ref<string>('all')
const searchQuery = ref('')
let refreshInterval: ReturnType<typeof setInterval> | null = null

const tableColumns = [
  { key: 'domain', label: 'Domain', sortable: true, align: 'left' as const },
  { key: 'error_pattern', label: 'Error', align: 'left' as const },
  { key: 'action', label: 'Action', align: 'left' as const },
  { key: 'source', label: 'Source', align: 'left' as const },
  { key: 'status', label: 'Status', align: 'left' as const },
  { key: 'created_at', label: 'Time', align: 'left' as const }
]

const stats = computed(() => {
  if (!attemptStats.value) return []
  const s = attemptStats.value
  const successRate = s.success_rate?.toFixed(1) || '0'
  return [
    { label: 'Total', value: s.total },
    { label: 'Pending', value: s.pending, color: 'text-yellow-600 dark:text-yellow-400' },
    { label: 'Success', value: s.success, color: 'text-green-600 dark:text-green-400' },
    { label: 'Failed', value: s.failed, color: 'text-red-600 dark:text-red-400' },
    { label: 'Success Rate', value: `${successRate}%`, color: 'text-blue-600 dark:text-blue-400' }
  ]
})

const filteredAttempts = computed(() => {
  let result = attempts.value
  
  if (statusFilter.value !== 'all') {
    result = result.filter(a => a.status === statusFilter.value)
  }
  
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(a => 
      a.domain?.toLowerCase().includes(query) || 
      a.url?.toLowerCase().includes(query) ||
      a.error_pattern?.toLowerCase().includes(query)
    )
  }
  
  return result
})

const fetchData = async () => {
  loading.value = true
  try {
    const [attemptsRes, statsRes] = await Promise.all([
      getRecoveryAttempts({ limit: 100 }),
      getRecoveryAttemptStats()
    ])
    attempts.value = attemptsRes.attempts || []
    totalAttempts.value = attemptsRes.total
    attemptStats.value = statsRes
  } catch (error) {
    console.error('Failed to fetch recovery attempts:', error)
    toast.error('Failed to load recovery history')
  } finally {
    loading.value = false
  }
}

const handleRefresh = async () => {
  await fetchData()
  toast.success('Data refreshed')
}

const getStatusColor = (status: string) => {
  switch(status) {
    case 'pending': return 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 border-yellow-500/20'
    case 'success': return 'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20'
    case 'failed': return 'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20'
    default: return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
  }
}

const getSourceColor = (source: string) => {
  switch(source) {
    case 'rule': return 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20'
    case 'ai': return 'bg-purple-500/10 text-purple-600 dark:text-purple-400 border-purple-500/20'
    case 'default': return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
    default: return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
  }
}

const getSourceIcon = (source: string) => {
  switch(source) {
    case 'rule': return Cog
    case 'ai': return Bot
    case 'default': return Zap
    default: return Activity
  }
}

const formatDate = (dateString?: string) => {
  if (!dateString) return 'N/A'
  const date = new Date(dateString)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  
  if (seconds < 60) return 'Just now'
  if (minutes < 60) return `${minutes}m ago`
  if (hours < 24) return `${hours}h ago`
  
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}d ago`
  
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

const formatErrorPattern = (pattern: string) => {
  if (!pattern) return 'Unknown'
  return pattern.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}

const formatAction = (action: string) => {
  if (!action) return 'Pending'
  return action.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}

onMounted(() => {
  fetchData()
  // Auto-refresh every 10 seconds
  refreshInterval = setInterval(fetchData, 10000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>

<template>
  <PageLayout>
    <!-- Header -->
    <PageHeader 
      title="Recovery History" 
      description="Track error recovery attempts and their outcomes in real-time"
      :show-help-icon="true"
    >
      <template #actions>
        <Button @click="handleRefresh" variant="outline" size="sm" class="gap-2">
          <RefreshCw class="w-4 h-4" />
          Refresh
        </Button>
      </template>
    </PageHeader>

    <!-- Stats -->
    <StatsBar :stats="stats" />

    <!-- Filters -->
    <FilterBar 
      search-placeholder="Search by domain, URL, or pattern..." 
      :search-value="searchQuery"
      @update:search-value="searchQuery = $event"
    >
      <template #filters>
        <Select v-model="statusFilter">
          <SelectTrigger class="w-[140px] h-9">
            <div class="flex items-center gap-2">
              <SlidersHorizontal class="w-4 h-4" />
              <SelectValue placeholder="Status" />
            </div>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Status</SelectItem>
            <SelectItem value="pending">Pending</SelectItem>
            <SelectItem value="success">Success</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
          </SelectContent>
        </Select>
      </template>
    </FilterBar>

    <!-- Table -->
    <div class="flex-1 overflow-auto">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <div v-else-if="filteredAttempts.length === 0" class="py-12 text-center px-6">
        <CheckCircle2 class="h-12 w-12 text-green-500 mx-auto mb-3" />
        <p class="text-muted-foreground">No recovery attempts found</p>
        <p class="text-sm text-muted-foreground mt-1">Recovery attempts will appear here when errors are detected.</p>
      </div>

      <DataTable
        v-else
        :data="filteredAttempts"
        :columns="tableColumns"
      >
        <template #row="{ row }">
          <td class="px-6 py-3">
            <div class="min-w-0">
              <div class="font-medium text-sm truncate">{{ row.domain || 'Unknown' }}</div>
              <div class="text-xs text-muted-foreground truncate max-w-[200px]">{{ row.url }}</div>
            </div>
          </td>
          <td class="px-6 py-3">
            <Badge variant="outline" class="text-xs font-medium">
              {{ formatErrorPattern(row.error_pattern) }}
            </Badge>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm">
              {{ formatAction(row.action) }}
            </div>
          </td>
          <td class="px-6 py-3">
            <Badge 
              variant="outline"
              :class="getSourceColor(row.source)"
              class="text-xs font-medium capitalize gap-1"
            >
              <component :is="getSourceIcon(row.source)" class="w-3 h-3" />
              {{ row.source || 'pending' }}
            </Badge>
          </td>
          <td class="px-6 py-3">
            <Badge 
              variant="outline"
              :class="getStatusColor(row.status)"
              class="text-xs font-medium capitalize"
            >
              <div class="w-1.5 h-1.5 rounded-full mr-1.5" :class="{
                'bg-yellow-500': row.status === 'pending',
                'bg-green-500': row.status === 'success',
                'bg-red-500': row.status === 'failed'
              }"></div>
              {{ row.status }}
            </Badge>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm text-muted-foreground">
              {{ formatDate(row.created_at) }}
            </div>
          </td>
        </template>
      </DataTable>
    </div>
  </PageLayout>
</template>
