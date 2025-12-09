<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { schedulesApi, type Schedule, type CronPreset } from '@/api/schedules'
import { workflowsApi } from '@/api/workflows'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import DataTable from '@/components/ui/data-table.vue'
import PageLayout from '@/components/layout/PageLayout.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import FilterBar from '@/components/layout/FilterBar.vue'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { ScrollArea } from '@/components/ui/scroll-area'
import { 
  Calendar,
  Plus,
  Loader2, 
  RefreshCw,
  Trash2,
  Edit,
  PlayCircle,
  Clock,
  Pause,
  Zap,
  Timer,
  Search,
  Sparkles,
  Check
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const router = useRouter()

const loading = ref(true)
const schedules = ref<Schedule[]>([])
const workflows = ref<{ id: string; name: string }[]>([])
const presets = ref<CronPreset[]>([])
const searchQuery = ref('')
const statusFilter = ref('all')
const togglingIds = ref<Set<string>>(new Set())
const runningIds = ref<Set<string>>(new Set())

// Dialog state
const dialogOpen = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const dialogLoading = ref(false)
const editingScheduleId = ref<string | null>(null)

// Workflow search state
const workflowSearchQuery = ref('')
const showWorkflowDropdown = ref(false)

const formData = ref({
  workflow_id: '',
  name: '',
  cron_expression: '',
  timezone: 'UTC'
})

const timezones = [
  { value: 'UTC', label: 'UTC', offset: '+00:00' },
  { value: 'America/New_York', label: 'New York', offset: '-05:00' },
  { value: 'America/Los_Angeles', label: 'Los Angeles', offset: '-08:00' },
  { value: 'Europe/London', label: 'London', offset: '+00:00' },
  { value: 'Europe/Paris', label: 'Paris', offset: '+01:00' },
  { value: 'Asia/Tokyo', label: 'Tokyo', offset: '+09:00' },
  { value: 'Asia/Shanghai', label: 'Shanghai', offset: '+08:00' },
  { value: 'Asia/Singapore', label: 'Singapore', offset: '+08:00' }
]

const tableColumns = [
  { key: 'workflow', label: 'Workflow', sortable: true, align: 'left' as const },
  { key: 'schedule', label: 'Schedule', align: 'left' as const },
  { key: 'timing', label: 'Timing', align: 'left' as const },
  { key: 'status', label: 'Status', align: 'center' as const },
  { key: 'next_run', label: 'Next Run', align: 'left' as const },
  { key: 'last_run', label: 'Last Run', align: 'left' as const },
  { key: 'actions', label: 'Actions', align: 'right' as const }
]

const stats = computed(() => {
  const active = schedules.value.filter(s => s.is_enabled).length
  const paused = schedules.value.filter(s => !s.is_enabled).length
  
  return [
    { label: 'Total Schedules', value: schedules.value.length, icon: Calendar, color: '' },
    { label: 'Active', value: active, icon: Zap, color: 'text-emerald-600 dark:text-emerald-400' },
    { label: 'Paused', value: paused, icon: Pause, color: 'text-amber-600 dark:text-amber-400' }
  ]
})

const filteredSchedules = computed(() => {
  let result = schedules.value
  if (statusFilter.value === 'active') result = result.filter(s => s.is_enabled)
  else if (statusFilter.value === 'paused') result = result.filter(s => !s.is_enabled)
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(s => 
      s.name.toLowerCase().includes(query) ||
      s.workflow_name?.toLowerCase().includes(query) ||
      s.cron_expression.toLowerCase().includes(query)
    )
  }
  return result
})

const filteredWorkflows = computed(() => {
  if (!workflowSearchQuery.value) return workflows.value
  const query = workflowSearchQuery.value.toLowerCase()
  return workflows.value.filter(w => w.name.toLowerCase().includes(query))
})

const selectedWorkflowName = computed(() => {
  const wf = workflows.value.find(w => w.id === formData.value.workflow_id)
  return wf?.name || ''
})

