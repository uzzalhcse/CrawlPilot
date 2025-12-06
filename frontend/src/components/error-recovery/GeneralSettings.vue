<template>
  <div class="space-y-6">
    <div class="bg-card border rounded-lg p-6">
      <h3 class="text-sm font-semibold mb-4">Smart Triggering</h3>
      <p class="text-xs text-muted-foreground mb-4">
        Configure when the recovery system kicks in. These settings control the sensitivity of error detection.
      </p>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Window Size</label>
          <input
            v-model.number="config['recovery.window_size']"
            type="number"
            min="10"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          />
          <p class="text-[10px] text-muted-foreground mt-1">
            How many recent requests to analyze per domain. <br/>
            <span class="italic">Example: 100 means look at the last 100 requests.</span>
          </p>
        </div>
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Error Rate Threshold</label>
          <input
            v-model.number="config['recovery.error_rate_threshold']"
            type="number"
            step="0.01"
            min="0"
            max="1"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          />
          <p class="text-[10px] text-muted-foreground mt-1">
            Trigger recovery if errors exceed this rate. <br/>
            <span class="italic">Example: 0.10 means trigger if >10% of requests fail.</span>
          </p>
        </div>
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Consecutive Errors</label>
          <input
            v-model.number="config['recovery.consecutive_threshold']"
            type="number"
            min="1"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          />
          <p class="text-[10px] text-muted-foreground mt-1">
            Trigger immediately after N failures in a row. <br/>
            <span class="italic">Example: 3 means trigger on the 3rd consecutive error.</span>
          </p>
        </div>
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Max Attempts</label>
          <input
            v-model.number="config['recovery.max_attempts']"
            type="number"
            min="1"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          />
          <p class="text-[10px] text-muted-foreground mt-1">
            Max recovery tries per task before giving up. <br/>
            <span class="italic">Example: 3 means try to recover 3 times, then send to DLQ.</span>
          </p>
        </div>
      </div>
    </div>

    <div class="bg-card border rounded-lg p-6">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h3 class="text-sm font-semibold">AI Reasoning (Fallback)</h3>
          <p class="text-xs text-muted-foreground mt-1">
            Use an LLM to analyze unknown errors and suggest fixes when no rules match.
          </p>
        </div>
        <label class="relative inline-flex items-center cursor-pointer">
          <input
            v-model="config['ai.enabled']"
            type="checkbox"
            class="sr-only peer"
          />
          <div class="w-9 h-5 bg-muted peer-focus:ring-2 peer-focus:ring-primary rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-background after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-primary"></div>
        </label>
      </div>
      
      <div v-if="config['ai.enabled']" class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Provider</label>
          <select
            v-model="config['ai.provider']"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          >
            <option value="gemini">Gemini (Google)</option>
            <option value="ollama">Ollama (Local)</option>
            <option value="openai">OpenAI (GPT-4)</option>
          </select>
          <p class="text-[10px] text-muted-foreground mt-1">Select the AI model provider.</p>
        </div>
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Model</label>
          <input
            v-model="config['ai.model']"
            type="text"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
            placeholder="gemini-2.0-flash"
          />
          <p class="text-[10px] text-muted-foreground mt-1">Specific model name (e.g. gpt-4o, llama3).</p>
        </div>
        <div>
            <label class="block text-xs font-medium text-muted-foreground mb-1.5">Endpoint (Optional)</label>
            <input
              v-model="config['ai.endpoint']"
              type="text"
              class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
              placeholder="http://localhost:11434"
            />
            <p class="text-[10px] text-muted-foreground mt-1">Custom API URL (required for Ollama).</p>
          </div>
          <div>
            <label class="block text-xs font-medium text-muted-foreground mb-1.5">Timeout (sec)</label>
            <input
              v-model.number="config['ai.timeout']"
              type="number"
              min="1"
              class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
            />
            <p class="text-[10px] text-muted-foreground mt-1">Max time to wait for AI response.</p>
          </div>
      </div>
    </div>

    <div class="bg-card border rounded-lg p-6">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h3 class="text-sm font-semibold">Learning System</h3>
          <p class="text-xs text-muted-foreground mt-1">
            Automatically create new rules from successful AI recovery actions.
          </p>
        </div>
        <label class="relative inline-flex items-center cursor-pointer">
          <input
            v-model="config['learning.enabled']"
            type="checkbox"
            class="sr-only peer"
          />
          <div class="w-9 h-5 bg-muted peer-focus:ring-2 peer-focus:ring-primary rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-background after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-primary"></div>
        </label>
      </div>
      
      <div v-if="config['learning.enabled']" class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Promotion Threshold</label>
          <input
            v-model.number="config['learning.promotion_threshold']"
            type="number"
            min="1"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          />
          <p class="text-[10px] text-muted-foreground mt-1">
            Promote to permanent rule after N successful uses. <br/>
            <span class="italic">Example: 3 means if AI fixes it the same way 3 times, make it a rule.</span>
          </p>
        </div>
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Cleanup Days</label>
          <input
            v-model.number="config['learning.cleanup_days']"
            type="number"
            min="1"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          />
          <p class="text-[10px] text-muted-foreground mt-1">Delete unused learned rules after N days.</p>
        </div>
      </div>
    </div>

    <div class="flex justify-end">
      <Button @click="saveConfig" variant="default" size="sm" :disabled="loading">
        <Save class="w-4 h-4 mr-2" />
        {{ loading ? 'Saving...' : 'Save Settings' }}
      </Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, onMounted, ref } from 'vue'
import { useErrorRecoveryStore } from '@/stores/errorRecovery'
import { Button } from '@/components/ui/button'
import { Save } from 'lucide-vue-next'

const store = useErrorRecoveryStore()
const loading = ref(false)

// Initialize with defaults matching backend migration
const config = reactive<Record<string, any>>({
  'recovery.window_size': 100,
  'recovery.error_rate_threshold': 0.10,
  'recovery.consecutive_threshold': 3,
  'recovery.max_attempts': 3,
  'ai.enabled': true,
  'ai.provider': 'ollama',
  'ai.model': 'qwen2.5',
  'ai.endpoint': 'http://localhost:11434',
  'ai.timeout': 30,
  'learning.enabled': true,
  'learning.promotion_threshold': 3,
  'learning.cleanup_days': 7,
})

async function saveConfig() {
  loading.value = true
  try {
    // Convert boolean strings back to booleans if needed, or ensure correct types
    // The v-model handles types for inputs, but let's be safe
    await store.updateMultipleConfigs(config)
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    const fetchedConfig = await store.fetchAllConfigs()
    if (fetchedConfig) {
      // Merge fetched config into reactive state
      Object.keys(fetchedConfig).forEach(key => {
        if (key in config) {
            // Handle boolean conversion if backend returns strings "true"/"false"
            let val = fetchedConfig[key]
            if (val === 'true') val = true
            if (val === 'false') val = false
            // Handle number conversion
            if (!isNaN(Number(val)) && typeof val === 'string' && val.trim() !== '') {
                // Check if it should be a number (based on default config type)
                if (typeof config[key] === 'number') {
                    val = Number(val)
                }
            }
            config[key] = val
        }
      })
    }
  } catch (error) {
    console.error('Failed to load configs', error)
  }
})
</script>
