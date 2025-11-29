<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
    <div class="bg-background border rounded-lg w-full max-w-3xl max-h-[85vh] overflow-hidden flex flex-col">
      <div class="px-6 py-4 border-b flex justify-between items-center shrink-0">
        <div>
          <h2 class="text-lg font-semibold">{{ isEditMode ? 'Edit Rule' : 'Create Rule' }}</h2>
          <p class="text-xs text-muted-foreground mt-0.5">Configure error matching and recovery actions</p>
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
              <label class="block text-sm font-medium mb-1.5">Name</label>
              <input
                v-model="formData.name"
                type="text"
                class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
                placeholder="my_custom_rule"
              />
            </div>
            <div>
              <label class="block text-sm font-medium mb-1.5">Priority</label>
              <input
                v-model.number="formData.priority"
                type="number"
                min="1"
                max="10"
                class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
              />
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium mb-1.5">Description</label>
            <textarea
              v-model="formData.description"
              rows="2"
              class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary resize-none"
              placeholder="What does this rule do?"
            />
          </div>
        </div>

        <!-- Conditions -->
        <div>
          <div class="flex justify-between items-center mb-3">
            <label class="text-sm font-medium">Conditions</label>
            <Button @click="addCondition" size="sm" variant="outline" class="h-8">
              <Plus class="w-3 h-3 mr-1" />
              Add
            </Button>
          </div>
          <div class="space-y-2">
            <div
              v-for="(condition, idx) in formData.conditions"
              :key="idx"
              class="flex items-center gap-2 p-3 bg-card border rounded-md"
            >
              <select
                v-model="condition.field"
                class="flex-1 px-2 py-1.5 text-sm bg-background border rounded"
              >
                <option value="error_type">Error Type</option>
                <option value="status_code">Status Code</option>
                <option value="domain">Domain</option>
                <option value="response_body">Response Body</option>
              </select>
              <select
                v-model="condition.operator"
                class="flex-1 px-2 py-1.5 text-sm bg-background border rounded"
              >
                <option value="equals">Equals</option>
                <option value="contains">Contains</option>
                <option value="regex">Regex</option>
                <option value="gt">{">"}</option>
                <option value="lt">{"<"}</option>
              </select>
              <input
                v-model="condition.value"
                type="text"
                class="flex-1 px-2 py-1.5 text-sm bg-background border rounded"
                placeholder="Value"
              />
              <Button
                @click="removeCondition(idx)"
                size="sm"
                variant="ghost"
                class="h-8 w-8 p-0 text-destructive hover:text-destructive"
              >
                <X class="w-4 h-4" />
              </Button>
            </div>
          </div>
        </div>

        <!-- Actions -->
        <div>
          <div class="flex justify-between items-center mb-3">
            <label class="text-sm font-medium">Actions</label>
            <Button @click="addAction" size="sm" variant="outline" class="h-8">
              <Plus class="w-3 h-3 mr-1" />
              Add
            </Button>
          </div>
          <div class="space-y-2">
            <div
              v-for="(action, idx) in formData.actions"
              :key="idx"
              class="p-3 bg-card border rounded-md space-y-2"
            >
              <div class="flex items-center gap-2">
                <select
                  v-model="action.type"
                  class="flex-1 px-2 py-1.5 text-sm bg-background border rounded"
                >
                  <option value="enable_stealth">Enable Stealth</option>
                  <option value="rotate_proxy">Rotate Proxy</option>
                  <option value="wait">Wait</option>
                  <option value="reduce_workers">Reduce Workers</option>
                  <option value="add_delay">Add Delay</option>
                  <option value="adjust_timeout">Adjust Timeout</option>
                </select>
                <Button
                  @click="removeAction(idx)"
                  size="sm"
                  variant="ghost"
                  class="h-8 w-8 p-0 text-destructive hover:text-destructive"
                >
                  <X class="w-4 h-4" />
                </Button>
              </div>
              <input
                v-model="action.parametersJson"
                type="text"
                class="w-full px-2 py-1.5 text-sm bg-background border rounded font-mono"
                placeholder='{"duration": 10}'
              />
            </div>
          </div>
        </div>

        <!-- Context -->
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium mb-1.5">Domain Pattern</label>
            <input
              v-model="formData.context.domain_pattern"
              type="text"
              class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
              placeholder="*.example.com or *"
            />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">Max Retries</label>
            <input
              v-model.number="formData.context.max_retries"
              type="number"
              min="1"
              class="w-full px-3 py-2 text-sm bg-background border rounded-md focus:ring-1 focus:ring-primary focus:border-primary"
            />
          </div>
        </div>
      </div>

      <div class="px-6 py-4 border-t flex justify-end gap-2 shrink-0">
        <Button @click="$emit('close')" variant="outline" size="sm">
          Cancel
        </Button>
        <Button @click="handleSave" variant="default" size="sm">
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
import { Plus, X } from 'lucide-vue-next'

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
  priority: props.rule?.priority || 5,
  conditions: props.rule?.conditions || [],
  actions: props.rule?.actions?.map(a => ({
    ...a,
    parametersJson: JSON.stringify(a.parameters),
  })) || [],
  context: {
    domain_pattern: props.rule?.context?.domain_pattern || '*',
    variables: props.rule?.context?.variables || {},
    max_retries: props.rule?.context?.max_retries || 3,
  },
  confidence: props.rule?.confidence || 0.8,
  created_by: props.rule?.created_by || 'custom',
})

function addCondition() {
  formData.conditions.push({
    field: 'status_code',
    operator: 'equals',
    value: '',
  })
}

function removeCondition(idx: number) {
  formData.conditions.splice(idx, 1)
}

function addAction() {
  formData.actions.push({
    type: 'wait',
    parametersJson: '{"duration": 10}',
  })
}

function removeAction(idx: number) {
  formData.actions.splice(idx, 1)
}

function handleSave() {
  const payload: Partial<ContextAwareRule> = {
    name: formData.name,
    description: formData.description,
    priority: formData.priority,
    conditions: formData.conditions,
    actions: formData.actions.map(a => ({
      type: a.type,
      parameters: JSON.parse(a.parametersJson || '{}'),
    })),
    context: formData.context,
    confidence: formData.confidence,
    created_by: formData.created_by,
  }

  emit('save', payload)
}
</script>
