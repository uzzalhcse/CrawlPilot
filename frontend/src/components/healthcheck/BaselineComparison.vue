<template>
  <div class="baseline-comparison">
    <!-- Loading State -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>Loading baseline comparison...</p>
    </div>

    <!-- Empty State - No Baseline -->
    <div v-else-if="error && error.includes('No baseline')" class="empty-state">
      <div class="empty-icon">📊</div>
      <h3>No Baseline Set</h3>
      <p>Set a baseline to track metric changes over time</p>
      <button @click="$emit('set-baseline')" class="btn-primary">
        Set Current Report as Baseline
      </button>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-state">
      <div class="error-icon">⚠️</div>
      <p>{{ error }}</p>
      <button @click="loadComparison" class="btn-secondary">
        Try Again
      </button>
    </div>

    <!-- Comparison Content -->
    <div v-else-if="comparison" class="comparison-content">
      <!-- Header -->
      <div class="comparison-header">
        <div>
          <h3>Baseline Comparison</h3>
          <p class="baseline-date">
            Baseline set: {{ formatDate(comparison.baseline.started_at) }}
          </p>
        </div>
        <div class="overall-status" :class="`status-${getOverallStatus()}`">
          <span class="status-icon">{{ getStatusIcon() }}</span>
          <span class="status-text">{{ getOverallStatus() }}</span>
        </div>
      </div>

      <!-- Metrics Grid -->
      <div class="metrics-grid">
        <div 
          v-for="comp in comparison.comparisons" 
          :key="comp.metric"
          class="metric-card"
          :class="`status-${comp.status}`"
        >
          <!-- Metric Header -->
          <div class="metric-header">
            <span class="metric-name">{{ formatMetricName(comp.metric) }}</span>
            <span class="status-badge" :class="`badge-${comp.status}`">
              <span class="badge-icon">{{ getMetricIcon(comp.status) }}</span>
              {{ comp.status }}
            </span>
          </div>

          <!-- Values Comparison -->
          <div class="value-comparison">
            <div class="value-box baseline">
              <span class="value-label">Baseline</span>
              <span class="value-number">{{ formatValue(comp.baseline) }}</span>
            </div>

            <div class="change-indicator" :class="`change-${comp.status}`">
              <span class="arrow">{{ getChangeArrow(comp.status) }}</span>
              <span v-if="comp.change_percent !== undefined && comp.change_percent !== 0" class="change-percent">
                {{ Math.abs(comp.change_percent).toFixed(1) }}%
              </span>
            </div>

            <div class="value-box current">
              <span class="value-label">Current</span>
              <span class="value-number">{{ formatValue(comp.current) }}</span>
            </div>
          </div>

          <!-- Change Summary -->
          <div v-if="comp.change_percent !== undefined && comp.change_percent !== 0" class="metric-summary">
            {{ getChangeSummary(comp) }}
          </div>
        </div>
      </div>

      <!-- Timeline Info -->
      <div class="timeline-info">
        <div class="timeline-item">
          <span class="timeline-label">Current Report:</span>
          <span class="timeline-value">{{ formatDate(comparison.current.started_at) }}</span>
        </div>
        <div class="timeline-item">
          <span class="timeline-label">Time Since Baseline:</span>
          <span class="timeline-value">{{ getTimeSinceBaseline() }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { workflowsApi } from '@/api/workflows'
import type { ComparisonResponse } from '@/types'

const props = defineProps<{
  reportId: string
}>()

defineEmits<{
  'set-baseline': []
}>()

const comparison = ref<ComparisonResponse | null>(null)
const loading = ref(false)
const error = ref('')

onMounted(async () => {
  await loadComparison()
})

async function loadComparison() {
  loading.value = true
  error.value = ''
  
  try {
    const response = await workflowsApi.compareWithBaseline(props.reportId)
    comparison.value = response.data
  } catch (err: any) {
    if (err.response?.status === 404) {
      error.value = 'No baseline found for this workflow'
    } else {
      error.value = 'Failed to load comparison'
    }
  } finally {
    loading.value = false
  }
}

function formatMetricName(metric: string): string {
  return metric
    .replace(/_/g, ' ')
    .replace(/\b\w/g, l => l.toUpperCase())
}

function formatValue(value: any): string {
  if (typeof value === 'number') {
    return value.toString()
  }
  return String(value)
}

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleString()
}

