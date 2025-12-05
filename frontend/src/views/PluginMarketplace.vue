<template>
  <div class="min-h-screen bg-[#0F1115] text-gray-200 font-sans">
    <ScrollArea class="h-screen">
      <div class="container mx-auto max-w-[1600px] px-6 py-8">
        
        <!-- LANDING MODE -->
        <div v-if="viewMode === 'landing'" class="space-y-16 py-16">
          <!-- Hero Section -->
          <div class="text-center space-y-8 max-w-4xl mx-auto">
            <h1 class="text-5xl md:text-7xl font-bold tracking-tight text-white">
              Discover Plugins
            </h1>
            <p class="text-xl text-gray-400 max-w-2xl mx-auto leading-relaxed">
              Hundreds of pre-built plugins to automate your work. <br>
              Scrape websites, extract data, and automate workflows.
            </p>
            
            <!-- Hero Search -->
            <div class="relative max-w-2xl mx-auto group">
              <Search class="absolute left-5 top-1/2 -translate-y-1/2 h-6 w-6 text-gray-500 group-focus-within:text-blue-500 transition-colors" />
              <Input
                v-model="searchQuery"
                type="text"
                placeholder="Search for scrapers, automation tools..."
                class="h-16 pl-14 bg-[#1a1d21] border-gray-800 hover:border-gray-700 focus-visible:ring-blue-500/20 focus-visible:border-blue-500 text-lg rounded-2xl shadow-2xl shadow-black/50 placeholder:text-gray-600"
                @input="switchToSearch"
              />
            </div>

            <!-- Categories -->
            <div class="flex flex-wrap justify-center gap-3 pt-4">
              <Button
                v-for="category in categories"
                :key="category.id"
                variant="outline"
                size="sm"
                class="rounded-full border-gray-800 bg-[#1a1d21] text-gray-400 hover:bg-[#222529] hover:text-white hover:border-gray-700 transition-all px-4 h-9"
                @click="selectCategory(category.id)"
              >
                {{ category.name }}
              </Button>
            </div>
          </div>

          <!-- Featured Section -->
          <div class="space-y-8">
            <div class="flex items-center justify-between px-1">
              <h2 class="text-2xl font-bold text-white tracking-tight">Featured Plugins</h2>
              <Button variant="ghost" class="text-gray-400 hover:text-white gap-2 group" @click="switchToSearch">
                View all
                <ArrowRight class="h-4 w-4 group-hover:translate-x-1 transition-transform" />
              </Button>
            </div>
            
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
              <PluginCard
                v-for="plugin in featuredPlugins"
                :key="plugin.id"
                :plugin="plugin"
                @click="openPluginDetail(plugin)"
              />
            </div>
          </div>
        </div>

        <!-- SEARCH / VIEW ALL MODE -->
        <div v-else class="space-y-8">
          <!-- Top Bar -->
          <div class="flex items-center justify-between gap-6">
            <div class="flex items-center gap-4 flex-1">
              <Button variant="ghost" size="icon" class="-ml-3 text-gray-400 hover:text-white hover:bg-white/5" @click="viewMode = 'landing'">
                <ArrowLeft class="h-5 w-5" />
              </Button>
              <div class="relative flex-1 max-w-lg">
                <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-500" />
                <Input
                  v-model="searchQuery"
                  type="text"
                  placeholder="Search for Plugins"
                  class="h-10 pl-10 bg-[#1a1d21] border-gray-800 hover:border-gray-700 focus-visible:ring-blue-500/20 focus-visible:border-blue-500 transition-all rounded-lg text-sm placeholder:text-gray-600"
                  @input="debouncedSearch"
                  auto-focus
                />
              </div>
            </div>
            <div class="flex items-center gap-3">
              <Button variant="outline" size="sm" class="border-gray-800 bg-transparent text-gray-400 hover:text-white hover:bg-white/5 hover:border-gray-700">
                <Filter class="h-4 w-4 mr-2" />
                Filter
              </Button>
              <Button 
                size="sm" 
                class="bg-blue-600 hover:bg-blue-700 text-white shadow-lg shadow-blue-600/20" 
                @click="showCreateDialog = true"
              >
                <Plus class="h-4 w-4 mr-2" />
                Develop new
              </Button>
            </div>
          </div>

          <!-- Filters Row -->
          <div class="flex items-center justify-between gap-4 pb-4 border-b border-gray-800/50">
            <div class="flex items-center gap-2 overflow-x-auto no-scrollbar">
              <!-- Categories -->
              <Select :model-value="selectedCategory || 'all'" @update:model-value="(v) => selectCategory(v === 'all' ? null : v)">
                <SelectTrigger class="h-8 min-w-[130px] bg-transparent border-transparent hover:bg-white/5 text-gray-400 hover:text-white transition-colors rounded-md text-sm font-medium">
                  <SelectValue placeholder="All categories" />
                </SelectTrigger>
                <SelectContent class="bg-[#1a1d21] border-gray-800 text-gray-300">
                  <SelectItem value="all" class="focus:bg-white/5 focus:text-white">All categories</SelectItem>
                  <SelectItem v-for="category in categories" :key="category.id" :value="category.id" class="focus:bg-white/5 focus:text-white">
                    {{ category.name }}
                  </SelectItem>
                </SelectContent>
              </Select>

              <div class="h-4 w-px bg-gray-800 mx-2"></div>

              <!-- Phase Type -->
              <Select :model-value="filters.phase_type" @update:model-value="(v) => updateFilter('phase_type', v)">
                <SelectTrigger class="h-8 min-w-[130px] bg-transparent border-transparent hover:bg-white/5 text-gray-400 hover:text-white transition-colors rounded-md text-sm font-medium">
                  <SelectValue>
                    {{ formatFilterLabel(filters.phase_type, 'All pricing models') }}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent class="bg-[#1a1d21] border-gray-800 text-gray-300">
                  <SelectItem value="all" class="focus:bg-white/5 focus:text-white">All pricing models</SelectItem>
                  <SelectItem value="discovery" class="focus:bg-white/5 focus:text-white">Discovery</SelectItem>
                  <SelectItem value="extraction" class="focus:bg-white/5 focus:text-white">Extraction</SelectItem>
                  <SelectItem value="processing" class="focus:bg-white/5 focus:text-white">Processing</SelectItem>
                </SelectContent>
              </Select>

              <!-- Plugin Type -->
              <Select :model-value="filters.plugin_type" @update:model-value="(v) => updateFilter('plugin_type', v)">
                <SelectTrigger class="h-8 min-w-[130px] bg-transparent border-transparent hover:bg-white/5 text-gray-400 hover:text-white transition-colors rounded-md text-sm font-medium">
                  <SelectValue>
                    {{ formatFilterLabel(filters.plugin_type, 'All developers') }}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent class="bg-[#1a1d21] border-gray-800 text-gray-300">
                  <SelectItem value="all" class="focus:bg-white/5 focus:text-white">All developers</SelectItem>
                  <SelectItem value="official" class="focus:bg-white/5 focus:text-white">Official</SelectItem>
                  <SelectItem value="community" class="focus:bg-white/5 focus:text-white">Community</SelectItem>
                </SelectContent>
              </Select>

              <!-- Sort -->
              <Select :model-value="filters.sort_by" @update:model-value="(v) => updateFilter('sort_by', v)">
                <SelectTrigger class="h-8 min-w-[130px] bg-transparent border-transparent hover:bg-white/5 text-gray-400 hover:text-white transition-colors rounded-md text-sm font-medium">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent class="bg-[#1a1d21] border-gray-800 text-gray-300">
                  <SelectItem value="popular" class="focus:bg-white/5 focus:text-white">Most relevant</SelectItem>
                  <SelectItem value="rating" class="focus:bg-white/5 focus:text-white">Highest rated</SelectItem>
                  <SelectItem value="recent" class="focus:bg-white/5 focus:text-white">Newest</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="text-sm font-medium text-gray-500">
              {{ plugins.length }} Plugins
            </div>
          </div>

          <!-- Content Section -->
          <div>
            <!-- Loading State -->
            <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
              <div v-for="i in 8" :key="i" class="h-[220px] rounded-xl bg-[#1a1d21] animate-pulse border border-gray-800" />
            </div>

            <!-- Empty State -->
            <div v-else-if="!plugins || plugins.length === 0" class="text-center py-24">
              <div class="inline-flex items-center justify-center w-20 h-20 rounded-full bg-[#1a1d21] mb-6 border border-gray-800">
                <Search class="h-10 w-10 text-gray-600" />
              </div>
              <h3 class="text-xl font-semibold mb-3 text-white">No plugins found</h3>
              <p class="text-gray-500 max-w-sm mx-auto">We couldn't find any plugins matching your search. Try adjusting your filters or search terms.</p>
              <Button variant="link" class="mt-6 text-blue-500 hover:text-blue-400" @click="clearFilters">
                Clear all filters
              </Button>
            </div>

            <!-- Plugin Grid -->
            <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
              <PluginCard
                v-for="plugin in plugins"
                :key="plugin.id"
                :plugin="plugin"
                @click="openPluginDetail(plugin)"
              />
            </div>
          </div>
        </div>
      </div>
    </ScrollArea>

    <!-- Create Plugin Dialog -->
    <CreatePluginDialog v-model:open="showCreateDialog" @created="handlePluginCreated" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { Search, Plus, ArrowRight, ArrowLeft, Filter } from 'lucide-vue-next'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { 
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import PluginCard from '@/components/plugins/PluginCard.vue'
import CreatePluginDialog from '@/components/plugins/CreatePluginDialog.vue'
import pluginAPI from '@/lib/plugin-api'
import type { Plugin, PluginCategory, PluginFilters, PhaseType, PluginType } from '@/types'

// UI State interface allowing 'all' values
interface FilterState {
  sort_by: 'popular' | 'recent' | 'rating' | 'name'
  limit: number
  phase_type: PhaseType | 'all'
  plugin_type: PluginType | 'all'
  verified: boolean
}

const router = useRouter()
const viewMode = ref<'landing' | 'search'>('landing')
const showCreateDialog = ref(false)
const plugins = ref<Plugin[]>([])
const categories = ref<PluginCategory[]>([])
const loading = ref(false)
const searchQuery = ref('')
const selectedCategory = ref<string | null>(null)

const filters = ref<FilterState>({
  sort_by: 'popular',
  limit: 50,
  phase_type: 'all',
  plugin_type: 'all',
  verified: false
})

const featuredPlugins = computed(() => (plugins.value || []).slice(0, 8))

const formatFilterLabel = (value: string, defaultLabel: string) => {
  if (value === 'all') return defaultLabel
  return value.charAt(0).toUpperCase() + value.slice(1)
}

// Update filter helper
const updateFilter = <K extends keyof FilterState>(key: K, value: any) => {
  if (value === null) return
  filters.value[key] = value
  if (viewMode.value === 'landing') viewMode.value = 'search'
  applyFilters()
}

// Switch to search mode
const switchToSearch = () => {
  if (viewMode.value !== 'search') {
    viewMode.value = 'search'
    applyFilters()
  }
}

// Debounced search
let searchTimeout: ReturnType<typeof setTimeout>
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
    // Map UI state to API filters
    const queryFilters: PluginFilters = {
      sort_by: filters.value.sort_by,
      limit: filters.value.limit,
      verified: filters.value.verified
    }
    
    if (filters.value.phase_type !== 'all') {
      queryFilters.phase_type = filters.value.phase_type
    }
    
    if (filters.value.plugin_type !== 'all') {
      queryFilters.plugin_type = filters.value.plugin_type
    }
    
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

const loadCategories = async () => {
  try {
    categories.value = await pluginAPI.getCategories()
  } catch (error) {
    console.error('Failed to load categories:', error)
  }
}

const applyFilters = () => {
  loadPlugins()
}

const selectCategory = (categoryId: any) => {
  selectedCategory.value = categoryId === 'all' ? null : categoryId
  if (viewMode.value === 'landing') viewMode.value = 'search'
  applyFilters()
}

const clearFilters = () => {
  searchQuery.value = ''
  selectedCategory.value = null
  filters.value = {
    sort_by: 'popular',
    limit: 50,
    phase_type: 'all',
    plugin_type: 'all',
    verified: false
  }
  applyFilters()
}

// Open plugin detail
const openPluginDetail = (plugin: Plugin) => {
  router.push({ name: 'plugin-detail', params: { id: plugin.id } })
}

// Handle plugin created
const handlePluginCreated = (pluginId: string) => {
  // Reload plugins to show the newly created one
  loadPlugins()
}

onMounted(() => {
  loadCategories()
  loadPlugins()
})
</script>
