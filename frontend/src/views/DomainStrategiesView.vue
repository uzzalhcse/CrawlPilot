<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { 
  getDomainStrategies, 
  clearDomainStrategy, 
  clearAllDomainStrategies,
  createDomainStrategy,
  updateDomainStrategy,
  type DomainStrategy, 
  type DomainStrategyStats 
} from '@/api/domainStrategies'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { 
  Brain, 
  Loader2, 
  RefreshCw,
  Trash2,
  Zap,
  Server,
  Home,
  Smartphone,
  CheckCircle2,
  Pencil,
  Plus
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const loading = ref(true)
const strategies = ref<DomainStrategy[]>([])
const stats = ref<DomainStrategyStats | null>(null)
const totalStrategies = ref(0)
const statusFilter = ref<string>('all')
const searchQuery = ref('')
const clearingDomain = ref<string | null>(null)
let refreshInterval: ReturnType<typeof setInterval> | null = null

// Edit/Create dialog state
const showEditDialog = ref(false)
const editingStrategy = ref<DomainStrategy | null>(null)
const editForm = ref({
  domain: '',
  recommended_tier: '1',
  adaptive_delay_ms: 0,
  session_requirement: '0'
})
const saving = ref(false)

const tableColumns = [
  { key: 'domain', label: 'Domain', sortable: true, align: 'left' as const },
  { key: 'recommended_tier', label: 'Tier', align: 'left' as const },
  { key: 'success_rate', label: 'Success Rate', align: 'left' as const },
  { key: 'sample_size', label: 'Samples', align: 'left' as const },
  { key: 'learning_status', label: 'Status', align: 'left' as const },
  { key: 'updated_at', label: 'Updated', align: 'left' as const },
  { key: 'actions', label: '', align: 'right' as const }
]

const statsDisplay = computed(() => {
  if (!stats.value) return []
  const s = stats.value
  return [
    { label: 'Total Domains', value: s.total, icon: Brain },
    { label: 'Stable', value: s.stable, color: 'text-green-600 dark:text-green-400' },
    { label: 'Learning', value: s.learning, color: 'text-blue-600 dark:text-blue-400' },
    { label: 'Needs Review', value: s.needs_review, color: 'text-yellow-600 dark:text-yellow-400' },
    { label: 'Avg Success', value: `${s.avg_success_rate?.toFixed(1) || 0}%`, color: 'text-purple-600 dark:text-purple-400' }
  ]
})

const filteredStrategies = computed(() => {
  let result = strategies.value
  
  if (statusFilter.value !== 'all') {
    result = result.filter(s => s.learning_status === statusFilter.value)
  }
  
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(s => s.domain?.toLowerCase().includes(query))
  }
  
  return result
})

const isCreating = computed(() => editingStrategy.value === null)

const fetchData = async () => {
  loading.value = true
  try {
    const res = await getDomainStrategies({ limit: 200 })
    strategies.value = res.strategies || []
    totalStrategies.value = res.total
    stats.value = res.stats
  } catch (error) {
    console.error('Failed to fetch domain strategies:', error)
    toast.error('Failed to load domain strategies')
  } finally {
    loading.value = false
  }
}

const handleRefresh = async () => {
  await fetchData()
  toast.success('Data refreshed')
}

const handleClearDomain = async (domain: string) => {
  clearingDomain.value = domain
  try {
    await clearDomainStrategy(domain)
    toast.success(`Cleared learning for ${domain}`)
    await fetchData()
  } catch (error) {
    toast.error('Failed to clear domain')
  } finally {
    clearingDomain.value = null
  }
}

const handleClearAllWithConfirm = async () => {
  if (!confirm('Clear all domain learning? This will reset all tier strategies. Next crawls will start from Tier 0.')) {
    return
  }
  try {
    const result = await clearAllDomainStrategies()
    toast.success(`Cleared ${result.deleted} domains`)
    await fetchData()
  } catch (error) {
    toast.error('Failed to clear all domains')
  }
}

// Edit/Create handlers
const openCreateDialog = () => {
  editingStrategy.value = null
  editForm.value = {
    domain: '',
    recommended_tier: '1',
    adaptive_delay_ms: 0,
    session_requirement: '0'
  }
  showEditDialog.value = true
}

const openEditDialog = (strategy: DomainStrategy) => {
  editingStrategy.value = strategy
  editForm.value = {
    domain: strategy.domain,
    recommended_tier: String(strategy.recommended_tier),
    adaptive_delay_ms: strategy.adaptive_delay_ms || 0,
    session_requirement: String(strategy.session_requirement || 0)
  }
  showEditDialog.value = true
}

