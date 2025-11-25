<template>
  <Dialog :open="isOpen" @update:open="(val) => !val && close()">
    <DialogContent class="max-w-5xl h-[85vh] p-0 gap-0 flex flex-col">
      <!-- Header -->
      <DialogHeader class="px-6 py-4 border-b flex-shrink-0">
        <DialogTitle class="flex items-center gap-2.5 text-xl">
          <div class="p-2 rounded-lg bg-primary/10">
            <Camera class="h-5 w-5 text-primary" />
          </div>
          Diagnostic Snapshot
        </DialogTitle>
      </DialogHeader>

      <!-- Loading State -->
      <div v-if="loading" class="flex flex-col items-center justify-center gap-4 py-16 flex-1">
        <RefreshCw class="h-10 w-10 animate-spin text-primary" />
        <p class="text-sm text-muted-foreground">Loading snapshot...</p>
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="flex flex-col items-center justify-center gap-4 py-16 flex-1">
        <div class="p-3 rounded-full bg-red-50 dark:bg-red-950">
          <AlertCircle class="h-10 w-10 text-red-600 dark:text-red-400" />
        </div>
        <p class="text-sm font-medium text-red-600 dark:text-red-400">{{ error }}</p>
      </div>

      <!-- Content -->
      <div v-else-if="snapshot && snapshot.id" class="flex flex-col flex-1 overflow-hidden">
        <!-- Tabs -->
        <Tabs v-model="activeTab" class="flex-1 flex flex-col overflow-hidden">
          <div class="border-b px border-b flex-shrink-0">
            <TabsList class="h-12 w-full justify-start rounded-none bg-transparent p-0 border-0">
              <TabsTrigger 
                value="screenshot" 
                class="data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-primary rounded-none px-4 h-12 gap-2"
              >
                <Camera class="h-4 w-4" />
                Screenshot
              </TabsTrigger>
              <TabsTrigger 
                value="dom"
                class="data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-primary rounded-none px-4 h-12 gap-2"
              >
                <FileCode class="h-4 w-4" />
                DOM
              </TabsTrigger>
              <TabsTrigger 
                value="console"
                class="data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-primary rounded-none px-4 h-12 gap-2"
              >
                <Terminal class="h-4 w-4" />
                Console
              </TabsTrigger>
              <TabsTrigger 
                value="details"
                class="data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-primary rounded-none px-4 h-12 gap-2"
              >
                <Info class="h-4 w-4" />
                Details
              </TabsTrigger>
            </TabsList>
          </div>

          <ScrollArea class="flex-1">
            <!-- Screenshot Tab -->
            <TabsContent value="screenshot" class="m-0 p-6 space-y-4">
              <div v-if="!snapshot.screenshot_path" class="flex flex-col items-center justify-center gap-3 py-12 text-muted-foreground">
                <ImageOff class="h-12 w-12 opacity-30" />
                <p class="text-sm">No screenshot available</p>
              </div>
              <div v-else class="space-y-3">
                <div class="flex gap-2">
                  <Button @click="viewFullScreen" variant="outline" size="sm">
                    <Maximize2 class="h-4 w-4 mr-2" />
                    Full Screen
                  </Button>
                </div>
                <Card class="overflow-hidden border-2">
                  <img 
                    :src="screenshotUrl" 
                    alt="Page screenshot"
                    :class="['w-full cursor-zoom-in transition-transform', imageZoom && 'scale-150 origin-top-left cursor-zoom-out']"
                    @click="imageZoom = !imageZoom"
                  />
                </Card>
              </div>
            </TabsContent>

            <!-- DOM Tab -->
            <TabsContent value="dom" class="m-0 p-6 space-y-4">
              <div class="flex gap-2">
                <Button @click="viewDOM" size="sm">
                  <Eye class="h-4 w-4 mr-2" />
                  View in New Tab
                </Button>
                <Button @click="downloadDOM" variant="outline" size="sm">
                  <Download class="h-4 w-4 mr-2" />
                  Download HTML
                </Button>
              </div>
              <Card class="p-4 border-2">
                <div class="flex items-center gap-2 text-sm text-muted-foreground">
                  <FileCode class="h-5 w-5" />
                  <span>Full HTML snapshot with styles and scripts</span>
                </div>
              </Card>
            </TabsContent>

            <!-- Console Tab -->
            <TabsContent value="console" class="m-0 p-6 space-y-3">
              <div v-if="snapshot.console_logs && snapshot.console_logs.length > 0" class="space-y-2">
                <Card
                  v-for="(log, index) in snapshot.console_logs"
                  :key="index"
                  :class="[
                    'p-4 border-l-4',
                    {
                      'border-l-red-500 bg-red-50 dark:bg-red-950 border-red-200 dark:border-red-800': log.type === 'error',
                      'border-l-amber-500 bg-amber-50 dark:bg-amber-950 border-amber-200 dark:border-amber-800': log.type === 'warn',
                      'border-l-blue-500 bg-blue-50 dark:bg-blue-950 border-blue-200 dark:border-blue-800': log.type === 'info' || log.type === 'log',
                    }
                  ]"
                >
                  <div class="flex gap-3">
                    <AlertCircle v-if="log.type === 'error'" class="h-4 w-4 mt-0.5 flex-shrink-0" />
                    <AlertTriangle v-else-if="log.type === 'warn'" class="h-4 w-4 mt-0.5 flex-shrink-0" />
                    <Info v-else class="h-4 w-4 mt-0.5 flex-shrink-0" />
                    <div class="flex-1 space-y-1">
                      <div class="font-mono text-sm font-medium">{{ log.message }}</div>
                      <div class="flex gap-2 text-xs opacity-75">
                        <span>{{ log.type }}</span>
                        <span>•</span>
                        <span>{{ formatDate(log.timestamp) }}</span>
                      </div>
                    </div>
                  </div>
                </Card>
              </div>
              <div v-else class="flex flex-col items-center justify-center gap-3 py-12 text-muted-foreground">
                <Terminal class="h-12 w-12 opacity-30" />
                <p class="text-sm">No console logs captured</p>
              </div>
            </TabsContent>

            <!-- Details Tab -->
            <TabsContent value="details" class="m-0 p-6 space-y-6">
              <!-- AI Auto-Fix Section -->
              <div class="space-y-3">
                <h3 class="text-base font-semibold flex items-center gap-2">
                  <div class="p-1.5 rounded bg-purple-500/10">
                    <Sparkles class="h-4 w-4 text-purple-600 dark:text-purple-400" />
                  </div>
                  AI Auto-Fix
                </h3>
                <AutoFixPanel :snapshot="snapshot" />
              </div>

              <Separator />

              <div class="grid gap-6 md:grid-cols-2">
                <!-- Page Info -->
                <Card class="p-4 border-2">
                  <h4 class="font-semibold mb-4 text-sm">Page Information</h4>
                  <div class="space-y-3">
                    <div class="space-y-1">
                      <div class="text-xs font-medium text-muted-foreground">URL</div>
                      <a :href="snapshot.url" target="_blank" class="flex items-center gap-1.5 text-sm text-primary hover:underline break-all">
                        {{ snapshot.url }}
                        <ExternalLink class="h-3 w-3 flex-shrink-0" />
                      </a>
                    </div>
                    <div v-if="snapshot.page_title" class="space-y-1">
                      <div class="text-xs font-medium text-muted-foreground">Title</div>
                      <div class="text-sm">{{ snapshot.page_title }}</div>
                    </div>
                    <div v-if="snapshot.status_code" class="space-y-1">
                      <div class="text-xs font-medium text-muted-foreground">Status Code</div>
                      <Badge variant="outline">{{ snapshot.status_code }}</Badge>
                    </div>
                  </div>
                </Card>

                <!-- Error Info -->
                <Card class="p-4 border-2">
                  <h4 class="font-semibold mb-4 text-sm">Error Details</h4>
                  <div class="space-y-3">
                    <div v-if="snapshot.selector_value" class="space-y-2">
                      <div class="text-xs font-medium text-muted-foreground">Selector</div>
                      <div class="flex items-start gap-2 flex-wrap">
                        <code class="text-xs bg-muted px-2 py-1 rounded border font-mono break-all flex-1 min-w-0">{{ snapshot.selector_value }}</code>
                        <Badge 
                          v-if="snapshot.field_required !== undefined"
                          :variant="snapshot.field_required ? 'destructive' : 'secondary'"
                          :title="snapshot.field_required ? 'Required field - causes failure' : 'Optional field - causes warning'"
                          class="flex-shrink-0"
                        >
                          {{ snapshot.field_required ? 'REQUIRED' : 'OPTIONAL' }}
                        </Badge>
                      </div>
                    </div>
                    <div v-if="snapshot.elements_found !== undefined" class="space-y-1">
                      <div class="text-xs font-medium text-muted-foreground">Elements Found</div>
                      <div class="text-sm">{{ snapshot.elements_found }}</div>
                    </div>
                    <div v-if="snapshot.error_message" class="space-y-1">
                      <div class="text-xs font-medium text-muted-foreground">Error</div>
                      <div class="text-sm text-red-600 dark:text-red-400 break-words">{{ snapshot.error_message }}</div>
                    </div>
                  </div>
                </Card>

                <!-- Metadata -->
                <Card v-if="snapshot.metadata" class="p-4 border-2 md:col-span-2">
                  <h4 class="font-semibold mb-4 text-sm">Metadata</h4>
                  <div class="grid gap-3 md:grid-cols-2">
                    <div v-for="(value, key) in snapshot.metadata" :key="key" class="space-y-1 min-w-0">
                      <div class="text-xs font-medium text-muted-foreground">{{ formatKey(key) }}</div>
                      <div class="text-sm break-words whitespace-pre-wrap max-w-full overflow-hidden">{{ formatValue(value) }}</div>
                    </div>
                  </div>
                </Card>
              </div>
            </TabsContent>
          </ScrollArea>
        </Tabs>
      </div>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Camera, FileCode, Terminal, Info, AlertCircle, AlertTriangle, Eye, Download, ExternalLink, ImageOff, Maximize2, RefreshCw, Sparkles } from 'lucide-vue-next'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { workflowsApi } from '@/api/workflows'
