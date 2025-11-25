<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-container">
      <!-- Modal Header -->
      <div class="modal-header">
        <div class="header-left">
          <div class="plugin-icon-large">
            {{ getPhaseIcon(plugin.phase_type) }}
          </div>
          <div class="header-title">
            <h2 class="plugin-title">{{ plugin.name }}</h2>
            <div class="plugin-badges">
              <span v-if="plugin.is_verified" class="badge badge-verified">
                <svg viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M6.267 3.455a3.066 3.066 0 001.745-.723 3.066 3.066 0 013.976 0 3.066 3.066 0 001.745.723 3.066 3.066 0 012.812 2.812c.051.643.304 1.254.723 1.745a3.066 3.066 0 010 3.976 3.066 3.066 0 00-.723 1.745 3.066 3.066 0 01-2.812 2.812 3.066 3.066 0 00-1.745.723 3.066 3.066 0 01-3.976 0 3.066 3.066 0 00-1.745-.723 3.066 3.066 0 01-2.812-2.812 3.066 3.066 0 00-.723-1.745 3.066 3.066 0 010-3.976 3.066 3.066 0 00.723-1.745  3.066 3.066 0 012.812-2.812zm7.44 5.252a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
                </svg>
                Verified
              </span>
              <span v-if="plugin.plugin_type === 'official'" class="badge badge-official">Official</span>
            </div>
          </div>
        </div>
        <button class="close-btn" @click="$emit('close')">
          <svg viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
          </svg>
        </button>
      </div>

      <!-- Modal Body -->
      <div class="modal-body">
        <!-- Info Section -->
        <div class="info-section">
          <p class="plugin-description">{{ plugin.description }}</p>

          <div class="plugin-stats">
            <div class="stat-item">
              <div class="stat-value">{{ formatNumber(plugin.total_downloads) }}</div>
              <div class="stat-label">Downloads</div>
            </div>
            <div class="stat-item">
              <div class="stat-value">{{ formatNumber(plugin.total_installs) }}</div>
              <div class="stat-label">Installs</div>
            </div>
            <div class="stat-item">
              <div class="stat-value">{{ plugin.average_rating.toFixed(1) }} ⭐</div>
              <div class="stat-label">Rating</div>
            </div>
          </div>

          <div class="plugin-details">
            <div class="detail-row">
              <span class="detail-label">Author:</span>
              <span class="detail-value">{{ plugin.author_name }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Phase Type:</span>
              <span class="detail-value">{{ formatPhaseType(plugin.phase_type) }}</span>
            </div>
            <div class="detail-row" v-if="plugin.category">
              <span class="detail-label">Category:</span>
              <span class="detail-value">{{ plugin.category }}</span>
            </div>
            <div class="detail-row" v-if="plugin.repository_url">
              <span class="detail-label">Repository:</span>
              <a :href="plugin.repository_url" target="_blank" class="detail-link">
                View on GitHub →
              </a>
            </div>
            <div class="detail-row" v-if="plugin.documentation_url">
              <span class="detail-label">Documentation:</span>
              <a :href="plugin.documentation_url" target="_blank" class="detail-link">
                Read Docs →
              </a>
            </div>
          </div>

          <!-- Tags -->
          <div v-if="plugin.tags && plugin.tags.length > 0" class="plugin-tags">
            <span v-for="tag in plugin.tags" :key="tag" class="tag">{{ tag }}</span>
          </div>
        </div>

        <!-- Version Section -->
        <div class="version-section">
          <h3 class="section-title">Latest Version</h3>
          <div v-if="loadingVersion" class="loading">Loading version...</div>
          <div v-else-if="latestVersion" class="version-info">
            <div class="version-header">
              <span class="version-number">v{{ latestVersion.version }}</span>
              <span v-if="latestVersion.is_stable" class="version-badge">Stable</span>
            </div>
            <p v-if="latestVersion.changelog" class="version-changelog">
              {{ latestVersion.changelog }}
            </p>
            <div class="version-meta">
              Published {{ formatDate(latestVersion.published_at) }}
            </div>
          </div>
        </div>

        <!-- Actions -->
        <div class="action-section">
          <button
            v-if="!isInstalled"
            :disabled="installing"
            class="btn btn-primary"
            @click="handleInstall"
          >
            <svg v-if="!installing" class="btn-icon" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm3.293-7.707a1 1 0 011.414 0L9 10.586V3a1 1 0 112 0v7.586l1.293-1.293a1 1 0 111.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z" clip-rule="evenodd" />
            </svg>
            <div v-else class="btn-spinner"></div>
            {{ installing ? 'Installing...' : 'Install Plugin' }}
          </button>
          <button
            v-else
            :disabled="uninstalling"
            class="btn btn-secondary"
            @click="handleUninstall"
          >
            <svg v-if="!uninstalling" class="btn-icon" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
            </svg>
            <div v-else class="btn-spinner"></div>
            {{ uninstalling ? 'Uninstalling...' : 'Uninstall' }}
          </button>

          <button class="btn btn-outline" @click="$emit('close')">
            Close
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { Plugin, PluginVersion } from '@/types'
import pluginAPI from '@/lib/plugin-api'

const props = defineProps<{
  plugin: Plugin
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'installed'): void
}>()

const latestVersion = ref<PluginVersion | null>(null)
const loadingVersion = ref(false)
const isInstalled = ref(false)
const installing = ref(false)
const uninstalling = ref(false)

const loadLatestVersion = async () => {
  loadingVersion.value = true
  try {
    const versions = await pluginAPI.listVersions(props.plugin.id)
    if (versions.length > 0) {
      latestVersion.value = versions[0]
    }
  } catch (error) {
    console.error('Failed to load version:', error)
  } finally {
    loadingVersion.value = false
  }
}

