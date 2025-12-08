<script setup lang="ts">
import { ref, watch, computed, onMounted } from 'vue'
import type { WorkflowNode, WorkflowConfig } from '@/types'
import { getNodeTemplate } from '@/config/nodeTemplates'
import { useBrowserProfilesStore } from '@/stores/browserProfiles'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area/index'
// Tabs removed - using button toggle instead
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import MonacoEditor from '@/components/ui/MonacoEditor.vue'
import { Code2, FileText, X, Trash2, Settings2, Save, Globe } from 'lucide-vue-next'

// New imports from the provided snippet
import { Textarea } from '@/components/ui/textarea'
// import { Switch } from '@/components/ui/switch'

// Import modular components
import NodeBasicInfo from './config-forms/NodeBasicInfo.vue'
import SimpleParamForm from './config-forms/SimpleParamForm.vue'
import FieldArrayManager from './config-forms/FieldArrayManager.vue'
import IndependentArrayManager from './config-forms/IndependentArrayManager.vue'
import FieldActionsManager from './config-forms/FieldActionsManager.vue'
// ExtractionBuilder removed - fields are now managed as canvas nodes



interface Props {
  node: WorkflowNode | null
  workflowConfig?: Partial<WorkflowConfig>
}

interface Emits {
  (e: 'update', node: WorkflowNode): void
  (e: 'update:workflowConfig', config: Partial<WorkflowConfig>): void
  (e: 'delete'): void
  (e: 'close'): void
  (e: 'save'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const profilesStore = useBrowserProfilesStore()
onMounted(() => {
  profilesStore.fetchProfiles()
})

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

// Filter browser profiles by selected default driver
const filteredProfiles = computed(() => {
  const driverType = props.workflowConfig?.default_driver || 'playwright'
  
  if (driverType === 'http') {
    return [] // HTTP driver doesn't use browser profiles
  }
  
  // Filter profiles matching the driver type
  return profilesStore.profiles.filter(p => p.driver_type === driverType)
})



function updateWorkflowConfig(key: keyof WorkflowConfig, value: any) {
  let finalValue = value
  if (key === 'browser_profile_id' && value === 'default_profile') {
    finalValue = null
  }
  const newConfig = { ...props.workflowConfig, [key]: finalValue }
  emit('update:workflowConfig', newConfig)
}



</script>

<template>
  <div class="h-full flex flex-col bg-card border-l border-border w-96 shadow-xl z-20">
    <!-- Workflow Settings (When no node selected) -->
    <div v-if="!localNode" class="flex flex-col h-full">
      <!-- Header -->
      <div class="p-4 border-b border-border">
        <div class="flex items-center gap-2 mb-1">
          <Settings2 class="w-5 h-5 text-primary" />
          <h2 class="font-semibold text-lg">Workflow Settings</h2>
        </div>
        <p class="text-xs text-muted-foreground">Global configuration for this workflow</p>
      </div>

      <ScrollArea class="flex-1">
        <div class="p-4 space-y-6">
          
          <!-- Workflow Description -->
          <div class="space-y-2">
            <Label>Description</Label>
            <Textarea
              :model-value="workflowConfig?.description || ''"
              @update:model-value="updateWorkflowConfig('description', $event)"
              placeholder="Add a description for this workflow..."
              rows="2"
              class="text-sm resize-none"
            />
          </div>

          <Separator />
          
          <!-- Browser Configuration -->
          <div class="space-y-4">
            <div class="flex items-center gap-2 text-sm font-medium text-foreground">
              <Globe class="w-4 h-4" />
              Browser Configuration
            </div>
            
            <!-- Default Driver Selection -->
            <div class="space-y-2">
              <Label>Default Driver</Label>
              <Select 
                :model-value="workflowConfig?.default_driver || 'playwright'" 
                @update:model-value="(val) => updateWorkflowConfig('default_driver', val)"
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select driver">
                    {{ 
                      workflowConfig?.default_driver === 'chromedp' ? 'Chromedp (Chrome DevTools Protocol)'
                        : workflowConfig?.default_driver === 'http' ? 'HTTP Client (No Browser)'
                        : 'Playwright (Default)'
                    }}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="playwright">Playwright (Default)</SelectItem>
                  <SelectItem value="chromedp">Chromedp (Chrome DevTools Protocol)</SelectItem>
                  <SelectItem value="http">HTTP Client (No Browser)</SelectItem>
                </SelectContent>
              </Select>
              <p class="text-[10px] text-muted-foreground">
                Default browser automation driver for this workflow.
              </p>
            </div>

            <!-- Browser Profile (hidden for HTTP driver) -->
            <div v-if="workflowConfig?.default_driver !== 'http'" class="space-y-2">
              <Label>Browser Profile</Label>
              <Select 
                :model-value="workflowConfig?.browser_profile_id || ''" 
                @update:model-value="(val) => updateWorkflowConfig('browser_profile_id', val)"
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select a browser profile">
                    {{ 
                      workflowConfig?.browser_profile_id 
                        ? (profilesStore.profiles.find(p => p.id === workflowConfig.browser_profile_id)?.name || 'Unknown Profile')
                        : 'Default Profile' 
                    }}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <!-- We use a special value for clearing/default -->
                  <SelectItem value="default_profile">Default Profile</SelectItem>
                  <template v-for="profile in filteredProfiles" :key="profile.id">
                    <SelectItem :value="profile.id">
                      <div class="flex items-center gap-2">
                        <span class="w-2 h-2 rounded-full" :class="profile.status === 'active' ? 'bg-green-500' : 'bg-gray-300'"></span>
                        {{ profile.name }}
                        <span class="text-xs text-muted-foreground">({{ profile.driver_type }})</span>
                      </div>
                    </SelectItem>
                  </template>
                </SelectContent>
              </Select>
              <p class="text-[10px] text-muted-foreground">
                Select which browser profile (fingerprint, cookies, etc.) to use for this workflow.
              </p>
            </div>

            <!-- HTTP Driver Browser Name Selection -->
            <div v-if="workflowConfig?.default_driver === 'http'" class="space-y-2">
              <Label>Default Browser (JA3 Fingerprint)</Label>
              <Select 
                :model-value="workflowConfig?.default_browser_name || 'chrome'" 
                @update:model-value="(val) => updateWorkflowConfig('default_browser_name', val)"
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select browser fingerprint">
                    {{ 
                      workflowConfig?.default_browser_name === 'firefox' ? 'Firefox'
                        : workflowConfig?.default_browser_name === 'safari' ? 'Safari'
                        : workflowConfig?.default_browser_name === 'edge' ? 'Edge'
                        : workflowConfig?.default_browser_name === 'ios' ? 'iOS Safari'
                        : workflowConfig?.default_browser_name === 'android' ? 'Android Chrome'
                        : 'Chrome (Default)'
                    }}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="chrome">Chrome (Default)</SelectItem>
                  <SelectItem value="firefox">Firefox</SelectItem>
                  <SelectItem value="safari">Safari</SelectItem>
                  <SelectItem value="edge">Edge</SelectItem>
                  <SelectItem value="ios">iOS Safari</SelectItem>
                  <SelectItem value="android">Android Chrome</SelectItem>
                </SelectContent>
              </Select>
              <p class="text-[10px] text-muted-foreground">
                TLS fingerprint + User-Agent for HTTP requests. Mimics the selected browser's network signature.
              </p>
            </div>
          </div>

          <Separator />

          <!-- Crawling Configuration -->
          <div class="space-y-4">
             <div class="flex items-center gap-2 text-sm font-medium text-foreground">
              <Settings2 class="w-4 h-4" />
              Crawling Limits
            </div>

            <div class="space-y-2">
              <Label>Max Depth</Label>
              <Input 
                type="number" 
                :model-value="workflowConfig?.max_depth || 3"
                @update:model-value="(val) => updateWorkflowConfig('max_depth', Number(val))"
              />
            </div>

            <div class="space-y-2">
              <Label>Rate Limit Delay (ms)</Label>
              <Input 
                type="number" 
                :model-value="workflowConfig?.rate_limit_delay || 1000"
                @update:model-value="(val) => updateWorkflowConfig('rate_limit_delay', Number(val))"
              />
            </div>
          </div>

        </div>
      </ScrollArea>
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

                 <Separator class="my-4" />

                 <!-- Pre-Extraction Actions -->
                 <FieldActionsManager
                   :model-value="localNode.data.params.actions || []"
                   @update:model-value="localNode.data.params.actions = $event"
                 />
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

              <!-- Probe Configuration (for extract_links and extract nodes) -->
              <div v-if="['extract_links', 'extract'].includes(localNode.data.nodeType)" class="space-y-4">
                <Separator />
                <div class="space-y-2">
                  <div class="flex items-center gap-2 text-sm font-medium text-foreground">
                    <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
                    Probe Configuration
                  </div>
                  <p class="text-[10px] text-muted-foreground">
                    Sample URL for testing this node during probe execution.
                  </p>
                </div>
                <div class="space-y-2">
                  <Label>Probe Sample URL</Label>
                  <Input 
                    :model-value="localNode.data.params.probe_url || ''"
                    @update:model-value="updateParam('probe_url', $event)"
                    placeholder="https://example.com/sample-page"
                  />
                  <p class="text-[10px] text-muted-foreground">
                    When running probes, this URL will be used to test this node's selectors.
                  </p>
                </div>
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
