<script setup lang="ts">
import { ref, watch } from 'vue'
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
  Loader2
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'

// State
const url = ref('https://example.com/products')
const advancedSettingsOpen = ref(false)
const selectedOutputFormat = ref('html')
const isRunning = ref(false)
const hasResult = ref(false)

// Driver & Profile selection
const selectedDriver = ref('camoufox')
const selectedProfile = ref('')

const drivers = [
  { id: 'camoufox', label: 'Camoufox (Stealth)' },
  { id: 'playwright', label: 'Playwright' },
  { id: 'puppeteer', label: 'Puppeteer' },
  { id: 'http', label: 'HTTP Client (Fast)' },
]

const profiles = [
  { id: '', label: 'No Profile' },
  { id: 'profile-1', label: 'Default Profile' },
  { id: 'profile-2', label: 'US Residential' },
  { id: 'profile-3', label: 'EU Mobile' },
]

// Sample result data for demonstration
const sampleHtml = ref('<html><head><title>Sample Page</title></head><body><h1>Hello World</h1><p>This is sample content.</p></body></html>')
const sampleMarkdown = ref('# Hello World\n\nThis is sample content extracted from the page.\n\n## Products\n- Product 1: $29.99\n- Product 2: $49.99')

const outputFormats = [
  { id: 'html', label: 'HTML', icon: FileText },
  { id: 'markdown', label: 'Markdown', icon: FileCode },
  { id: 'screenshot', label: 'Screenshot', icon: ImageIcon },
]

// Sync preview with output format
watch(selectedOutputFormat, (newFormat) => {
  // Preview automatically shows selected format
})

const runScraper = async () => {
  isRunning.value = true
  // Simulate API call
  await new Promise(resolve => setTimeout(resolve, 1500))
  hasResult.value = true
  isRunning.value = false
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

          <!-- AI Unblocker Banner -->
          <div class="rounded-xl border border-emerald-500/30 bg-emerald-500/5 p-4 flex items-center gap-3">
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
                    <option v-for="driver in drivers" :key="driver.id" :value="driver.id">
                      {{ driver.label }}
                    </option>
                  </select>
                  <ChevronDown class="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
                </div>
              </div>

              <!-- Profile Selection -->
              <div>
                <label class="text-xs font-medium text-muted-foreground mb-2 block">Profile</label>
                <div class="relative">
                  <select
                    v-model="selectedProfile"
                    class="w-full px-4 py-2.5 text-sm bg-background border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary transition-colors appearance-none cursor-pointer"
                  >
                    <option v-for="profile in profiles" :key="profile.id" :value="profile.id">
                      {{ profile.label }}
                    </option>
                  </select>
                  <ChevronDown class="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
                </div>
                <p class="text-xs text-muted-foreground mt-2 flex items-start gap-1.5">
                  <User class="w-3 h-3 mt-0.5 shrink-0" />
                  <span>Select a browser profile to run the scraper with (cookies, fingerprint, and settings will load automatically).</span>
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
              <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Advanced Settings</span>
              <component 
                :is="advancedSettingsOpen ? ChevronDown : ChevronRight" 
                class="w-4 h-4 text-muted-foreground" 
              />
            </button>
            <div v-if="advancedSettingsOpen" class="px-5 pb-5 space-y-4 border-t">
              <div class="pt-4">
                <label class="text-xs font-medium text-muted-foreground">Timeout (seconds)</label>
                <input
                  type="number"
                  value="30"
                  class="w-full mt-2 px-3 py-2 text-sm bg-background border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/50"
                />
              </div>
              <div>
                <label class="text-xs font-medium text-muted-foreground">Wait For Selector</label>
                <input
                  type="text"
                  placeholder=".product-list, #main-content"
                  class="w-full mt-2 px-3 py-2 text-sm bg-background border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/50"
                />
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
          <div class="p-5 border-b">
            <h2 class="text-sm font-semibold">Output Preview</h2>
            <p class="text-xs text-muted-foreground mt-0.5">Your result will appear here after running a request</p>
          </div>

          <!-- Content Area -->
          <div class="flex-1 p-5">
            <!-- Empty State -->
            <div 
              v-if="!hasResult" 
              class="h-full min-h-[400px] rounded-lg bg-muted/30 border border-dashed border-muted-foreground/20 flex flex-col items-center justify-center"
            >
              <div class="p-4 rounded-full bg-muted/50 mb-3">
                <component :is="outputFormats.find(f => f.id === selectedOutputFormat)?.icon || FileText" class="w-8 h-8 text-muted-foreground/50" />
              </div>
              <p class="text-sm font-medium text-muted-foreground">No preview available</p>
              <p class="text-xs text-muted-foreground/70 mt-1">Run a request to see the {{ selectedOutputFormat.toUpperCase() }} output here</p>
            </div>

            <!-- Result Content -->
            <div v-else class="h-full min-h-[400px]">
              <!-- HTML Output -->
              <div v-if="selectedOutputFormat === 'html'" class="h-full">
                <div class="h-full rounded-lg bg-muted/30 border p-4 overflow-auto">
                  <pre class="text-xs font-mono text-foreground/80 whitespace-pre-wrap">{{ sampleHtml }}</pre>
                </div>
              </div>

              <!-- Markdown Output -->
              <div v-if="selectedOutputFormat === 'markdown'" class="h-full">
                <div class="h-full rounded-lg bg-muted/30 border p-4 overflow-auto">
                  <pre class="text-xs font-mono text-foreground/80 whitespace-pre-wrap">{{ sampleMarkdown }}</pre>
                </div>
              </div>

              <!-- Screenshot Output -->
              <div v-if="selectedOutputFormat === 'screenshot'" class="h-full">
                <div class="h-full rounded-lg bg-muted/30 border p-4 flex items-center justify-center">
                  <div class="text-center">
                    <div class="w-full max-w-md mx-auto rounded-lg border-2 border-dashed border-muted-foreground/20 p-8">
                      <ImageIcon class="w-12 h-12 text-muted-foreground/50 mx-auto mb-3" />
                      <p class="text-sm text-muted-foreground">Screenshot preview</p>
                      <p class="text-xs text-muted-foreground/70 mt-1">1920x1080px</p>
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

