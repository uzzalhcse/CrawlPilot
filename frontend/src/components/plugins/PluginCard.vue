<template>
  <div class="plugin-card" @click="$emit('click')">
    <!-- Plugin Header -->
    <div class="plugin-header">
      <div class="plugin-icon">
        {{ getPhaseIcon(plugin.phase_type) }}
      </div>
      <div class="plugin-badges">
        <span v-if="plugin.is_verified" class="badge badge-verified" title="Verified Plugin">
          <svg viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M6.267 3.455a3.066 3.066 0 001.745-.723 3.066 3.066 0 013.976 0 3.066 3.066 0 001.745.723 3.066 3.066 0 012.812 2.812c.051.643.304 1.254.723 1.745a3.066 3.066 0 010 3.976 3.066 3.066 0 00-.723 1.745 3.066 3.066 0 01-2.812 2.812 3.066 3.066 0 00-1.745.723 3.066 3.066 0 01-3.976 0 3.066 3.066 0 00-1.745-.723 3.066 3.066 0 01-2.812-2.812 3.066 3.066 0 00-.723-1.745 3.066 3.066 0 010-3.976 3.066 3.066 0 00.723-1.745 3.066 3.066 0 012.812-2.812zm7.44 5.252a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
          </svg>
        </span>
        <span v-if="plugin.plugin_type === 'official'" class="badge badge-official">Official</span>
      </div>
    </div>

    <!-- Plugin Info -->
    <div class="plugin-info">
      <h3 class="plugin-name">{{ plugin.name }}</h3>
      <p class="plugin-description">{{ truncateDescription(plugin.description) }}</p>
    </div>

    <!-- Plugin Meta -->
    <div class="plugin-meta">
      <div class="meta-row">
        <span class="meta-item">
          <svg class="meta-icon" viewBox="0 0 20 20" fill="currentColor">
            <path d="M10 12a2 2 0 100-4 2 2 0 000 4z" />
            <path fill-rule="evenodd" d="M.458 10C1.732 5.943 5.522 3 10 3s8.268 2.943 9.542 7c-1.274 4.057-5.064 7-9.542 7S1.732 14.057.458 10zM14 10a4 4 0 11-8 0 4 4 0 018 0z" clip-rule="evenodd" />
          </svg>
          {{ formatNumber(plugin.total_downloads) }}
        </span>
        <span class="meta-item">
          <svg class="meta-icon"viewBox="0 0 20 20" fill="currentColor">
            <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
          </svg>
          {{ plugin.average_rating.toFixed(1) }}
        </span>
      </div>
      <div class="meta-row">
        <span class="author">by {{ plugin.author_name }}</span>
      </div>
    </div>

    <!-- Category & Phase Type Pills -->
    <div class="plugin-tags">
      <span class="tag tag-phase">{{ formatPhaseType(plugin.phase_type) }}</span>
      <span v-if="plugin.category" class="tag tag-category">{{ plugin.category }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Plugin } from '@/types'

defineProps<{
  plugin: Plugin
}>()

defineEmits<{
  (e: 'click'): void
}>()

const truncateDescription = (desc: string, maxLength = 120): string => {
  if (desc.length <= maxLength) return desc
  return desc.substring(0, maxLength).trim() + '...'
}

const formatNumber = (num: number): string => {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num.toString()
}

const formatPhaseType = (type: string): string => {
  return type.charAt(0).toUpperCase() + type.slice(1)
}

const getPhaseIcon = (phaseType: string): string => {
  const icons: Record<string, string> = {
    discovery: '🔍',
    extraction: '📦',
    processing: '⚙️',
    custom: '🔧'
  }
  return icons[phaseType] || '📄'
}
</script>

<style scoped>
.plugin-card {
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 1.5rem;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  height: 100%;
}

.plugin-card:hover {
  border-color: #8b5cf6;
  box-shadow: 0 4px 12px rgba(139, 92, 246, 0.1);
  transform: translateY(-2px);
}

/* Header */
.plugin-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.plugin-icon {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.5rem;
}

.plugin-badges {
  display: flex;
  gap: 0.5rem;
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.25rem 0.5rem;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
}

.badge-verified {
  background: #dbeafe;
  color: #1e40af;
}

.badge-verified svg {
  width: 14px;
  height: 14px;
}

.badge-official {
  background: #fef3c7;
  color: #b45309;
}

/* Info */
.plugin-info {
  flex: 1;
}

.plugin-name {
  font-size: 1.125rem;
  font-weight: 600;
  color: #111827;
  margin: 0 0 0.5rem;
  line-height: 1.4;
}

.plugin-description {
  font-size: 0.875rem;
  color: #6b7280;
  line-height: 1.5;
  margin: 0;
}

/* Meta */
.plugin-meta {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.meta-row {
  display: flex;
  align-items: center;
  gap: 1rem;
  font-size: 0.875rem;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  color: #6b7280;
  font-weight: 500;
}

.meta-icon {
  width: 16px;
  height: 16px;
  color: #9ca3af;
}

.author {
  color: #6b7280;
  font-size: 0.875rem;
}

/* Tags */
.plugin-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  padding-top: 0.5rem;
  border-top: 1px solid #f3f4f6;
}

.tag {
  padding: 0.25rem 0.75rem;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 500;
}

.tag-phase {
  background: #ede9fe;
  color: #7c3aed;
}

.tag-category {
  background: #f3f4f6;
  color: #374151;
}
</style>
