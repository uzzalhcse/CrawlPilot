<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useBrowserProfilesStore } from '@/stores/browserProfiles'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import PageLayout from '@/components/layout/PageLayout.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import StatsBar from '@/components/layout/StatsBar.vue'
import TabBar from '@/components/layout/TabBar.vue'
import { Loader2, Play, Pencil, Copy, Trash2, Chrome, Globe, StopCircle } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const router = useRouter()
const route = useRoute()
const profilesStore = useBrowserProfilesStore()

const loading = ref(true)
const showDeleteDialog = ref(false)
const isRunning = ref(false)
const launching = ref(false)
const activeTab = ref('overview')

const profile = computed(() => profilesStore.currentProfile)

const tabs = [
  { id: 'overview', label: 'Overview' },
  { id: 'fingerprint', label: 'Fingerprint' },
  { id: 'proxy', label: 'Proxy & Network' }
]

const stats = computed(() => {
  if (!profile.value) return []
  return [
    { label: 'Times Used', value: profile.value.usage_count || 0 },
    { label: 'Browser Type', value: profile.value.browser_type.toUpperCase() },
    { label: 'Status', value: profile.value.status.toUpperCase(), color: profile.value.status === 'active' ? 'text-green-600 dark:text-green-400' : '' }
  ]
})

const getBrowserIcon = (browserType: string) => {
  switch(browserType) {
    case 'chromium': return Chrome
    case 'firefox': return Globe
    case 'webkit': return Globe
    default: return Chrome
  }
}

onMounted(async () => {
  try {
    await profilesStore.fetchProfileById(route.params.id as string)
  } catch (error) {
    toast.error('Failed to load profile')
    router.push('/browser-profiles')
  } finally {
    loading.value = false
  }
})

const handleEdit = () => {
  router.push(`/browser-profiles/${profile.value?.id}/edit`)
}

const toggleLaunch = async () => {
  if (!profile.value || launching.value) return
  
  launching.value = true
  try {
    if (isRunning.value) {
      await profilesStore.stopProfile(profile.value.id)
      isRunning.value = false
      toast.success('Profile stopped')
    } else {
      await profilesStore.launchProfile(profile.value.id, false)
      isRunning.value = true
      toast.success('Profile launched')
    }
  } catch (error) {
    toast.error(`Failed to ${isRunning.value ? 'stop' : 'launch'} profile`)
  } finally {
    launching.value = false
  }
}

const handleDuplicate = async () => {
  try {
    await profilesStore.duplicateProfile(route.params.id as string)
    toast.success('Profile duplicated successfully')
    router.push('/browser-profiles')
  } catch (error) {
    toast.error('Failed to duplicate profile')
  }
}

const handleDelete = () => {
  showDeleteDialog.value = true
}

const confirmDelete = async () => {
  try {
    await profilesStore.deleteProfile(route.params.id as string)
    toast.success('Profile deleted successfully')
    router.push('/browser-profiles')
  } catch (error) {
    toast.error('Failed to delete profile')
  }
}

