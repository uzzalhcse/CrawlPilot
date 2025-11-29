<template>
  <div class="space-y-6">
    <div class="bg-card border rounded-lg p-6">
      <h3 class="text-sm font-semibold mb-4">Pattern Analyzer</h3>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Window Size</label>
          <input
            v-model.number="config.window_size"
            type="number"
            min="10"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          />
          <p class="text-xs text-muted-foreground mt-1">Requests to analyze</p>
        </div>
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Error Rate Threshold</label>
          <input
            v-model.number="config.error_rate_threshold"
            type="number"
            step="0.01"
            min="0"
            max="1"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          />
          <p class="text-xs text-muted-foreground mt-1">0.10 = 10% error rate</p>
        </div>
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Consecutive Errors</label>
          <input
            v-model.number="config.consecutive_limit"
            type="number"
            min="1"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          />
          <p class="text-xs text-muted-foreground mt-1">Activate after N consecutive</p>
        </div>
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Same Error Threshold</label>
          <input
            v-model.number="config.same_error_threshold"
            type="number"
            min="1"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          />
          <p class="text-xs text-muted-foreground mt-1">Activate when repeats N times</p>
        </div>
      </div>
    </div>

    <div class="bg-card border rounded-lg p-6">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-sm font-semibold">AI Reasoning (Fallback)</h3>
        <label class="relative inline-flex items-center cursor-pointer">
          <input
            v-model="config.ai_enabled"
            type="checkbox"
            class="sr-only peer"
          />
          <div class="w-9 h-5 bg-muted peer-focus:ring-2 peer-focus:ring-primary rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-background after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-primary"></div>
        </label>
      </div>
      
      <div v-if="config.ai_enabled" class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Provider</label>
          <select
            v-model="config.ai_provider"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          >
            <option value="gemini">Gemini</option>
            <option value="openrouter">OpenRouter</option>
          </select>
        </div>
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Model</label>
          <input
            v-model="config.ai_model"
            type="text"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
            placeholder="gemini-2.0-flash"
          />
        </div>
      </div>
    </div>

    <div class="bg-card border rounded-lg p-6">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-sm font-semibold">Learning System</h3>
        <label class="relative inline-flex items-center cursor-pointer">
          <input
            v-model="config.learning_enabled"
            type="checkbox"
            class="sr-only peer"
          />
          <div class="w-9 h-5 bg-muted peer-focus:ring-2 peer-focus:ring-primary rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-background after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-primary"></div>
        </label>
      </div>
      
      <p class="text-xs text-muted-foreground mb-4">Auto-create rules from successful AI solutions</p>
      
      <div v-if="config.learning_enabled" class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Min Success Rate</label>
          <input
            v-model.number="config.min_success_rate"
            type="number"
            step="0.01"
            min="0"
            max="1"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          />
          <p class="text-xs text-muted-foreground mt-1">0.90 = 90% success required</p>
        </div>
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Min Usage Count</label>
          <input
            v-model.number="config.min_usage_count"
            type="number"
            min="1"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          />
          <p class="text-xs text-muted-foreground mt-1">Create after N successful uses</p>
        </div>
      </div>
    </div>

    <div class="flex justify-end">
      <Button @click="saveConfig" variant="default" size="sm">
        <Save class="w-4 h-4 mr-2" />
        Save Settings
      </Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, onMounted } from 'vue'
import { useErrorRecoveryStore } from '@/stores/errorRecovery'
import { Button } from '@/components/ui/button'
import { Save } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const store = useErrorRecoveryStore()

const config = reactive({
  window_size: 100,
  error_rate_threshold: 0.10,
  consecutive_limit: 5,
  same_error_threshold: 10,
  ai_enabled: true,
  ai_provider: 'gemini',
  ai_model: 'gemini-2.0-flash',
  learning_enabled: true,
  min_success_rate: 0.90,
  min_usage_count: 5,
})

async function saveConfig() {
  try {
    await store.updateConfig('analyzer', {
      window_size: config.window_size,
      error_rate_threshold: config.error_rate_threshold,
      consecutive_limit: config.consecutive_limit,
      same_error_threshold: config.same_error_threshold,
    })
    
    await store.updateConfig('ai', {
      enabled: config.ai_enabled,
      provider: config.ai_provider,
      model: config.ai_model,
    })
    
    await store.updateConfig('learning', {
      enabled: config.learning_enabled,
      min_success_rate: config.min_success_rate,
      min_usage_count: config.min_usage_count,
    })
  } catch (error) {
    // Toast shown by store
  }
}

onMounted(async () => {
  try {
    const analyzer = await store.fetchConfig('analyzer')
    if (analyzer) Object.assign(config, analyzer)
    
    const ai = await store.fetchConfig('ai')
    if (ai) {
      config.ai_enabled = ai.enabled
      config.ai_provider = ai.provider
      config.ai_model = ai.model
    }
    
    const learning = await store.fetchConfig('learning')
    if (learning) {
      config.learning_enabled = learning.enabled
      config.min_success_rate = learning.min_success_rate
      config.min_usage_count = learning.min_usage_count
    }
  } catch (error) {
    // Config not found, use defaults
  }
})
</script>