const fetchData = async () => {
  loading.value = true
  try {
    const [schedulesRes, workflowsRes, presetsRes] = await Promise.all([
      schedulesApi.list({ limit: 100 }),
      workflowsApi.list({ limit: 100 }),
      schedulesApi.getPresets()
    ])
    schedules.value = schedulesRes.data.schedules
    workflows.value = workflowsRes.data.workflows.map(w => ({ id: w.id, name: w.name }))
    presets.value = presetsRes.data.presets
  } catch (error) {
    console.error('Failed to fetch schedules:', error)
    toast.error('Failed to load schedules')
  } finally {
    loading.value = false
  }
}

const openCreateDialog = () => {
  dialogMode.value = 'create'
  editingScheduleId.value = null
  workflowSearchQuery.value = ''
  showWorkflowDropdown.value = false
  formData.value = { workflow_id: '', name: '', cron_expression: '', timezone: 'UTC' }
  dialogOpen.value = true
}

const openEditDialog = (schedule: Schedule) => {
  dialogMode.value = 'edit'
  editingScheduleId.value = schedule.id
  workflowSearchQuery.value = ''
  showWorkflowDropdown.value = false
  formData.value = {
    workflow_id: schedule.workflow_id,
    name: schedule.name,
    cron_expression: schedule.cron_expression,
    timezone: schedule.timezone || 'UTC'
  }
  dialogOpen.value = true
}

const handlePresetSelect = (preset: CronPreset) => {
  formData.value.cron_expression = preset.expression
}

const selectWorkflow = (workflowId: string) => {
  formData.value.workflow_id = workflowId
  showWorkflowDropdown.value = false
  workflowSearchQuery.value = ''
}

const handleSave = async () => {
  if (!formData.value.workflow_id || !formData.value.name || !formData.value.cron_expression) {
    toast.error('Please fill in all required fields')
    return
  }
  dialogLoading.value = true
  try {
    if (dialogMode.value === 'create') {
      await schedulesApi.create(formData.value)
      toast.success('Schedule created successfully')
    } else {
      await schedulesApi.update(editingScheduleId.value!, {
        name: formData.value.name,
        cron_expression: formData.value.cron_expression,
        timezone: formData.value.timezone
      })
      toast.success('Schedule updated successfully')
    }
    dialogOpen.value = false
    await fetchData()
  } catch (error: any) {
    console.error('Failed to save schedule:', error)
    toast.error(error.response?.data?.error || 'Failed to save schedule')
  } finally {
    dialogLoading.value = false
  }
}

const handleToggle = async (schedule: Schedule) => {
  if (togglingIds.value.has(schedule.id)) return
  togglingIds.value.add(schedule.id)
  try {
    await schedulesApi.toggle(schedule.id)
    schedule.is_enabled = !schedule.is_enabled
    toast.success(schedule.is_enabled ? 'Schedule enabled' : 'Schedule paused')
  } catch (error) {
    console.error('Failed to toggle schedule:', error)
    toast.error('Failed to toggle schedule')
  } finally {
    togglingIds.value.delete(schedule.id)
  }
}

const handleRunNow = async (schedule: Schedule) => {
  if (runningIds.value.has(schedule.id)) return
  runningIds.value.add(schedule.id)
  try {
    const res = await schedulesApi.runNow(schedule.id)
    toast.success('Execution started')
    router.push(`/executions/${res.data.execution_id}`)
  } catch (error: any) {
    console.error('Failed to run schedule:', error)
    toast.error(error.response?.data?.error || 'Failed to run schedule')
  } finally {
    runningIds.value.delete(schedule.id)
  }
}

const handleDelete = async (schedule: Schedule) => {
  if (!confirm(`Are you sure you want to delete "${schedule.name}"?`)) return
  try {
    await schedulesApi.delete(schedule.id)
    toast.success('Schedule deleted')
    await fetchData()
  } catch (error) {
    console.error('Failed to delete schedule:', error)
    toast.error('Failed to delete schedule')
  }
}

