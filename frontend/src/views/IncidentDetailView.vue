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
  FileText,
  RefreshCw,
  XCircle,
  Play
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const resolving = ref(false)
const incident = ref<Incident | null>(null)

const incidentId = computed(() => route.params.id as string)

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

const formatDate = (dateString?: string) => {
  if (!dateString) return 'N/A'
  return new Date(dateString).toLocaleString()
}

const formatErrorPattern = (pattern: string) => {
  return pattern.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
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
        <a :href="incident.url" target="_blank" class="inline-flex">
          <Button variant="outline">
            <ExternalLink class="w-4 h-4 mr-2" />
            Open URL
          </Button>
        </a>
      </div>

      <!-- Info Cards Grid -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <!-- Basic Info -->
        <Card>
          <CardHeader>
            <CardTitle class="text-lg flex items-center gap-2">
              <Globe class="w-5 h-5" />
              Basic Information
            </CardTitle>
          </CardHeader>
          <CardContent class="space-y-3">
            <div>
              <div class="text-sm text-muted-foreground">Domain</div>
              <div class="font-medium">{{ incident.domain }}</div>
            </div>
            <div>
              <div class="text-sm text-muted-foreground">URL</div>
              <div class="font-mono text-sm truncate">{{ incident.url }}</div>
            </div>
            <div>
              <div class="text-sm text-muted-foreground">Error Pattern</div>
              <Badge variant="outline">{{ formatErrorPattern(incident.error_pattern) }}</Badge>
            </div>
            <div>
              <div class="text-sm text-muted-foreground">Status Code</div>
              <div class="font-medium">{{ incident.status_code || 'N/A' }}</div>
            </div>
          </CardContent>
        </Card>

        <!-- Timeline -->
        <Card>
          <CardHeader>
            <CardTitle class="text-lg flex items-center gap-2">
              <Clock class="w-5 h-5" />
              Timeline
            </CardTitle>
          </CardHeader>
          <CardContent class="space-y-3">
            <div>
              <div class="text-sm text-muted-foreground">Created</div>
              <div class="font-medium">{{ formatDate(incident.created_at) }}</div>
            </div>
            <div>
              <div class="text-sm text-muted-foreground">Last Updated</div>
              <div class="font-medium">{{ formatDate(incident.updated_at) }}</div>
            </div>
            <div v-if="incident.resolved_at">
              <div class="text-sm text-muted-foreground">Resolved At</div>
              <div class="font-medium text-green-600">{{ formatDate(incident.resolved_at) }}</div>
            </div>
            <div v-if="incident.assigned_to">
              <div class="text-sm text-muted-foreground">Assigned To</div>
              <div class="font-medium">{{ incident.assigned_to }}</div>
            </div>
          </CardContent>
        </Card>

        <!-- Error Message -->
        <Card class="md:col-span-2">
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

        <!-- Recovery Attempts -->
        <Card class="md:col-span-2">
          <CardHeader>
            <CardTitle class="text-lg flex items-center gap-2">
              <RefreshCw class="w-5 h-5" />
              Recovery Attempts
              <Badge variant="outline" class="ml-2">{{ incident.recovery_attempts?.length || 0 }}</Badge>
            </CardTitle>
            <CardDescription>Automated recovery actions that were tried</CardDescription>
          </CardHeader>
          <CardContent>
            <div v-if="!incident.recovery_attempts?.length" class="text-muted-foreground text-sm">
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

        <!-- Page Content Preview -->
        <Card v-if="incident.page_content" class="md:col-span-2">
          <CardHeader>
            <CardTitle class="text-lg flex items-center gap-2">
              <FileText class="w-5 h-5" />
              Page Content Preview
            </CardTitle>
          </CardHeader>
          <CardContent>
            <pre class="bg-muted p-4 rounded-lg text-xs overflow-x-auto max-h-64 whitespace-pre-wrap">{{ incident.page_content.slice(0, 2000) }}{{ incident.page_content.length > 2000 ? '...' : '' }}</pre>
          </CardContent>
        </Card>

        <!-- Resolution -->
        <Card v-if="incident.resolution" class="md:col-span-2">
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
    </div>
  </PageLayout>
</template>
