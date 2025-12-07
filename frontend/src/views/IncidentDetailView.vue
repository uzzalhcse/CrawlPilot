<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getIncident, resolveIncident, updateIncidentStatus, type Incident } from '@/api/incidents'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import PageLayout from '@/components/layout/PageLayout.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import { 
  ArrowLeft,
  CheckCircle2, 
  Clock, 
  Loader2,
  ExternalLink,
  AlertTriangle,
  Globe,
  RefreshCw,
  XCircle,
  Play,
  Bug,
  Lightbulb,
  Image,
  FileCode,
  Workflow,
  Crosshair,
  Bot
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const resolving = ref(false)
const incident = ref<Incident | null>(null)

const incidentId = computed(() => route.params.id as string)

// Parse dom_snapshot JSON to extract probe details
const probeDetails = computed(() => {
  if (!incident.value?.dom_snapshot) return null
  try {
    return JSON.parse(incident.value.dom_snapshot)
  } catch {
    return null
  }
})



// Extract snapshot paths
const snapshotPaths = computed(() => {
  return probeDetails.value?.snapshot_paths || []
})

// Extract probe result phases
const probePhases = computed(() => {
  return probeDetails.value?.probe_result?.phases || []
})

// Get nodes with issues (0 links/elements or failed)
const problematicNodes = computed(() => {
  const nodes: any[] = []
  for (const phase of probePhases.value) {
    for (const node of phase.nodes || []) {
      if (node.status === 'failed' || 
          (node.node_type === 'extract_links' && node.links_found === 0) ||
          (node.node_type === 'extract' && node.element_count === 0)) {
        nodes.push({ ...node, phaseName: phase.phase_name })
      }
    }
  }
  return nodes
})

const fetchIncident = async () => {
  loading.value = true
  try {
    incident.value = await getIncident(incidentId.value)
  } catch (error) {
    console.error('Failed to fetch incident:', error)
    toast.error('Failed to load incident')
    router.push('/incidents')
  } finally {
    loading.value = false
  }
}

const handleResolve = async () => {
  if (!incident.value || resolving.value) return
  
  resolving.value = true
  try {
    await resolveIncident(incident.value.id, 'Manually resolved from detail view')
    toast.success('Incident resolved')
    await fetchIncident()
  } catch (error) {
    console.error('Failed to resolve incident:', error)
    toast.error('Failed to resolve incident')
  } finally {
    resolving.value = false
  }
}

