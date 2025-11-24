<template>
  <TransitionRoot appear :show="isOpen" as="template">
    <Dialog as="div" @close="close" class="relative z-50">
      <TransitionChild
        as="template"
        enter="duration-300 ease-out"
        enter-from="opacity-0"
        enter-to="opacity-100"
        leave="duration-200 ease-in"
        leave-from="opacity-100"
        leave-to="opacity-0"
      >
        <div class="fixed inset-0 bg-black/70" />
      </TransitionChild>

      <div class="fixed inset-0 overflow-y-auto">
        <div class="flex min-h-full items-center justify-center p-4">
          <TransitionChild
            as="template"
            enter="duration-300 ease-out"
            enter-from="opacity-0 scale-95"
            enter-to="opacity-100 scale-100"
            leave="duration-200 ease-in"
            leave-from="opacity-100 scale-100"
            leave-to="opacity-0 scale-95"
          >
            <DialogPanel class="snapshot-modal">
              <!-- Header -->
              <div class="modal-header">
                <DialogTitle class="modal-title">
                  <Camera class="w-6 h-6" />
                  <span>Diagnostic Snapshot</span>
                </DialogTitle>
                <button @click="close" class="close-button">
                  <X class="w-5 h-5" />
                </button>
              </div>

              <!-- Loading State -->
              <div v-if="loading" class="loading-state">
                <div class="spinner" />
                <p>Loading snapshot...</p>
              </div>

              <!-- Error State -->
              <div v-else-if="error" class="error-state">
                <AlertCircle class="w-12 h-12 text-red-500" />
                <p class="text-red-600">{{ error }}</p>
              </div>

              <!-- Content -->
              <div v-else-if="snapshot && snapshot.id" class="modal-content">                <!-- Tabs -->
                <div class="tabs">
                  <button
                    v-for="tab in tabs"
                    :key="tab.id"
                    @click="activeTab = tab.id"
                    :class="['tab', { active: activeTab === tab.id }]"
                  >
                    <component :is="tab.icon" class="w-4 h-4" />
                    <span>{{ tab.label }}</span>
                  </button>
                </div>

                <!--Tab Content -->
                <div class="tab-content">
                  <!-- Screenshot Tab -->
                  <div v-if="activeTab === 'screenshot'" class="screenshot-tab">
                    <div v-if="!snapshot.screenshot_path" class="empty-state">
                      <ImageOff class="w-12 h-12 text-gray-400" />
                      <p>No screenshot available</p>
                    </div>
                    <div v-else class="screenshot-container">
                      <div class="screenshot-actions">
                        <button @click="viewFullScreen" class="action-button" title="Open in new tab">
                          <Maximize2 class="w-4 h-4" />
                          <span>Full Screen</span>
                        </button>
                      </div>
                      <img 
                        :src="screenshotUrl" 
                        alt="Page screenshot"
                        :class="['screenshot-image', { 'zoomed': imageZoom }]"
                        @click="imageZoom = !imageZoom"
                      />
                    </div>
                  </div>

                  <!-- DOM Tab -->
                  <div v-if="activeTab === 'dom'" class="dom-tab">
                    <div class="dom-actions">
                      <button @click="viewDOM" class="action-button">
                        <Eye class="w-4 h-4" />
                        <span>View in New Tab</span>
                      </button>
                      <button @click="downloadDOM" class="action-button">
                        <Download class="w-4 h-4" />
                        <span>Download HTML</span>
                      </button>
                    </div>
                    <div class="dom-info">
                      <FileCode class="w-5 h-5" />
                      <span>Full HTML snapshot with styles and scripts</span>
                    </div>
                  </div>

                  <!-- Console Tab -->
                  <div v-if="activeTab === 'console'" class="console-tab">
                    <div v-if="snapshot.console_logs && snapshot.console_logs.length > 0" class="console-logs">
                      <div
                        v-for="(log, index) in snapshot.console_logs"
                        :key="index"
                        :class="['console-log', `log-${log.type}`]"
                      >
                        <div class="log-icon">
                          <AlertCircle v-if="log.type === 'error'" class="w-4 h-4" />
                          <AlertTriangle v-else-if="log.type === 'warn'" class="w-4 h-4" />
                          <Info v-else class="w-4 h-4" />
                        </div>
                        <div class="log-content">
                          <div class="log-message">{{ log.message }}</div>
                          <div class="log-meta">
                            <span>{{ log.type }}</span>
                            <span>•</span>
                            <span>{{ formatDate(log.timestamp) }}</span>
                          </div>
                        </div>
                      </div>
                    </div>
                    <div v-else class="empty-state">
                      <Terminal class="w-12 h-12" />
                      <p>No console logs captured</p>
                    </div>
                  </div>

                  <!-- Details Tab -->
                  <div v-if="activeTab === 'details'" class="details-tab">
                    <div class="detail-grid">
                      <!-- Page Info -->
                      <div class="detail-section">
                        <h4 class="section-title">Page Information</h4>
                        <div class="detail-item">
                          <span class="detail-label">URL:</span>
                          <a :href="snapshot.url" target="_blank" class="detail-value link">
                            {{ snapshot.url }}
                            <ExternalLink class="w-3 h-3" />
                          </a>
                        </div>
                        <div class="detail-item" v-if="snapshot.page_title">
                          <span class="detail-label">Title:</span>
                          <span class="detail-value">{{ snapshot.page_title }}</span>
                        </div>
                        <div class="detail-item" v-if="snapshot.status_code">
                          <span class="detail-label">Status Code:</span>
                          <span class="detail-value">{{ snapshot.status_code }}</span>
                        </div>
                      </div>

                      <!-- Error Info -->
                      <div class="detail-section">
                        <h4 class="section-title">Error Details</h4>
                        <div class="detail-item" v-if="snapshot.selector_value">
                          <span class="detail-label">Selector:</span>
                          <code class="detail-value code">{{ snapshot.selector_value }}</code>
                        </div>
                        <div class="detail-item">
                          <span class="detail-label">Elements Found:</span>
                          <span class="detail-value">{{ snapshot.elements_found }}</span>
                        </div>
                        <div class="detail-item" v-if="snapshot.error_message">
                          <span class="detail-label">Error:</span>
                          <span class="detail-value error">{{ snapshot.error_message }}</span>
                        </div>
                      </div>

                      <!-- Metadata -->
                      <div class="detail-section" v-if="snapshot.metadata">
                        <h4 class="section-title">Metadata</h4>
                        <div class="detail-item" v-for="(value, key) in snapshot.metadata" :key="key">
                          <span class="detail-label">{{ formatKey(key) }}:</span>
                          <span class="detail-value">{{ formatValue(value) }}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { X, Camera, FileCode, Terminal, Info, AlertCircle, AlertTriangle, Eye, Download, ExternalLink, ImageOff, Maximize2 } from 'lucide-vue-next'
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue'
import { workflowsApi } from '@/api/workflows'
import type { HealthCheckSnapshot } from '@/types'

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

