<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useBrowserProfilesStore } from '@/stores/browserProfiles'
import { browserProfilesApi } from '@/api/browserProfiles'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import PageLayout from '@/components/layout/PageLayout.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Loader2, Sparkles, TestTube, Chrome, Globe } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const router = useRouter()
const route = useRoute()
const profilesStore = useBrowserProfilesStore()

const isEdit = ref(false)
const loading = ref(false)
const testing = ref(false)

// Form data
const formData = ref({
  name: '',
  description: '',
  browser_type: 'chromium' as 'chromium' | 'firefox' | 'webkit',
  folder: '',
  tags: [] as string[],
  executable_path: '',
  cdp_endpoint: '',
  user_agent: '',
  platform: 'Win32',
  screen_width: 1920,
  screen_height: 1080,
  timezone: 'America/New_York',
  locale: 'en-US',
  languages: ['en-US', 'en'],
  webgl_vendor: 'Intel Inc.',
  webgl_renderer: 'Intel Iris OpenGL Engine',
  canvas_noise: true,
  hardware_concurrency: 4,
  device_memory: 8,
  do_not_track: false,
  disable_webrtc: false,
  proxy_enabled: false,
  proxy_type: 'http',
  proxy_server: '',
  proxy_username: '',
  proxy_password: '',
  clear_on_close: true
})

const browserTypes = ref([
  { value: 'chromium', label: 'Chromium', icon: Chrome, description: 'Google Chrome, Microsoft Edge, Brave' },
  { value: 'firefox', label: 'Firefox', icon: Globe, description: 'Mozilla Firefox' },
  { value: 'webkit', label: 'WebKit', icon: Globe, description: 'Safari (macOS/iOS)' }
])

const platforms = ['Win32', 'MacIntel', 'Linux x86_64', 'Linux armv7l']
const resolutions = [
  { width: 1920, height: 1080, label: '1920x1080 (Full HD)' },
  { width: 1366, height: 768, label: '1366x768' },
  { width: 1440, height: 900, label: '1440x900' },
  { width: 2560, height: 1440, label: '2560x1440 (2K)' }
]

const timezones = [
  'America/New_York',
  'America/Los_Angeles', 
  'Europe/London',
  'Europe/Paris',
  'Asia/Tokyo',
  'Asia/Shanghai'
]

onMounted(async () => {
  if (route.params.id) {
    isEdit.value = true
    loading.value = true
    try {
      const profile = await profilesStore.fetchProfileById(route.params.id as string)
      // Populate form with existing data
      Object.assign(formData.value, profile)
    } catch (error) {
      toast.error('Failed to load profile')
      router.push('/browser-profiles')
    } finally {
      loading.value = false
    }
  }
})

const generateRandomFingerprint = async () => {
  loading.value = true
  try {
    const fingerprint = await profilesStore.generateFingerprint(formData.value.browser_type)
    formData.value.user_agent = fingerprint.UserAgent
    formData.value.platform = fingerprint.Platform
    formData.value.screen_width = fingerprint.ScreenWidth
    formData.value.screen_height = fingerprint.ScreenHeight
    formData.value.timezone = fingerprint.Timezone
    formData.value.locale = fingerprint.Locale
    formData.value.languages = fingerprint.Languages
    formData.value.webgl_vendor = fingerprint.WebGLVendor
    formData.value.webgl_renderer = fingerprint.WebGLRenderer
    formData.value.hardware_concurrency = fingerprint.HardwareConcurrency
    formData.value.device_memory = fingerprint.DeviceMemory
    toast.success('Fingerprint generated successfully')
  } catch (error) {
    toast.error('Failed to generate fingerprint')
  } finally {
    loading.value = false
  }
}

const testBrowserConfig = async () => {
  testing.value = true
  try {
    const result = await browserProfilesApi.testBrowserConfig({
      browser_type: formData.value.browser_type,
      executable_path: formData.value.executable_path || undefined,
      cdp_endpoint: formData.value.cdp_endpoint || undefined
    })
    if (result.data.success) {
      toast.success('Browser configuration is valid')
    } else {
      toast.error(result.data.error || 'Test failed')
    }
  } catch (error: any) {
    toast.error(error.response?.data?.error || 'Test failed')
  } finally {
    testing.value = false
  }
}

