<template>
  <PageLayout>
    <!-- Header -->
    <PageHeader 
      title="Proxy Management" 
      description="Manage and monitor proxy servers for rotation and error recovery"
      :show-help-icon="true"
    >
      <template #actions>
        <Button @click="showProxyEditor = true" variant="default" size="sm">
          <Plus class="w-4 h-4 mr-2" />
          Add Proxy
        </Button>
      </template>
    </PageHeader>

    <!-- Stats -->
    <StatsBar :stats="proxyStats" />

    <!-- Filters -->
    <FilterBar 
      search-placeholder="Search proxies..." 
      :search-value="searchQuery"
      @update:search-value="searchQuery = $event"
    >
      <template #filters>
        <Select v-model="proxyStatusFilter">
          <SelectTrigger class="w-[140px] h-9">
            <div class="flex items-center gap-2">
              <SlidersHorizontal class="w-4 h-4" />
              <SelectValue placeholder="Status" />
            </div>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Proxies</SelectItem>
            <SelectItem value="healthy">Healthy</SelectItem>
            <SelectItem value="unhealthy">Unhealthy</SelectItem>
          </SelectContent>
        </Select>
      </template>
    </FilterBar>

    <!-- Content -->
    <div class="flex-1 overflow-auto">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <div v-else-if="filteredProxies.length === 0" class="py-12 text-center">
        <p class="text-muted-foreground">No proxies found</p>
      </div>

      <div v-else class="grid gap-3 p-6">
        <div
          v-for="proxy in filteredProxies"
          :key="proxy.id"
          class="group bg-card border rounded-lg p-4 hover:border-primary/50 transition-colors"
        >
          <div class="flex items-start justify-between gap-4">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg flex items-center justify-center"
                :class="proxy.is_healthy ? 'bg-green-500/10' : 'bg-red-500/10'">
                <Wifi :class="proxy.is_healthy ? 'text-green-500' : 'text-red-500'" class="w-5 h-5" />
              </div>
              <div class="min-w-0">
                <div class="flex items-center gap-2 mb-1">
                  <h3 class="text-sm font-medium">{{ proxy.proxy_address }}:{{ proxy.port }}</h3>
                  <Badge 
                    variant="outline"
                    :class="proxy.is_healthy ? 'bg-green-500/10 text-green-600 border-green-500/20' : 'bg-red-500/10 text-red-600 border-red-500/20'"
                    class="text-xs"
                  >
                    {{ proxy.is_healthy ? 'Healthy' : 'Unhealthy' }}
                  </Badge>
                  <Badge v-if="proxy.proxy_type" variant="outline" class="text-xs">
                    {{ proxy.proxy_type }}
                  </Badge>
                </div>
                <div class="flex flex-wrap gap-3 text-xs text-muted-foreground">
                  <span v-if="proxy.country_code">{{ proxy.country_code }}</span>
                  <span>{{ proxy.success_count }} successes</span>
                  <span>{{ proxy.failure_count }} failures</span>
                </div>
              </div>
            </div>
            
            <div class="flex items-center gap-1">
              <Button 
                @click="handleToggleProxy(proxy)" 
                size="sm" 
                variant="ghost" 
                class="h-8 w-8 p-0"
                :disabled="togglingIds.has(proxy.id)"
              >
                <Loader2 v-if="togglingIds.has(proxy.id)" class="h-4 w-4 animate-spin" />
                <Power v-else :class="proxy.is_healthy ? 'text-green-500' : 'text-muted-foreground'" class="h-4 w-4" />
              </Button>
              <Button @click="handleDeleteProxy(proxy)" size="sm" variant="ghost" class="h-8 w-8 p-0 text-destructive hover:text-destructive">
                <Trash2 class="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Simple Proxy Add Dialog -->
    <div v-if="showProxyEditor" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-background border rounded-lg p-6 max-w-md w-full mx-4">
        <h3 class="text-lg font-semibold mb-4">Add Proxy</h3>
        <div class="space-y-4">
          <div>
            <label class="text-sm font-medium">Server Address</label>
            <input v-model="newProxy.proxy_address" type="text" placeholder="192.168.1.1" 
              class="w-full mt-1 px-3 py-2 border rounded-md bg-background" />
          </div>
          <div>
            <label class="text-sm font-medium">Port</label>
            <input v-model.number="newProxy.port" type="number" placeholder="8080" 
              class="w-full mt-1 px-3 py-2 border rounded-md bg-background" />
          </div>
          <div>
            <label class="text-sm font-medium">Username (optional)</label>
            <input v-model="newProxy.username" type="text" 
              class="w-full mt-1 px-3 py-2 border rounded-md bg-background" />
          </div>
          <div>
            <label class="text-sm font-medium">Password (optional)</label>
            <input v-model="newProxy.password" type="password" 
              class="w-full mt-1 px-3 py-2 border rounded-md bg-background" />
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-6">
          <Button variant="outline" @click="showProxyEditor = false">Cancel</Button>
          <Button @click="handleAddProxy" :disabled="!newProxy.proxy_address || !newProxy.port">Add Proxy</Button>
        </div>
      </div>
    </div>
  </PageLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getProxies, createProxy, deleteProxy, toggleProxy, type Proxy } from '@/api/recovery'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import PageLayout from '@/components/layout/PageLayout.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import StatsBar from '@/components/layout/StatsBar.vue'