const saveStrategy = async () => {
  saving.value = true
  try {
    if (isCreating.value) {
      await createDomainStrategy({
        domain: editForm.value.domain,
        recommended_tier: parseInt(editForm.value.recommended_tier)
      })
      toast.success(`Created strategy for ${editForm.value.domain}`)
    } else {
      await updateDomainStrategy(editingStrategy.value!.domain, {
        recommended_tier: parseInt(editForm.value.recommended_tier),
        adaptive_delay_ms: editForm.value.adaptive_delay_ms,
        session_requirement: parseInt(editForm.value.session_requirement)
      })
      toast.success(`Updated strategy for ${editingStrategy.value!.domain}`)
    }
    showEditDialog.value = false
    await fetchData()
  } catch (error) {
    toast.error(isCreating.value ? 'Failed to create strategy' : 'Failed to update strategy')
  } finally {
    saving.value = false
  }
}

// Quick tier change handler
const handleQuickTierChange = async (domain: string, newTier: string) => {
  try {
    await updateDomainStrategy(domain, { recommended_tier: parseInt(newTier) })
    toast.success(`Updated tier for ${domain}`)
    await fetchData()
  } catch (error) {
    toast.error('Failed to update tier')
  }
}

const getTierIcon = (tier: number) => {
  switch(tier) {
    case 0: return Home
    case 1: return Server
    case 2: return Zap
    case 3: return Smartphone
    default: return Server
  }
}

const getTierLabel = (tier: number) => {
  switch(tier) {
    case 0: return 'Direct'
    case 1: return 'Datacenter'
    case 2: return 'Residential'
    case 3: return 'Mobile'
    default: return `Tier ${tier}`
  }
}

const getTierColor = (tier: number) => {
  switch(tier) {
    case 0: return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
    case 1: return 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20'
    case 2: return 'bg-purple-500/10 text-purple-600 dark:text-purple-400 border-purple-500/20'
    case 3: return 'bg-orange-500/10 text-orange-600 dark:text-orange-400 border-orange-500/20'
    default: return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
  }
}

const getStatusColor = (status: string) => {
  switch(status) {
    case 'stable': return 'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20'
    case 'learning': return 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20'
    case 'new': return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
    case 'needs_review': return 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 border-yellow-500/20'
    default: return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
  }
}

