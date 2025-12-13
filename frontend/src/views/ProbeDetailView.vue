<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { probesApi, type ProbeResult, type ProbeSnapshot } from '@/api/probes'
import { workflowsApi } from '@/api/workflows'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import PageLayout from '@/components/layout/PageLayout.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import { 
  CheckCircle2, 
  AlertTriangle,
  XCircle,
  Clock,
  Image,
  ExternalLink,
  PlayCircle,
  Loader2
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const route = useRoute()

const workflowId = computed(() => route.params.workflowId as string)
const probeId = computed(() => route.params.probeId as string)

const loading = ref(true)
const probe = ref<ProbeResult | null>(null)
const workflowName = ref('')
const expandedPhases = ref<Set<string>>(new Set())
const selectedSnapshot = ref<ProbeSnapshot | null>(null)
const showSnapshotDialog = ref(false)
const running = ref(false)

const stats = computed(() => {
  if (!probe.value) return []
  const phases = probe.value.phases || []
  const nodes = phases.flatMap(p => p.nodes || [])
  
  return [
    { label: 'Duration', value: formatDuration(probe.value.duration_ms) },
    { label: 'Phases', value: phases.length },
    { label: 'Nodes Passed', value: nodes.filter(n => n.status === 'passed').length, color: 'text-green-600 dark:text-green-400' },
    { label: 'Nodes Failed', value: nodes.filter(n => n.status === 'failed').length, color: 'text-red-600 dark:text-red-400' }
  ]
})

const fetchData = async () => {
  loading.value = true
  try {
    // Fetch workflow name
    const workflowRes = await workflowsApi.getById(workflowId.value)
    workflowName.value = workflowRes.data.name

    // Fetch probe results
    const res = await probesApi.getProbeResults(workflowId.value, 20)
    const found = res.data.results.find(p => p.id === probeId.value)
    if (found) {
      probe.value = found
      // Expand failed phases by default
      found.phases?.forEach(phase => {
        if (phase.status === 'failed') {
          expandedPhases.value.add(phase.phase_id)
        }
      })
    }
  } catch (error) {
    console.error('Failed to fetch probe:', error)
    toast.error('Failed to load probe details')
  } finally {
    loading.value = false
  }
}

const handleRunProbe = async () => {
  if (running.value) return
  running.value = true
  try {
    await probesApi.runProbe(workflowId.value)
    toast.success('Probe check started')
    setTimeout(fetchData, 3000)
  } catch (error) {
    console.error('Failed to run probe:', error)
    toast.error('Failed to start probe check')
  } finally {
    running.value = false
  }
}



const openSnapshot = (snapshot: ProbeSnapshot) => {
  selectedSnapshot.value = snapshot
  showSnapshotDialog.value = true
  // Fetch DOM content for preview (only for local paths, not GCS URLs)
  if (!isGcsUrl(snapshot.dom_path)) {
    fetchDomContent(snapshot.dom_path)
  } else {
    domContent.value = 'DOM stored in GCS. Click "Open in GCS Console" to view.'
  }
}

// Check if path is a GCS console URL (used in staging/production)
const isGcsUrl = (path: string) => {
  return path && (path.startsWith('https://console.cloud.google.com') || path.startsWith('gs://'))
}

// Build URL to fetch snapshot file from backend (for local paths)
const getSnapshotFileUrl = (path: string) => {
  if (isGcsUrl(path)) {
    return path // Return GCS URL directly
  }
  return `/api/v1/snapshots/file?path=${encodeURIComponent(path)}`
}

// Ref to store DOM content for preview
const domContent = ref('')
const domPreview = computed(() => {
  if (!domContent.value) return 'Loading DOM content...'
  // Limit preview to first 5000 chars
  if (domContent.value.length > 5000) {
    return domContent.value.substring(0, 5000) + '\n\n... (truncated)'
  }
  return domContent.value
})

// Fetch DOM content from backend (for local storage only)
const fetchDomContent = async (path: string) => {
  if (!path) {
    domContent.value = ''
    return
  }
  if (isGcsUrl(path)) {
    domContent.value = 'DOM stored in GCS. Click "Open in GCS Console" to view.'
    return
  }
  try {
    const response = await fetch(getSnapshotFileUrl(path))
    if (response.ok) {
      domContent.value = await response.text()
    } else {
      domContent.value = 'Failed to load DOM content'
    }
  } catch (error) {
    domContent.value = 'Error loading DOM content'
  }
}

