<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useBrowserProfilesStore } from '@/stores/browserProfiles'
import type { BrowserProfile } from '@/api/browserProfiles'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import DataTable from '@/components/ui/data-table.vue'
import PageLayout from '@/components/layout/PageLayout.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import StatsBar from '@/components/layout/StatsBar.vue'
import TabBar from '@/components/layout/TabBar.vue'
import FilterBar from '@/components/layout/FilterBar.vue'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { Plus, Play, StopCircle, Pencil, Trash2, Loader2, SlidersHorizontal, Copy, Chrome, Globe } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const router = useRouter()
const profilesStore = useBrowserProfilesStore()

const showDeleteDialog = ref(false)
const selectedProfile = ref<any>(null)
const statusFilter = ref<string>('all')
const browserTypeFilter = ref<string>('all')
const searchQuery = ref('')
const activeTab = ref('all')
const launchingProfiles = ref<Set<string>>(new Set()) // Track profiles being launched/stopped (UI state only)

const tableColumns = [
  { key: 'name', label: 'Name', sortable: true, align: 'left' as const },
  { key: 'browser_type', label: 'Browser Type', align: 'left' as const },
  { key: 'status', label: 'Status', align: 'left' as const },
  { key: 'last_used', label: 'Last Used', align: 'left' as const },
  { key: 'actions', label: 'Actions', align: 'right' as const }
]

const tabs = [
  { id: 'all', label: 'All Profiles' },
  { id: 'recent', label: 'Recently Used' }
]

const stats = computed(() => {
  const runningCount = profilesStore.profiles.filter((p: BrowserProfile) => p.status === 'running').length
  return [
    { label: 'Total Profiles', value: profilesStore.profiles.length },
    { label: 'Active', value: profilesStore.activeProfiles.length, color: 'text-green-600 dark:text-green-400' },
    { label: 'Running', value: runningCount, color: 'text-blue-600 dark:text-blue-400' }
  ]
})

const filteredProfiles = computed(() => {
  let result = activeTab.value === 'recent' 
    ? profilesStore.recentlyUsedProfiles 
    : profilesStore.profiles
  
  if (statusFilter.value !== 'all') {
    result = result.filter((p: any) => p.status === statusFilter.value)
  }

  if (browserTypeFilter.value !== 'all') {
    result = result.filter((p: any) => p.browser_type === browserTypeFilter.value)
  }  
  
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter((p: any) => 
      p.name.toLowerCase().includes(query) || 
      p.description?.toLowerCase().includes(query)
    )
  }
  
  return result
})

const handleCreateProfile = () => {
  router.push('/browser-profiles/create')
}

const handleEditProfile = (profile: any) => {
  router.push(`/browser-profiles/${profile.id}/edit`)
}

const handleDeleteProfile = (profile: any) => {
  selectedProfile.value = profile
  showDeleteDialog.value = true
}

const confirmDelete = async () => {
  if (!selectedProfile.value) return
  
  try {
    await profilesStore.deleteProfile(selectedProfile.value.id)
    toast.success('Profile deleted successfully')
    showDeleteDialog.value = false
    selectedProfile.value = null
  } catch (error) {
    console.error('Failed to delete profile:', error)
    toast.error('Failed to delete profile')
  }
}

const handleDuplicateProfile = async (profile: any) => {
  try {
    await profilesStore.duplicateProfile(profile.id)
    toast.success('Profile duplicated successfully')
  } catch (error) {
    console.error('Failed to duplicate profile:', error)
    toast.error('Failed to duplicate profile')
  }
}

const toggleProfileLaunch = async (profile: any) => {
  const isRunning = profile.status === 'running'
  
  if (launchingProfiles.value.has(profile.id)) return // Prevent double-click
  
  launchingProfiles.value.add(profile.id)
  
  try {
    if (isRunning) {
      // Stop the profile
      await profilesStore.stopProfile(profile.id)
      toast.success('Profile stopped')
    } else {
      // Launch the profile
      await profilesStore.launchProfile(profile.id, false)
      toast.success('Profile launched')
    }
    // Refresh profiles to get updated status from database
    await profilesStore.fetchProfiles()
  } catch (error) {
    console.error('Failed to toggle profile:', error)
    toast.error(`Failed to ${isRunning ? 'stop' : 'launch'} profile`)
  } finally {
    launchingProfiles.value.delete(profile.id)
  }
}

const handleViewDetails = (profile: any) => {
  router.push(`/browser-profiles/${profile.id}`)
}

const getBrowserIcon = (browserType: string) => {
  switch(browserType) {
    case 'chromium': return Chrome
    case 'firefox': return Globe
    case 'webkit': return Globe
    default: return Chrome
  }
}

const getBrowserIconColor = (browserType: string) => {
  switch(browserType) {
    case 'chromium': return 'text-blue-600 dark:text-blue-400'
    case 'firefox': return 'text-orange-600 dark:text-orange-400'
    case 'webkit': return 'text-purple-600 dark:text-purple-400'
    default: return 'text-gray-600 dark:text-gray-400'
  }
}

