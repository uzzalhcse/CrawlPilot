<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
    <div class="bg-background border rounded-lg w-full max-w-2xl max-h-[85vh] overflow-hidden flex flex-col">
      <div class="px-6 py-4 border-b flex justify-between items-center shrink-0">
        <div>
          <h2 class="text-lg font-semibold">{{ isEditMode ? 'Edit Rule' : 'Create Rule' }}</h2>
          <p class="text-xs text-muted-foreground mt-0.5">Configure error pattern matching and recovery action</p>
        </div>
        <Button @click="$emit('close')" size="sm" variant="ghost" class="h-8 w-8 p-0">
          <X class="w-4 h-4" />
        </Button>
      </div>

      <div class="flex-1 overflow-y-auto p-6 space-y-6">
        <!-- Basic Info -->
        <div class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium mb-1.5">Name *</label>
              <input
                v-model="formData.name"
                type="text"
                class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
                placeholder="my_custom_rule"
              />
            </div>
            <div>
              <label class="block text-sm font-medium mb-1.5">Priority (lower = higher)</label>
              <input
                v-model.number="formData.priority"
                type="number"
                min="1"
                max="100"
                class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
              />
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium mb-1.5">Description</label>
            <input
              v-model="formData.description"
              type="text"
              class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
              placeholder="What does this rule do?"
            />
          </div>
        </div>

        <!-- Pattern Matching -->
        <div class="bg-card border rounded-lg p-4 space-y-4">
          <h3 class="text-sm font-semibold flex items-center gap-2">
            <AlertTriangle class="w-4 h-4 text-orange-500" />
            Error Pattern
          </h3>
          <p class="text-xs text-muted-foreground">Define what kind of error triggers this rule.</p>
          
          <div>
            <label class="block text-sm font-medium mb-1.5">Pattern Type *</label>
            <select
              v-model="formData.pattern"
              class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
            >
              <option value="blocked">🚫 Blocked / IP Ban (403, Access Denied)</option>
              <option value="rate_limited">⏱️ Rate Limited (429, Too Many Requests)</option>
              <option value="captcha">🔒 Captcha Detected (Recaptcha, hCaptcha)</option>
              <option value="timeout">⌛ Timeout (Request took too long)</option>
              <option value="connection_error">🔌 Connection Error (Reset, Refused)</option>
              <option value="server_error">💥 Server Error (500, 502, 503)</option>
              <option value="layout_changed">📐 Layout Changed (Selectors failed)</option>
              <option value="auth_required">🔐 Auth Required (401, Login page)</option>
              <option value="not_found">🔍 Not Found (404)</option>
              <option value="unknown">❓ Unknown Error (Fallback)</option>
            </select>
          </div>

          <!-- Conditions -->
          <div class="space-y-3">
            <label class="block text-sm font-medium">Additional Conditions (optional)</label>
            <p class="text-xs text-muted-foreground">Refine when this rule applies. Leave empty to apply to all errors of this type.</p>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-xs text-muted-foreground mb-1">Domain</label>
                <input
                  v-model="conditions.domain"
                  type="text"
                  class="w-full px-3 py-2 text-sm bg-background border rounded-md"
                  placeholder="e.g. amazon.com"
                />
                <p class="text-[10px] text-muted-foreground mt-1">Matches specific domain</p>
              </div>
              <div>
                <label class="block text-xs text-muted-foreground mb-1">Status Code</label>
                <input
                  v-model.number="conditions.status_code"
                  type="number"
                  min="100"
                  max="599"
                  class="w-full px-3 py-2 text-sm bg-background border rounded-md"
                  placeholder="e.g. 403"
                />
                <p class="text-[10px] text-muted-foreground mt-1">Exact HTTP status match</p>
              </div>
              <div>
                <label class="block text-xs text-muted-foreground mb-1">URL Pattern</label>
                <input
                  v-model="conditions.url_pattern"
                  type="text"
                  class="w-full px-3 py-2 text-sm bg-background border rounded-md"
                  placeholder="e.g. /product/*"
                />
                <p class="text-[10px] text-muted-foreground mt-1">Matches URL path</p>
              </div>
              <div>
                <label class="block text-xs text-muted-foreground mb-1">Error Contains</label>
                <input
                  v-model="conditions.error_contains"
                  type="text"
                  class="w-full px-3 py-2 text-sm bg-background border rounded-md"
                  placeholder="e.g. 'captcha'"
                />
                <p class="text-[10px] text-muted-foreground mt-1">Text found in error message</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Recovery Action -->
        <div class="bg-card border rounded-lg p-4 space-y-4">
          <h3 class="text-sm font-semibold flex items-center gap-2">
            <Zap class="w-4 h-4 text-blue-500" />
            Recovery Action
          </h3>
          <p class="text-xs text-muted-foreground">What should the system do when this error occurs?</p>
          
          <div>
            <label class="block text-sm font-medium mb-1.5">Action Type *</label>
            <select
              v-model="formData.action"
              class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
            >
              <option value="switch_proxy">🔄 Switch Proxy (Try a different IP)</option>
              <option value="add_delay">⏳ Add Delay (Wait before retrying)</option>
              <option value="retry">🔁 Retry (Immediate retry)</option>
              <option value="retry_with_browser">🌐 Retry with Browser (If using HTTP client)</option>
              <option value="rotate_user_agent">👤 Rotate User Agent (Change browser fingerprint)</option>
              <option value="clear_cookies">🍪 Clear Cookies (Reset session)</option>
              <option value="skip_domain">⏭️ Skip Domain (Stop crawling this site temporarily)</option>
              <option value="send_to_dlq">📨 Send to Dead Letter Queue (Manual review)</option>
            </select>
          </div>

          <!-- Dynamic Action Parameters -->
          <div v-if="formData.action === 'add_delay'">
            <label class="block text-sm font-medium mb-1.5">Delay Seconds</label>
            <input
              v-model.number="actionParams.seconds"
              type="number"
              min="1"
              class="w-full px-3 py-2 text-sm bg-background border rounded-md"
              placeholder="30"
            />
            <p class="text-xs text-muted-foreground mt-1">How long to wait before the next attempt.</p>
          </div>

          <div v-if="formData.action === 'skip_domain'">
            <label class="block text-sm font-medium mb-1.5">Block Duration (seconds)</label>
            <input
              v-model.number="actionParams.block_duration"
              type="number"
              min="60"
              class="w-full px-3 py-2 text-sm bg-background border rounded-md"
              placeholder="3600"
            />
            <p class="text-xs text-muted-foreground mt-1">Prevent any requests to this domain for this duration (e.g. 3600s = 1 hour).</p>
          </div>

          <div v-if="formData.action === 'send_to_dlq'">
            <label class="block text-sm font-medium mb-1.5">DLQ Category</label>
            <select
              v-model="actionParams.category"
              class="w-full px-3 py-2 text-sm bg-background border rounded-md"
            >
              <option value="captcha">Captcha (Needs solving)</option>
              <option value="blocked">Blocked (Hard block)</option>
              <option value="auth_required">Auth Required (Login needed)</option>
              <option value="manual_review">Manual Review (Unknown issue)</option>
            </select>
            <p class="text-xs text-muted-foreground mt-1">Categorize this failure for the manual review dashboard.</p>
          </div>

          <div>
            <label class="block text-sm font-medium mb-1.5">Reason (for logging)</label>
            <input
              v-model="actionParams.reason"
              type="text"
              class="w-full px-3 py-2 text-sm bg-background border rounded-md"
              placeholder="e.g. IP blocked by target site"
            />
          </div>
        </div>

        <!-- Retry Settings -->
        <div class="grid grid-cols-3 gap-4">
          <div>
            <label class="block text-sm font-medium mb-1.5">Max Retries</label>
            <input
              v-model.number="formData.max_retries"
              type="number"
              min="1"
              max="10"
              class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
            />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">Retry Delay (sec)</label>
            <input
              v-model.number="formData.retry_delay"
              type="number"
              min="0"
              class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
            />
          </div>
          <div class="flex items-end pb-2">
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                v-model="formData.enabled"
                type="checkbox"
                class="w-4 h-4 rounded border-gray-300"
              />
              <span class="text-sm font-medium">Enabled</span>
            </label>
          </div>
        </div>
      </div>

      <div class="px-6 py-4 border-t flex justify-end gap-2 shrink-0">
        <Button @click="$emit('close')" variant="outline" size="sm">
          Cancel
        </Button>
        <Button @click="handleSave" variant="default" size="sm" :disabled="!isValid">
          {{ isEditMode ? 'Update' : 'Create' }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, computed } from 'vue'
