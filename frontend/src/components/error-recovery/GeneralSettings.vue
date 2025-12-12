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

    <div class="bg-card border rounded-lg p-6">
      <h3 class="text-sm font-semibold mb-4">Tiered Proxy System</h3>
      <p class="text-xs text-muted-foreground mb-4">
        Configure how the system escalates through proxy tiers when requests are blocked. 
        Tier 0 = Direct, Tier 1 = Datacenter, Tier 2 = Residential, Tier 3 = Mobile.
      </p>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Escalate After Failures</label>
          <input
            v-model.number="config['tiered_proxy.escalate_after_failures']"
            type="number"
            min="1"
            max="10"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          />
          <p class="text-[10px] text-muted-foreground mt-1">
            Number of failures before escalating to next proxy tier. <br/>
            <span class="italic">Example: 2 means after 2 failures on Tier 0, try Tier 1.</span>
          </p>
        </div>
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Tier Cooldown (min)</label>
          <input
            v-model.number="config['tiered_proxy.tier_cooldown_minutes']"
            type="number"
            min="1"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          />
          <p class="text-[10px] text-muted-foreground mt-1">
            Time to wait before retrying a lower tier. <br/>
            <span class="italic">Example: 10 means wait 10 minutes before trying Tier 0 again.</span>
          </p>
        </div>
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Min Samples for Confidence</label>
          <input
            v-model.number="config['tiered_proxy.min_samples_for_confidence']"
            type="number"
            min="5"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          />
          <p class="text-[10px] text-muted-foreground mt-1">
            Minimum requests before considering a tier stable for a domain.
          </p>
        </div>
        <div>
          <label class="block text-xs font-medium text-muted-foreground mb-1.5">Success Rate Threshold</label>
          <input
            v-model.number="config['tiered_proxy.success_rate_threshold']"
            type="number"
            step="0.05"
            min="0.5"
            max="1"
            class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
          />
          <p class="text-[10px] text-muted-foreground mt-1">
            Success rate to consider a tier working for a domain. <br/>
            <span class="italic">Example: 0.8 means 80% success rate is acceptable.</span>
          </p>
        </div>
      </div>
    </div>

    <!-- Protected Domains section removed - now managed via domain_strategies table -->
    <!-- TODO: Create a dedicated "Domain Strategies" page to view/manage learned domain tiers -->

    <div class="flex justify-end">
      <Button @click="saveConfig" variant="default" size="sm" :disabled="loading">
        <Save class="w-4 h-4 mr-2" />
        {{ loading ? 'Saving...' : 'Save Settings' }}
      </Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, onMounted, ref, computed } from 'vue'
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
  // Tiered proxy settings
  'tiered_proxy.escalate_after_failures': 2,
  'tiered_proxy.tier_cooldown_minutes': 10,
  'tiered_proxy.min_samples_for_confidence': 20,
  'tiered_proxy.success_rate_threshold': 0.8,
})

async function saveConfig() {
  loading.value = true
  try {
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
