<template>
  <div class="plugin-marketplace">
    <!-- Header -->
    <div class="marketplace-header">
      <div class="header-content">
        <h1 class="marketplace-title">Plugin Marketplace</h1>
        <p class="marketplace-subtitle">
          Extend Crawlify with powerful plugins for discovery and extraction
        </p>
      </div>

      <!-- Search Bar -->
      <div class="search-container">
        <div class="search-box">
          <svg class="search-icon" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clip-rule="evenodd" />
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search plugins..."
            class="search-input"
            @input="debouncedSearch"
          />
        </div>
      </div>
    </div>

    <div class="marketplace-container">
      <!-- Sidebar with Categories -->
      <aside class="marketplace-sidebar">
        <div class="sidebar-section">
          <h3 class="sidebar-title">Categories</h3>
          <div class="category-list">
            <button
              v-for="category in categories"
              :key="category.id"
              :class="['category-btn', { active: selectedCategory === category.id }]"
              @click="selectCategory(category.id)"
            >
              <span class="category-icon">{{ getCategoryIcon(category.id) }}</span>
              <span class="category-name">{{ category.name }}</span>
            </button>
          </div>
        </div>

        <div class="sidebar-section">
          <h3 class="sidebar-title">Filters</h3>

          <!-- Phase Type Filter -->
          <div class="filter-group">
            <label class="filter-label">Phase Type</label>
            <select v-model="filters.phase_type" class="filter-select" @change="applyFilters">
              <option value="">All Phases</option>
              <option value="discovery">Discovery</option>
              <option value="extraction">Extraction</option>
              <option value="processing">Processing</option>
            </select>
          </div>

          <!-- Plugin Type Filter -->
          <div class="filter-group">
            <label class="filter-label">Type</label>
            <select v-model="filters.plugin_type" class="filter-select" @change="applyFilters">
              <option value="">All Types</option>
              <option value="official">Official</option>
              <option value="community">Community</option>
            </select>
          </div>

          <!-- Verified Filter -->
          <div class="filter-group">
            <label class="filter-checkbox">
              <input
                v-model="filters.verified"
                type="checkbox"
                @change="applyFilters"
              />
              <span>Verified Only</span>
            </label>
          </div>
        </div>
      </aside>

      <!-- Main Content -->
      <main class="marketplace-main">
        <!--  Sort & View Controls -->
        <div class="controls-bar">
          <div class="results-info">
            {{ plugins.length }} {{ plugins.length === 1 ? 'plugin' : 'plugins' }}
          </div>

          <div class="sort-controls">
            <label class="sort-label">Sort by:</label>
            <select v-model="filters.sort_by" class="sort-select" @change="applyFilters">
              <option value="popular">Most Popular</option>
              <option value="rating">Highest Rated</option>
              <option value="recent">Recently Added</option>
              <option value="name">Name</option>
            </select>
          </div>
        </div>

        <!-- Loading State -->
        <div v-if="loading" class="loading-state">
          <div class="spinner"></div>
          <p>Loading plugins...</p>
        </div>

        <!-- Empty State -->
        <div v-else-if="plugins.length === 0 && !loading" class="empty-state">
          <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <circle cx="11" cy="11" r="8"></circle>
            <path d="m21 21-4.35-4.35"></path>
          </svg>
          <h3>No plugins found</h3>
          <p>Try adjusting your filters or search query</p>
        </div>

        <!-- Plugin Grid -->
        <div v-else class="plugin-grid">
          <PluginCard
            v-for="plugin in plugins"
            :key="plugin.id"
            :plugin="plugin"
            @click="openPluginDetail(plugin)"
          />
        </div>
      </main>
    </div>

    <!-- Plugin Detail Modal -->
    <PluginDetailModal
      v-if="selectedPlugin"
      :plugin="selectedPlugin"
      @close="selectedPlugin = null"
      @installed="onPluginInstalled"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import type { Plugin, PluginCategory, PluginFilters } from '@/types'
import pluginAPI from '@/lib/plugin-api'
import PluginCard from '@/components/plugins/PluginCard.vue'
import PluginDetailModal from '@/components/plugins/PluginDetailModal.vue'

const plugins = ref<Plugin[]>([])
const categories = ref<PluginCategory[]>([])
const loading = ref(false)
const searchQuery = ref('')
const selectedCategory = ref<string | null>(null)
const selectedPlugin = ref<Plugin | null>(null)

const filters = ref<PluginFilters>({
  sort_by: 'popular',
  limit: 50
})

// Debounced search
let searchTimeout: NodeJS.Timeout
const debouncedSearch = () => {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    applyFilters()
  }, 300)
}

// Load plugins
const loadPlugins = async () => {
  loading.value = true
  try {
    const queryFilters = { ...filters.value }
    
    if (searchQuery.value.trim()) {
      queryFilters.q = searchQuery.value.trim()
    }
    
    if (selectedCategory.value) {
      queryFilters.category = selectedCategory.value
    }

    plugins.value = await pluginAPI.listPlugins(queryFilters)
  } catch (error) {
    console.error('Failed to load plugins:', error)
  } finally {
    loading.value = false
  }
}

