<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { WorkflowNode } from '@/types'
import { getNodeTemplate } from '@/config/nodeTemplates'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import MonacoEditor from '@/components/ui/MonacoEditor.vue'
import { Code2, FileText } from 'lucide-vue-next'

// Import modular components
import NodeBasicInfo from './config-forms/NodeBasicInfo.vue'
import SimpleParamForm from './config-forms/SimpleParamForm.vue'
import FieldArrayManager from './config-forms/FieldArrayManager.vue'

interface Props {
  node: WorkflowNode | null
  open: boolean
}

interface Emits {
  (e: 'update:open', value: boolean): void
  (e: 'save', node: WorkflowNode): void
  (e: 'delete'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

type ConfigMode = 'form' | 'json'

const localNode = ref<WorkflowNode | null>(null)
const configMode = ref<ConfigMode>('form')
const jsonValue = ref('')
const jsonError = ref<string | null>(null)

// Watch for node changes
watch(
  () => props.node,
  (newNode) => {
    if (newNode) {
      localNode.value = JSON.parse(JSON.stringify(newNode))
      jsonValue.value = JSON.stringify(newNode.data.params, null, 2)
      
      // Auto-select mode based on complexity
      configMode.value = detectConfigMode(newNode)
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

// Filter out field_array types for separate rendering
const simpleParams = computed(() => {
  if (!nodeTemplate.value?.paramSchema) return []
  return nodeTemplate.value.paramSchema.filter(
    field => field.type !== 'field_array' && field.type !== 'sequence_steps'
  )
})

// Smart mode detection
function detectConfigMode(node: WorkflowNode): ConfigMode {
  const params = node.data.params
  const paramKeys = Object.keys(params)
  
  // JSON mode for complex nodes
  if (
    // Nodes with many parameters
    paramKeys.length > 5 ||
    // Nodes with array parameters
    paramKeys.some(key => Array.isArray(params[key])) ||
    // Nodes with nested objects
    paramKeys.some(key => {
      const value = params[key]
      return value !== null && 
             typeof value === 'object' && 
             !Array.isArray(value) &&
             Object.keys(value).length > 0
    }) ||
    // Specific complex node types
    ['extract', 'sequence', 'conditional'].includes(node.data.nodeType)
  ) {
    return 'json'
  }
  
  return 'form'
}

function handleClose() {
  emit('update:open', false)
}

function handleSave() {
  if (!localNode.value) return
  
  // If in JSON mode, parse and update params
  if (configMode.value === 'json') {
    try {
      const parsed = JSON.parse(jsonValue.value)
      localNode.value.data.params = parsed
      jsonError.value = null
    } catch (error: any) {
      jsonError.value = error.message
      return
    }
  }
  
  emit('save', localNode.value)
  emit('update:open', false)
}

function handleModeToggle(mode: ConfigMode) {
  if (mode === 'json' && localNode.value) {
    // Switch to JSON: Update JSON value from current params
    jsonValue.value = JSON.stringify(localNode.value.data.params, null, 2)
  } else if (mode === 'form' && localNode.value) {
    // Switch to Form: Parse JSON and update params
    try {
      const parsed = JSON.parse(jsonValue.value)
      localNode.value.data.params = parsed
      jsonError.value = null
    } catch (error: any) {
      jsonError.value = error.message
      return
    }
  }
  
  configMode.value = mode
}

function formatJSON() {
  try {
    const parsed = JSON.parse(jsonValue.value)
    jsonValue.value = JSON.stringify(parsed, null, 2)
    jsonError.value = null
  } catch (error: any) {
    jsonError.value = error.message
  }
}

function updateParam(key: string, value: any) {
  if (!localNode.value) return
  localNode.value.data.params[key] = value
}

function updateLabel(value: string | number) {
  if (!localNode.value) return
  localNode.value.data.label = String(value)
}

// Keyboard shortcuts
function handleKeyDown(event: KeyboardEvent) {
  // ESC to close
  if (event.key === 'Escape') {
    handleClose()
  }
  
  // Cmd+S or Ctrl+S to save
  if ((event.metaKey || event.ctrlKey) && event.key === 's') {
    event.preventDefault()
    handleSave()
  }
}

watch(() => props.open, (isOpen) => {
  if (isOpen) {
    window.addEventListener('keydown', handleKeyDown)
  } else {
    window.removeEventListener('keydown', handleKeyDown)
  }
})
</script>

<template>
  <Dialog :open="open" @update:open="(val) => emit('update:open', val)">
    <DialogContent 
      class="max-w-[80vw] h-[85vh] flex flex-col gap-0 p-0"
      :class="{ 'backdrop-blur-sm': open }"
    >
      <!-- Header -->
      <DialogHeader class="p-6 pb-4 border-b space-y-1">
        <div class="flex items-start justify-between">
          <div class="flex-1 min-w-0 pr-4">
            <DialogTitle class="text-xl font-semibold mb-1">
              Configure Node
            </DialogTitle>
            <DialogDescription class="text-sm flex items-center gap-2">
              <span class="px-2 py-0.5 bg-primary/10 text-primary rounded font-mono text-xs">
                {{ localNode?.data.nodeType }}
              </span>
              <span class="text-muted-foreground">{{ nodeTemplate?.description }}</span>
            </DialogDescription>
          </div>
          
          <!-- Mode Toggle -->
          <div class="flex items-center gap-2 bg-muted rounded-lg p-1">
            <button
              @click="handleModeToggle('form')"
              :class="[
                'px-3 py-1.5 rounded text-sm font-medium transition-colors flex items-center gap-2',
                configMode === 'form' 
                  ? 'bg-background shadow-sm' 
                  : 'hover:bg-background/50'
              ]"
            >
              <FileText class="w-4 h-4" />
              Form
            </button>
            <button
              @click="handleModeToggle('json')"
              :class="[
                'px-3 py-1.5 rounded text-sm font-medium transition-colors flex items-center gap-2',
                configMode === 'json' 
                  ? 'bg-background shadow-sm' 
                  : 'hover:bg-background/50'
              ]"
            >
              <Code2 class="w-4 h-4" />
              JSON
            </button>
          </div>
        </div>
      </DialogHeader>

      <!-- Content -->
      <div class="flex-1 overflow-hidden p-6">
        <!-- Form Mode -->
        <div v-if="configMode === 'form' && localNode" class="h-full overflow-y-auto space-y-6">
          <!-- Node Basic Info -->
          <NodeBasicInfo
            :label="localNode.data.label"
            :node-type="localNode.data.nodeType"
            @update:label="updateLabel"
          />

          <!-- Parameters -->
          <div v-if="nodeTemplate?.paramSchema && nodeTemplate.paramSchema.length > 0">
            <!-- Check if we have field_array type -->
            <template v-for="field in nodeTemplate.paramSchema" :key="field.key">
              <!-- Field Array Manager for complex extraction fields -->
              <div v-if="field.type === 'field_array'" class="space-y-2">
                <div class="font-semibold text-sm border-b border-border pb-2">
                  {{ field.label }}
                  <span v-if="field.required" class="text-red-500">*</span>
                </div>
                <p v-if="field.description" class="text-xs text-muted-foreground mb-3">
                  {{ field.description }}
                </p>
                <FieldArrayManager
                  :model-value="localNode.data.params[field.key] || {}"
                  :schema="field.arrayItemSchema || []"
                  :param-key="field.key"
                  @update:model-value="updateParam(field.key, $event)"
                />
              </div>
              
              <!-- Simple params (everything except field_array) will be handled by SimpleParamForm below -->
            </template>
            
            <!-- Simple Parameters Form (non-field_array params) -->
            <SimpleParamForm
              v-if="simpleParams.length > 0"
              :params="localNode.data.params"
              :schema="simpleParams"
              @update:params="updateParam"
            />
          </div>
        </div>

        <!-- JSON Mode -->
        <div v-else-if="configMode === 'json'" class="h-full flex flex-col gap-3">
          <!-- Toolbar -->
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <Button 
                size="sm" 
                variant="outline"
                @click="formatJSON"
              >
                Format JSON
              </Button>
              <span v-if="jsonError" class="text-xs text-red-500">
                {{ jsonError }}
              </span>
            </div>
          </div>

          <!-- Monaco Editor -->
          <div class="flex-1 border rounded-lg overflow-hidden">
            <MonacoEditor
              v-model="jsonValue"
              language="json"
              :height="600"
              theme="vs-dark"
            />
          </div>
        </div>
      </div>

      <!-- Footer -->
      <DialogFooter class="p-6 pt-4 border-t bg-muted/30">
        <div class="flex items-center justify-between w-full">
          <div class="text-xs text-muted-foreground">
            <kbd class="px-2 py-1 bg-background border rounded text-xs">ESC</kbd> to cancel
            <span class="mx-2">•</span>
            <kbd class="px-2 py-1 bg-background border rounded text-xs">⌘S</kbd> to save
          </div>
          <div class="flex gap-2">
            <Button variant="outline" @click="handleClose">
              Cancel
            </Button>
            <Button @click="handleSave">
              Save Changes
            </Button>
          </div>
        </div>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