// Download/open snapshot in new tab
const downloadSnapshot = (type: 'screenshot' | 'dom') => {
  if (!selectedSnapshot.value) return
  const path = type === 'screenshot' 
    ? selectedSnapshot.value.screenshot_path 
    : selectedSnapshot.value.dom_path
  window.open(getSnapshotFileUrl(path), '_blank')
}

// Handle image load error
const handleImageError = (event: Event) => {
  const img = event.target as HTMLImageElement
  img.src = 'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" width="400" height="200"><rect fill="%23f3f4f6" width="100%" height="100%"/><text fill="%239ca3af" x="50%" y="50%" text-anchor="middle" dy=".3em" font-family="sans-serif" font-size="14">Screenshot not available</text></svg>'
}

const getStatusColor = (status: string) => {
  switch(status) {
    case 'healthy':
    case 'passed': return 'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20'
    case 'degraded': return 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20'
    case 'broken':
    case 'failed': return 'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20'
    case 'skipped': return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
    default: return 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20'
  }
}

const getStatusIcon = (status: string) => {
  switch(status) {
    case 'healthy':
    case 'passed': return CheckCircle2
    case 'degraded': return AlertTriangle
    case 'broken':
    case 'failed': return XCircle
    default: return Clock
  }
}

const formatDuration = (ms?: number) => {
  if (!ms) return 'N/A'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(2)}s`
}

const formatDate = (dateString?: string) => {
  if (!dateString) return 'N/A'
  return new Date(dateString).toLocaleString()
}

onMounted(fetchData)
</script>

<template>
  <PageLayout>
    <!-- Header -->
    <PageHeader 
      :title="workflowName || 'Loading...'"
      description="Probe result details with phase and node breakdown"
    >
      <template #breadcrumb>
        <div class="flex items-center text-sm text-muted-foreground">
          <router-link to="/probes" class="hover:text-foreground transition-colors">Probes</router-link>
          <span class="mx-2">/</span>
          <span class="text-foreground">{{ workflowName }}</span>
        </div>
      </template>
      <template #actions>
        <Button @click="handleRunProbe" variant="outline" size="sm" :disabled="running">
          <Loader2 v-if="running" class="w-4 h-4 mr-2 animate-spin" />
          <PlayCircle v-else class="w-4 h-4 mr-2" />
          Run Probe
        </Button>
      </template>
    </PageHeader>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <template v-else-if="probe">
      <!-- Overall Status Card -->
      <div class="p-4 border-b">
        <Card class="p-4">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-4">
              <div 
                class="w-14 h-14 rounded-xl flex items-center justify-center"
                :class="{
                  'bg-green-500/10': probe.status === 'healthy',
                  'bg-amber-500/10': probe.status === 'degraded',
                  'bg-red-500/10': probe.status === 'broken'
                }"
              >
                <component 
                  :is="getStatusIcon(probe.status)" 
                  class="w-7 h-7" 
                  :class="{
                    'text-green-500': probe.status === 'healthy',
                    'text-amber-500': probe.status === 'degraded',
                    'text-red-500': probe.status === 'broken'
                  }"
                />
              </div>
              <div>
                <div class="flex items-center gap-2">
                  <h3 class="text-lg font-semibold capitalize">{{ probe.status }}</h3>
                  <Badge :class="getStatusColor(probe.status)" class="text-xs">
                    {{ probe.status }}
                  </Badge>
                </div>
                <p class="text-sm text-muted-foreground mt-1">
                  Checked {{ formatDate(probe.created_at) }}
                </p>
              </div>
            </div>
            <div class="flex gap-6 text-sm">
              <div v-for="stat in stats" :key="stat.label" class="text-center">
                <div class="font-semibold text-lg" :class="stat.color">{{ stat.value }}</div>
                <div class="text-muted-foreground text-xs">{{ stat.label }}</div>
              </div>
            </div>
          </div>
        </Card>
      </div>

      <!-- Timeline View -->
      <div class="flex-1 overflow-auto p-4">
        <div class="relative">
          <!-- Timeline Line -->
          <div class="absolute left-6 top-0 bottom-0 w-0.5 bg-border"></div>
          
          <!-- Timeline Items -->
          <div class="space-y-0">
            <template v-for="phase in probe.phases" :key="phase.phase_id">
              <!-- Phase Header -->
              <div class="relative flex items-start gap-4 pb-2">
                <div 
                  class="w-12 h-12 rounded-xl flex items-center justify-center z-10 shrink-0"
                  :class="{
                    'bg-green-500/20': phase.status === 'passed',
                    'bg-red-500/20': phase.status === 'failed',
                    'bg-gray-500/20': phase.status === 'skipped'
                  }"
                >
                  <component 
                    :is="getStatusIcon(phase.status)" 
                    class="w-5 h-5" 
                    :class="{
                      'text-green-500': phase.status === 'passed',
                      'text-red-500': phase.status === 'failed',
                      'text-gray-500': phase.status === 'skipped'
                    }"
                  />
                </div>
                <div class="flex-1 pt-2">
                  <div class="flex items-center justify-between">
                    <div>
                      <h3 class="font-semibold text-base">{{ phase.phase_name }}</h3>
                      <p class="text-xs text-muted-foreground">
                        {{ phase.nodes?.length || 0 }} nodes • {{ formatDuration(phase.duration_ms) }}
                      </p>
                    </div>
                    <div class="flex items-center gap-2">
                      <a 
                        v-if="phase.sample_url"
                        :href="phase.sample_url" 
                        target="_blank" 
                        class="text-xs text-primary hover:underline flex items-center gap-1"
                      >
                        <ExternalLink class="w-3 h-3" />
                        Sample URL
                      </a>
                      <Badge :class="getStatusColor(phase.status)" class="text-xs capitalize">
                        {{ phase.status }}
                      </Badge>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Phase Nodes -->
              <div class="ml-6 pl-10 border-l-2 border-muted space-y-1 pb-4">
                <div 
                  v-for="node in phase.nodes" 
                  :key="node.node_id"
                  class="relative group"
                >
                  <!-- Node connector dot -->
                  <div 
                    class="absolute -left-[13px] top-3 w-2 h-2 rounded-full border-2 border-background"
                    :class="{
                      'bg-green-500': node.status === 'passed',
                      'bg-red-500': node.status === 'failed',
                      'bg-gray-400': node.status === 'skipped'
                    }"
                  ></div>
                  
                  <!-- Node Card -->
                  <Card 
                    class="p-3 hover:bg-muted/30 transition-colors"
                    :class="{
                      'border-red-500/30': node.status === 'failed',
                      'border-green-500/20': node.status === 'passed'
                    }"
                  >
                    <div class="flex items-start justify-between gap-4">
                      <div class="flex items-start gap-3 min-w-0 flex-1">
                        <div 
                          class="w-7 h-7 rounded-lg flex items-center justify-center shrink-0 mt-0.5"
                          :class="{
                            'bg-green-500/10': node.status === 'passed',
                            'bg-red-500/10': node.status === 'failed',
                            'bg-gray-500/10': node.status === 'skipped'
                          }"
                        >
                          <component 
                            :is="getStatusIcon(node.status)" 
                            class="w-3.5 h-3.5" 
                            :class="{
                              'text-green-500': node.status === 'passed',
                              'text-red-500': node.status === 'failed',
                              'text-gray-500': node.status === 'skipped'
                            }"
                          />
                        </div>
                        <div class="min-w-0 flex-1">
                          <div class="flex items-center gap-2">
                            <span class="font-medium text-sm">{{ node.node_name }}</span>
                            <Badge variant="outline" class="text-[10px] py-0 px-1.5">{{ node.node_type }}</Badge>
                          </div>
                          <div class="flex items-center gap-3 text-xs text-muted-foreground mt-1">
                            <span v-if="node.element_count">{{ node.element_count }} elements</span>
                            <span v-if="node.links_found">{{ node.links_found }} links</span>
                            <span>{{ formatDuration(node.duration_ms) }}</span>
                          </div>
                          <div v-if="node.error" class="text-xs text-red-500 mt-2 p-2 bg-red-50 dark:bg-red-900/20 rounded">
                            {{ node.error }}
                          </div>
                          <code v-if="node.selector" class="text-[10px] text-muted-foreground font-mono mt-2 bg-muted px-2 py-1 rounded block truncate">
                            {{ node.selector }}
                          </code>
                          
                          <!-- Field-level results for extract nodes -->
                          <div v-if="node.fields && node.fields.length > 0" class="mt-3 space-y-1.5">
                            <div class="text-xs font-medium text-muted-foreground mb-1.5">Extracted Fields:</div>
                            <div 
                              v-for="field in node.fields" 
                              :key="field.name"
                              class="flex items-start gap-2 text-xs p-1.5 rounded"
                              :class="{
                                'bg-green-50 dark:bg-green-900/10': field.status === 'ok',
                                'bg-amber-50 dark:bg-amber-900/10': field.status === 'empty',
                                'bg-red-50 dark:bg-red-900/10': field.status === 'error'
                              }"
                            >
                              <div 
                                class="w-4 h-4 rounded-full flex items-center justify-center shrink-0 mt-0.5"
                                :class="{
                                  'bg-green-500/20 text-green-600': field.status === 'ok',
                                  'bg-amber-500/20 text-amber-600': field.status === 'empty',
                                  'bg-red-500/20 text-red-600': field.status === 'error'
                                }"
                              >
                                <CheckCircle2 v-if="field.status === 'ok'" class="w-2.5 h-2.5" />
                                <AlertTriangle v-else-if="field.status === 'empty'" class="w-2.5 h-2.5" />
                                <XCircle v-else class="w-2.5 h-2.5" />
                              </div>
                              <div class="min-w-0 flex-1">
                                <div class="flex items-center gap-2">
                                  <span class="font-medium">{{ field.name }}</span>
                                  <Badge 
                                    class="text-[9px] py-0 px-1"
                                    :class="{
                                      'bg-green-500/10 text-green-600 border-green-500/20': field.status === 'ok',
                                      'bg-amber-500/10 text-amber-600 border-amber-500/20': field.status === 'empty',
                                      'bg-red-500/10 text-red-600 border-red-500/20': field.status === 'error'
                                    }"
                                  >
                                    {{ field.status }}
                                  </Badge>
                                </div>
                                <div v-if="field.value" class="text-muted-foreground truncate mt-0.5" :title="field.value">
                                  {{ field.value }}
                                </div>
                                <code v-if="field.selector" class="text-[9px] text-muted-foreground/70 font-mono block truncate mt-0.5">
                                  {{ field.selector }}
                                </code>
                                <div v-if="field.error" class="text-red-500 mt-0.5">{{ field.error }}</div>
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                      <Button 
                        v-if="node.snapshot"
                        @click="openSnapshot(node.snapshot)"
                        size="sm"
                        variant="ghost"
                        class="h-7 px-2 shrink-0"
                      >
                        <Image class="w-3.5 h-3.5 mr-1" />
                        Snapshot
                      </Button>
                    </div>
                  </Card>
                </div>
              </div>
            </template>
          </div>
        </div>
      </div>
    </template>

    <!-- Snapshot Dialog -->
    <Dialog v-model:open="showSnapshotDialog">
      <DialogContent class="max-w-5xl max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>Node Snapshot</DialogTitle>
        </DialogHeader>
        <div v-if="selectedSnapshot" class="flex-1 overflow-y-auto space-y-4">
          <!-- Snapshot Metadata -->
          <div class="bg-muted rounded-lg p-4">
            <div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
              <div>
                <div class="text-xs text-muted-foreground">Page Title</div>
                <div class="font-medium truncate" :title="selectedSnapshot.page_title">{{ selectedSnapshot.page_title || 'N/A' }}</div>
              </div>
              <div>
                <div class="text-xs text-muted-foreground">Page URL</div>
                <a :href="selectedSnapshot.page_url" target="_blank" class="font-medium text-primary truncate block hover:underline" :title="selectedSnapshot.page_url">{{ selectedSnapshot.page_url }}</a>
              </div>
              <div>
                <div class="text-xs text-muted-foreground">DOM Size</div>
                <div class="font-medium">{{ (selectedSnapshot.dom_size / 1024).toFixed(1) }} KB</div>
              </div>
              <div>
                <div class="text-xs text-muted-foreground">Screenshot Size</div>
                <div class="font-medium">{{ selectedSnapshot.image_width }}x{{ selectedSnapshot.image_height }}</div>
              </div>
            </div>
          </div>

          <!-- Screenshot Preview -->
          <div v-if="selectedSnapshot.screenshot_path" class="space-y-2">
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-semibold">Screenshot</h3>
              <Button variant="outline" size="sm" @click="downloadSnapshot('screenshot')">
                <ExternalLink class="w-3 h-3 mr-1" />
                Open Full Size
              </Button>
            </div>
            <div class="border rounded-lg overflow-hidden bg-muted/30">
              <img 
                :src="getSnapshotFileUrl(selectedSnapshot.screenshot_path)" 
                :alt="selectedSnapshot.page_title || 'Screenshot'"
                class="w-full max-h-[400px] object-contain"
                @error="handleImageError"
              />
            </div>
          </div>

          <!-- DOM Preview -->
          <div v-if="selectedSnapshot.dom_path" class="space-y-2">
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-semibold">HTML DOM</h3>
              <Button variant="outline" size="sm" @click="downloadSnapshot('dom')">
                <ExternalLink class="w-3 h-3 mr-1" />
                Open in New Tab
              </Button>
            </div>
            <div class="border rounded-lg bg-muted/30 max-h-[300px] overflow-auto">
              <pre class="text-xs font-mono p-4 whitespace-pre-wrap break-all">{{ domPreview }}</pre>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  </PageLayout>
</template>
