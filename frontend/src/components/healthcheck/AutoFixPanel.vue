<template>
  <div v-if="snapshot" class="autofix-panel">
    <button 
      @click="analyzeWithAI" 
      :disabled="analyzing || suggestions.length > 0"
      class="analyze-button"
    >
      <Sparkles class="w-4 h-4" />
      <span v-if="analyzing">Analyzing with AI...</span>
      <span v-else-if="suggestions.length > 0">AI Analyzed</span>
      <span v-else>Analyze with AI</span>
    </button>

    <div v-if="suggestions.length > 0" class="suggestions-list">
      <div v-for="suggestion in suggestions" :key="suggestion.id" class="suggestion-card">
        <div class="suggestion-header">
          <div class="confidence">
            <TrendingUp class="w-4 h-4" />
            <span>{{ (suggestion.confidence_score * 100).toFixed(0) }}% confidence</span>
          </div>
          <span :class="['badge', suggestion.status]">{{ suggestion.status }}</span>
        </div>

        <div class="suggestion-body">
          <div class="selector-comparison">
            <div class="current">
              <label>Current (Failed):</label>
              <code>{{ snapshot.selector_value }}</code>
            </div>
            <ArrowRight class="w-4 h-4 arrow" />
            <div class="suggested">
              <label>Suggested:</label>
              <code>{{ suggestion.suggested_selector }}</code>
            </div>
          </div>

          <div class="explanation">
            <Info class="w-4 h-4" />
            <p>{{ suggestion.fix_explanation }}</p>
          </div>

          <!-- Verification Result -->
          <div v-if="suggestion.verification_result" class="verification-result">
            <div class="verification-header">
              <div class="verification-status" :class="{ 'valid': suggestion.verification_result.is_valid, 'invalid': !suggestion.verification_result.is_valid }">
                <Check v-if="suggestion.verification_result.is_valid" class="w-4 h-4" />
                <AlertTriangle v-else class="w-4 h-4" />
                <span>{{ suggestion.verification_result.is_valid ? 'Verified' : 'Verification Failed' }}</span>
              </div>
              <span class="element-count" v-if="suggestion.verification_result.is_valid">
                {{ suggestion.verification_result.elements_found }} elements found
              </span>
            </div>
            
            <div v-if="suggestion.verification_result.error_message" class="verification-error">
              {{ suggestion.verification_result.error_message }}
            </div>

            <div v-if="suggestion.verification_result.data_preview && suggestion.verification_result.data_preview.length > 0" class="data-preview">
              <span class="preview-label">Data Preview:</span>
              <ul class="preview-list">
                <li v-for="(item, index) in suggestion.verification_result.data_preview" :key="index">
                  {{ item }}
                </li>
              </ul>
            </div>
          </div>

          <div v-if="suggestion.status === 'pending'" class="actions">
            <button @click="approve(suggestion.id)" class="btn-approve">
              <Check class="w-4 h-4" />
              Approve
            </button>
            <button @click="reject(suggestion.id)" class="btn-reject">
              <X class="w-4 h-4" />
              Reject
            </button>
          </div>

          <div v-if="suggestion.status === 'approved'" class="actions">
            <button @click="apply(suggestion.id)" class="btn-apply">
              <Wand2 class="w-4 h-4" />
              Apply Fix
            </button>
          </div>

          <div v-if="suggestion.status === 'applied'" class="actions">
            <button @click="revert(suggestion.id)" class="btn-revert">
              <RotateCcw class="w-4 h-4" />
              Revert
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Sparkles, TrendingUp, ArrowRight, Info, Check, X, Wand2, RotateCcw, AlertTriangle } from 'lucide-vue-next'
import { workflowsApi } from '@/api/workflows'
import type { HealthCheckSnapshot, FixSuggestion } from '@/types'

interface Props {
  snapshot: HealthCheckSnapshot | null
}

const props = defineProps<Props>()

const analyzing = ref(false)
const suggestions = ref<FixSuggestion[]>([])

onMounted(async () => {
  if (props.snapshot) {
    await loadSuggestions()
  }
})

async function loadSuggestions() {
  if (!props.snapshot) return
  try {
    const res = await workflowsApi.getSuggestions(props.snapshot.id)
    suggestions.value = res.data || []
  } catch (err) {
    console.error('Failed to load suggestions:', err)
  }
}

