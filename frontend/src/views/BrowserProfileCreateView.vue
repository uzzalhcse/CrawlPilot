<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
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
  driver_type: 'playwright' as 'playwright' | 'chromedp' | 'camoufox' | 'http',
  browser_type: 'chromium' as 'chromium' | 'firefox' | 'webkit',
  folder: '',
  tags: [] as string[],
  executable_path: '',
  cdp_endpoint: '',
  target_os: 'linux' as 'windows' | 'macos' | 'linux' | 'android' | 'ios',
  screen_width: 1920,
  screen_height: 1080,
  timezone: 'America/New_York',
  locale: 'en-US',
  languages: ['en-US', 'en'],
  do_not_track: false,
  disable_webrtc: false,
  proxy_enabled: false,
  proxy_type: 'http',
  proxy_server: '',
  proxy_username: '',
  proxy_password: '',
  clear_on_close: true,
  // Camoufox-specific
  geo_ip: '',
  virtual_headless: false,
  force_scope_access: false,
  auto_captcha_solve: false,
  block_images: false,
  block_webgl: false,
  humanize: 0,
  include_default_addons: true,
  enable_cache: false,
  user_data_dir: ''
})

// Driver types with browser compatibility info (HTTP excluded - uses inline browser_name)
const driverTypes = ref([
  { value: 'playwright', label: 'Playwright', description: 'All browsers (Chromium, Firefox, WebKit)', browsers: ['chromium', 'firefox', 'webkit'] },
  { value: 'camoufox', label: 'Camoufox', description: 'Stealth Firefox (Anti-detect)', browsers: ['firefox'] },
  { value: 'chromedp', label: 'Chromedp', description: 'Chromium only (CDP)', browsers: ['chromium'] }
])

const browserTypes = ref([
  { value: 'chromium', label: 'Chromium', icon: Chrome, description: 'Google Chrome, Microsoft Edge, Brave' },
  { value: 'firefox', label: 'Firefox', icon: Globe, description: 'Mozilla Firefox' },
  { value: 'webkit', label: 'WebKit', icon: Globe, description: 'Safari (macOS/iOS)' }
])

const platforms = [
  { value: 'windows', label: 'Windows' },
  { value: 'macos', label: 'macOS' },
  { value: 'linux', label: 'Linux' },
  { value: 'android', label: 'Android' },
  { value: 'ios', label: 'iOS' }
]
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
    formData.value.screen_width = fingerprint.ScreenWidth
    formData.value.screen_height = fingerprint.ScreenHeight
    formData.value.timezone = fingerprint.Timezone
    formData.value.locale = fingerprint.Locale
    formData.value.languages = fingerprint.Languages
    toast.success('Fingerprint generated successfully')
  } catch (error) {
    toast.error('Failed to generate fingerprint')
  } finally {
    loading.value = false
  }
}

