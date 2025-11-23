<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useExecutionsStore } from '@/stores/executions'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { 
  ArrowLeft, 
  Loader2, 
  CheckCircle, 
  XCircle, 
  Clock,
  StopCircle,
  Download,
  RefreshCw,
  Play,
  FileJson,
  ChevronLeft,
  ChevronRight,
  Maximize2
} from 'lucide-vue-next'
import ExecutionLiveView from '@/components/execution/ExecutionLiveView.vue'

const route = useRoute()
const router = useRouter()
const executionsStore = useExecutionsStore()
const executionId = route.params.id as string

const refreshInterval = ref<number | null>(null)
const activeTab = ref('live')

// Pagination State
const currentPage = ref(1)
const pageSize = ref(50)

// Dialog State
const isDialogOpen = ref(false)
const selectedItemData = ref<any>(null)
const selectedItemTitle = ref('')

// Real execution data from store
const execution = computed(() => executionsStore.currentExecution)
const extractedData = computed(() => executionsStore.extractedData)
const totalItems = computed(() => executionsStore.extractedDataTotal)

// Parse the JSON string data into objects
const parsedExtractedData = computed(() => {
  if (!extractedData.value) return []
  return extractedData.value.map(item => {
    let parsedData = item.data
    if (typeof item.data === 'string') {
      try {
        parsedData = JSON.parse(item.data)
      } catch (e) {
        console.error('Failed to parse item data:', e)
        parsedData = { error: 'Invalid JSON', raw: item.data }
      }
    }
    return {
      ...item,
      parsedData
    }
  })
})

// Dynamic columns for the data table
const dataColumns = computed(() => {
  if (!parsedExtractedData.value || parsedExtractedData.value.length === 0) return []
  
  const keys = new Set<string>()
  parsedExtractedData.value.forEach(item => {
    if (item.parsedData) {
      Object.keys(item.parsedData).forEach(key => keys.add(key))
    }
  })
  
  return Array.from(keys).sort()
})

const renderCell = (value: any) => {
  if (value === null || value === undefined) return '-'
  if (typeof value === 'object') return 'View Details'
  if (String(value).startsWith('http')) {
    return value 
  }
  return String(value)
}

const isComplexValue = (value: any) => {
  return typeof value === 'object' && value !== null
}

const openDetailDialog = (key: string, value: any) => {
  selectedItemTitle.value = key
  selectedItemData.value = value
  isDialogOpen.value = true
}

// Auto-switch tab based on status
const updateActiveTab = () => {
  if (execution.value) {
    if (execution.value.status === 'running') {
      activeTab.value = 'live'
    } else if (execution.value.status === 'completed' && activeTab.value === 'live') {
      // Switch to data tab when completed and data is available
      activeTab.value = 'data'
    }
  }
}

// Watch for status changes to auto-switch tabs
watch(() => execution.value?.status, (newStatus, oldStatus) => {
  if (newStatus === 'completed' && oldStatus === 'running') {
    // Execution just completed, switch to data tab and reload data
    activeTab.value = 'data'
    loadExtractedData()
  }
})

const progressPercentage = computed(() => {
  const stats = executionsStore.executionStats
  if (!stats || stats.total_urls === 0) return 0
  return Math.round((stats.completed / stats.total_urls) * 100)
})

const getStatusVariant = (status: string) => {
  switch (status) {
    case 'running':
      return 'default'
    case 'completed':
      return 'default'
    case 'failed':
      return 'destructive'
    default:
      return 'secondary'
  }
}

const getStatusIcon = (status: string) => {
  switch (status) {
    case 'running':
      return Loader2
    case 'completed':
      return CheckCircle
    case 'failed':
      return XCircle
    default:
      return Clock
  }
}

const formatDate = (dateString: string) => {
  if (!dateString) return '-'
  return new Date(dateString).toLocaleString()
}

const handleBack = () => {
  router.push('/executions')
}

const handleStop = async () => {
  try {
    await executionsStore.stopExecution(executionId)
    await loadExecutionData()
  } catch (error) {
    console.error('Failed to stop execution:', error)
  }
}