import FilterBar from '@/components/layout/FilterBar.vue'
import { Plus, Trash2, Loader2, SlidersHorizontal, Wifi, Power } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const proxyStatusFilter = ref('all')
const searchQuery = ref('')
const showProxyEditor = ref(false)
const loading = ref(false)
const proxies = ref<Proxy[]>([])
const togglingIds = ref<Set<string>>(new Set())
const newProxy = ref({ proxy_address: '', port: 0, username: '', password: '', server: '' })

const proxyStats = computed(() => {
  const healthy = proxies.value.filter(p => p.is_healthy).length
  return [
    { label: 'Total Proxies', value: proxies.value.length },
    { label: 'Healthy', value: healthy, color: 'text-green-600 dark:text-green-400' },
    { label: 'Unhealthy', value: proxies.value.length - healthy, color: 'text-red-600 dark:text-red-400' }
  ]
})

const filteredProxies = computed(() => {
  let result = proxies.value
  if (proxyStatusFilter.value === 'healthy') {
    result = result.filter(p => p.is_healthy)
  } else if (proxyStatusFilter.value === 'unhealthy') {
    result = result.filter(p => !p.is_healthy)
  }
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(p => p.proxy_address.toLowerCase().includes(query) || p.country_code?.toLowerCase().includes(query))
  }
  return result
})

async function fetchProxies() {
  loading.value = true
  try {
    const res = await getProxies()
    proxies.value = res.proxies || []
  } catch (error) {
    console.error('Failed to fetch proxies:', error)
    toast.error('Failed to load proxies')
  } finally {
    loading.value = false
  }
}

async function handleAddProxy() {
  try {
    newProxy.value.server = newProxy.value.proxy_address
    await createProxy(newProxy.value)
    toast.success('Proxy added')
    showProxyEditor.value = false
    newProxy.value = { proxy_address: '', port: 0, username: '', password: '', server: '' }
    await fetchProxies()
  } catch (error) {
    toast.error('Failed to add proxy')
  }
}

async function handleDeleteProxy(proxy: Proxy) {
  if (confirm(`Delete proxy ${proxy.proxy_address}:${proxy.port}?`)) {
    try {
      await deleteProxy(proxy.id)
      toast.success('Proxy deleted')
      await fetchProxies()
    } catch (error) {
      toast.error('Failed to delete proxy')
    }
  }
}

async function handleToggleProxy(proxy: Proxy) {
  if (togglingIds.value.has(proxy.id)) return
  togglingIds.value.add(proxy.id)
  try {
    await toggleProxy(proxy.id, !proxy.is_healthy)
    await fetchProxies()
  } catch (error) {
    toast.error('Failed to toggle proxy')
  } finally {
    togglingIds.value.delete(proxy.id)
  }
}

onMounted(fetchProxies)
</script>