const getSuccessRateColor = (rate: number) => {
  if (rate >= 0.9) return 'text-green-600 dark:text-green-400'
  if (rate >= 0.7) return 'text-yellow-600 dark:text-yellow-400'
  return 'text-red-600 dark:text-red-400'
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

onMounted(() => {
  fetchData()
  refreshInterval = setInterval(fetchData, 30000)
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
      title="Domain Strategies" 
      description="Learned anti-bot bypass strategies per domain"
      :show-help-icon="true"
    >
      <template #actions>
        <Button 
          variant="outline" 
          size="sm" 
          class="gap-2 text-red-600 hover:text-red-700"
          @click="handleClearAllWithConfirm"
        >
          <Trash2 class="w-4 h-4" />
          Clear All
        </Button>
        <Button @click="handleRefresh" variant="outline" size="sm" class="gap-2">
          <RefreshCw class="w-4 h-4" />
          Refresh
        </Button>
        <Button @click="openCreateDialog" variant="default" size="sm" class="gap-2">
          <Plus class="w-4 h-4" />
          Add Domain
        </Button>
      </template>
    </PageHeader>

    <!-- Stats -->
    <StatsBar :stats="statsDisplay" />

    <!-- Filters -->
    <FilterBar 
      search-placeholder="Search by domain..." 
      :search-value="searchQuery"
      @update:search-value="searchQuery = $event"
    >
      <template #filters>
        <Select v-model="statusFilter">
          <SelectTrigger class="w-[140px] h-9">
            <div class="flex items-center gap-2">
              <Brain class="w-4 h-4" />
              <SelectValue placeholder="Status" />
            </div>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Status</SelectItem>
            <SelectItem value="stable">Stable</SelectItem>
            <SelectItem value="learning">Learning</SelectItem>
            <SelectItem value="new">New</SelectItem>
            <SelectItem value="needs_review">Needs Review</SelectItem>
          </SelectContent>
        </Select>
      </template>
    </FilterBar>

    <!-- Table -->
    <div class="flex-1 overflow-auto">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <div v-else-if="filteredStrategies.length === 0" class="py-12 text-center px-6">
        <CheckCircle2 class="h-12 w-12 text-green-500 mx-auto mb-3" />
        <p class="text-muted-foreground">No domain strategies found</p>
        <p class="text-sm text-muted-foreground mt-1">Strategies are learned automatically when crawling protected sites.</p>
      </div>

      <DataTable
        v-else
        :data="filteredStrategies"
        :columns="tableColumns"
      >
        <template #row="{ row }">
          <td class="px-6 py-3">
            <div class="min-w-0">
              <div class="font-medium text-sm truncate max-w-[250px]">{{ row.domain }}</div>
              <div class="text-xs text-muted-foreground">
                {{ row.total_requests.toLocaleString() }} requests
              </div>
            </div>
          </td>
          <td class="px-6 py-3">
            <Select :model-value="String(row.recommended_tier)" @update:model-value="(val) => val && handleQuickTierChange(row.domain, String(val))">
              <SelectTrigger class="w-[130px] h-8">
                <div class="flex items-center gap-2">
                  <component :is="getTierIcon(row.recommended_tier)" class="w-3 h-3" />
                  <span class="text-xs">{{ getTierLabel(row.recommended_tier) }}</span>
                </div>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="0">Direct</SelectItem>
                <SelectItem value="1">Datacenter</SelectItem>
                <SelectItem value="2">Residential</SelectItem>
                <SelectItem value="3">Mobile</SelectItem>
              </SelectContent>
            </Select>
          </td>
          <td class="px-6 py-3">
            <div class="flex items-center gap-2">
              <div class="w-16 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                <div 
                  class="h-full transition-all" 
                  :class="row.success_rate >= 0.9 ? 'bg-green-500' : row.success_rate >= 0.7 ? 'bg-yellow-500' : 'bg-red-500'"
                  :style="{ width: `${row.success_rate * 100}%` }"
                ></div>
              </div>
              <span :class="getSuccessRateColor(row.success_rate)" class="text-sm font-medium">
                {{ (row.success_rate * 100).toFixed(0) }}%
              </span>
            </div>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm">{{ row.sample_size.toLocaleString() }}</div>
          </td>
          <td class="px-6 py-3">
            <Badge 
              variant="outline"
              :class="getStatusColor(row.learning_status)"
              class="text-xs font-medium capitalize"
            >
              <div class="w-1.5 h-1.5 rounded-full mr-1.5" :class="{
                'bg-green-500': row.learning_status === 'stable',
                'bg-blue-500': row.learning_status === 'learning',
                'bg-gray-400': row.learning_status === 'new',
                'bg-yellow-500': row.learning_status === 'needs_review'
              }"></div>
              {{ row.learning_status.replace('_', ' ') }}
            </Badge>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm text-muted-foreground">
              {{ formatDate(row.updated_at) }}
            </div>
          </td>
          <td class="px-6 py-3 text-right">
            <div class="flex items-center justify-end gap-1">
              <Button 
                variant="ghost" 
                size="sm" 
                class="hover:bg-muted"
                @click="openEditDialog(row)"
              >
                <Pencil class="w-4 h-4" />
              </Button>
              <Button 
                variant="ghost" 
                size="sm" 
                class="text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-950/20"
                @click="handleClearDomain(row.domain)"
                :disabled="clearingDomain === row.domain"
              >
                <Loader2 v-if="clearingDomain === row.domain" class="w-4 h-4 animate-spin" />
                <Trash2 v-else class="w-4 h-4" />
              </Button>
            </div>
          </td>
        </template>
      </DataTable>
    </div>
  </PageLayout>

  <!-- Edit/Create Dialog -->
  <Dialog v-model:open="showEditDialog">
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>{{ isCreating ? 'Add Domain Strategy' : 'Edit Domain Strategy' }}</DialogTitle>
        <DialogDescription>
          {{ isCreating ? 'Pre-configure proxy tier for a domain before crawling.' : 'Modify the proxy strategy for this domain.' }}
        </DialogDescription>
      </DialogHeader>

      <div class="grid gap-4 py-4">
        <!-- Domain (only editable when creating) -->
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="domain" class="text-right">Domain</Label>
          <Input
            id="domain"
            v-model="editForm.domain"
            :disabled="!isCreating"
            placeholder="example.com"
            class="col-span-3"
          />
        </div>

        <!-- Recommended Tier -->
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="tier" class="text-right">Proxy Tier</Label>
          <Select v-model="editForm.recommended_tier">
            <SelectTrigger class="col-span-3">
              <SelectValue placeholder="Select tier" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="0">Tier 0 - Direct (No Proxy)</SelectItem>
              <SelectItem value="1">Tier 1 - Datacenter</SelectItem>
              <SelectItem value="2">Tier 2 - Residential</SelectItem>
              <SelectItem value="3">Tier 3 - Mobile</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <!-- Adaptive Delay (for editing) -->
        <div v-if="!isCreating" class="grid grid-cols-4 items-center gap-4">
          <Label for="delay" class="text-right">Delay (ms)</Label>
          <Input
            id="delay"
            v-model.number="editForm.adaptive_delay_ms"
            type="number"
            min="0"
            placeholder="0"
            class="col-span-3"
          />
        </div>

        <!-- Session Requirement (for editing) -->
        <div v-if="!isCreating" class="grid grid-cols-4 items-center gap-4">
          <Label for="session" class="text-right">Session</Label>
          <Select v-model="editForm.session_requirement">
            <SelectTrigger class="col-span-3">
              <SelectValue placeholder="Session requirement" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="0">Unknown</SelectItem>
              <SelectItem value="1">Required</SelectItem>
              <SelectItem value="2">Optional</SelectItem>
              <SelectItem value="3">Not Needed</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="showEditDialog = false">Cancel</Button>
        <Button @click="saveStrategy" :disabled="saving || (isCreating && !editForm.domain)">
          <Loader2 v-if="saving" class="w-4 h-4 mr-2 animate-spin" />
          {{ isCreating ? 'Create' : 'Save Changes' }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
