<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { WorkflowNode } from '@/types'
import { getNodeTemplate } from '@/config/nodeTemplates'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area/index'
// Tabs removed - using button toggle instead
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import MonacoEditor from '@/components/ui/MonacoEditor.vue'
import { Code2, FileText, X, Trash2, Settings2, Save } from 'lucide-vue-next'

// New imports from the provided snippet
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'

// Import modular components
import NodeBasicInfo from './config-forms/NodeBasicInfo.vue'
import SimpleParamForm from './config-forms/SimpleParamForm.vue'
import FieldArrayManager from './config-forms/FieldArrayManager.vue'
import IndependentArrayManager from './config-forms/IndependentArrayManager.vue'
// ExtractionBuilder removed - fields are now managed as canvas nodes

// Visual Selector API
import { selectorApi } from '@/api/selector'

interface Props {
  node: WorkflowNode | null
}

interface Emits {
  (e: 'update', node: WorkflowNode): void
  (e: 'delete'): void
  (e: 'close'): void
  (e: 'save'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const activeTab = ref('settings')
const jsonValue = ref('')
const jsonError = ref<string | null>(null)

// Local copy of the node to avoid direct mutation of props
const localNode = ref<WorkflowNode | null>(null)

// Watch for node changes
watch(
  () => props.node,
  (newNode) => {
    if (newNode) {
      // If IDs don't match, it's a different node - always update
      if (!localNode.value || localNode.value.id !== newNode.id) {
        localNode.value = JSON.parse(JSON.stringify(newNode))
        jsonValue.value = JSON.stringify(newNode.data.params, null, 2)
        
        // Reset tab to settings unless it's a complex node that forces JSON
        if (shouldForceJson(newNode)) {
          activeTab.value = 'json'
        } else {
          activeTab.value = 'settings'
        }
      } else {
        // Same node ID - check if content actually changed to avoid infinite loop
        const currentStr = JSON.stringify(localNode.value)
        const newStr = JSON.stringify(newNode)
        
        if (currentStr !== newStr) {
           localNode.value = JSON.parse(newStr)
           jsonValue.value = JSON.stringify(newNode.data.params, null, 2)
        }
      }
    } else {
      localNode.value = null
    }
  },
  { immediate: true, deep: true }
)

// Emit updates whenever localNode changes
watch(
  localNode,
  (newNode) => {
    if (newNode) {
      emit('update', newNode)
    }
  },
  { deep: true }
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

function shouldForceJson(node: WorkflowNode): boolean {
  const params = node.data.params
  const paramKeys = Object.keys(params)
  
  // JSON mode for complex nodes
  if (
    // Nodes with many parameters
    paramKeys.length > 8 ||
    // Nodes with nested objects (excluding 'fields' for extract nodes)
    paramKeys.some(key => {
      // Skip 'fields' param for extract nodes as we have a custom builder for it
      if (node.data.nodeType === 'extract' && key === 'fields') return false
      
      const value = params[key]
      return value !== null && 
             typeof value === 'object' && 
             !Array.isArray(value) &&
             Object.keys(value).length > 0
    }) ||
    // Specific complex node types
    ['sequence', 'conditional'].includes(node.data.nodeType)
  ) {
    return true
  }
  return false
}

function handleJsonChange(value: string) {
  jsonValue.value = value
  try {
    const parsed = JSON.parse(value)
    if (localNode.value) {
      localNode.value.data.params = parsed
      jsonError.value = null
    }
  } catch (e: any) {
    jsonError.value = e.message
  }
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

const extractionMode = computed(() => {
  if (!localNode.value) return 'single'
  const params = localNode.value.data.params
  
  if (params.extractions && params.extractions.length > 0) return 'key_value'
  if (params.multiple) return 'multiple'
  return 'single'
})

function updateExtractionMode(mode: string) {
  if (!localNode.value) return
  
  const params = localNode.value.data.params
  
  switch (mode) {
    case 'single':
      params.multiple = false
      params.extractions = null // Clear extractions
      if (!params.type) params.type = 'text'
      break
    case 'multiple':
      params.multiple = true
      params.extractions = null // Clear extractions
      if (!params.type) params.type = 'text'
      break
    case 'key_value':
      params.multiple = false 
      if (!params.extractions) params.extractions = []
      break
  }
}

// Visual Selector state
const visualSelectorSessionId = ref<string | null>(null)
const isVisualSelectorOpen = ref(false)
let stopPolling: (() => void) | null = null

// Visual Selector for Extract Node
async function openVisualSelector() {
  let url = prompt('Enter the URL to open for element selection:')
  if (!url) return
  
  // Add protocol if missing
  if (!url.startsWith('http://') && !url.startsWith('https://')) {
    url = 'https://' + url
  }
  
  try {
    // Convert existing fields to SelectedField format
    const existingFields = convertToSelectedFields(localNode.value?.data.params.fields || {})
    
    // Create session
    const session = await selectorApi.createSession(url, existingFields)
    visualSelectorSessionId.value = session.session_id
    isVisualSelectorOpen.value = true
    
    console.log('Visual Selector session created:', session.session_id)
    
    // Start polling for selected fields
    stopPolling = await selectorApi.pollForFields(
      session.session_id,
      2000,
      (selectedFields) => {
        if (selectedFields.length > 0) {
          importFromVisualSelector(selectedFields)
        }
      },
      (error) => {
        console.error('Visual selector polling error:', error)
        closeVisualSelector()
      }
    )
    
  } catch (error: any) {
    console.error('Visual selector error:', error)
    const errorMsg = error.response?.data?.error || error.message || 'Unknown error'
    alert(`Failed to open Visual Selector:\n${errorMsg}`)
  }
}

function convertToSelectedFields(nodeFields: Record<string, any>): any[] {
  const selected: any[] = []
  
  for (const [fieldName, fieldConfig] of Object.entries(nodeFields)) {
    const config = fieldConfig as any
    
    // Handle key-value pairs
    if (config.extractions && Array.isArray(config.extractions)) {
      selected.push({
        name: fieldName,
        selector: config.selector || '',
        type: config.type || 'text',
        multiple: config.multiple || false,
        mode: 'key-value-pairs',
        attributes: {
          extractions: config.extractions
        }
      })
    }
    // Handle regular fields
    else {
      selected.push({
        name: fieldName,
        selector: config.selector || '',
        type: config.type || 'text',
        attribute: config.attribute,
        multiple: config.multiple || false,
        mode: config.multiple ? 'list' : 'single'
      })
    }
  }
  
  return selected
}

function importFromVisualSelector(selectedFields: any[]) {
  if (!localNode.value) return
  
  const newFields: Record<string, any> = {}
  
  selectedFields.forEach(field => {
    const fieldConfig: any = {
      selector: field.selector,
      type: field.type,
      multiple: field.multiple || false
    }
    
    if (field.attribute) {
      fieldConfig.attribute = field.attribute
    }
    
    // Handle key-value pairs
    if (field.mode === 'key-value-pairs' && field.attributes?.extractions) {
      fieldConfig.extractions = field.attributes.extractions
      fieldConfig.multiple = false // Key-value pairs are not "multiple"
    }
    
    newFields[field.name] = fieldConfig
  })
  
  // Check if fields actually changed
  const currentFields = localNode.value.data.params.fields || {}
  if (JSON.stringify(currentFields) === JSON.stringify(newFields)) {
    return // No changes, skip update
  }

  console.log('📥 [Visual Selector] Importing fields:', newFields)
  console.log('📝 [Visual Selector] Current fields:', currentFields)
  
  // Update the node's fields
  localNode.value.data.params.fields = newFields
  
  console.log('✅ [Visual Selector] Fields updated, emitting update event')
  
  // Watcher on localNode will handle the emit automatically
}

function closeVisualSelector() {
  if (stopPolling) {
    stopPolling()
    stopPolling = null
  }
  
  if (visualSelectorSessionId.value) {
    selectorApi.closeSession(visualSelectorSessionId.value).catch(err => {
      console.error('Error closing visual selector session:', err)
    })
    visualSelectorSessionId.value = null
  }
  
  isVisualSelectorOpen.value = false
}

</script>

<template>
  <div class="h-full flex flex-col bg-card border-l border-border w-96 shadow-xl z-20">
    <!-- Empty State -->
    <div v-if="!localNode" class="flex-1 flex flex-col items-center justify-center text-muted-foreground p-6 text-center">
      <div class="w-12 h-12 rounded-full bg-muted flex items-center justify-center mb-4">
        <Settings2 class="w-6 h-6 opacity-50" />
      </div>
      <h3 class="font-medium text-foreground mb-1">No Node Selected</h3>
      <p class="text-sm">Select a node on the canvas to configure its properties.</p>
    </div>

    <!-- Content -->
    <div v-else class="flex flex-col h-full">
      <!-- Header -->
      <div class="p-4 border-b border-border">
        <div class="flex items-center justify-between mb-2">
          <h2 class="font-semibold text-lg truncate">{{ localNode.data.label }}</h2>
          <div class="flex items-center gap-1">
            <Button variant="ghost" size="icon" class="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10" @click="emit('delete')">
              <Trash2 class="w-4 h-4" />
            </Button>
            <Button variant="ghost" size="icon" class="h-8 w-8" @click="emit('close')">
              <X class="w-4 h-4" />
            </Button>
          </div>
        </div>
        <div class="flex items-center gap-2 text-xs text-muted-foreground">
          <span class="px-2 py-0.5 rounded-md bg-primary/10 text-primary font-medium capitalize">
            {{ localNode.data.nodeType }}
          </span>
          <span class="w-1 h-1 rounded-full bg-border"></span>
          <span class="font-mono opacity-70 text-[10px]" :title="localNode.id">
            {{ localNode.id.substring(0, 16) }}...
          </span>
        </div>
      </div>

      <!-- Mode Toggle (Tab Alternative) -->
      <div class="px-4 py-3 border-b border-border">
        <div class="flex items-center bg-muted rounded-lg p-1 w-fit">
          <Button 
            variant="ghost" 
            size="sm" 
            :class="{ 'bg-background shadow-sm': activeTab === 'settings' }"
            @click="activeTab = 'settings'"
          >
            <FileText class="w-4 h-4 mr-2" />
            Settings
          </Button>
          <Button 
            variant="ghost" 
            size="sm" 
            :class="{ 'bg-background shadow-sm': activeTab === 'json' }"
            @click="activeTab = 'json'"
          >
            <Code2 class="w-4 h-4 mr-2" />
            JSON
          </Button>
        </div>
      </div>

        <!-- Settings Tab -->
        <div v-if="activeTab === 'settings'" class="flex-1 overflow-hidden flex flex-col">
          <ScrollArea class="flex-1">
            <div class="p-4 space-y-6">
              <!-- Basic Info -->
              <NodeBasicInfo
                :label="localNode.data.label"
                :node-type="localNode.data.nodeType"
                @update:label="updateLabel"
              />

              <Separator />

              <!-- Special Form for Extraction Field Node -->
              <div v-if="localNode.data.nodeType === 'extractField'" class="space-y-4">
                 <!-- Extraction Mode Selection -->
                 <div class="space-y-2">
                   <Label>Extraction Mode</Label>
                   <Select 
                     :model-value="extractionMode"
                     @update:model-value="(val) => updateExtractionMode(String(val))"
                   >
                     <SelectTrigger>
                       <SelectValue placeholder="Select mode" />
                     </SelectTrigger>
                     <SelectContent>
                       <SelectItem value="single">Single Value</SelectItem>
                       <SelectItem value="multiple">List/Array</SelectItem>
                       <SelectItem value="key_value">Key-Value Pairs</SelectItem>
                     </SelectContent>
                   </Select>
                 </div>

                 <!-- Single & Multiple Mode Fields -->
                 <template v-if="extractionMode !== 'key_value'">
                   <div class="space-y-2">
                    <Label>CSS Selector</Label>
                    <Input 
                      v-model="localNode.data.params.selector" 
                      placeholder="e.g. .product-title, h1"
                    />
                  </div>

                  <div class="grid grid-cols-2 gap-4">
                    <div class="space-y-2">
                      <Label>Type</Label>
                      <Select v-model="localNode.data.params.type">
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="text">Text Content</SelectItem>
                          <SelectItem value="html">Inner HTML</SelectItem>
                          <SelectItem value="attribute">Attribute</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    <div class="space-y-2">
                      <Label>Transform</Label>
                      <Select v-model="localNode.data.params.transform">
                        <SelectTrigger>
                          <SelectValue placeholder="None" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="none">None</SelectItem>
                          <SelectItem value="trim">Trim Whitespace</SelectItem>
                          <SelectItem value="lowercase">Lowercase</SelectItem>
                          <SelectItem value="uppercase">Uppercase</SelectItem>
                          <SelectItem value="clean_html">Clean HTML</SelectItem>
                          <SelectItem value="number">To Number</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                  <div v-if="localNode.data.params.type === 'attribute'" class="space-y-2">
                    <Label>Attribute Name</Label>
                    <Input 
                      v-model="localNode.data.params.attribute" 
                      placeholder="e.g. href, src"
                    />
                  </div>

                  <!-- Advanced Options (Limit/Default) -->
                  <Accordion type="single" collapsible class="w-full">
                    <AccordionItem value="advanced" class="border-b-0">
                      <AccordionTrigger class="text-xs text-muted-foreground py-2">
                        Advanced Options
                      </AccordionTrigger>
                      <AccordionContent class="space-y-4 pt-2">
                        <div v-if="extractionMode === 'multiple'" class="space-y-2">
                          <Label>Limit (0 for unlimited)</Label>
                          <Input 
                            type="number"
                            v-model.number="localNode.data.params.limit" 
                            placeholder="0"
                          />
                        </div>

                        <div class="space-y-2">
                          <Label>Default Value</Label>
                          <Input 
                            v-model="localNode.data.params.default_value" 
                            placeholder="Value if extraction fails"
                          />
                        </div>
                      </AccordionContent>
                    </AccordionItem>
                  </Accordion>
                 </template>

                 <!-- Key-Value Pair Mode -->
                 <div v-else class="space-y-2">
                   <Label>Independent Extractions</Label>
                   <IndependentArrayManager
                     :model-value="localNode.data.params.extractions || []"
                     @update:model-value="localNode.data.params.extractions = $event"
                   />
                 </div>
              </div>

              <!-- Extract Node - Visual Selector Only (TOP) -->
              <div v-if="localNode.data.nodeType === 'extract'" class="space-y-4 mb-6">
                <!-- Visual Selector Button -->
                <div class="space-y-2">
                  <Label>Field Configuration</Label>
                  <Button 
                    variant="default" 
                    class="w-full bg-blue-600 hover:bg-blue-700"
                    @click="openVisualSelector"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 mr-2" viewBox="0 0 20 20" fill="currentColor">
                      <path d="M11 3a1 1 0 100 2h2.586l-6.293 6.293a1 1 0 101.414 1.414L15 6.414V9a1 1 0 102 0V4a1 1 0 00-1-1h-5z" />
                      <path d="M5 5a2 2 0 00-2 2v8a2 2 0 002 2h8a2 2 0 002-2v-3a1 1 0 10-2 0v3H5V7h3a1 1 0 000-2H5z" />
                    </svg>
                    Visual Selector
                  </Button>
                </div>
                
                <!-- Info Message -->
                <div class="flex items-start gap-3 p-4 rounded-lg bg-primary/5 border border-primary/20">
                  <div class="flex-shrink-0 mt-0.5">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 20 20" fill="currentColor">
                      <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
                    </svg>
                  </div>
                  <div class="flex-1 text-sm">
                    <div class="font-medium text-foreground mb-1">Fields managed on canvas</div>
                    <div class="text-muted-foreground text-xs leading-relaxed">
                      After using Visual Selector, individual field nodes will appear on the canvas for detailed configuration.
                    </div>
                  </div>
                </div>

                <Separator />
              </div>

              <!-- Standard Parameters -->
              <div v-if="nodeTemplate?.paramSchema && nodeTemplate.paramSchema.length > 0">
                <template v-for="field in nodeTemplate.paramSchema" :key="field.key">
                  <!-- Field Array Manager -->
                  <div 
                    v-if="field.type === 'field_array' && !(localNode.data.nodeType === 'extract' && field.key === 'fields')" 
                    class="space-y-2"
                  >
                    <div class="font-semibold text-sm mb-2">{{ field.label }}</div>
                    <FieldArrayManager
                      :model-value="localNode.data.params[field.key] || {}"
                      :schema="field.arrayItemSchema || []"
                      :param-key="field.key"
                      @update:model-value="updateParam(field.key, $event)"
                    />
                  </div>
                </template>
                
                <!-- Simple Parameters -->
                <SimpleParamForm
                  v-if="simpleParams.length > 0"
                  :params="localNode.data.params"
                  :schema="simpleParams"
                  @update:params="updateParam"
                />
              </div>
            </div>
          </ScrollArea>
        </div>

        <!-- JSON Tab -->
        <div v-if="activeTab === 'json'" class="flex-1 overflow-hidden flex flex-col">
          <div class="p-2 border-b border-border flex justify-between items-center bg-muted/10">
            <span class="text-xs text-muted-foreground">Edit raw configuration</span>
            <Button size="sm" variant="ghost" class="h-6 text-xs" @click="formatJSON">
              Format
            </Button>
          </div>
          <div class="flex-1 relative">
            <monaco-editor
              v-model="jsonValue"
              language="json"
              theme="vs-dark"
              @change="handleJsonChange"
            />
          </div>
          <div v-if="jsonError" class="p-2 bg-destructive/10 text-destructive text-xs border-t border-destructive/20">
            {{ jsonError }}
          </div>
        </div>
      
      <!-- Footer with Save Button -->
      <div class="p-4 border-t border-border bg-muted/20">
        <Button @click="emit('save')" variant="default" size="default" class="w-full">
          <Save class="w-4 h-4 mr-2" />
          Save Workflow
        </Button>
      </div>
    </div>
  </div>
</template>
