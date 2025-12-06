<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { 
  getIncidents, 
  getIncidentStats, 
  resolveIncident,
  updateIncidentStatus,
  type Incident, 
  type IncidentStats 
} from '@/api/incidents'
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
  AlertTriangle, 
  CheckCircle2, 
  Clock, 
  Eye, 
  Loader2, 
  SlidersHorizontal,
  XCircle,
  AlertCircle,
  ExternalLink
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const router = useRouter()

const loading = ref(true)
const incidents = ref<Incident[]>([])
const incidentStats = ref<IncidentStats | null>(null)
const totalIncidents = ref(0)
const statusFilter = ref<string>('all')
const priorityFilter = ref<string>('all')
const searchQuery = ref('')
const resolvingIds = ref<Set<string>>(new Set())

const tableColumns = [
  { key: 'domain', label: 'Domain', sortable: true, align: 'left' as const },
  { key: 'error_pattern', label: 'Error Pattern', align: 'left' as const },
  { key: 'status', label: 'Status', align: 'left' as const },
  { key: 'priority', label: 'Priority', align: 'left' as const },
  { key: 'created_at', label: 'Created', align: 'left' as const },
  { key: 'actions', label: 'Actions', align: 'right' as const }
]

const stats = computed(() => {
  if (!incidentStats.value) return []
  const s = incidentStats.value
  return [
    { label: 'Total', value: s.total },
    { label: 'Open', value: s.by_status?.open || 0, color: 'text-red-600 dark:text-red-400' },
    { label: 'In Progress', value: s.by_status?.in_progress || 0, color: 'text-yellow-600 dark:text-yellow-400' },
    { label: 'Resolved', value: s.by_status?.resolved || 0, color: 'text-green-600 dark:text-green-400' }
  ]
})

const filteredIncidents = computed(() => {
  let result = incidents.value
  
  if (statusFilter.value !== 'all') {
    result = result.filter(i => i.status === statusFilter.value)
  }

  if (priorityFilter.value !== 'all') {
    result = result.filter(i => i.priority === priorityFilter.value)
  }  
  
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(i => 
      i.domain.toLowerCase().includes(query) || 
      i.url.toLowerCase().includes(query) ||
      i.error_pattern.toLowerCase().includes(query)
    )
  }
  
  return result
})

const fetchData = async () => {
  loading.value = true
  try {
    const [incidentsRes, statsRes] = await Promise.all([
      getIncidents({ limit: 100 }),
      getIncidentStats()
    ])
    incidents.value = incidentsRes.incidents || []
    totalIncidents.value = incidentsRes.total
    incidentStats.value = statsRes
  } catch (error) {
    console.error('Failed to fetch incidents:', error)
    toast.error('Failed to load incidents')
  } finally {
    loading.value = false
  }
}

const handleViewDetails = (incident: Incident) => {
  router.push(`/incidents/${incident.id}`)
}

const handleResolve = async (incident: Incident) => {
  if (resolvingIds.value.has(incident.id)) return
  
  resolvingIds.value.add(incident.id)
  try {
    await resolveIncident(incident.id, 'Manually resolved from dashboard')
    toast.success('Incident resolved')
    await fetchData()
  } catch (error) {
    console.error('Failed to resolve incident:', error)
    toast.error('Failed to resolve incident')
  } finally {
    resolvingIds.value.delete(incident.id)
  }
}

const handleIgnore = async (incident: Incident) => {
  try {
    await updateIncidentStatus(incident.id, 'ignored')
    toast.success('Incident ignored')
    await fetchData()
  } catch (error) {
    console.error('Failed to ignore incident:', error)
    toast.error('Failed to ignore incident')
  }
}

const getStatusColor = (status: string) => {
  switch(status) {
    case 'open': return 'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20'
    case 'in_progress': return 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 border-yellow-500/20'
    case 'resolved': return 'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20'
    case 'ignored': return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
    default: return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
  }
}

const getPriorityColor = (priority: string) => {
  switch(priority) {
    case 'critical': return 'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20'
    case 'high': return 'bg-orange-500/10 text-orange-600 dark:text-orange-400 border-orange-500/20'
    case 'medium': return 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 border-yellow-500/20'
    case 'low': return 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20'
    default: return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
  }
}

const getPriorityIcon = (priority: string) => {
  switch(priority) {
    case 'critical': return AlertTriangle
    case 'high': return AlertCircle
    default: return Clock
  }
}

