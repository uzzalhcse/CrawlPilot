<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { 
  getRecoveryAttempts, 
  getRecoveryAttemptStats,
  type RecoveryAttempt, 
  type RecoveryAttemptStats 
} from '@/api/recovery'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger
} from '@/components/ui/accordion'
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
  CheckCircle2, 
  Loader2, 
  SlidersHorizontal,
  RefreshCw
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const loading = ref(true)
const attempts = ref<RecoveryAttempt[]>([])
const attemptStats = ref<RecoveryAttemptStats | null>(null)
const totalAttempts = ref(0)
const statusFilter = ref<string>('all')
const searchQuery = ref('')

const stats = computed(() => {
  if (!attemptStats.value) return []
  const s = attemptStats.value
  const successRate = s.success_rate?.toFixed(1) || '0'
  const detected = attempts.value.filter(a => a.status === 'detected').length
  return [
    { label: 'Total', value: s.total },
    { label: 'Detected', value: detected, color: 'text-cyan-600 dark:text-cyan-400' },
    { label: 'Pending', value: s.pending || 0, color: 'text-yellow-600 dark:text-yellow-400' },
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
    case 'detected': return 'bg-cyan-500/10 text-cyan-600 dark:text-cyan-400 border-cyan-500/20'
    case 'pending': return 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 border-yellow-500/20'
    case 'success': return 'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20'
    case 'failed': return 'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20'
    default: return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
  }
}

const getStatusDot = (status: string) => {
  switch(status) {
    case 'detected': return 'bg-cyan-500'
    case 'pending': return 'bg-yellow-500'
    case 'success': return 'bg-green-500'
    case 'failed': return 'bg-red-500'
    default: return 'bg-gray-500'
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

const formatAction = (action?: string, status?: string) => {
  if (status === 'detected' || (!action && status === 'detected')) return 'No Action'
  if (!action || action === 'pending') return 'Pending'
  return action.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}

const getTierLabel = (tier: number) => {
  switch(tier) {
    case 0: return 'Direct'
    case 1: return 'DC'
    case 2: return 'Res'
    case 3: return 'Mobile'
    default: return tier ? `T${tier}` : '-'
  }
}

const getTierColor = (tier: number) => {
  switch(tier) {
    case 1: return 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20'
    case 2: return 'bg-purple-500/10 text-purple-600 dark:text-purple-400 border-purple-500/20'
    case 3: return 'bg-orange-500/10 text-orange-600 dark:text-orange-400 border-orange-500/20'
    default: return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
  }
}

onMounted(() => {
  fetchData()
})
</script>

<template>
  <PageLayout>
    <!-- Header -->
    <PageHeader 
      title="Recovery History" 
      description="Track error recovery attempts and their outcomes in real-time"
    >
      <template #actions>
        <Button variant="outline" size="sm" @click="handleRefresh">
          <RefreshCw class="w-4 h-4 mr-2" />
          Refresh
        </Button>
      </template>
    </PageHeader>

    <!-- Stats -->
    <StatsBar :stats="stats" />

    <!-- Filter -->
    <FilterBar 
      v-model="searchQuery"
      placeholder="Search by domain, URL, or pattern..."
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
            <SelectItem value="detected">Detected</SelectItem>
            <SelectItem value="pending">Pending</SelectItem>
            <SelectItem value="success">Success</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
          </SelectContent>
        </Select>
      </template>
    </FilterBar>

    <!-- Content -->
    <div class="flex-1 overflow-auto">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <div v-else-if="filteredAttempts.length === 0" class="py-12 text-center px-6">
        <CheckCircle2 class="h-12 w-12 text-green-500 mx-auto mb-3" />
        <p class="text-muted-foreground">No recovery attempts found</p>
        <p class="text-sm text-muted-foreground mt-1">Recovery attempts will appear here when errors are detected.</p>
      </div>

      <!-- Accordion Table -->
      <div v-else class="rounded-lg border border-border bg-card">
        <!-- Table Header -->
        <div class="grid grid-cols-12 gap-2 px-4 py-3 bg-muted/50 border-b border-border text-xs font-medium text-muted-foreground uppercase tracking-wide">
          <div class="col-span-3">Domain</div>
          <div class="col-span-2">Error</div>
          <div class="col-span-3">Reason</div>
          <div class="col-span-1">Conf</div>
          <div class="col-span-2">Status</div>
          <div class="col-span-1">Time</div>
        </div>

        <!-- Accordion Rows -->
        <Accordion type="single" collapsible class="w-full">
          <AccordionItem 
            v-for="attempt in filteredAttempts" 
            :key="attempt.id" 
            :value="attempt.id"
            class="border-b border-border last:border-b-0"
          >
            <AccordionTrigger class="px-4 py-3 hover:bg-muted/30 transition-colors [&[data-state=open]]:bg-muted/30">
              <div class="grid grid-cols-12 gap-2 w-full text-left items-center">
                <!-- Domain -->
                <div class="col-span-3 min-w-0">
                  <div class="font-medium text-sm truncate">{{ attempt.domain || 'Unknown' }}</div>
                  <div class="text-xs text-muted-foreground truncate">{{ attempt.url }}</div>
                </div>
                <!-- Error -->
                <div class="col-span-2">
                  <Badge variant="outline" class="text-xs font-medium">
                    {{ formatErrorPattern(attempt.error_pattern) }}
                  </Badge>
                </div>
                <!-- Reason -->
                <div class="col-span-3">
                  <div class="text-xs text-muted-foreground truncate" :title="attempt.trigger_reason">
                    {{ attempt.trigger_reason || '-' }}
                  </div>
                </div>
                <!-- Confidence -->
                <div class="col-span-1">
                  <div v-if="attempt.confidence" class="text-sm font-medium" :class="{
                    'text-green-600': attempt.confidence >= 0.8,
                    'text-yellow-600': attempt.confidence >= 0.5 && attempt.confidence < 0.8,
                    'text-red-600': attempt.confidence < 0.5
                  }">
                    {{ (attempt.confidence * 100).toFixed(0) }}%
                  </div>
                  <span v-else class="text-muted-foreground text-xs">-</span>
                </div>
                <!-- Status -->
                <div class="col-span-2">
                  <Badge variant="outline" :class="getStatusColor(attempt.status)" class="text-xs font-medium capitalize">
                    <div class="w-1.5 h-1.5 rounded-full mr-1.5" :class="getStatusDot(attempt.status)"></div>
                    {{ attempt.status }}
                  </Badge>
                </div>
                <!-- Time -->
                <div class="col-span-1 text-sm text-muted-foreground">
                  {{ formatDate(attempt.created_at) }}
                </div>
              </div>
            </AccordionTrigger>
            <AccordionContent class="px-4 pb-4">
              <div class="bg-muted/20 rounded-lg p-4 mt-2">
                <div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                  <div>
                    <div class="text-muted-foreground text-xs mb-1">Action</div>
                    <div class="font-medium">{{ formatAction(attempt.action, attempt.status) }}</div>
                  </div>
                  <div>
                    <div class="text-muted-foreground text-xs mb-1">Proxy ID</div>
                    <div class="font-mono text-xs">{{ attempt.proxy_id || '-' }}</div>
                  </div>
                  <div>
                    <div class="text-muted-foreground text-xs mb-1">Proxy Tier</div>
                    <div v-if="attempt.proxy_tier > 0">
                      <Badge variant="outline" :class="getTierColor(attempt.proxy_tier)" class="text-xs">
                        {{ getTierLabel(attempt.proxy_tier) }}
                      </Badge>
                    </div>
                    <div v-else class="text-muted-foreground">-</div>
                  </div>
                  <div>
                    <div class="text-muted-foreground text-xs mb-1">Tier Escalation</div>
                    <div v-if="attempt.tier_from && attempt.tier_to">
                      <Badge variant="outline" class="text-xs bg-blue-500/10 text-blue-600">
                        {{ getTierLabel(attempt.tier_from) }} → {{ getTierLabel(attempt.tier_to) }}
                      </Badge>
                    </div>
                    <div v-else class="text-muted-foreground">-</div>
                  </div>
                  <div>
                    <div class="text-muted-foreground text-xs mb-1">Retry Count</div>
                    <div>{{ attempt.retry_count || 0 }}</div>
                  </div>
                  <div>
                    <div class="text-muted-foreground text-xs mb-1">Retry Delay</div>
                    <div>{{ attempt.retry_delay_ms ? `${attempt.retry_delay_ms}ms` : '-' }}</div>
                  </div>
                  <div>
                    <div class="text-muted-foreground text-xs mb-1">Duration</div>
                    <div>{{ attempt.duration_ms ? `${attempt.duration_ms}ms` : '-' }}</div>
                  </div>
                  <div>
                    <div class="text-muted-foreground text-xs mb-1">Source</div>
                    <div class="capitalize">{{ attempt.source || '-' }}</div>
                  </div>
                </div>
                <div class="mt-3 pt-3 border-t border-border">
                  <div class="text-muted-foreground text-xs mb-1">Full URL</div>
                  <div class="text-xs font-mono break-all">{{ attempt.url }}</div>
                </div>
                <div v-if="attempt.error_message" class="mt-3 pt-3 border-t border-border">
                  <div class="text-muted-foreground text-xs mb-1">Error Message</div>
                  <div class="text-xs font-mono text-red-500">{{ attempt.error_message }}</div>
                </div>
              </div>
            </AccordionContent>
          </AccordionItem>
        </Accordion>
      </div>
    </div>
  </PageLayout>
</template>
