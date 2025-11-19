<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { WorkflowNode } from '@/types'
import { getNodeTemplate } from '@/config/nodeTemplates'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { X, Settings } from 'lucide-vue-next'

interface Props {
  node: WorkflowNode | null
}

interface Emits {
  (e: 'update:node', node: WorkflowNode): void
  (e: 'close'): void
  (e: 'delete'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const localNode = ref<WorkflowNode | null>(null)

watch(
  () => props.node,
  (newNode) => {
    if (newNode) {
      localNode.value = JSON.parse(JSON.stringify(newNode))
    } else {
      localNode.value = null
    }
  },
  { immediate: true, deep: true }
)

const nodeTemplate = computed(() => {
  if (!localNode.value) return null
  return getNodeTemplate(localNode.value.data.nodeType)
})

const paramSchema = computed(() => {
  return nodeTemplate.value?.paramSchema || []
})

function updateParam(key: string, value: any) {
  if (!localNode.value) return
  localNode.value.data.params[key] = value
}

function updateLabel(value: string) {
  if (!localNode.value) return
  localNode.value.data.label = value
}

function updateOptional(value: boolean) {
  if (!localNode.value) return
  localNode.value.data.optional = value
}

function updateRetry(field: 'max_retries' | 'delay', value: number) {
  if (!localNode.value) return
  if (!localNode.value.data.retry) {
    localNode.value.data.retry = { max_retries: 0, delay: 0 }
  }
  localNode.value.data.retry[field] = value
}

function saveChanges() {
  if (localNode.value) {
    emit('update:node', localNode.value)
  }
}

function deleteNode() {
  emit('delete')
}

function parseJsonParam(value: string, key: string) {
  try {
    const parsed = JSON.parse(value)
    updateParam(key, parsed)
  } catch (e) {
    console.error('Invalid JSON:', e)
  }
}
</script>

<template>
  <div v-if="localNode" class="h-full flex flex-col bg-background border-l border-border">
    <!-- Header -->
    <div class="p-4 border-b border-border flex items-center justify-between bg-muted/30">
      <div>
        <h3 class="font-semibold text-lg">Node Configuration</h3>
        <p class="text-xs text-muted-foreground mt-0.5">Configure node parameters</p>
      </div>
      <Button variant="ghost" size="icon" @click="emit('close')">
        <X class="w-4 h-4" />
      </Button>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-y-auto p-4 space-y-6 scroll-smooth">
      <!-- Node Label -->
      <div class="space-y-2">
        <Label for="node-label">Node Label</Label>
        <Input
          id="node-label"
          :model-value="localNode.data.label"
          @update:model-value="updateLabel"
          placeholder="Enter node label"
        />
      </div>

      <!-- Node Type (Read-only) -->
      <div class="space-y-2">
        <Label>Node Type</Label>
        <div class="p-2.5 bg-muted border border-border rounded-md text-sm font-mono">
          {{ localNode.data.nodeType }}
        </div>
      </div>

      <!-- Parameters -->
      <div v-if="paramSchema.length > 0" class="space-y-4">
        <div class="flex items-center gap-2 pb-2 border-b border-border">
          <div class="font-semibold text-sm">Parameters</div>
          <div class="text-xs text-muted-foreground">({{ paramSchema.length }})</div>
        </div>

        <div
          v-for="field in paramSchema"
          :key="field.key"
          class="space-y-2"
        >
          <Label :for="`param-${field.key}`">
            {{ field.label }}
            <span v-if="field.required" class="text-red-500">*</span>
          </Label>

          <!-- Text Input -->
          <Input
            v-if="field.type === 'text'"
            :id="`param-${field.key}`"
            :model-value="localNode.data.params[field.key] || ''"
            @update:model-value="(val: string) => updateParam(field.key, val)"
            :placeholder="field.placeholder"
          />

          <!-- Number Input -->
          <Input
            v-else-if="field.type === 'number'"
            :id="`param-${field.key}`"
            type="number"
            :model-value="localNode.data.params[field.key] ?? field.defaultValue ?? 0"
            @update:model-value="(val) => updateParam(field.key, Number(val))"
            :placeholder="field.placeholder"
          />

          <!-- Textarea -->
          <Textarea
            v-else-if="field.type === 'textarea'"
            :id="`param-${field.key}`"
            :model-value="
              typeof localNode.data.params[field.key] === 'object'
                ? JSON.stringify(localNode.data.params[field.key], null, 2)
                : localNode.data.params[field.key] || ''
            "
            @blur="(e: Event) => parseJsonParam(((e.target as HTMLTextAreaElement).value), field.key)"
            :placeholder="field.placeholder"
            rows="6"
            class="font-mono text-sm"
          />

          <!-- Select -->
          <Select
            v-else-if="field.type === 'select' && field.options"
            :model-value="localNode.data.params[field.key] || field.defaultValue"
            @update:model-value="(val) => updateParam(field.key, val)"
          >
            <SelectTrigger :id="`param-${field.key}`">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="option in field.options"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </SelectItem>
            </SelectContent>
          </Select>

          <!-- Boolean Switch -->
          <div v-else-if="field.type === 'boolean'" class="flex items-center space-x-2">
            <Switch
              :id="`param-${field.key}`"
              :checked="localNode.data.params[field.key] ?? field.defaultValue ?? false"
              @update:checked="(val: boolean) => updateParam(field.key, val)"
            />
          </div>

          <!-- Description -->
          <p v-if="field.description" class="text-xs text-muted-foreground">
            {{ field.description }}
          </p>
        </div>
      </div>

      <!-- Advanced Options -->
      <div class="space-y-4 border-t border-border pt-4">
        <div class="font-semibold text-sm">Advanced Options</div>

        <!-- Optional -->
        <div class="flex items-center justify-between">
          <Label for="optional">Optional Node</Label>
          <Switch
            id="optional"
            :checked="localNode.data.optional || false"
            @update:checked="updateOptional"
          />
        </div>

        <!-- Retry Config -->
        <div class="space-y-3">
          <div class="text-sm font-medium">Retry Configuration</div>
          <div class="space-y-2">
            <Label for="max-retries">Max Retries</Label>
            <Input
              id="max-retries"
              type="number"
              :model-value="localNode.data.retry?.max_retries || 0"
              @update:model-value="(val) => updateRetry('max_retries', Number(val))"
              min="0"
            />
          </div>
          <div class="space-y-2">
            <Label for="retry-delay">Delay (ms)</Label>
            <Input
              id="retry-delay"
              type="number"
              :model-value="localNode.data.retry?.delay || 0"
              @update:model-value="(val) => updateRetry('delay', Number(val))"
              min="0"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- Footer -->
    <div class="p-4 border-t border-border bg-muted/30 flex gap-2">
      <Button @click="saveChanges" class="flex-1" size="default">
        Save Changes
      </Button>
      <Button @click="deleteNode" variant="destructive" size="default">
        Delete
      </Button>
    </div>
  </div>

  <!-- Empty State -->
  <div v-else class="h-full flex flex-col items-center justify-center bg-muted/20 border-l border-border gap-3">
    <div class="w-16 h-16 rounded-full bg-muted flex items-center justify-center">
      <Settings class="w-8 h-8 text-muted-foreground" />
    </div>
    <div class="text-center">
      <p class="font-medium">No Node Selected</p>
      <p class="text-sm text-muted-foreground mt-1">Select a node to configure its parameters</p>
    </div>
  </div>
</template>