function getOverallStatus(): string {
  if (!comparison.value) return 'unchanged'
  
  const hasImproved = comparison.value.comparisons.some(c => c.status === 'improved')
  const hasDegraded = comparison.value.comparisons.some(c => c.status === 'degraded')
  
  if (hasDegraded) return 'degraded'
  if (hasImproved) return 'improved'
  return 'unchanged'
}

function getStatusIcon(): string {
  const status = getOverallStatus()
  return status === 'improved' ? '✓' : status === 'degraded' ? '⚠' : '='
}

function getMetricIcon(status: string): string {
  return status === 'improved' ? '↑' : status === 'degraded' ? '↓' : '='
}

function getChangeArrow(status: string): string {
  return status === 'improved' ? '→' : status === 'degraded' ? '→' : '='
}

function getChangeSummary(comp: any): string {
  const abs = Math.abs(comp.change_percent).toFixed(1)
  if (comp.status === 'improved') {
    return `Improved by ${abs}%`
  } else if (comp.status === 'degraded') {
    return `Decreased by ${abs}%`
  }
  return 'No change'
}

function getTimeSinceBaseline(): string {
  if (!comparison.value) return ''
  
  const baseline = new Date(comparison.value.baseline.started_at)
  const current = new Date(comparison.value.current.started_at)
  const diff = current.getTime() - baseline.getTime()
  
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))
  
  if (days > 0) {
    return `${days} day${days > 1 ? 's' : ''} ${hours}h`
  }
  return `${hours} hour${hours > 1 ? 's' : ''}`
}

defineExpose({ loadComparison })
</script>

<style scoped>
.baseline-comparison {
  min-height: 400px;
}

/* Loading & Empty States */
.loading-state,
.empty-state,
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  text-align: center;
}

.spinner {
  width: 48px;
  height: 48px;
  border: 4px solid hsl(var(--border));
  border-top-color: hsl(var(--primary));
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin-bottom: 1rem;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.empty-icon,
.error-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
}

.empty-state h3 {
  font-size: 1.25rem;
  font-weight: 600;
  color: hsl(var(--foreground));
  margin-bottom: 0.5rem;
}

.empty-state p {
  color: hsl(var(--muted-foreground));
  margin-bottom: 1.5rem;
}

.btn-primary,
.btn-secondary {
  padding: 0.65rem 1.5rem;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-primary {
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
  border: none;
}

.btn-primary:hover {
  background: hsl(var(--primary) / 0.9);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px hsl(var(--primary) / 0.3);
}

.btn-secondary {
  background: transparent;
  border: 1px solid hsl(var(--border));
  color: hsl(var(--foreground));
}

.btn-secondary:hover {
  background: hsl(var(--muted) / 0.5);
}

/* Comparison Header */
.comparison-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 1.5rem;
  background: hsl(var(--muted) / 0.3);
  border-radius: 8px 8px 0 0;
  border-bottom: 2px solid hsl(var(--border));
}

.comparison-header h3 {
  font-size: 1.25rem;
  font-weight: 600;
  color: hsl(var(--foreground));
  margin: 0 0 0.25rem 0;
}

.baseline-date {
  font-size: 0.875rem;
  color: hsl(var(--muted-foreground));
  margin: 0;
}

.overall-status {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border-radius: 6px;
  font-weight: 600;
  font-size: 0.9rem;
  text-transform: capitalize;
}