const handleStatusChange = async (status: Incident['status']) => {
  if (!incident.value) return
  
  try {
    await updateIncidentStatus(incident.value.id, status)
    toast.success(`Status updated to ${status}`)
    await fetchIncident()
  } catch (error) {
    console.error('Failed to update status:', error)
    toast.error('Failed to update status')
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

const formatDate = (dateString?: string | number) => {
  if (!dateString) return 'N/A'
  // Handle Unix timestamp
  if (typeof dateString === 'number') {
    return new Date(dateString * 1000).toLocaleString()
  }
  return new Date(dateString).toLocaleString()
}

const formatErrorPattern = (pattern: string) => {
  return pattern.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}

const formatBytes = (bytes?: number) => {
  if (!bytes) return 'N/A'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

onMounted(fetchIncident)
</script>

<template>
  <PageLayout>
    <!-- Header -->
    <PageHeader 
      :title="incident?.domain || 'Loading...'" 
      :description="incident ? `Incident #${incident.id.slice(0, 8)}` : ''"
    >
      <template #back>
        <Button variant="ghost" size="sm" @click="router.push('/incidents')" class="mr-2">
          <ArrowLeft class="w-4 h-4 mr-1" />
          Back
        </Button>
      </template>
      <template #actions>
        <div class="flex items-center gap-2">
          <Badge v-if="incident" :class="getStatusColor(incident.status)" class="text-sm capitalize">
            {{ incident.status.replace('_', ' ') }}
          </Badge>
          <Badge v-if="incident" :class="getPriorityColor(incident.priority)" class="text-sm capitalize">
            {{ incident.priority }} Priority
          </Badge>
        </div>
      </template>
    </PageHeader>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-24">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Content -->
    <div v-else-if="incident" class="p-6 space-y-6">
      <!-- Action Buttons -->
      <div class="flex items-center gap-2 flex-wrap">
        <Button 
          v-if="incident.status === 'open' || incident.status === 'in_progress'"
          @click="handleResolve"
          :disabled="resolving"
          class="bg-green-600 hover:bg-green-700"
        >
          <Loader2 v-if="resolving" class="w-4 h-4 mr-2 animate-spin" />
          <CheckCircle2 v-else class="w-4 h-4 mr-2" />
          Resolve
        </Button>
        <Button 
          v-if="incident.status === 'open'"
          @click="handleStatusChange('in_progress')"
          variant="outline"
        >
          <Play class="w-4 h-4 mr-2" />
          Start Investigation
        </Button>
        <Button 
          v-if="incident.status === 'open'"
          @click="handleStatusChange('ignored')"
          variant="outline"
        >
          <XCircle class="w-4 h-4 mr-2" />
          Ignore
        </Button>
        <a v-if="incident.url" :href="incident.url" target="_blank" class="inline-flex">
          <Button variant="outline">
            <ExternalLink class="w-4 h-4 mr-2" />
            Open URL
          </Button>
        </a>
      </div>

      <!-- Info Cards Grid -->
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- Basic Info -->
        <Card>
          <CardHeader class="pb-3">
            <CardTitle class="text-base flex items-center gap-2">
              <Globe class="w-4 h-4" />
              Basic Information
            </CardTitle>
          </CardHeader>
          <CardContent class="space-y-3 text-sm">
            <div>
              <div class="text-muted-foreground mb-0.5">Domain</div>
              <div class="font-medium">{{ incident.domain || 'N/A' }}</div>
            </div>
            <div>
              <div class="text-muted-foreground mb-0.5">URL</div>
              <div class="font-mono text-xs break-all">{{ incident.url || 'N/A' }}</div>
            </div>
            <div>
              <div class="text-muted-foreground mb-0.5">Error Pattern</div>
              <Badge variant="outline">{{ formatErrorPattern(incident.error_pattern) }}</Badge>
            </div>
            <div>
              <div class="text-muted-foreground mb-0.5">Workflow ID</div>
              <code class="text-xs bg-muted px-1 py-0.5 rounded">{{ incident.workflow_id?.slice(0, 8) || 'N/A' }}</code>
            </div>
          </CardContent>
        </Card>

        <!-- Timeline -->
        <Card>
          <CardHeader class="pb-3">
            <CardTitle class="text-base flex items-center gap-2">
              <Clock class="w-4 h-4" />
              Timeline
            </CardTitle>
          </CardHeader>
          <CardContent class="space-y-3 text-sm">
            <div>
              <div class="text-muted-foreground mb-0.5">First Error</div>
              <div class="font-medium">{{ formatDate(incident.first_error_at) }}</div>
            </div>
            <div>
              <div class="text-muted-foreground mb-0.5">Last Error</div>
              <div class="font-medium">{{ formatDate(incident.last_error_at) }}</div>
            </div>
            <div>
              <div class="text-muted-foreground mb-0.5">Created</div>
              <div class="font-medium">{{ formatDate(incident.created_at) }}</div>
            </div>
            <div v-if="incident.resolved_at">
              <div class="text-muted-foreground mb-0.5">Resolved</div>
              <div class="font-medium text-green-600">{{ formatDate(incident.resolved_at) }}</div>
            </div>
          </CardContent>
        </Card>

        <!-- AI Status -->
        <Card>
          <CardHeader class="pb-3">
            <CardTitle class="text-base flex items-center gap-2">
              <Bot class="w-4 h-4" />
              AI Analysis
            </CardTitle>
          </CardHeader>
          <CardContent class="space-y-3 text-sm">
            <div>
              <div class="text-muted-foreground mb-0.5">AI Enabled</div>
              <Badge :variant="incident.ai_enabled ? 'default' : 'secondary'">
                {{ incident.ai_enabled ? 'Yes' : 'No' }}
              </Badge>
            </div>
            <div v-if="incident.ai_provider">
              <div class="text-muted-foreground mb-0.5">Provider</div>
              <div class="font-medium capitalize">{{ incident.ai_provider }}</div>
            </div>
            <div v-if="incident.ai_failure_reason">
              <div class="text-muted-foreground mb-0.5">Status</div>
              <div class="text-xs text-amber-600 dark:text-amber-400">{{ incident.ai_failure_reason }}</div>
            </div>
            <div v-if="incident.ai_reasoning">
              <div class="text-muted-foreground mb-0.5">Reasoning</div>
              <div class="text-xs">{{ incident.ai_reasoning }}</div>
            </div>
          </CardContent>
        </Card>
      </div>

      <!-- Affected Nodes (from probe) -->
      <Card v-if="problematicNodes.length > 0">
        <CardHeader>
          <CardTitle class="text-lg flex items-center gap-2">
            <Bug class="w-5 h-5 text-red-500" />
            Affected Nodes
            <Badge variant="destructive" class="ml-2">{{ problematicNodes.length }}</Badge>
          </CardTitle>
          <CardDescription>Nodes that failed or returned no results during probe</CardDescription>
        </CardHeader>
        <CardContent class="space-y-3">
          <div 
            v-for="(node, idx) in problematicNodes" 
            :key="idx"
            class="border rounded-lg p-4 space-y-3"
          >
            <div class="flex items-center justify-between flex-wrap gap-2">
              <div class="flex items-center gap-2">
                <Badge variant="outline">{{ node.phaseName }}</Badge>
                <span class="font-medium">{{ node.node_name }}</span>
                <Badge variant="secondary" class="text-xs">{{ node.node_type }}</Badge>
              </div>
              <Badge :variant="node.status === 'failed' ? 'destructive' : 'outline'" class="text-xs">
                {{ node.status === 'failed' ? 'Failed' : '0 results' }}
              </Badge>
            </div>
            
            <!-- Selector -->
            <div v-if="node.selector" class="space-y-1">
              <div class="flex items-center gap-2 text-sm text-muted-foreground">
                <Crosshair class="w-3.5 h-3.5" />
                CSS Selector
              </div>
              <code class="block bg-muted px-3 py-2 rounded text-xs font-mono break-all">{{ node.selector }}</code>
            </div>
            
            <!-- Selector Context -->
            <div v-if="node.selector_context" class="space-y-1">
              <div class="flex items-center gap-2 text-sm text-muted-foreground">
                <FileCode class="w-3.5 h-3.5" />
                HTML Context
              </div>
              <pre class="bg-muted/50 border px-3 py-2 rounded text-xs overflow-x-auto max-h-24">{{ node.selector_context }}</pre>
            </div>
            
            <!-- Snapshot Info -->
            <div v-if="node.snapshot" class="flex flex-wrap gap-4 text-xs text-muted-foreground">
              <div v-if="node.snapshot.page_title" class="flex items-center gap-1">
                <span class="font-medium">Page:</span> {{ node.snapshot.page_title }}
              </div>
              <div v-if="node.snapshot.dom_size" class="flex items-center gap-1">
                <span class="font-medium">DOM Size:</span> {{ formatBytes(node.snapshot.dom_size) }}
              </div>
              <div v-if="node.snapshot.image_width" class="flex items-center gap-1">
                <span class="font-medium">Screenshot:</span> {{ node.snapshot.image_width }}x{{ node.snapshot.image_height }}
              </div>
            </div>
            
            <!-- Error message if any -->
            <div v-if="node.error" class="text-sm text-red-600 dark:text-red-400 bg-red-500/10 p-2 rounded">
              {{ node.error }}
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- Error Message -->
      <Card>
        <CardHeader>
          <CardTitle class="text-lg flex items-center gap-2">
            <AlertTriangle class="w-5 h-5 text-red-500" />
            Error Details
          </CardTitle>
        </CardHeader>
        <CardContent>
          <pre class="bg-muted p-4 rounded-lg text-sm overflow-x-auto whitespace-pre-wrap">{{ incident.error_message || 'No error message' }}</pre>
        </CardContent>
      </Card>

      <!-- Suggested Actions -->
      <Card v-if="incident.suggested_actions?.length">
        <CardHeader>
          <CardTitle class="text-lg flex items-center gap-2">
            <Lightbulb class="w-5 h-5 text-yellow-500" />
            Suggested Actions
            <Badge variant="outline" class="ml-2">{{ incident.suggested_actions.length }}</Badge>
          </CardTitle>
          <CardDescription>Recommended steps to investigate and resolve this incident</CardDescription>
        </CardHeader>
        <CardContent>
          <ul class="space-y-2">
            <li 
              v-for="(action, idx) in incident.suggested_actions" 
              :key="idx"
              class="flex items-start gap-3 text-sm"
            >
              <span class="flex-shrink-0 w-6 h-6 rounded-full bg-primary/10 text-primary text-xs flex items-center justify-center font-medium">
                {{ idx + 1 }}
              </span>
              <span>{{ action }}</span>
            </li>
          </ul>
        </CardContent>
      </Card>

      <!-- Snapshot Paths -->
      <Card v-if="snapshotPaths.length > 0">
        <CardHeader>
          <CardTitle class="text-lg flex items-center gap-2">
            <Image class="w-5 h-5" />
            Snapshot Files
            <Badge variant="outline" class="ml-2">{{ snapshotPaths.length }}</Badge>
          </CardTitle>
          <CardDescription>Local snapshot files captured during probe execution</CardDescription>
        </CardHeader>
        <CardContent>
          <div class="space-y-2">
            <div 
              v-for="(path, idx) in snapshotPaths" 
              :key="idx"
              class="flex items-center gap-2 text-sm"
            >
              <Badge variant="secondary" class="text-xs shrink-0">
                {{ path.endsWith('.png') ? 'Screenshot' : 'DOM' }}
              </Badge>
              <code class="text-xs bg-muted px-2 py-1 rounded break-all">{{ path }}</code>
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- Probe Phases Summary -->
      <Card v-if="probePhases.length > 0">
        <CardHeader>
          <CardTitle class="text-lg flex items-center gap-2">
            <Workflow class="w-5 h-5" />
            Probe Phases
            <Badge variant="outline" class="ml-2">{{ probePhases.length }}</Badge>
          </CardTitle>
          <CardDescription>Workflow phases executed during probe</CardDescription>
        </CardHeader>
        <CardContent>
          <div class="space-y-3">
            <div 
              v-for="(phase, idx) in probePhases" 
              :key="idx"
              class="border rounded-lg p-3"
            >
              <div class="flex items-center justify-between mb-2">
                <div class="flex items-center gap-2">
                  <span class="font-medium">{{ phase.phase_name }}</span>
                  <Badge :variant="phase.status === 'passed' ? 'outline' : 'destructive'" class="text-xs">
                    {{ phase.status }}
                  </Badge>
                </div>
                <span class="text-xs text-muted-foreground">{{ phase.duration_ms }}ms</span>
              </div>
              <div class="text-xs text-muted-foreground">
                {{ phase.nodes?.length || 0 }} nodes executed
                <span v-if="phase.sample_url" class="ml-2">
                  • <a :href="phase.sample_url" target="_blank" class="hover:underline">{{ phase.sample_url }}</a>
                </span>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- Recovery Attempts -->
      <Card>
        <CardHeader>
          <CardTitle class="text-lg flex items-center gap-2">
            <RefreshCw class="w-5 h-5" />
            Recovery Attempts
            <Badge variant="outline" class="ml-2">{{ incident.recovery_attempts?.length || 0 }}</Badge>
          </CardTitle>
          <CardDescription>Automated recovery actions that were tried</CardDescription>
        </CardHeader>
        <CardContent>
          <div v-if="!incident.recovery_attempts?.length" class="text-muted-foreground text-sm py-4 text-center">
            No recovery attempts recorded
          </div>
          <div v-else class="space-y-3">
            <div 
              v-for="(attempt, index) in incident.recovery_attempts" 
              :key="index"
              class="flex items-start gap-3 p-3 border rounded-lg"
            >
              <div class="w-8 h-8 rounded-full flex items-center justify-center shrink-0"
                :class="attempt.success ? 'bg-green-500/10' : 'bg-red-500/10'">
                <CheckCircle2 v-if="attempt.success" class="w-4 h-4 text-green-500" />
                <XCircle v-else class="w-4 h-4 text-red-500" />
              </div>
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2 mb-1">
                  <span class="font-medium capitalize">{{ attempt.action.replace(/_/g, ' ') }}</span>
                  <Badge variant="outline" class="text-xs">{{ attempt.source }}</Badge>
                </div>
                <div class="text-sm text-muted-foreground">
                  {{ formatDate(attempt.timestamp) }}
                </div>
                <div v-if="Object.keys(attempt.params || {}).length > 0" class="mt-2">
                  <pre class="text-xs bg-muted p-2 rounded overflow-x-auto">{{ JSON.stringify(attempt.params, null, 2) }}</pre>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- Resolution -->
      <Card v-if="incident.resolution">
        <CardHeader>
          <CardTitle class="text-lg flex items-center gap-2">
            <CheckCircle2 class="w-5 h-5 text-green-500" />
            Resolution
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p>{{ incident.resolution }}</p>
        </CardContent>
      </Card>
    </div>
  </PageLayout>
</template>