// Load categories
const loadCategories = async () => {
  try {
    categories.value = await pluginAPI.getCategories()
  } catch (error) {
    console.error('Failed to load categories:', error)
  }
}

// Select category
const selectCategory = (categoryId: string) => {
  if (selectedCategory.value === categoryId) {
    selectedCategory.value = null
  } else {
    selectedCategory.value = categoryId
  }
  applyFilters()
}

// Apply filters
const applyFilters = () => {
  loadPlugins()
}

// Open plugin detail
const openPluginDetail = (plugin: Plugin) => {
  selectedPlugin.value = plugin
}

// Handle plugin installed
const onPluginInstalled = () => {
  // Reload plugins to update install status
  loadPlugins()
}

// Get category icon
const getCategoryIcon = (categoryId: string): string => {
  const icons: Record<string, string> = {
    'ecommerce': '🛒',
    'social-media': '📱',
    'news': '📰',
    'data-extraction': '📊',
    'authentication': '🔐',
    'pagination': '📄',
    'javascript-heavy': '⚡',
    'api-integration': '🔌',
    'general': '⚙️'
  }
  return icons[categoryId] || '📦'
}

onMounted(() => {
  loadCategories()
  loadPlugins()
})
</script>

<style scoped>
.plugin-marketplace {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 2rem;
}

.marketplace-header {
  max-width: 1400px;
  margin: 0 auto 2rem;
}

.header-content {
  text-align: center;
  margin-bottom: 2rem;
}

.marketplace-title {
  font-size: 2.5rem;
  font-weight: 700;
  color: white;
  margin: 0 0 0.5rem;
}

.marketplace-subtitle {
  font-size: 1.125rem;
  color: rgba(255, 255, 255, 0.9);
  margin: 0;
}

.search-container {
  max-width: 600px;
  margin: 0 auto;
}

.search-box {
  position: relative;
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
}

.search-icon {
  position: absolute;
  left: 1rem;
  top: 50%;
  transform: translateY(-50%);
  width: 20px;
  height: 20px;
  color: #9ca3af;
}

.search-input {
  width: 100%;
  padding: 0.875rem 1rem 0.875rem 3rem;
  border: none;
  border-radius: 12px;
  font-size: 1rem;
  outline: none;
}

.marketplace-container {
  max-width: 1400px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 2rem;
  align-items: start;
}

/* Sidebar */
.marketplace-sidebar {
  background: white;
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  position: sticky;
  top: 2rem;
}

.sidebar-section {
  margin-bottom: 2rem;
}

.sidebar-section:last-child {
  margin-bottom: 0;
}

.sidebar-title {
  font-size: 0.875rem;
  font-weight: 600;
  text-transform: uppercase;
  color: #6b7280;
  margin: 0 0 1rem;
  letter-spacing: 0.05em;
}

.category-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.category-btn {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  background: transparent;
  border: 2px solid transparent;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  text-align: left;
  font-size: 0.9375rem;
}

.category-btn:hover {
  background: #f9fafb;
  border-color: #e5e7eb;
}

.category-btn.active {
  background: #ede9fe;
  border-color: #8b5cf6;
  color: #7c3aed;
}

.category-icon {
  font-size: 1.25rem;
}

.category-name {
  flex: 1;
  font-weight: 500;
}

.filter-group {
  margin-bottom: 1rem;
}

.filter-label {
  display: block;
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
  margin-bottom: 0.5rem;
}

.filter-select {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 0.875rem;
  outline: none;
  transition: border 0.2s;
}

.filter-select:focus {
  border-color: #8b5cf6;
}

.filter-checkbox {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  font-size: 0.875rem;
  color: #374151;
}

.filter-checkbox input {
  width: 16px;
  height: 16px;
  cursor: pointer;
}

/* Main Content */
.marketplace-main {
  background: white;
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  min-height: 500px;
}

.controls-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid #e5e7eb;
}

.results-info {
  font-size: 0.875rem;
  color: #6b7280;
  font-weight: 500;
}

.sort-controls {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.sort-label {
  font-size: 0.875rem;
  color: #6b7280;
}

.sort-select {
  padding: 0.5rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 0.875rem;
  outline: none;
  cursor: pointer;
  transition: border 0.2s;
}

.sort-select:focus {
  border-color: #8b5cf6;
}

/* Loading & Empty States */
.loading-state,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  color: #6b7280;
}

.spinner {
  width: 48px;
  height: 48px;
  border: 4px solid #e5e7eb;
  border-top-color: #8b5cf6;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin-bottom: 1rem;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.empty-icon {
  width: 64px;
  height: 64px;
  color: #d1d5db;
  margin-bottom: 1rem;
}

.empty-state h3 {
  font-size: 1.125rem;
  font-weight: 600;
  color: #374151;
  margin: 0 0 0.5rem;
}

.empty-state p {
  font-size: 0.875rem;
  color: #6b7280;
  margin: 0;
}

/* Plugin Grid */
.plugin-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 1.5rem;
}

@media (max-width: 1024px) {
  .marketplace-container {
    grid-template-columns: 1fr;
  }
  
  .marketplace-sidebar {
    position: static;
  }
}
</style>