const formatDate = (dateString?: string) => {
  if (!dateString) return 'Never'
  const date = new Date(dateString)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  
  if (days === 0) return 'Today'
  if (days === 1) return 'Yesterday'
  if (days < 7) return `${days} days ago`
  
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

onMounted(async () => {
  try {
    await profilesStore.fetchProfiles()
  } catch (error) {
    console.error('Failed to load browser profiles:', error)
  }
})
</script>

<template>
  <PageLayout>
    <!-- Header -->
    <PageHeader 
      title="Browser Profiles" 
      description="Manage browser profiles with custom fingerprints and configurations"
      :show-help-icon="true"
    >
      <template #actions>
        <Button @click="handleCreateProfile" variant="default" class="bg-primary hover:bg-primary/90">
          <Plus class="w-4 h-4 mr-2" />
          Create Profile
        </Button>
      </template>
    </PageHeader>

    <!-- Stats -->
    <StatsBar :stats="stats" />

    <!-- Tabs -->
    <TabBar :tabs="tabs" v-model="activeTab" />

    <!-- Filters -->
    <FilterBar 
      search-placeholder="Search by profile name" 
      :search-value="searchQuery"
      @update:search-value="searchQuery = $event"
    >
      <template #filters>
        <Select v-model="statusFilter">
          <SelectTrigger class="w-[140px] h-9">
            <div class="flex items-center gap-2">
              <SlidersHorizontal class="w-4 h-4" />
              <SelectValue placeholder="Status" />
            </div>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All status</SelectItem>
            <SelectItem value="active">Active</SelectItem>
            <SelectItem value="inactive">Inactive</SelectItem>
          </SelectContent>
        </Select>

        <Select v-model="browserTypeFilter">
          <SelectTrigger class="w-[160px] h-9">
            <SelectValue placeholder="Browser Type" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All browsers</SelectItem>
            <SelectItem value="chromium">Chromium</SelectItem>
            <SelectItem value="firefox">Firefox</SelectItem>
            <SelectItem value="webkit">WebKit</SelectItem>
          </SelectContent>
        </Select>
      </template>
    </FilterBar>

    <!-- Table -->
    <div class="flex-1 overflow-auto">
      <div v-if="profilesStore.loading" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <div v-else-if="filteredProfiles.length === 0" class="py-12 text-center px-6">
        <p class="text-muted-foreground">No browser profiles found</p>
      </div>

      <DataTable
        v-else
        :data="filteredProfiles"
        :columns="tableColumns"
        :on-row-click="handleViewDetails"
      >
        <template #row="{ row }">
          <td class="px-6 py-3">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center shrink-0">
                <component :is="getBrowserIcon(row.browser_type)" :class="['w-5 h-5', getBrowserIconColor(row.browser_type)]" />
              </div>
              <div class="min-w-0">
                <div class="font-medium text-sm truncate">{{ row.name }}</div>
                <div class="text-xs text-muted-foreground truncate">{{ row.id }}</div>
              </div>
            </div>
          </td>
          <td class="px-6 py-3">
            <Badge 
              variant="outline"
              :class="{
                'bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20': row.browser_type === 'chromium',
                'bg-orange-500/10 text-orange-600 dark:text-orange-400 border-orange-500/20': row.browser_type === 'firefox',
                'bg-purple-500/10 text-purple-600 dark:text-purple-400 border-purple-500/20': row.browser_type === 'webkit'
              }"
              class="text-xs font-medium capitalize"
            >
              {{ row.browser_type }}
            </Badge>
          </td>
          <td class="px-6 py-3" @click.stop>
            <Badge 
              variant="outline"
              :class="{
                'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20': row.status === 'active',
                'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20': row.status === 'inactive'
              }"
              class="text-xs font-medium"
            >
              <div class="w-1.5 h-1.5 rounded-full mr-1.5" :class="{
                'bg-green-500': row.status === 'active',
                'bg-gray-500': row.status === 'inactive'
              }"></div>
              {{ row.status }}
            </Badge>
          </td>
          <td class="px-6 py-3">
            <div class="text-sm text-muted-foreground">
              {{ formatDate(row.last_used_at) }}
            </div>
          </td>
          <td class="px-6 py-3 text-right" @click.stop>
            <div class="flex items-center justify-end gap-1">
              <Button 
                @click="toggleProfileLaunch(row)"
                size="sm"
                variant="ghost"
                :class="[
                  'h-8 w-8 p-0',
                  row.status === 'running' ? 'text-blue-600 dark:text-blue-400' : ''
                ]"
                :disabled="(row.status !== 'active' && row.status !== 'running') || launchingProfiles.has(row.id)"
              >
                <Loader2 v-if="launchingProfiles.has(row.id)" class="h-4 w-4 animate-spin" />
                <StopCircle v-else-if="row.status === 'running'" class="h-4 w-4" />
                <Play v-else class="h-4 w-4" />
              </Button>
              <Button 
                @click="handleDuplicateProfile(row)"
                size="sm"
                variant="ghost"
                class="h-8 w-8 p-0"
              >
                <Copy class="h-4 w-4" />
              </Button>
              <Button 
                @click="handleEditProfile(row)"
                size="sm"
                variant="ghost"
                class="h-8 w-8 p-0"
              >
                <Pencil class="h-4 w-4" />
              </Button>
              <Button 
                @click="handleDeleteProfile(row)"
                size="sm"
                variant="ghost"
                class="h-8 w-8 p-0 text-destructive hover:text-destructive"
              >
                <Trash2 class="h-4 w-4" />
              </Button>
            </div>
          </td>
        </template>
      </DataTable>
    </div>

    <!-- Delete Confirmation Dialog -->
    <div v-if="showDeleteDialog" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-background border rounded-lg p-6 max-w-md w-full mx-4">
        <h3 class="text-lg font-semibold mb-2">Delete Profile?</h3>
        <p class="text-sm text-muted-foreground mb-4">
          Are you sure you want to delete "{{ selectedProfile?.name }}"? This action cannot be undone.
        </p>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="showDeleteDialog = false">Cancel</Button>
          <Button variant="destructive" @click="confirmDelete">Delete</Button>
        </div>
      </div>
    </div>
  </PageLayout>
</template>