import type { ContextAwareRule } from '@/stores/errorRecovery'
import { Button } from '@/components/ui/button'
import { X, AlertTriangle, Zap } from 'lucide-vue-next'

const props = defineProps<{
  rule: ContextAwareRule | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save', rule: Partial<ContextAwareRule>): void
}>()

const isEditMode = computed(() => !!props.rule)

const formData = reactive({
  name: props.rule?.name || '',
  description: props.rule?.description || '',
  priority: props.rule?.priority || 100,
  enabled: props.rule?.enabled ?? true,
  pattern: props.rule?.pattern || 'blocked',
  action: props.rule?.action || 'switch_proxy',
  max_retries: props.rule?.max_retries || 3,
  retry_delay: props.rule?.retry_delay || 5,
})

// Conditions - matches backend RuleCondition fields
const conditions = reactive({
  domain: (props.rule?.conditions as any)?.domain || '',
  status_code: (props.rule?.conditions as any)?.status_code || null,
  url_pattern: (props.rule?.conditions as any)?.url_pattern || '',
  error_contains: (props.rule?.conditions as any)?.error_contains || '',
})

// Action params 
const actionParams = reactive({
  seconds: (props.rule?.action_params as any)?.seconds || 30,
  block_duration: (props.rule?.action_params as any)?.block_duration || 3600,
  category: (props.rule?.action_params as any)?.category || 'manual_review',
  reason: (props.rule?.action_params as any)?.reason || '',
})