// Auto-enable force_scope_access when auto_captcha_solve is enabled
// force_scope_access is REQUIRED for CAPTCHA solving to work
watch(() => formData.value.auto_captcha_solve, (newVal) => {
  if (newVal) {
    formData.value.force_scope_access = true
  }
})

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

          <!-- Driver Type Selector -->
          <div class="space-y-2">
            <Label>Driver Type <span class="text-destructive">*</span></Label>
            <div class="grid grid-cols-3 gap-3">
              <button
                v-for="driver in driverTypes"
                :key="driver.value"
                type="button"
                @click="formData.driver_type = driver.value as any; if (driver.value === 'chromedp') formData.browser_type = 'chromium'; if (driver.value === 'camoufox') formData.browser_type = 'firefox'"
                :class="[
                  'p-3 border-2 rounded-lg text-left transition-all',
                  formData.driver_type === driver.value 
                    ? 'border-primary bg-primary/5' 
                    : 'border-border hover:border-primary/50'
                ]"
              >
                <div class="font-medium text-sm">{{ driver.label }}</div>
                <div class="text-xs text-muted-foreground mt-1">{{ driver.description }}</div>
              </button>
            </div>
          </div>

          <div class="space-y-2">
            <Label>Browser Type <span class="text-destructive">*</span></Label>
            <div class="grid grid-cols-3 gap-3">
              <button
                v-for="browser in browserTypes"
                :key="browser.value"
                type="button"
                @click="formData.browser_type = browser.value as any"
                :disabled="formData.driver_type === 'chromedp' && browser.value !== 'chromium'"
                :class="[
                  'p-4 border-2 rounded-lg text-left transition-all',
                  formData.browser_type === browser.value 
                    ? 'border-primary bg-primary/5' 
                    : 'border-border hover:border-primary/50',
                  formData.driver_type === 'chromedp' && browser.value !== 'chromium'
                    ? 'opacity-50 cursor-not-allowed'
                    : formData.driver_type === 'camoufox' && browser.value !== 'firefox'
                    ? 'opacity-50 cursor-not-allowed'
                    : ''
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

        <!-- Camoufox Settings -->
        <div v-if="formData.driver_type === 'camoufox'" class="bg-card border rounded-lg p-6 space-y-4">
          <h3 class="text-lg font-semibold">Camoufox Settings</h3>
          
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="space-y-2">
              <Label>GeoIP (Optional)</Label>
              <Input v-model="formData.geo_ip" placeholder="IP address or 'auto'" />
              <p class="text-xs text-muted-foreground">Leave empty or use 'auto' for automatic detection.</p>
            </div>
            
            <div class="space-y-2">
              <Label>Humanize Mouse (Seconds)</Label>
              <Input type="number" v-model.number="formData.humanize" min="0" step="0.1" />
              <p class="text-xs text-muted-foreground">0 to disable. Adds realistic mouse movement delays.</p>
            </div>

            <div class="space-y-2">
              <Label>User Data Dir (Optional)</Label>
              <Input v-model="formData.user_data_dir" placeholder="/path/to/data/dir" />
              <p class="text-xs text-muted-foreground">Persistent storage for cookies/local storage.</p>
            </div>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="flex items-center justify-between border p-3 rounded-lg">
              <div class="space-y-0.5">
                <Label>Virtual Headless</Label>
                <p class="text-xs text-muted-foreground">Use Xvfb virtual display (Linux only)</p>
              </div>
              <Switch v-model="formData.virtual_headless" />
            </div>

            <div class="flex items-center justify-between border p-3 rounded-lg" :class="{ 'opacity-60': formData.auto_captcha_solve }">
              <div class="space-y-0.5">
                <Label>Force Scope Access</Label>
                <p class="text-xs text-muted-foreground">
                  {{ formData.auto_captcha_solve 
                    ? 'Required for Auto CAPTCHA Solve (auto-enabled)' 
                    : 'Access closed Shadow DOM (for CAPTCHA)' }}
                </p>
              </div>
              <Switch v-model="formData.force_scope_access" :disabled="formData.auto_captcha_solve" />
            </div>

            <div class="flex items-center justify-between border p-3 rounded-lg">
              <div class="space-y-0.5">
                <Label>Auto CAPTCHA Solve</Label>
                <p class="text-xs text-muted-foreground">Enable auto-solving middleware</p>
              </div>
              <Switch v-model="formData.auto_captcha_solve" />
            </div>

            <div class="flex items-center justify-between border p-3 rounded-lg">
              <div class="space-y-0.5">
                <Label>Block Images</Label>
                <p class="text-xs text-muted-foreground">Block image loading for speed</p>
              </div>
              <Switch v-model="formData.block_images" />
            </div>

            <div class="flex items-center justify-between border p-3 rounded-lg">
              <div class="space-y-0.5">
                <Label>Block WebGL</Label>
                <p class="text-xs text-muted-foreground">Disable WebGL (may increase stealth)</p>
              </div>
              <Switch v-model="formData.block_webgl" />
            </div>

            <div class="flex items-center justify-between border p-3 rounded-lg">
              <div class="space-y-0.5">
                <Label>Include Default Addons</Label>
                <p class="text-xs text-muted-foreground">Install uBlock Origin etc.</p>
              </div>
              <Switch v-model="formData.include_default_addons" />
            </div>

            <div class="flex items-center justify-between border p-3 rounded-lg">
              <div class="space-y-0.5">
                <Label>Enable Cache</Label>
                <p class="text-xs text-muted-foreground">Enable browser caching</p>
              </div>
              <Switch v-model="formData.enable_cache" />
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

          <div class="grid gap-4 md:grid-cols-3">
            <div class="space-y-2">
              <Label for="target_os">Target OS</Label>
              <Select v-model="formData.target_os">
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="platform in platforms" :key="platform.value" :value="platform.value">
                    {{ platform.label }}
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
              <p v-if="formData.geo_ip === 'auto'" class="text-xs text-yellow-500 font-medium">
                Warning: Overwritten by GeoIP 'auto'
              </p>
            </div>

            <div class="space-y-2">
              <Label for="locale">Locale</Label>
              <Input id="locale" v-model="formData.locale" placeholder="en-US" />
              <p v-if="formData.geo_ip === 'auto'" class="text-xs text-yellow-500 font-medium">
                Warning: Overwritten by GeoIP 'auto'
              </p>
            </div>
          </div>

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

          <div class="flex items-center space-x-2 pt-2">
            <Switch id="disable_webrtc" v-model="formData.disable_webrtc" />
            <Label for="disable_webrtc" class="cursor-pointer">Disable WebRTC</Label>
          </div>
        </div>

        <!-- Proxy Configuration -->
        <div class="bg-card border rounded-lg p-6 space-y-4">
          <h3 class="text-lg font-semibold mb-4">Proxy Configuration</h3>

          <div class="flex items-center space-x-2">
            <Switch id="proxy_enabled" v-model="formData.proxy_enabled" />
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
