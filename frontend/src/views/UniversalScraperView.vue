<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { 
  Zap, 
  ChevronDown,
  ChevronRight,
  Play,
  FileText,
  Image as ImageIcon,
  FileCode,
  Info,
  Sparkles,
  Monitor,
  User,
  Loader2,
  AlertCircle
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { useBrowserProfilesStore } from '@/stores/browserProfiles'
import { 
  submitScrape, 
  pollScrapeResult, 
  type ScrapeResult
} from '@/api/scraper'

// State
const url = ref('https://aqua-has.com/product/dms10a/')
const advancedSettingsOpen = ref(false)
const selectedOutputFormat = ref('html')
const isRunning = ref(false)
const hasResult = ref(false)
const timeout = ref(30)
const waitForSelector = ref('')
const headless = ref(true)
const error = ref('')
const useProxy = ref(false)
const proxyTier = ref(1)

// Proxy tier options
const proxyTiers = [
  { value: 1, label: 'Tier 1 - Datacenter' },
  { value: 2, label: 'Tier 2 - Residential' },
  { value: 3, label: 'Tier 3 - Mobile' },
]

// Check if any advanced settings are modified from defaults
const hasAdvancedSettings = computed(() => {
  return timeout.value !== 30 || waitForSelector.value !== '' || !headless.value || useProxy.value
})

// Driver & Profile selection
const selectedDriver = ref('http')
const selectedProfile = ref('')

// Use drivers store for dynamic driver list
import { useDriversStore } from '@/stores/drivers'
const driversStore = useDriversStore()

// Use browser profiles store
const profilesStore = useBrowserProfilesStore()

// Load drivers and profiles on mount
onMounted(async () => {
  // Fetch drivers from API
  await driversStore.fetchDrivers()
  
  // Fetch browser profiles
  if (profilesStore.profiles.length === 0) {
    try {
      await profilesStore.fetchProfiles()
    } catch (err) {
      console.error('Failed to load profiles:', err)
    }
  }
})

// Filter profiles based on selected driver
const filteredProfiles = computed(() => {
  if (!selectedDriver.value || selectedDriver.value === 'http') {
    // HTTP driver doesn't use profiles
    return []
  }
  return profilesStore.profiles.filter(p => 
    p.driver_type === selectedDriver.value || !p.driver_type
  )
})

// Reset profile when driver changes if current profile is incompatible
watch(selectedDriver, (newDriver) => {
  if (newDriver === 'http') {
    selectedProfile.value = ''
  } else if (selectedProfile.value) {
    const profile = profilesStore.profiles.find(p => p.id === selectedProfile.value)
    if (profile && profile.driver_type && profile.driver_type !== newDriver) {
      selectedProfile.value = ''
    }
  }
})

// Result data
const resultContent = ref('')
const resultScreenshot = ref('')
const resultStatus = ref<'pending' | 'running' | 'completed' | 'failed'>('pending')
const resultDuration = ref(0)

const outputFormats = [
  { id: 'html', label: 'HTML', icon: FileText },
  { id: 'markdown', label: 'Markdown', icon: FileCode },
  { id: 'screenshot', label: 'Screenshot', icon: ImageIcon },
]

const runScraper = async () => {
  if (!url.value) {
    error.value = 'Please enter a URL'
    return
  }

  error.value = ''
  isRunning.value = true
  hasResult.value = false
  resultContent.value = ''
  resultScreenshot.value = ''
  resultStatus.value = 'pending'

  try {
    // Submit scrape request
    const response = await submitScrape({
      url: url.value,
      driver: selectedDriver.value,
      profile_id: selectedProfile.value || undefined,
      output_format: selectedOutputFormat.value,
      timeout: timeout.value,
      wait_for_selector: waitForSelector.value || undefined,
      headless: selectedDriver.value !== 'http' ? headless.value : undefined,
      use_proxy: useProxy.value,
      proxy_tier: useProxy.value ? proxyTier.value : undefined
    })

    // Poll for result
    const result = await pollScrapeResult(
      response.scrape_id,
      (update: ScrapeResult) => {
        resultStatus.value = update.status
        if (update.duration_ms) {
          resultDuration.value = update.duration_ms
        }
      },
      120, // max 2 minutes
      500  // poll every 500ms
    )

    // Update result
    hasResult.value = true
    if (result.content) {
      resultContent.value = result.content
    }
    if (result.screenshot) {
      resultScreenshot.value = result.screenshot
    }
    if (result.error) {
      error.value = result.error
    }
    resultDuration.value = result.duration_ms

  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Scrape failed'
    hasResult.value = false
  } finally {
    isRunning.value = false
  }
}
</script>

<template>
  <div class="min-h-screen">
    <div class="p-6">
      <!-- Header -->
      <div class="mb-6">
        <h1 class="text-2xl font-semibold flex items-center gap-2">
          <Sparkles class="w-6 h-6 text-primary" />
          Universal Scraper
        </h1>
        <p class="text-muted-foreground text-sm mt-0.5">Extract data from any webpage with AI-powered parsing</p>
      </div>

      <!-- Two Column Layout -->
      <div class="grid grid-cols-2 gap-6">
        <!-- Left Panel: Input -->
        <div class="space-y-4">
          <!-- URL Input Card -->
          <div class="rounded-xl border bg-card p-5">
            <h2 class="text-sm font-semibold mb-3">Enter a URL to extract data from</h2>
            <input
              v-model="url"
              type="text"
              placeholder="https://example.com/products"
              class="w-full px-4 py-3 text-sm bg-background border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary transition-colors"
            />
          </div>

          <!-- Error Display -->
          <div v-if="error" class="rounded-xl border border-red-500/30 bg-red-500/5 p-4 flex items-center gap-3">
            <AlertCircle class="w-4 h-4 text-red-500 shrink-0" />
            <span class="text-sm text-red-600 dark:text-red-400">{{ error }}</span>
          </div>

          <!-- AI Unblocker Banner -->
          <div v-if="0" class="rounded-xl border border-emerald-500/30 bg-emerald-500/5 p-4 flex items-center gap-3">
            <div class="p-2 rounded-lg bg-emerald-500/10">
              <Zap class="w-4 h-4 text-emerald-500" />
            </div>
            <span class="text-sm text-emerald-600 dark:text-emerald-400">AI Web Unblocker is included for all requests</span>
          </div>

          <!-- Execution Environment Card -->
          <div class="rounded-xl border bg-card p-5">
            <div class="flex items-center gap-3 mb-4">
              <Monitor class="w-4 h-4 text-muted-foreground" />
              <span class="text-sm font-semibold">Execution Environment</span>
            </div>
            
            <!-- Driver Selection -->
            <div class="space-y-4">
              <div>
                <label class="text-xs font-medium text-muted-foreground mb-2 block">Driver</label>
                <div class="relative">
                  <select
                    v-model="selectedDriver"
                    class="w-full px-4 py-2.5 text-sm bg-background border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary transition-colors appearance-none cursor-pointer"
                  >
                    <option v-for="driver in driversStore.drivers" :key="driver.id" :value="driver.id">
                      {{ driver.name }}
                    </option>
                  </select>
                  <ChevronDown class="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
                </div>
                <p class="text-xs text-muted-foreground mt-1.5">
                  {{ driversStore.getDriverById(selectedDriver)?.description }}
                </p>
              </div>

              <!-- Profile Selection (only for browser drivers) -->
              <div v-if="selectedDriver !== 'http'">
                <label class="text-xs font-medium text-muted-foreground mb-2 block">Browser Profile</label>
                <div class="relative">
                  <select
                    v-model="selectedProfile"
                    class="w-full px-4 py-2.5 text-sm bg-background border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary transition-colors appearance-none cursor-pointer"
                  >
                    <option value="">No Profile (Default Settings)</option>
                    <option v-for="profile in filteredProfiles" :key="profile.id" :value="profile.id">
                      {{ profile.name }} ({{ profile.browser_type || 'chromium' }})
                    </option>
                  </select>
                  <ChevronDown class="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
                </div>
                <p class="text-xs text-muted-foreground mt-2 flex items-start gap-1.5">
                  <User class="w-3 h-3 mt-0.5 shrink-0" />
                  <span>Select a browser profile to use custom fingerprint and proxy settings.</span>
                </p>
              </div>
            </div>
          </div>

          <!-- Advanced Settings -->
          <div class="rounded-xl border bg-card overflow-hidden">
            <button
              @click="advancedSettingsOpen = !advancedSettingsOpen"
              class="w-full px-5 py-4 flex items-center justify-between hover:bg-muted/30 transition-colors"
            >
              <div class="flex items-center gap-2">
                <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Advanced Settings</span>
                <span v-if="hasAdvancedSettings" class="px-1.5 py-0.5 text-[10px] font-medium bg-primary/10 text-primary rounded">Modified</span>
              </div>
              <component 
                :is="advancedSettingsOpen ? ChevronDown : ChevronRight" 
                class="w-4 h-4 text-muted-foreground" 
              />
            </button>
            <div v-if="advancedSettingsOpen" class="px-5 pb-5 border-t">
              <!-- General Settings -->
              <div class="pt-4 space-y-4">
                <div class="grid grid-cols-2 gap-4">
                  <!-- Timeout -->
                  <div>
                    <label class="text-xs font-medium text-muted-foreground mb-1.5 block">Request Timeout</label>
                    <div class="relative">
                      <input
                        v-model.number="timeout"
                        type="number"
                        min="5"
                        max="300"
                        class="w-full px-3 py-2 text-sm bg-background border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/50 pr-12"
                      />
                      <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">sec</span>
                    </div>
                    <p class="text-[10px] text-muted-foreground/70 mt-1">5-300 seconds</p>
                  </div>


                </div>

                <!-- Wait For Selector (browser drivers only) -->
              <div v-if="selectedDriver !== 'http'" class="pt-2 border-t border-dashed">
                <label class="text-xs font-medium text-muted-foreground mb-1.5 block">Wait For Selector</label>
                <input
                  v-model="waitForSelector"
                  type="text"
                  placeholder=".product-list, #main-content, [data-loaded]"
                  class="w-full px-3 py-2 text-sm bg-background border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/50 font-mono text-xs"
                />
                <p class="text-[10px] text-muted-foreground/70 mt-1">Wait for element before extracting content</p>
                
                <!-- Headless Toggle -->
                <label class="flex items-center gap-3 cursor-pointer group mt-4">
                  <input
                    v-model="headless"
                    type="checkbox"
                    class="w-4 h-4 rounded border-input text-primary focus:ring-primary/50"
                  />
                  <div>
                    <span class="text-sm font-medium group-hover:text-foreground transition-colors">Headless Mode</span>
                    <p class="text-[10px] text-muted-foreground/70">Run browser without visible window (faster)</p>
                  </div>
                </label>
              </div>

              <!-- Proxy Settings -->
              <div class="pt-4 border-t border-dashed">
                <label class="flex items-center gap-3 cursor-pointer group">
                  <input
                    v-model="useProxy"
                    type="checkbox"
                    class="w-4 h-4 rounded border-input text-primary focus:ring-primary/50"
                  />
                  <div>
                    <span class="text-sm font-medium group-hover:text-foreground transition-colors">Use Proxy</span>
                    <p class="text-[10px] text-muted-foreground/70">Route request through a proxy server</p>
                  </div>
                </label>
                
                <!-- Proxy Tier Selection (shown when Use Proxy is checked) -->
                <div v-if="useProxy" class="mt-3 ml-7">
                  <label class="text-xs font-medium text-muted-foreground mb-1.5 block">Proxy Tier</label>
                  <div class="relative">
                    <select
                      v-model="proxyTier"
                      class="w-full px-3 py-2 text-sm bg-background border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/50 appearance-none cursor-pointer"
                    >
                      <option v-for="tier in proxyTiers" :key="tier.value" :value="tier.value">
                        {{ tier.label }}
                      </option>
                    </select>
                    <ChevronDown class="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
                  </div>
                  <p class="text-[10px] text-muted-foreground/70 mt-1">Higher tiers are more expensive but less likely to be blocked</p>
                </div>
              </div>
              </div>
            </div>
          </div>

          <!-- Output Format Selector -->
          <div class="rounded-xl border bg-card p-5">
            <div class="flex items-center gap-2 mb-3">
              <span class="text-sm font-semibold">Output Format</span>
              <Info class="w-3.5 h-3.5 text-muted-foreground" />
            </div>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="format in outputFormats"
                :key="format.id"
                @click="selectedOutputFormat = format.id"
                :class="[
                  'flex items-center gap-2 px-4 py-2 rounded-full text-sm font-medium transition-all duration-200',
                  selectedOutputFormat === format.id
                    ? 'bg-foreground text-background'
                    : 'bg-muted/50 text-muted-foreground hover:bg-muted hover:text-foreground'
                ]"
              >
                <component :is="format.icon" class="w-4 h-4" />
                {{ format.label }}
              </button>
            </div>
          </div>

          <!-- Run Button -->
          <div class="flex items-center gap-4">
            <Button
              @click="runScraper"
              :disabled="isRunning"
              class="flex-1 h-11 bg-emerald-500 hover:bg-emerald-600 text-white font-medium"
            >
              <Play v-if="!isRunning" class="w-4 h-4 mr-2" />
              <Loader2 v-else class="w-4 h-4 mr-2 animate-spin" />
              {{ isRunning ? 'Running...' : 'Run & Preview Result' }}
            </Button>
          </div>
        </div>

        <!-- Right Panel: Preview -->
        <div class="rounded-xl border bg-card overflow-hidden flex flex-col">
          <!-- Header -->
          <div class="p-5 border-b flex items-center justify-between">
            <div>
              <h2 class="text-sm font-semibold">Output Preview</h2>
              <p class="text-xs text-muted-foreground mt-0.5">
                {{ hasResult ? `Completed in ${resultDuration}ms` : 'Your result will appear here after running a request' }}
              </p>
            </div>
            <div v-if="isRunning" class="flex items-center gap-2 text-xs text-muted-foreground">
              <Loader2 class="w-3 h-3 animate-spin" />
              <span class="capitalize">{{ resultStatus }}</span>
            </div>
          </div>

          <!-- Content Area -->
          <div class="flex-1 p-5">
            <!-- Empty State -->
            <div 
              v-if="!hasResult && !isRunning" 
              class="h-full min-h-[400px] rounded-lg bg-muted/30 border border-dashed border-muted-foreground/20 flex flex-col items-center justify-center"
            >
              <div class="p-4 rounded-full bg-muted/50 mb-3">
                <component :is="outputFormats.find(f => f.id === selectedOutputFormat)?.icon || FileText" class="w-8 h-8 text-muted-foreground/50" />
              </div>
              <p class="text-sm font-medium text-muted-foreground">No preview available</p>
              <p class="text-xs text-muted-foreground/70 mt-1">Run a request to see the {{ selectedOutputFormat.toUpperCase() }} output here</p>
            </div>

            <!-- Loading State -->
            <div 
              v-else-if="isRunning" 
              class="h-full min-h-[400px] rounded-lg bg-muted/30 border border-dashed border-muted-foreground/20 flex flex-col items-center justify-center"
            >
              <Loader2 class="w-8 h-8 text-primary animate-spin mb-3" />
              <p class="text-sm font-medium text-muted-foreground">Processing request...</p>
              <p class="text-xs text-muted-foreground/70 mt-1 capitalize">Status: {{ resultStatus }}</p>
            </div>

            <!-- Result Content -->
            <div v-else-if="hasResult" class="h-full min-h-[400px]">
              <!-- HTML Output -->
              <div v-if="selectedOutputFormat === 'html'" class="h-full">
                <div class="h-full rounded-lg bg-muted/30 border p-4 overflow-auto">
                  <pre class="text-xs font-mono text-foreground/80 whitespace-pre-wrap">{{ resultContent }}</pre>
                </div>
              </div>

              <!-- Markdown Output -->
              <div v-if="selectedOutputFormat === 'markdown'" class="h-full">
                <div class="h-full rounded-lg bg-muted/30 border p-4 overflow-auto">
                  <pre class="text-xs font-mono text-foreground/80 whitespace-pre-wrap">{{ resultContent }}</pre>
                </div>
              </div>

              <!-- Screenshot Output -->
              <div v-if="selectedOutputFormat === 'screenshot'" class="h-full">
                <div class="h-full rounded-lg bg-muted/30 border p-4 flex items-center justify-center">
                  <div v-if="resultScreenshot" class="w-full">
                    <img :src="`data:image/png;base64,${resultScreenshot}`" alt="Screenshot" class="max-w-full rounded-lg border" />
                  </div>
                  <div v-else class="text-center">
                    <div class="w-full max-w-md mx-auto rounded-lg border-2 border-dashed border-muted-foreground/20 p-8">
                      <ImageIcon class="w-12 h-12 text-muted-foreground/50 mx-auto mb-3" />
                      <p class="text-sm text-muted-foreground">No screenshot available</p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