const formatDate = (dateString?: string) => {
  if (!dateString) return 'Never'
  return new Date(dateString).toLocaleDateString('en-US', { 
    month: 'short', 
    day: 'numeric', 
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}
</script>

<template>
  <PageLayout>
    <div v-if="loading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <template v-else-if="profile">
      <PageHeader 
        :title="profile.name"
        :description="profile.description || 'Browser profile'"
      >
        <template #title-prefix>
          <component :is="getBrowserIcon(profile.browser_type)" class="w-6 h-6 mr-3 text-primary" />
        </template>
        <template #title-suffix>
          <Badge 
            variant="outline"
            :class="{
              'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20': profile.status === 'active',
              'bg-gray-500/10 text-gray-600 dark:text-gray-400 border-gray-500/20': profile.status === 'inactive'
            }"
            class="text-xs font-medium ml-3"
          >
            {{ profile.status }}
          </Badge>
        </template>
        <template #actions>
          <Button variant="outline" size="default" @click="toggleLaunch" :disabled="launching || profile.status !== 'active'">
            <Loader2 v-if="launching" class="w-4 h-4 mr-2 animate-spin" />
            <StopCircle v-else-if="isRunning" class="w-4 h-4 mr-2" />
            <Play v-else class="w-4 h-4 mr-2" />
            {{ launching ? 'Processing...' : (isRunning ? 'Stop' : 'Launch') }}
          </Button>
          <Button variant="outline" size="default" @click="handleDuplicate">
            <Copy class="w-4 h-4 mr-2" />
            Duplicate
          </Button>
          <Button variant="outline" size="default" @click="handleEdit">
            <Pencil class="w-4 h-4 mr-2" />
            Edit
          </Button>
          <Button variant="outline" size="default" @click="handleDelete" class="text-destructive hover:text-destructive">
            <Trash2 class="w-4 h-4 mr-2" />
            Delete
          </Button>
        </template>
      </PageHeader>

      <StatsBar :stats="stats" />

      <TabBar :tabs="tabs" v-model="activeTab" />

      <div class="flex-1 overflow-auto px-6 pb-6">
        <!-- Overview Tab -->
        <div v-if="activeTab === 'overview'" class="max-w-6xl space-y-6">
          <div class="grid gap-6 md:grid-cols-2">
            <!-- General Info -->
            <div class="bg-card border rounded-lg p-6">
              <h3 class="text-sm font-semibold mb-4 text-muted-foreground uppercase tracking-wide">General Information</h3>
              <div class="space-y-3">
                <div>
                  <div class="text-xs text-muted-foreground mb-1">Profile ID</div>
                  <div class="font-mono text-sm">{{ profile.id }}</div>
                </div>
                <div>
                  <div class="text-xs text-muted-foreground mb-1">Browser Type</div>
                  <div class="capitalize">{{ profile.browser_type }}</div>
                </div>
                <div v-if="profile.folder">
                  <div class="text-xs text-muted-foreground mb-1">Folder</div>
                  <div>{{ profile.folder }}</div>
                </div>
                <div>
                  <div class="text-xs text-muted-foreground mb-1">Created</div>
                  <div class="text-sm">{{ formatDate(profile.created_at) }}</div>
                </div>
                <div>
                  <div class="text-xs text-muted-foreground mb-1">Last Used</div>
                  <div class="text-sm">{{ formatDate(profile.last_used_at) }}</div>
                </div>
              </div>
            </div>

            <!-- Browser Configuration -->
            <div class="bg-card border rounded-lg p-6">
              <h3 class="text-sm font-semibold mb-4 text-muted-foreground uppercase tracking-wide">Browser Configuration</h3>
              <div class="space-y-3">
                <div v-if="profile.executable_path">
                  <div class="text-xs text-muted-foreground mb-1">Custom Executable</div>
                  <div class="font-mono text-xs break-all">{{ profile.executable_path }}</div>
                </div>
                <div v-if="profile.cdp_endpoint">
                  <div class="text-xs text-muted-foreground mb-1">CDP Endpoint</div>
                  <div class="font-mono text-xs break-all">{{ profile.cdp_endpoint }}</div>
                </div>
                <div v-if="!profile.executable_path && !profile.cdp_endpoint">
                  <div class="text-sm text-muted-foreground">Using default {{ profile.browser_type }} browser</div>
                </div>
              </div>
            </div>
          </div>

          <!-- User Agent -->
          <div class="bg-card border rounded-lg p-6">
            <h3 class="text-sm font-semibold mb-3 text-muted-foreground uppercase tracking-wide">User Agent</h3>
            <div class="font-mono text-xs break-all text-muted-foreground">{{ profile.user_agent }}</div>
          </div>
        </div>

        <!-- Fingerprint Tab -->
        <div v-if="activeTab === 'fingerprint'" class="max-w-6xl">
          <div class="grid gap-6 md:grid-cols-3">
            <div class="bg-card border rounded-lg p-4">
              <div class="text-xs text-muted-foreground mb-1">Platform</div>
              <div class="font-medium">{{ profile.platform }}</div>
            </div>
            <div class="bg-card border rounded-lg p-4">
              <div class="text-xs text-muted-foreground mb-1">Screen Resolution</div>
              <div class="font-medium">{{ profile.screen_width }}×{{ profile.screen_height }}</div>
            </div>
            <div class="bg-card border rounded-lg p-4">
              <div class="text-xs text-muted-foreground mb-1">Timezone</div>
              <div class="font-medium">{{ profile.timezone }}</div>
            </div>
            <div class="bg-card border rounded-lg p-4">
              <div class="text-xs text-muted-foreground mb-1">Locale</div>
              <div class="font-medium">{{ profile.locale }}</div>
            </div>
            <div class="bg-card border rounded-lg p-4">
              <div class="text-xs text-muted-foreground mb-1">CPU Cores</div>
              <div class="font-medium">{{ profile.hardware_concurrency }}</div>
            </div>
            <div class="bg-card border rounded-lg p-4">
              <div class="text-xs text-muted-foreground mb-1">Memory</div>
              <div class="font-medium">{{ profile.device_memory }} GB</div>
            </div>
            <div class="bg-card border rounded-lg p-4">
              <div class="text-xs text-muted-foreground mb-1">WebGL Vendor</div>
              <div class="font-medium text-sm">{{ profile.webgl_vendor }}</div>
            </div>
            <div class="bg-card border rounded-lg p-4">
              <div class="text-xs text-muted-foreground mb-1">WebGL Renderer</div>
              <div class="font-medium text-sm">{{ profile.webgl_renderer }}</div>
            </div>
            <div class="bg-card border rounded-lg p-4">
              <div class="text-xs text-muted-foreground mb-1">Canvas Noise</div>
              <Badge :variant="profile.canvas_noise ? 'default' : 'outline'" class="text-xs">
                {{ profile.canvas_noise ? 'Enabled' : 'Disabled' }}
              </Badge>
            </div>
            <div class="bg-card border rounded-lg p-4">
              <div class="text-xs text-muted-foreground mb-1">WebRTC</div>
              <Badge :variant="profile.disable_webrtc ? 'destructive' : 'default'" class="text-xs">
                {{ profile.disable_webrtc ? 'Disabled' : 'Enabled' }}
              </Badge>
            </div>
          </div>
        </div>

        <!-- Proxy Tab -->
        <div v-if="activeTab === 'proxy'" class="max-w-6xl">
          <div v-if="profile.proxy_enabled" class="grid gap-6 md:grid-cols-2">
            <div class="bg-card border rounded-lg p-6">
              <h3 class="text-sm font-semibold mb-4 text-muted-foreground uppercase tracking-wide">Proxy Configuration</h3>
              <div class="space-y-3">
                <div>
                  <div class="text-xs text-muted-foreground mb-1">Proxy Type</div>
                  <div class="font-medium uppercase">{{ profile.proxy_type }}</div>
                </div>
                <div>
                  <div class="text-xs text-muted-foreground mb-1">Proxy Server</div>
                  <div class="font-mono text-sm">{{ profile.proxy_server }}</div>
                </div>
                <div v-if="profile.proxy_username">
                  <div class="text-xs text-muted-foreground mb-1">Authentication</div>
                  <div class="text-sm">Configured</div>
                </div>
              </div>
            </div>
          </div>
          <div v-else class="bg-card border rounded-lg p-12 text-center">
            <p class="text-muted-foreground">No proxy configured for this profile</p>
          </div>
        </div>
      </div>
    </template>

    <!-- Delete Confirmation Dialog -->
    <div v-if="showDeleteDialog" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-background border rounded-lg p-6 max-w-md w-full mx-4">
        <h3 class="text-lg font-semibold mb-2">Delete Profile?</h3>
        <p class="text-sm text-muted-foreground mb-4">
          Are you sure you want to delete this profile? This action cannot be undone.
        </p>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="showDeleteDialog = false">Cancel</Button>
          <Button variant="destructive" @click="confirmDelete">Delete</Button>
        </div>
      </div>
    </div>
  </PageLayout>
</template>