const handleSubmit = async () => {
  if (!formData.value.name.trim()) {
    toast.error('Please enter a profile name')
    return
  }

  loading.value = true
  try {
    if (isEdit.value) {
      await profilesStore.updateProfile(route.params.id as string, formData.value)
      toast.success('Profile updated successfully')
    } else {
      await profilesStore.createProfile(formData.value)
      toast.success('Profile created successfully')
    }
    router.push('/browser-profiles')
  } catch (error: any) {
    toast.error(error.response?.data?.error || `Failed to ${isEdit.value ? 'update' : 'create'} profile`)
  } finally {
    loading.value = false
  }
}

const handleCancel = () => {
  router.push('/browser-profiles')
}
</script>

<template>
  <PageLayout>
    <PageHeader 
      :title="isEdit ? 'Edit Browser Profile' : 'Create Browser Profile'"
      :description="isEdit ? 'Update your browser profile configuration' : 'Create a new browser profile with custom fingerprints'"
    />

    <div class="flex-1 overflow-auto px-6 pb-6">
      <div v-if="loading && isEdit" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <form v-else @submit.prevent="handleSubmit" class="max-w-4xl space-y-8">
        <!-- General Information -->
        <div class="bg-card border rounded-lg p-6 space-y-4">
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-lg font-semibold">General Information</h3>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <Label for="name">Profile Name <span class="text-destructive">*</span></Label>
              <Input id="name" v-model="formData.name" placeholder="My Browser Profile" required />
            </div>

            <div class="space-y-2">
              <Label for="folder">Folder</Label>
              <Input id="folder" v-model="formData.folder" placeholder="default" />
            </div>
          </div>

          <div class="space-y-2">
            <Label for="description">Description</Label>
            <Textarea id="description" v-model="formData.description" placeholder="Profile description..." rows="3" />
          </div>
        </div>

        <!-- Browser Configuration -->
        <div class="bg-card border rounded-lg p-6 space-y-4">
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-lg font-semibold">Browser Configuration</h3>
            <Button type="button" variant="outline" size="sm" @click="testBrowserConfig" :disabled="testing">
              <TestTube class="w-4 h-4 mr-2" />
              {{ testing ? 'Testing...' : 'Test Config' }}
            </Button>
          </div>

          <div class="space-y-2">
            <Label>Browser Type <span class="text-destructive">*</span></Label>
            <div class="grid grid-cols-3 gap-3">
              <button
                v-for="browser in browserTypes"
                :key="browser.value"
                type="button"
                @click="formData.browser_type = browser.value as any"
                :class="[
                  'p-4 border-2 rounded-lg text-left transition-all',
                  formData.browser_type === browser.value 
                    ? 'border-primary bg-primary/5' 
                    : 'border-border hover:border-primary/50'
                ]"
              >
                <component :is="browser.icon" class="w-6 h-6 mb-2" :class="formData.browser_type === browser.value ? 'text-primary' : 'text-muted-foreground'" />
                <div class="font-medium text-sm">{{ browser.label }}</div>
                <div class="text-xs text-muted-foreground mt-1">{{ browser.description }}</div>
              </button>
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <Label for="executable_path">Custom Executable Path (Optional)</Label>
              <Input id="executable_path" v-model="formData.executable_path" placeholder="/usr/bin/google-chrome" />
              <p class="text-xs text-muted-foreground">Leave empty to use default browser</p>
            </div>

            <div class="space-y-2">
              <Label for="cdp_endpoint">CDP WebSocket Endpoint (Optional)</Label>
              <Input id="cdp_endpoint" v-model="formData.cdp_endpoint" placeholder="ws://localhost:9222/devtools/..." />
              <p class="text-xs text-muted-foreground">Connect to existing browser instance</p>
            </div>
          </div>
        </div>

        <!-- Fingerprint Settings -->
        <div class="bg-card border rounded-lg p-6 space-y-4">
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-lg font-semibold">Fingerprint Settings</h3>
            <Button type="button" variant="outline" size="sm" @click="generateRandomFingerprint" :disabled="loading">
              <Sparkles class="w-4 h-4 mr-2" />
              Generate Random
            </Button>
          </div>

          <div class="space-y-2">
            <Label for="user_agent">User Agent</Label>
            <Textarea id="user_agent" v-model="formData.user_agent" rows="2" placeholder="Mozilla/5.0..." />
          </div>

          <div class="grid gap-4 md:grid-cols-3">
            <div class="space-y-2">
              <Label for="platform">Platform</Label>
              <Select v-model="formData.platform">
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="platform in platforms" :key="platform" :value="platform">
                    {{ platform }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="space-y-2">
              <Label for="timezone">Timezone</Label>
              <Select v-model="formData.timezone">
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="tz in timezones" :key="tz" :value="tz">
                    {{ tz }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="space-y-2">
              <Label for="locale">Locale</Label>
              <Input id="locale" v-model="formData.locale" placeholder="en-US" />
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <Label>Screen Resolution</Label>
              <Select v-model="formData.screen_width">
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="res in resolutions" :key="res.width" :value="res.width">
                    {{ res.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div class="space-y-2">
                <Label for="hardware">CPU Cores</Label>
                <Input id="hardware" v-model.number="formData.hardware_concurrency" type="number" min="1" max="64" />
              </div>
              <div class="space-y-2">
                <Label for="memory">Memory (GB)</Label>
                <Input id="memory" v-model.number="formData.device_memory" type="number" min="1" max="128" />
              </div>
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <Label for="webgl_vendor">WebGL Vendor</Label>
              <Input id="webgl_vendor" v-model="formData.webgl_vendor" placeholder="Intel Inc." />
            </div>

            <div class="space-y-2">
              <Label for="webgl_renderer">WebGL Renderer</Label>
              <Input id="webgl_renderer" v-model="formData.webgl_renderer" placeholder="Intel Iris OpenGL Engine" />
            </div>
          </div>

          <div class="flex items-center space-x-2 pt-2">
            <Switch id="canvas_noise" v-model:checked="formData.canvas_noise" />
            <Label for="canvas_noise" class="cursor-pointer">Enable Canvas Noise</Label>
          </div>

          <div class="flex items-center space-x-2">
            <Switch id="disable_webrtc" v-model:checked="formData.disable_webrtc" />
            <Label for="disable_webrtc" class="cursor-pointer">Disable WebRTC</Label>
          </div>
        </div>

        <!-- Proxy Configuration -->
        <div class="bg-card border rounded-lg p-6 space-y-4">
          <h3 class="text-lg font-semibold mb-4">Proxy Configuration</h3>

          <div class="flex items-center space-x-2">
            <Switch id="proxy_enabled" v-model:checked="formData.proxy_enabled" />
            <Label for="proxy_enabled" class="cursor-pointer">Enable Proxy</Label>
          </div>

          <div v-if="formData.proxy_enabled" class="space-y-4 mt-4">
            <div class="grid gap-4 md:grid-cols-2">
              <div class="space-y-2">
                <Label for="proxy_type">Proxy Type</Label>
                <Select v-model="formData.proxy_type">
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="http">HTTP</SelectItem>
                    <SelectItem value="https">HTTPS</SelectItem>
                    <SelectItem value="socks5">SOCKS5</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div class="space-y-2">
                <Label for="proxy_server">Proxy Server</Label>
                <Input id="proxy_server" v-model="formData.proxy_server" placeholder="proxy.example.com:8080" />
              </div>
            </div>

            <div class="grid gap-4 md:grid-cols-2">
              <div class="space-y-2">
                <Label for="proxy_username">Username (Optional)</Label>
                <Input id="proxy_username" v-model="formData.proxy_username" placeholder="username" />
              </div>

              <div class="space-y-2">
                <Label for="proxy_password">Password (Optional)</Label>
                <Input id="proxy_password" v-model="formData.proxy_password" type="password" placeholder="password" />
              </div>
            </div>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex items-center justify-end gap-3 pt-4 border-t">
          <Button type="button" variant="outline" @click="handleCancel">Cancel</Button>
          <Button type="submit" :disabled="loading">
            <Loader2 v-if="loading" class="w-4 h-4 mr-2 animate-spin" />
            {{ isEdit ? 'Update Profile' : 'Create Profile' }}
          </Button>
        </div>
      </form>
    </div>
  </PageLayout>
</template>