const tabs = [
  { id: 'screenshot', label: 'Screenshot', icon: Camera },
  { id: 'dom', label: 'DOM', icon: FileCode },
  { id: 'console', label: 'Console', icon: Terminal },
  { id: 'details', label: 'Details', icon: Info }
]

const screenshotUrl = computed(() => {
  if (!snapshot.value?.id) return ''
  return workflowsApi.getScreenshotUrl(snapshot.value.id)
})

const domUrl = computed(() => {
  if (!snapshot.value?.id) return ''
  return workflowsApi.getDOMUrl(snapshot.value.id)
})

// Load snapshot when dialog opens
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

// Also watch when dialog opens/closes
watch(() => props.open, (isOpen) => {
  if (!isOpen) {
    // Reset state when closing
    snapshot.value = null
    error.value = null
    imageZoom.value = false
    activeTab.value = 'screenshot'
  } else if (props.snapshotId) {
    // Load snapshot when opening
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
  // State will be reset by the watch on props.open
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

<style scoped>
.snapshot-modal {
  @apply w-full max-w-5xl transform overflow-hidden rounded-2xl;
  @apply bg-white dark:bg-gray-800 shadow-2xl transition-all;
}

.modal-header {
  @apply flex items-center justify-between p-6 border-b;
  @apply border-gray-200 dark:border-gray-700;
}

.modal-title {
  @apply flex items-center gap-3 text-xl font-semibold;
  @apply text-gray-900 dark:text-white;
}

.close-button {
  @apply p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700;
  @apply transition-colors text-gray-500 dark:text-gray-400;
}

.loading-state, .error-state {
  @apply flex flex-col items-center justify-center gap-4 p-12;
}

.spinner {
  @apply w-12 h-12 border-4 border-blue-200 dark:border-blue-900;
  @apply border-t-blue-500 rounded-full animate-spin;
}

.modal-content {
  @apply flex flex-col;
  max-height: 80vh;
}

.tabs {
  @apply flex gap-2 p-4 border-b border-gray-200 dark:border-gray-700;
  @apply bg-gray-50 dark:bg-gray-900/50;
}

.tab {
  @apply flex items-center gap-2 px-4 py-2 rounded-lg font-medium;
  @apply text-gray-600 dark:text-gray-400 transition-all;
  @apply hover:bg-white dark:hover:bg-gray-800;
}

.tab.active {
  @apply bg-white dark:bg-gray-800 text-blue-600 dark:text-blue-400;
  @apply shadow-sm;
}

.tab-content {
  @apply flex-1 overflow-y-auto p-6;
}

.screenshot-container {
  @apply relative;
}

.screenshot-actions {
  @apply flex gap-2 mb-3;
}

.action-button {
  @apply flex items-center gap-2 px-3 py-2 rounded-lg;
  @apply bg-gray-100 dark:bg-gray-700;
  @apply hover:bg-gray-200 dark:hover:bg-gray-600;
  @apply text-gray-700 dark:text-gray-200;
  @apply transition-colors text-sm font-medium;
}

.screenshot-image {
  @apply w-full rounded-lg cursor-zoom-in transition-transform;
}

.screenshot-image.zoomed {
  @apply cursor-zoom-out scale-150 origin-top-left;
}

.empty-state {
  @apply flex flex-col items-center gap-3 text-gray-400 dark:text-gray-600;
}

.dom-tab {
  @apply space-y-4;
}

.dom-actions {
  @apply flex gap-3;
}

.action-button {
  @apply flex items-center gap-2 px-4 py-2 rounded-lg;
  @apply bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400;
  @apply hover:bg-blue-100 dark:hover:bg-blue-900/30 transition-colors;
  @apply font-medium;
}

.dom-info {
  @apply flex items-center gap-2 p-4 rounded-lg;
  @apply bg-gray-50 dark:bg-gray-900 text-gray-600 dark:text-gray-400;
}

.console-logs {
  @apply space-y-2;
}

.console-log {
  @apply flex gap-3 p-4 rounded-lg font-mono text-sm;
  @apply border-l-4;
}

.log-error {
  @apply bg-red-50 dark:bg-red-900/20 border-red-500 text-red-700 dark:text-red-400;
}

.log-warn {
  @apply bg-yellow-50 dark:bg-yellow-900/20 border-yellow-500 text-yellow-700 dark:text-yellow-400;
}

.log-info, .log-log {
  @apply bg-blue-50 dark:bg-blue-900/20 border-blue-500 text-blue-700 dark:text-blue-400;
}

.log-content {
  @apply flex-1 space-y-1;
}

.log-message {
  @apply font-medium;
}

.log-meta {
  @apply flex gap-2 text-xs opacity-75;
}

.details-tab {
  @apply space-y-6;
}

.detail-grid {
  @apply grid gap-6 md:grid-cols-2;
}

.detail-section {
  @apply space-y-3;
}

.section-title {
  @apply text-sm font-semibold text-gray-900 dark:text-white;
  @apply pb-2 border-b border-gray-200 dark:border-gray-700;
}

.detail-item {
  @apply flex flex-col gap-1;
}

.detail-label {
  @apply text-xs font-medium text-gray-500 dark:text-gray-400;
}

.detail-value {
  @apply text-sm text-gray-900 dark:text-gray-100;
}

.detail-value.link {
  @apply flex items-center gap-1 text-blue-600 dark:text-blue-400;
  @apply hover:underline;
}

.detail-value.code {
  @apply bg-gray-100 dark:bg-gray-900 p-2 rounded font-mono text-xs;
  @apply break-all;
}

.detail-value.error {
  @apply text-red-600 dark:text-red-400;
}
</style>