async function analyzeWithAI() {
  if (!props.snapshot) return
  analyzing.value = true
  try {
    const res = await workflowsApi.analyzeSnapshot(props.snapshot.id)
    suggestions.value = [res.data]
  } catch (err: any) {
    alert('AI analysis failed: ' + (err.response?.data?.error || err.message))
  } finally {
    analyzing.value = false
  }
}

async function approve(id: string) {
  await workflowsApi.approveSuggestion(id)
  await loadSuggestions()
}

async function reject(id: string) {
  await workflowsApi.rejectSuggestion(id)
  await loadSuggestions()
}

async function apply(id: string) {
  await workflowsApi.applySuggestion(id)
  await loadSuggestions()
  alert('Fix applied! Re-run health check to verify.')
}

async function revert(id: string) {
  await workflowsApi.revertSuggestion(id)
  await loadSuggestions()
}
</script>

<style scoped>
.autofix-panel {
  @apply space-y-4;
}

.analyze-button {
  @apply flex items-center gap-2 px-4 py-2.5 rounded-lg font-medium;
  @apply bg-gradient-to-r from-purple-500 to-blue-500 text-white;
  @apply hover:from-purple-600 hover:to-blue-600 transition-all;
  @apply disabled:opacity-50 disabled:cursor-not-allowed;
}

.suggestions-list {
  @apply space-y-3;
}

.suggestion-card {
  @apply bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-800 dark:to-gray-900;
  @apply p-4 rounded-lg border border-gray-200 dark:border-gray-700;
}

.suggestion-header {
  @apply flex items-center justify-between mb-3;
}

.confidence {
  @apply flex items-center gap-2 text-sm font-medium text-blue-600 dark:text-blue-400;
}

.badge {
  @apply px-2 py-1 text-xs rounded-full font-medium;
}

.badge.pending { @apply bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400; }
.badge.approved { @apply bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400; }
.badge.applied { @apply bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400; }
.badge.rejected { @apply bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400; }

.suggestion-body {
  @apply space-y-3;
}

.selector-comparison {
  @apply grid grid-cols-[1fr_auto_1fr] gap-3 items-center;
  @apply p-3 bg-white dark:bg-gray-950 rounded-lg;
}

.selector-comparison label {
  @apply text-xs font-medium text-gray-500 dark:text-gray-400 mb-1 block;
}

.selector-comparison code {
  @apply block p-2 bg-gray-50 dark:bg-gray-900 rounded text-sm font-mono;
  @apply border border-gray-200 dark:border-gray-800;
}

.selector-comparison .current code {
  @apply text-red-600 dark:text-red-400 border-red-200 dark:border-red-900;
}

.selector-comparison .suggested code {
  @apply text-green-600 dark:text-green-400 border-green-200 dark:border-green-900;
}

.arrow {
  @apply text-gray-400;
}

.explanation {
  @apply flex gap-2 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg;
  @apply text-sm text-gray-700 dark:text-gray-300;
}

.actions {
  @apply flex gap-2;
}

.actions button {
  @apply flex items-center gap-2 px-3 py-2 rounded-lg font-medium text-sm;
  @apply transition-colors;
}

.btn-approve {
  @apply bg-green-500 text-white hover:bg-green-600;
}

.btn-reject {
  @apply bg-red-500 text-white hover:bg-red-600;
}

.btn-apply {
  @apply bg-blue-500 text-white hover:bg-blue-600;
}

.btn-revert {
  @apply bg-gray-500 text-white hover:bg-gray-600;
}

.verification-result {
  @apply space-y-2 p-3 bg-gray-50 dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700;
}

.verification-header {
  @apply flex items-center justify-between;
}

.verification-status {
  @apply flex items-center gap-2 text-sm font-medium;
}

.verification-status.valid {
  @apply text-green-600 dark:text-green-400;
}

.verification-status.invalid {
  @apply text-red-600 dark:text-red-400;
}

.element-count {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

.verification-error {
  @apply text-sm text-red-600 dark:text-red-400;
}

.data-preview {
  @apply space-y-1;
}

.preview-label {
  @apply text-xs font-medium text-gray-500 dark:text-gray-400;
}

.preview-list {
  @apply list-disc list-inside text-xs text-gray-700 dark:text-gray-300 space-y-0.5 pl-2;
}
</style>