const formatDate = (dateString?: string) => {
  if (!dateString) return 'Never'
  const date = new Date(dateString)
  if (isNaN(date.getTime())) return 'Invalid'
  const now = new Date()
  const diff = date.getTime() - now.getTime()
  const absMinutes = Math.abs(Math.floor(diff / (1000 * 60)))
  const absHours = Math.floor(absMinutes / 60)
  const absDays = Math.floor(absHours / 24)
  if (diff > 0) {
    if (absMinutes < 60) return `in ${absMinutes}m`
    if (absHours < 24) return `in ${absHours}h`
    return `in ${absDays}d`
  } else {
    if (absMinutes < 60) return `${absMinutes}m ago`
    if (absHours < 24) return `${absHours}h ago`
    return `${absDays}d ago`
  }
}

const getCronPresetName = (cron: string) => {
  const preset = presets.value.find(p => p.expression === cron)
  return preset?.name || null
}

onMounted(fetchData)
</script>

<template>
  <PageLayout>
    <PageHeader 
      title="Scheduled Workflows" 
      description="Automate workflow executions with cron-based scheduling"
      :show-help-icon="true"
    >
      <template #actions>
        <div class="flex items-center gap-2">
          <Button @click="fetchData" variant="outline" size="sm" :disabled="loading">
            <RefreshCw class="w-4 h-4 mr-2" :class="{ 'animate-spin': loading }" />
            Refresh
          </Button>
          <Button @click="openCreateDialog" size="sm" class="bg-gradient-to-r from-primary to-primary/80 shadow-lg shadow-primary/20">
            <Plus class="w-4 h-4 mr-2" />
            New Schedule
          </Button>
        </div>
      </template>
    </PageHeader>

    <!-- Stats Cards -->
    <div class="px-6 py-4">
      <div class="grid grid-cols-3 gap-4">
        <div v-for="stat in stats" :key="stat.label" 
             class="relative rounded-xl border bg-card/50 p-4 hover:shadow-lg hover:border-primary/20 transition-all">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-xs font-medium text-muted-foreground uppercase tracking-wider">{{ stat.label }}</p>
              <p class="text-2xl font-bold mt-1" :class="stat.color">{{ stat.value }}</p>
            </div>
            <div class="h-12 w-12 rounded-xl bg-gradient-to-br from-primary/10 to-primary/5 flex items-center justify-center">
              <component :is="stat.icon" class="w-6 h-6 text-primary" />
            </div>
          </div>
        </div>
      </div>
    </div>

    <FilterBar search-placeholder="Search schedules..." :search-value="searchQuery" @update:search-value="searchQuery = $event">
      <template #filters>
        <Select v-model="statusFilter">
          <SelectTrigger class="w-[130px] h-9">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Status</SelectItem>
            <SelectItem value="active">Active</SelectItem>
            <SelectItem value="paused">Paused</SelectItem>
          </SelectContent>
        </Select>
      </template>
    </FilterBar>

    <div class="flex-1 overflow-auto">
      <div v-if="loading" class="flex items-center justify-center py-16">
        <Loader2 class="h-10 w-10 animate-spin text-primary" />
      </div>

      <div v-else-if="filteredSchedules.length === 0" class="py-16 text-center px-6">
        <div class="w-16 h-16 rounded-2xl bg-primary/10 flex items-center justify-center mx-auto mb-4">
          <Calendar class="h-8 w-8 text-primary" />
        </div>
        <h3 class="text-lg font-semibold mb-2">No schedules found</h3>
        <p class="text-muted-foreground mb-4">Create a schedule to automate workflow execution.</p>
        <Button @click="openCreateDialog"><Plus class="w-4 h-4 mr-2" />Create Schedule</Button>
      </div>

      <DataTable v-else :data="filteredSchedules" :columns="tableColumns">
        <template #row="{ row }">
          <td class="px-6 py-4">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-xl flex items-center justify-center" :class="row.is_enabled ? 'bg-emerald-500/10' : 'bg-gray-500/10'">
                <Timer class="w-5 h-5" :class="row.is_enabled ? 'text-emerald-500' : 'text-gray-400'" />
              </div>
              <div class="min-w-0">
                <div class="font-medium text-sm truncate max-w-[180px]">{{ row.workflow_name || 'Unknown' }}</div>
                <div class="text-xs text-muted-foreground font-mono">{{ row.workflow_id.substring(0, 8) }}...</div>
              </div>
            </div>
          </td>
          <td class="px-6 py-4"><div class="font-medium text-sm">{{ row.name }}</div></td>
          <td class="px-6 py-4">
            <div class="flex items-center gap-2">
              <code class="text-xs font-mono bg-muted px-2 py-0.5 rounded">{{ row.cron_expression }}</code>
              <Badge v-if="getCronPresetName(row.cron_expression)" variant="secondary" class="text-xs">{{ getCronPresetName(row.cron_expression) }}</Badge>
            </div>
          </td>
          <td class="px-6 py-4 text-center">
            <Badge :class="row.is_enabled ? 'bg-emerald-500/10 text-emerald-600 border-emerald-500/20' : 'bg-gray-500/10 text-gray-500 border-gray-500/20'" class="text-xs">
              <component :is="row.is_enabled ? Zap : Pause" class="w-3 h-3 mr-1" />
              {{ row.is_enabled ? 'Active' : 'Paused' }}
            </Badge>
          </td>
          <td class="px-6 py-4">
            <div class="flex items-center gap-2 text-sm">
              <Clock class="w-4 h-4 text-muted-foreground" />
              <span :class="row.next_run_at ? 'font-medium' : 'text-muted-foreground'">{{ formatDate(row.next_run_at) }}</span>
            </div>
          </td>
          <td class="px-6 py-4"><span class="text-sm text-muted-foreground">{{ formatDate(row.last_run_at) }}</span></td>
          <td class="px-6 py-4 text-right" @click.stop>
            <div class="flex items-center justify-end gap-2">
              <Switch :checked="row.is_enabled" :disabled="togglingIds.has(row.id)" @update:checked="handleToggle(row)" class="data-[state=checked]:bg-emerald-500" />
              <Button @click="handleRunNow(row)" size="sm" variant="ghost" class="h-8 w-8 p-0" :disabled="runningIds.has(row.id)" title="Run Now">
                <Loader2 v-if="runningIds.has(row.id)" class="h-4 w-4 animate-spin" /><PlayCircle v-else class="h-4 w-4" />
              </Button>
              <Button @click="openEditDialog(row)" size="sm" variant="ghost" class="h-8 w-8 p-0" title="Edit"><Edit class="h-4 w-4" /></Button>
              <Button @click="handleDelete(row)" size="sm" variant="ghost" class="h-8 w-8 p-0 text-destructive" title="Delete"><Trash2 class="h-4 w-4" /></Button>
            </div>
          </td>
        </template>
      </DataTable>
    </div>

    <!-- Dialog -->
    <Dialog v-model:open="dialogOpen">
      <DialogContent class="sm:max-w-[540px] p-0 overflow-hidden">
        <div class="bg-gradient-to-r from-primary/10 to-transparent p-6 border-b">
          <DialogHeader>
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-xl bg-primary/20 flex items-center justify-center">
                <Sparkles class="w-5 h-5 text-primary" />
              </div>
              <div>
                <DialogTitle>{{ dialogMode === 'create' ? 'Create Schedule' : 'Edit Schedule' }}</DialogTitle>
                <DialogDescription>{{ dialogMode === 'create' ? 'Set up automated workflow execution.' : 'Update schedule configuration.' }}</DialogDescription>
              </div>
            </div>
          </DialogHeader>
        </div>

        <div class="p-6 space-y-5">
          <!-- Searchable Workflow Selector -->
          <div class="space-y-2">
            <Label>Workflow <span class="text-destructive">*</span></Label>
            <div class="relative">
              <div class="relative">
                <Search class="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                <Input 
                  v-model="workflowSearchQuery" 
                  @focus="showWorkflowDropdown = true"
                  placeholder="Search workflows..." 
                  class="pl-9 h-11"
                  :disabled="dialogMode === 'edit'"
                />
              </div>
              <!-- Selected workflow display -->
              <div v-if="selectedWorkflowName && !showWorkflowDropdown && !workflowSearchQuery" 
                   class="absolute inset-0 flex items-center px-9 bg-background rounded-md border pointer-events-none">
                <span class="truncate">{{ selectedWorkflowName }}</span>
              </div>
              <!-- Dropdown -->
              <div v-if="showWorkflowDropdown && dialogMode === 'create'" 
                   class="absolute z-50 top-full left-0 right-0 mt-1 bg-popover border rounded-lg shadow-lg overflow-hidden">
                <ScrollArea class="max-h-[200px]">
                  <div v-if="filteredWorkflows.length === 0" class="py-4 text-center text-sm text-muted-foreground">
                    No workflows found
                  </div>
                  <div v-else class="p-1">
                    <button
                      v-for="wf in filteredWorkflows"
                      :key="wf.id"
                      @click="selectWorkflow(wf.id)"
                      class="flex items-center justify-between w-full px-3 py-2.5 text-sm rounded-md hover:bg-muted transition-colors text-left"
                    >
                      <div class="flex items-center gap-3 truncate">
                        <div class="w-7 h-7 rounded-lg bg-primary/10 flex items-center justify-center shrink-0">
                          <Timer class="w-3.5 h-3.5 text-primary" />
                        </div>
                        <span class="truncate">{{ wf.name }}</span>
                      </div>
                      <Check v-if="formData.workflow_id === wf.id" class="w-4 h-4 text-primary shrink-0" />
                    </button>
                  </div>
                </ScrollArea>
              </div>
            </div>
            <p v-if="dialogMode === 'edit'" class="text-xs text-muted-foreground">Workflow cannot be changed after creation</p>
          </div>

          <!-- Name -->
          <div class="space-y-2">
            <Label>Schedule Name <span class="text-destructive">*</span></Label>
            <Input v-model="formData.name" placeholder="e.g., Daily Product Sync" class="h-11" />
          </div>

          <!-- Cron -->
          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <Label>Cron Expression <span class="text-destructive">*</span></Label>
              <a href="https://crontab.guru/" target="_blank" class="text-xs text-primary hover:underline">Cron helper ↗</a>
            </div>
            <Input v-model="formData.cron_expression" placeholder="e.g., 0 0 * * *" class="h-11 font-mono" />
            <div class="space-y-2">
              <p class="text-xs text-muted-foreground">Quick presets:</p>
              <div class="flex flex-wrap gap-2">
                <Button
                  v-for="preset in presets.slice(0, 8)"
                  :key="preset.expression"
                  variant="outline"
                  size="sm"
                  :class="formData.cron_expression === preset.expression ? 'bg-primary/10 border-primary/50 text-primary' : ''"
                  class="h-7 text-xs"
                  @click="handlePresetSelect(preset)"
                >
                  {{ preset.name }}
                </Button>
              </div>
            </div>
          </div>

          <!-- Timezone -->
          <div class="space-y-2">
            <Label>Timezone</Label>
            <Select v-model="formData.timezone">
              <SelectTrigger class="h-11">
                <SelectValue placeholder="Select timezone" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="tz in timezones" :key="tz.value" :value="tz.value">
                  {{ tz.label }} ({{ tz.offset }})
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <DialogFooter class="px-6 py-4 bg-muted/30 border-t">
          <Button variant="outline" @click="dialogOpen = false">Cancel</Button>
          <Button @click="handleSave" :disabled="dialogLoading" class="bg-gradient-to-r from-primary to-primary/80">
            <Loader2 v-if="dialogLoading" class="w-4 h-4 mr-2 animate-spin" />
            {{ dialogMode === 'create' ? 'Create Schedule' : 'Save Changes' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </PageLayout>
</template>

<style scoped>
/* Click outside to close dropdown */
.relative:focus-within .absolute.z-50 {
  display: block;
}
</style>