import type { HealthCheckSnapshot } from '@/types'
import AutoFixPanel from './AutoFixPanel.vue'

interface Props {
  snapshotId: string | null
  open: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
}>()

const snapshot = ref<HealthCheckSnapshot | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const activeTab = ref('screenshot')
const imageZoom = ref(false)

const isOpen = computed(() => props.open)

const screenshotUrl = computed(() => {
  if (!snapshot.value?.id) return ''
  return workflowsApi.getScreenshotUrl(snapshot.value.id)
})

const domUrl = computed(() => {
  if (!snapshot.value?.id) return ''
  return workflowsApi.getDOMUrl(snapshot.value.id)
})

watch(() => props.snapshotId, async (newId) => {
  if (!newId || !props.open) {
    snapshot.value = null
    return
  }

  loading.value = true
  error.value = null
  
  try {
    const response = await workflowsApi.getSnapshot(newId)
    snapshot.value = response.data
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Failed to load snapshot'
    snapshot.value = null
  } finally {
    loading.value = false
  }
})

watch(() => props.open, (isOpen) => {
  if (!isOpen) {
    snapshot.value = null
    error.value = null
    imageZoom.value = false
    activeTab.value = 'screenshot'
  } else if (props.snapshotId) {
    loadSnapshot()
  }
})

async function loadSnapshot() {
  if (!props.snapshotId) return
  
  loading.value = true
  error.value = null
  
  try {
    const response = await workflowsApi.getSnapshot(props.snapshotId)
    snapshot.value = response.data
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Failed to load snapshot'
    snapshot.value = null
  } finally {
    loading.value = false
  }
}

function close() {
  emit('close')
}

function viewDOM() {
  window.open(domUrl.value, '_blank')
}

function downloadDOM() {
  if (!snapshot.value?.dom_snapshot_path) return
  
  const link = document.createElement('a')
  link.href = domUrl.value
  link.download = `dom-${snapshot.value.node_id}.html`
  link.click()
}

function viewFullScreen() {
  if (!snapshot.value?.screenshot_path) return
  window.open(screenshotUrl.value, '_blank')
}

function formatDate(date: string) {
  return new Date(date).toLocaleString()
}

function formatKey(key: string): string {
  return key.split('_').map(word => 
    word.charAt(0).toUpperCase() + word.slice(1)
  ).join(' ')
}

function formatValue(value: any): string {
  if (typeof value === 'object') {
    return JSON.stringify(value, null, 2)
  }
  return String(value)
}
</script>