const handleDownloadData = () => {
  const dataStr = JSON.stringify(parsedExtractedData.value, null, 2)
  const dataUri = 'data:application/json;charset=utf-8,'+ encodeURIComponent(dataStr)
  const exportFileDefaultName = `execution_${executionId}_data.json`
  
  const linkElement = document.createElement('a')
  linkElement.setAttribute('href', dataUri)
  linkElement.setAttribute('download', exportFileDefaultName)
  linkElement.click()
}

const loadExecutionData = async () => {
  try {
    await Promise.all([
      executionsStore.fetchExecutionById(executionId),
      executionsStore.fetchExecutionStats(executionId),
      loadExtractedData()
    ])
  } catch (error) {
    console.error('Failed to load execution data:', error)
  }
}

const loadExtractedData = async () => {
  try {
    await executionsStore.fetchExtractedData(executionId, {
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value
    })
  } catch (error) {
    console.error('Failed to load extracted data:', error)
  }
}

const handlePageChange = (page: number) => {
  currentPage.value = page
  loadExtractedData()
}

const handlePageSizeChange = (value: string) => {
  pageSize.value = parseInt(value)
  currentPage.value = 1
  loadExtractedData()
}

const startAutoRefresh = () => {
  refreshInterval.value = window.setInterval(() => {
    if (execution.value && execution.value.status === 'running') {
      loadExecutionData()
    }
  }, 5000)
}

onMounted(async () => {
  await loadExecutionData()
  updateActiveTab()
  startAutoRefresh()
})

onUnmounted(() => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
  }
})
</script>