.overall-status.status-improved {
  background: hsl(142 76% 36% / 0.15);
  color: hsl(142 76% 36%);
  border: 1px solid hsl(142 76% 36% / 0.3);
}

.overall-status.status-degraded {
  background: hsl(var(--destructive) / 0.15);
  color: hsl(var(--destructive));
  border: 1px solid hsl(var(--destructive) / 0.3);
}

.overall-status.status-unchanged {
  background: hsl(var(--muted) / 0.5);
  color: hsl(var(--muted-foreground));
  border: 1px solid hsl(var(--border));
}

.status-icon {
  font-size: 1.2rem;
}

/* Metrics Grid */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 1.25rem;
  padding: 1.5rem;
  background: hsl(var(--card));
}

.metric-card {
  padding: 1.25rem;
  border: 2px solid hsl(var(--border));
  border-radius: 8px;
  background: hsl(var(--background));
  transition: all 0.2s;
}

.metric-card:hover {
  box-shadow: 0 4px 12px hsl(var(--foreground) / 0.1);
}

.metric-card.status-improved {
  border-color: hsl(142 76% 36% / 0.4);
  background: hsl(142 76% 36% / 0.03);
}

.metric-card.status-degraded {
  border-color: hsl(var(--destructive) / 0.4);
  background: hsl(var(--destructive) / 0.03);
}

.metric-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.25rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid hsl(var(--border) / 0.5);
}

.metric-name {
  font-weight: 600;
  font-size: 0.95rem;
  color: hsl(var(--foreground));
}

.status-badge {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.35rem 0.75rem;
  border-radius: 12px;
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: capitalize;
}

.badge-improved {
  background: hsl(142 76% 36% / 0.15);
  color: hsl(142 76% 36%);
}

.badge-degraded {
  background: hsl(var(--destructive) / 0.15);
  color: hsl(var(--destructive));
}

.badge-unchanged {
  background: hsl(var(--muted));
  color: hsl(var(--muted-foreground));
}

.badge-icon {
  font-size: 1rem;
}

/* Value Comparison */
.value-comparison {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: 1rem;
  align-items: center;
  margin-bottom: 1rem;
}

.value-box {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: 0.75rem;
  border-radius: 6px;
  text-align: center;
}

.value-box.baseline {
  background: hsl(var(--muted) / 0.3);
}

.value-box.current {
  background: hsl(var(--primary) / 0.1);
}

.value-label {
  font-size: 0.75rem;
  font-weight: 500;
  color: hsl(var(--muted-foreground));
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.value-number {
  font-size: 1.75rem;
  font-weight: 700;
  color: hsl(var(--foreground));
  line-height: 1;
}

.change-indicator {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
}

.arrow {
  font-size: 1.5rem;
  color: hsl(var(--muted-foreground));
}

.change-percent {
  font-size: 0.9rem;
  font-weight: 700;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

.change-improved .change-percent {
  background: hsl(142 76% 36% / 0.15);
  color: hsl(142 76% 36%);
}

.change-degraded .change-percent {
  background: hsl(var(--destructive) / 0.15);
  color: hsl(var(--destructive));
}

/* Metric Summary */
.metric-summary {
  padding: 0.65rem;
  background: hsl(var(--muted) / 0.3);
  border-radius: 4px;
  text-align: center;
  font-size: 0.85rem;
  color: hsl(var(--foreground));
  font-weight: 500;
}

/* Timeline Info */
.timeline-info {
  display: flex;
  justify-content: space-around;
  padding: 1.5rem;
  background: hsl(var(--muted) / 0.2);
  border-radius: 0 0 8px 8px;
  border-top: 1px solid hsl(var(--border));
}

.timeline-item {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  text-align: center;
}

.timeline-label {
  font-size: 0.8rem;
  color: hsl(var(--muted-foreground));
  font-weight: 500;
}

.timeline-value {
  font-size: 0.95rem;
  color: hsl(var(--foreground));
  font-weight: 600;
}
</style>