const formatDate = (dateString?: string) => {
  if (!dateString) return 'N/A'
  const date = new Date(dateString)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const hours = Math.floor(diff / (1000 * 60 * 60))
  
  if (hours < 1) return 'Just now'
  if (hours < 24) return `${hours}h ago`
  
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}d ago`
  
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

const formatErrorPattern = (pattern: string) => {
  return pattern.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}

onMounted(fetchData)
</script>

<template>
  <PageLayout>
    <!-- Header -->
    <PageHeader 
      title="Incident Investigation" 
      description="Review and resolve crawling incidents that require human attention"
      :show-help-icon="true"
    />

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
            <SelectItem value="open">Open</SelectItem>
            <SelectItem value="in_progress">In Progress</SelectItem>
            <SelectItem value="resolved">Resolved</SelectItem>
            <SelectItem value="ignored">Ignored</SelectItem>
          </SelectContent>
        </Select>

        <Select v-model="priorityFilter">
          <SelectTrigger class="w-[140px] h-9">
            <SelectValue placeholder="Priority" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Priority</SelectItem>
            <SelectItem value="critical">Critical</SelectItem>
            <SelectItem value="high">High</SelectItem>
            <SelectItem value="medium">Medium</SelectItem>
            <SelectItem value="low">Low</SelectItem>
          </SelectContent>
        </Select>
      </template>
    </FilterBar>

    <!-- Table -->
    <div class="flex-1 overflow-auto">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <div v-else-if="filteredIncidents.length === 0" class="py-12 text-center px-6">
        <CheckCircle2 class="h-12 w-12 text-green-500 mx-auto mb-3" />
        <p class="text-muted-foreground">No incidents found</p>
        <p class="text-sm text-muted-foreground mt-1">All clear! No issues requiring attention.</p>
      </div>

      <DataTable
        v-else
        :data="filteredIncidents"
        :columns="tableColumns"
        :on-row-click="handleViewDetails"
      >
        <template #row="{ row }">
          <td class="px-6 py-3">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg bg-red-500/10 flex items-center justify-center shrink-0">
                <component :is="getPriorityIcon(row.priority)" class="w-5 h-5 text-red-500" />
              </div>
              <div class="min-w-0">
                <div class="font-medium text-sm truncate">{{ row.domain }}</div>
                <div class="text-xs text-muted-foreground truncate max-w-[200px]">{{ row.url }}</div>
              </div>
            </div>
          </td>
          <td class="px-6 py-3">
            <Badge variant="outline" class="text-xs font-medium">
              {{ formatErrorPattern(row.error_pattern) }}
            </Badge>
          </td>
          <td class="px-6 py-3">
            <Badge 
              variant="outline"
              :class="getStatusColor(row.status)"
              class="text-xs font-medium capitalize"
            >
              <div class="w-1.5 h-1.5 rounded-full mr-1.5" :class="{
                'bg-red-500': row.status === 'open',
                'bg-yellow-500': row.status === 'in_progress',
                'bg-green-500': row.status === 'resolved',
                'bg-gray-500': row.status === 'ignored'
              }"></div>
              {{ row.status.replace('_', ' ') }}
            </Badge>
          </td>
          <td class="px-6 py-3">
            <Badge 
              variant="outline"
              :class="getPriorityColor(row.priority)"
              class="text-xs font-medium capitalize"
            >
              {{ row.priority }}
            </Badge>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm text-muted-foreground">
              {{ formatDate(row.created_at) }}
            </div>
          </td>
          <td class="px-6 py-3 text-right" @click.stop>
            <div class="flex items-center justify-end gap-1">
              <Button 
                @click="handleViewDetails(row)"
                size="sm"
                variant="ghost"
                class="h-8 w-8 p-0"
                title="View Details"
              >
                <Eye class="h-4 w-4" />
              </Button>
              <Button 
                v-if="row.status === 'open' || row.status === 'in_progress'"
                @click="handleResolve(row)"
                size="sm"
                variant="ghost"
                class="h-8 w-8 p-0 text-green-600 hover:text-green-600"
                :disabled="resolvingIds.has(row.id)"
                title="Resolve"
              >
                <Loader2 v-if="resolvingIds.has(row.id)" class="h-4 w-4 animate-spin" />
                <CheckCircle2 v-else class="h-4 w-4" />
              </Button>
              <Button 
                v-if="row.status === 'open'"
                @click="handleIgnore(row)"
                size="sm"
                variant="ghost"
                class="h-8 w-8 p-0 text-muted-foreground hover:text-muted-foreground"
                title="Ignore"
              >
                <XCircle class="h-4 w-4" />
              </Button>
              <a 
                :href="row.url" 
                target="_blank" 
                @click.stop
                class="inline-flex items-center justify-center h-8 w-8 rounded-md hover:bg-accent"
                title="Open URL"
              >
                <ExternalLink class="h-4 w-4 text-muted-foreground" />
              </a>
            </div>
          </td>
        </template>
      </DataTable>
    </div>
  </PageLayout>
</template>