const isValid = computed(() => {
  return formData.name && formData.pattern && formData.action
})

function handleSave() {
  // Build conditions array matching backend RuleCondition struct
  const conditionsArr: Array<{ field: string; operator: string; value: string }> = []
  if (conditions.domain) {
    conditionsArr.push({ field: 'domain', operator: 'contains', value: conditions.domain })
  }
  if (conditions.status_code) {
    conditionsArr.push({ field: 'status_code', operator: 'equals', value: String(conditions.status_code) })
  }
  if (conditions.url_pattern) {
    conditionsArr.push({ field: 'url_pattern', operator: 'contains', value: conditions.url_pattern })
  }
  if (conditions.error_contains) {
    conditionsArr.push({ field: 'error_contains', operator: 'contains', value: conditions.error_contains })
  }

  // Build action_params object based on action type
  const actionParamsObj: Record<string, any> = {}
  if (actionParams.reason) actionParamsObj.reason = actionParams.reason
  
  if (formData.action === 'add_delay') {
    actionParamsObj.seconds = actionParams.seconds
  }
  if (formData.action === 'skip_domain') {
    actionParamsObj.block_duration = actionParams.block_duration
  }
  if (formData.action === 'send_to_dlq') {
    actionParamsObj.category = actionParams.category
  }

  const payload: Partial<ContextAwareRule> = {
    name: formData.name,
    description: formData.description,
    priority: formData.priority,
    enabled: formData.enabled,
    pattern: formData.pattern,
    conditions: conditionsArr as any,
    action: formData.action,
    action_params: actionParamsObj,
    max_retries: formData.max_retries,
    retry_delay: formData.retry_delay,
  }

  emit('save', payload)
}
</script>
