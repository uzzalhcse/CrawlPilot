<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useExecutionsStore } from '@/stores/executions'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
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
  Loader2, 
  CheckCircle, 
  XCircle, 
  Clock,
  StopCircle,
  Download,
  Play,
  Pause,
  ChevronLeft,
  ChevronRight,
  Maximize2
} from 'lucide-vue-next'
import ExecutionLiveView from '@/components/execution/ExecutionLiveView.vue'
import ExecutionNodeTree from '@/components/execution/ExecutionNodeTree.vue'
import PageLayout from '@/components/layout/PageLayout.vue'
import PageHeader from '@/components/layout/PageHeader.vue'

const route = useRoute()
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


const getStatusVariant = (status: string) => {
  switch (status) {
    case 'running':
      return 'default'
    case 'paused':
      return 'secondary'
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
    case 'paused':
      return Pause
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


const handleStop = async () => {
  try {
    await executionsStore.stopExecution(executionId)
    await loadExecutionData()
  } catch (error) {
    console.error('Failed to stop execution:', error)
  }
}

const handlePause = async () => {
  try {
    await executionsStore.pauseExecution(executionId)
    await loadExecutionData()
  } catch (error) {
    console.error('Failed to pause execution:', error)
  }
}

const handleResume = async () => {
  try {
    await executionsStore.resumeExecution(executionId)
    await loadExecutionData()
  } catch (error) {
    console.error('Failed to resume execution:', error)
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

const handlePageSizeChange = (value: any) => {
  if (!value) return
  pageSize.value = parseInt(String(value))
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
  <PageLayout>
    <PageHeader
      :title="execution?.workflow_name || 'Execution Details'"
      :description="executionId"
    >
      <template #breadcrumb>
        <div class="flex items-center text-sm text-muted-foreground">
          <router-link to="/executions" class="hover:text-foreground transition-colors">Executions</router-link>
          <span class="mx-2">/</span>
          <span class="text-foreground font-mono text-xs">{{ executionId.slice(0, 8) }}...</span>
        </div>
      </template>
      <template #actions>
        <Button 
          v-if="execution?.status === 'paused'" 
          variant="default" 
          @click="handleResume"
          size="sm"
        >
          <Play class="mr-2 h-4 w-4" />
          Resume
        </Button>
        <Button 
          v-if="execution?.status === 'running'" 
          variant="outline" 
          @click="handlePause"
          size="sm"
        >
          <Pause class="mr-2 h-4 w-4" />
          Pause
        </Button>
        <Button 
          v-if="execution?.status === 'running'" 
          variant="destructive" 
          @click="handleStop"
          size="sm"
        >
          <StopCircle class="mr-2 h-4 w-4" />
          Stop
        </Button>
      </template>
    </PageHeader>

    <div v-if="executionsStore.loading && !execution" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <div v-else-if="execution" class="space-y-3">
      <!-- Status Bar -->
      <Card class="p-3 mt-3 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <Badge :variant="getStatusVariant(execution.status)" class="flex items-center gap-1.5 px-2 py-0.5 text-xs">
            <component 
              :is="getStatusIcon(execution.status)" 
              class="h-3 w-3"
              :class="{ 'animate-spin': execution.status === 'running' }"
            />
            <span class="uppercase">{{ execution.status }}</span>
          </Badge>
          <div class="text-xs text-muted-foreground">
            Started: {{ formatDate(execution.started_at) }}
          </div>
          <div v-if="execution.completed_at" class="text-xs text-muted-foreground">
            Completed: {{ formatDate(execution.completed_at) }}
          </div>
        </div>
        
        <div class="flex items-center gap-4">
          <div class="text-right">
            <div class="text-xs font-medium">{{ executionsStore.executionStats?.items_extracted || 0 }}</div>
            <div class="text-[10px] text-muted-foreground">Items Extracted</div>
          </div>
          <div class="text-right">
            <div class="text-xs font-medium">{{ executionsStore.executionStats?.total_urls || 0 }}</div>
            <div class="text-[10px] text-muted-foreground">Total URLs</div>
          </div>
          <div class="text-right">
            <div class="text-xs font-medium">{{ executionsStore.executionStats?.completed || 0 }}</div>
            <div class="text-[10px] text-muted-foreground">Completed</div>
          </div>
           <div class="text-right">
            <div class="text-xs font-medium text-red-500">{{ executionsStore.executionStats?.failed || 0 }}</div>
            <div class="text-[10px] text-muted-foreground">Failed</div>
          </div>
        </div>
      </Card>

      <!-- Main Content -->
      <Tabs v-model="activeTab" class="space-y-3 w-full max-w-full">
        <TabsList class="h-8">
          <TabsTrigger value="live" class="text-xs">Live View</TabsTrigger>
          <TabsTrigger value="data" class="text-xs">Extracted Data</TabsTrigger>
          <TabsTrigger value="tree" class="text-xs">Node Tree</TabsTrigger>
        </TabsList>

        <TabsContent value="live" class="space-y-3">
          <ExecutionLiveView 
            :execution-id="executionId" 
            :workflow-config="execution.workflow_config || {}" 
            :execution-status="execution.status"
          />
        </TabsContent>

        <TabsContent value="data" class="min-w-0 w-full max-w-full">
          <Card class="p-4 w-full max-w-full">
            <div class="mb-3 flex items-center justify-between">
              <h3 class="text-sm font-semibold">Extracted Data ({{ totalItems }} items)</h3>
              <div class="flex items-center gap-2">
                 <Select :model-value="String(pageSize)" @update:model-value="handlePageSizeChange">
                  <SelectTrigger class="w-[90px] h-8 text-xs">
                    <SelectValue placeholder="Page Size" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="10" class="text-xs">10 / page</SelectItem>
                    <SelectItem value="50" class="text-xs">50 / page</SelectItem>
                    <SelectItem value="100" class="text-xs">100 / page</SelectItem>
                  </SelectContent>
                </Select>
                <Button @click="handleDownloadData" size="sm" variant="outline" class="h-8 text-xs">
                  <Download class="mr-1.5 h-3 w-3" />
                  Download JSON
                </Button>
              </div>
            </div>
            
            <div v-if="parsedExtractedData.length === 0" class="py-8 text-center text-muted-foreground text-xs">
              No data extracted yet.
            </div>
            
            <div v-else class="space-y-3">
              <div class="rounded-md border overflow-x-auto w-full max-w-full">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead class="w-[160px] text-xs h-9">Timestamp</TableHead>
                      <TableHead v-for="col in dataColumns" :key="col" class="text-xs h-9">{{ col }}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow v-for="item in parsedExtractedData" :key="item.id">
                      <TableCell class="whitespace-nowrap text-muted-foreground text-[11px] py-2">
                        {{ formatDate(item.extracted_at) }}
                      </TableCell>
                      <TableCell v-for="col in dataColumns" :key="col" class="max-w-[300px] truncate text-xs py-2">
                        <template v-if="isComplexValue(item.parsedData[col])">
                          <Button 
                            variant="ghost" 
                            size="sm" 
                            class="h-6 text-[11px] px-2"
                            @click="openDetailDialog(col, item.parsedData[col])"
                          >
                            <Maximize2 class="mr-1 h-3 w-3" />
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
                <div class="text-xs text-muted-foreground">
                  Showing {{ (currentPage - 1) * pageSize + 1 }} to {{ Math.min(currentPage * pageSize, totalItems) }} of {{ totalItems }} entries
                </div>
                <div class="flex items-center gap-2">
                  <Button 
                    variant="outline" 
                    size="sm" 
                    :disabled="currentPage === 1"
                    @click="handlePageChange(currentPage - 1)"
                    class="h-8 text-xs"
                  >
                    <ChevronLeft class="h-3 w-3" />
                    Previous
                  </Button>
                  <div class="text-xs font-medium">Page {{ currentPage }}</div>
                  <Button 
                    variant="outline" 
                    size="sm" 
                    :disabled="currentPage * pageSize >= totalItems"
                    @click="handlePageChange(currentPage + 1)"
                    class="h-8 text-xs"
                  >
                    Next
                    <ChevronRight class="h-3 w-3" />
                  </Button>
                </div>
              </div>
            </div>
          </Card>
        </TabsContent>

        <TabsContent value="tree" class="min-w-0 w-full max-w-full">
          <Card class="p-4 w-full max-w-full">
            <ExecutionNodeTree />
          </Card>
        </TabsContent>
      </Tabs>
    </div>

    <div v-else-if="executionsStore.error" class="py-12 text-center">
      <p class="text-destructive mb-4">{{ executionsStore.error }}</p>
      <Button @click="loadExecutionData()" variant="outline">
        Retry
      </Button>
    </div>
  </PageLayout>

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
</template>