<template>
  <div class="container mx-auto py-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-4">
        <Button variant="ghost" size="icon" @click="handleBack">
          <ArrowLeft class="h-4 w-4" />
        </Button>
        <div>
          <h1 class="text-2xl font-bold tracking-tight">Execution Details</h1>
          <div class="flex items-center gap-2 text-muted-foreground">
            <span class="font-mono text-sm">{{ executionId }}</span>
            <span v-if="execution">• {{ execution.workflow_name }}</span>
          </div>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <Button 
          v-if="execution?.status === 'running'" 
          variant="destructive" 
          @click="handleStop"
          size="sm"
        >
          <StopCircle class="mr-2 h-4 w-4" />
          Stop Execution
        </Button>
      </div>
    </div>

    <div v-if="executionsStore.loading && !execution" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <template v-else-if="execution">
      <!-- Status Bar -->
      <Card class="p-4 flex items-center justify-between bg-muted/30">
        <div class="flex items-center gap-4">
          <Badge :variant="getStatusVariant(execution.status)" class="flex items-center gap-2 px-3 py-1 text-sm">
            <component 
              :is="getStatusIcon(execution.status)" 
              class="h-4 w-4"
              :class="{ 'animate-spin': execution.status === 'running' }"
            />
            <span class="uppercase">{{ execution.status }}</span>
          </Badge>
          <div class="text-sm text-muted-foreground">
            Started: {{ formatDate(execution.started_at) }}
          </div>
          <div v-if="execution.completed_at" class="text-sm text-muted-foreground">
            Completed: {{ formatDate(execution.completed_at) }}
          </div>
        </div>
        
        <div class="flex items-center gap-6">
          <div class="text-right">
            <div class="text-sm font-medium">{{ executionsStore.executionStats?.items_extracted || 0 }}</div>
            <div class="text-xs text-muted-foreground">Items Extracted</div>
          </div>
          <div class="text-right">
            <div class="text-sm font-medium">{{ executionsStore.executionStats?.total_urls || 0 }}</div>
            <div class="text-xs text-muted-foreground">Total URLs</div>
          </div>
          <div class="text-right">
            <div class="text-sm font-medium">{{ executionsStore.executionStats?.completed || 0 }}</div>
            <div class="text-xs text-muted-foreground">Completed</div>
          </div>
           <div class="text-right">
            <div class="text-sm font-medium text-red-500">{{ executionsStore.executionStats?.failed || 0 }}</div>
            <div class="text-xs text-muted-foreground">Failed</div>
          </div>
        </div>
      </Card>

      <!-- Main Content -->
      <Tabs v-model="activeTab" class="space-y-4">
        <TabsList>
          <TabsTrigger value="live" v-if="execution.status === 'running'">Live View</TabsTrigger>
          <TabsTrigger value="data">Extracted Data</TabsTrigger>
        </TabsList>

        <TabsContent value="live" v-if="execution.status === 'running'" class="space-y-4">
          <ExecutionLiveView 
            :execution-id="executionId" 
            :workflow-config="execution.workflow_config || {}" 
          />
        </TabsContent>

        <TabsContent value="data">
          <Card class="p-6">
            <div class="mb-4 flex items-center justify-between">
              <h3 class="text-lg font-semibold">Extracted Data ({{ totalItems }} items)</h3>
              <div class="flex items-center gap-2">
                 <Select :model-value="String(pageSize)" @update:model-value="handlePageSizeChange">
                  <SelectTrigger class="w-[100px]">
                    <SelectValue placeholder="Page Size" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="10">10 / page</SelectItem>
                    <SelectItem value="50">50 / page</SelectItem>
                    <SelectItem value="100">100 / page</SelectItem>
                  </SelectContent>
                </Select>
                <Button @click="handleDownloadData" size="sm" variant="outline">
                  <Download class="mr-2 h-4 w-4" />
                  Download JSON
                </Button>
              </div>
            </div>
            
            <div v-if="parsedExtractedData.length === 0" class="py-12 text-center text-muted-foreground">
              No data extracted yet.
            </div>
            
            <div v-else class="space-y-4">
              <div class="rounded-md border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead class="w-[180px]">Timestamp</TableHead>
                      <TableHead v-for="col in dataColumns" :key="col">{{ col }}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow v-for="item in parsedExtractedData" :key="item.id">
                      <TableCell class="whitespace-nowrap text-muted-foreground text-xs">
                        {{ formatDate(item.extracted_at) }}
                      </TableCell>
                      <TableCell v-for="col in dataColumns" :key="col" class="max-w-[300px] truncate">
                        <template v-if="isComplexValue(item.parsedData[col])">
                          <Button 
                            variant="ghost" 
                            size="sm" 
                            class="h-6 text-xs"
                            @click="openDetailDialog(col, item.parsedData[col])"
                          >
                            <Maximize2 class="mr-2 h-3 w-3" />
                            View Details
                          </Button>
                        </template>
                        <template v-else>
                          {{ renderCell(item.parsedData[col]) }}
                        </template>
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>

              <!-- Pagination Controls -->
              <div class="flex items-center justify-between">
                <div class="text-sm text-muted-foreground">
                  Showing {{ (currentPage - 1) * pageSize + 1 }} to {{ Math.min(currentPage * pageSize, totalItems) }} of {{ totalItems }} entries
                </div>
                <div class="flex items-center gap-2">
                  <Button 
                    variant="outline" 
                    size="sm" 
                    :disabled="currentPage === 1"
                    @click="handlePageChange(currentPage - 1)"
                  >
                    <ChevronLeft class="h-4 w-4" />
                    Previous
                  </Button>
                  <div class="text-sm font-medium">Page {{ currentPage }}</div>
                  <Button 
                    variant="outline" 
                    size="sm" 
                    :disabled="currentPage * pageSize >= totalItems"
                    @click="handlePageChange(currentPage + 1)"
                  >
                    Next
                    <ChevronRight class="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </div>
          </Card>
        </TabsContent>
      </Tabs>
    </template>

    <div v-else-if="executionsStore.error" class="py-12 text-center">
      <p class="text-destructive mb-4">{{ executionsStore.error }}</p>
      <Button @click="loadExecutionData()" variant="outline">
        Retry
      </Button>
    </div>

    <!-- Detail Dialog -->
    <Dialog :open="isDialogOpen" @update:open="isDialogOpen = $event">
      <DialogContent class="max-w-3xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Field Details: {{ selectedItemTitle }}</DialogTitle>
          <DialogDescription>
            Full content of the selected field.
          </DialogDescription>
        </DialogHeader>
        <div class="mt-4">
          <pre class="bg-muted p-4 rounded-md overflow-x-auto text-xs">{{ JSON.stringify(selectedItemData, null, 2) }}</pre>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>