const checkInstallStatus = async () => {
  try {
    isInstalled.value = await pluginAPI.isPluginInstalled(props.plugin.id)
  } catch (error) {
    console.error('Failed to check install status:', error)
  }
}

const handleInstall = async () => {
  installing.value = true
  try {
    await pluginAPI.installPlugin(props.plugin.id)
    isInstalled.value = true
    emit('installed')
  } catch (error) {
    console.error('Failed to install plugin:', error)
    alert('Failed to install plugin. Please try again.')
  } finally {
    installing.value = false
  }
}

const handleUninstall = async () => {
  if (!confirm(`Are you sure you want to uninstall ${props.plugin.name}?`)) {
    return
  }

  uninstalling.value = true
  try {
    await pluginAPI.uninstallPlugin(props.plugin.id)
    isInstalled.value = false
    emit('installed')
  } catch (error) {
    console.error('Failed to uninstall plugin:', error)
    alert('Failed to uninstall plugin. Please try again.')
  } finally {
    uninstalling.value = false
  }
}

const formatNumber = (num: number): string => {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num.toString()
}

const formatPhaseType = (type: string): string => {
  return type.charAt(0).toUpperCase() + type.slice(1)
}

const formatDate = (dateString: string): string => {
  const date = new Date(dateString)
  return date.toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })
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

onMounted(() => {
  loadLatestVersion()
  checkInstallStatus()
})
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 2rem;
  backdrop-filter: blur(4px);
}

.modal-container {
  background: white;
  border-radius: 16px;
  max-width: 700px;
  width: 100%;
  max-height: 90vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 2rem;
  border-bottom: 1px solid #e5e7eb;
}

.header-left {
  display: flex;
  gap: 1rem;
  flex: 1;
}

.plugin-icon-large {
  width: 64px;
  height: 64px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 2rem;
  flex-shrink: 0;
}

.header-title {
  flex: 1;
}

.plugin-title {
  font-size: 1.5rem;
  font-weight: 700;
  color: #111827;
  margin: 0 0 0.5rem;
}

.plugin-badges {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.75rem;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 600;
}

.badge-verified {
  background: #dbeafe;
  color: #1e40af;
}

.badge-verified svg {
  width: 16px;
  height: 16px;
}

.badge-official {
  background: #fef3c7;
  color: #b45309;
}

.close-btn {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  border: none;
  background: transparent;
  cursor: pointer;
  color: #6b7280;
  transition: all 0.2s;
  flex-shrink: 0;
}

.close-btn:hover {
  background: #f3f4f6;
  color: #111827;
}

.close-btn svg {
  width: 20px;
  height: 20px;
}

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 2rem;
}

/* Info Section */
.info-section {
  margin-bottom: 2rem;
}

.plugin-description {
  font-size: 1rem;
  line-height: 1.6;
  color: #374151;
  margin: 0 0 1.5rem;
}

.plugin-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.stat-item {
  text-align: center;
  padding: 1rem;
  background: #f9fafb;
  border-radius: 8px;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: #111827;
}

.stat-label {
  font-size: 0.875rem;
  color: #6b7280;
  margin-top: 0.25rem;
}

.plugin-details {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
}

.detail-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.detail-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: #6b7280;
  min-width: 120px;
}

.detail-value {
  font-size: 0.875rem;
  color: #111827;
}

.detail-link {
  font-size: 0.875rem;
  color: #8b5cf6;
  text-decoration: none;
  font-weight: 500;
  transition: color 0.2s;
}

.detail-link:hover {
  color: #7c3aed;
  text-decoration: underline;
}

.plugin-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.tag {
  padding: 0.375rem 0.75rem;
  background: #f3f4f6;
  border-radius: 6px;
  font-size: 0.875rem;
  color: #374151;
  font-weight: 500;
}

/* Version Section */
.version-section {
  margin-bottom: 2rem;
}

.section-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: #111827;
  margin: 0 0 1rem;
}

.loading {
  color: #6b7280;
  font-size: 0.875rem;
}

.version-info {
  padding: 1rem;
  background: #f9fafb;
  border-radius: 8px;
}

.version-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.version-number {
  font-size: 1rem;
  font-weight: 600;
  color: #111827;
}

.version-badge {
  padding: 0.25rem 0.5rem;
  background: #d1fae5;
  color: #065f46;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 600;
}

.version-changelog {
  font-size: 0.875rem;
  color: #374151;
  line-height: 1.5;
  margin: 0 0 0.5rem;
}

.version-meta {
  font-size: 0.75rem;
  color: #6b7280;
}

/* Actions */
.action-section {
  display: flex;
  gap: 1rem;
  padding-top: 1.5rem;
  border-top: 1px solid #e5e7eb;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.75rem 1.5rem;
  border-radius: 8px;
  font-size: 0.9375rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.btn-secondary {
  background: #fee2e2;
  color: #dc2626;
}

.btn-secondary:hover:not(:disabled) {
  background: #fecaca;
}

.btn-outline {
  background: transparent;
  border: 1px solid #d1d5db;
  color: #374151;
}

.btn-outline:hover {
  background: #f9fafb;
  border-color: #9ca3af;
}

.btn-icon {
  width: 18px;
  height: 18px;
}

.btn-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@media (max-width: 640px) {
  .modal-overlay {
    padding: 0;
  }

  .modal-container {
    max-height: 100vh;
    border-radius: 0;
  }

  .plugin-stats {
    grid-template-columns: 1fr;
  }

  .action-section {
    flex-direction: column;
  }
}
</style>
