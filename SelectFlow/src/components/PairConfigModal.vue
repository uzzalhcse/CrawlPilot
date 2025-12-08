<script setup lang="ts">
import { ref, computed } from 'vue'
import { generatePairSelector } from '../utils/selector'
import type { KeyValuePairConfig } from '../types'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { toast } from 'vue-sonner'

const props = defineProps<{
  open: boolean
  labelElement: HTMLElement
  valueElement: HTMLElement
  clickPosition?: { x: number, y: number }
  initialConfig?: {
    groupName: string
    config: KeyValuePairConfig
    outputFormat: 'object' | 'array'
  }
}>()

const emit = defineEmits<{
  (e: 'save', data: { 
    groupName: string, 
    config: KeyValuePairConfig,
    outputFormat: 'object' | 'array'
  }): void
  (e: 'addAnother'): void
  (e: 'cancel'): void
  (e: 'update:open', value: boolean): void
}>()

// Compute anchor position for Popover
const anchorStyle = computed(() => {
  if (!props.clickPosition) return { left: '0px', top: '0px' }
  // Since container is absolute, we need to use page coordinates (viewport + scroll)
  const scrollTop = window.scrollY || document.documentElement.scrollTop
  const scrollLeft = window.scrollX || document.documentElement.scrollLeft
  
  return {
    position: 'absolute' as const,
    left: `${props.clickPosition.x + scrollLeft}px`,
    top: `${props.clickPosition.y + scrollTop}px`,
    width: '1px',
    height: '1px',
    pointerEvents: 'none' as const
  }
})

// State
const groupName = ref('attributes')
const keyType = ref<'text' | 'attr' | 'html'>('text')
const valueType = ref<'text' | 'attr' | 'html'>('text')
const keyAttribute = ref('')
const valueAttribute = ref('')
const transform = ref<'trim' | 'lowercase' | 'uppercase' | 'none'>('trim')
const outputFormat = ref<'object' | 'array'>('array')
const customKeySelector = ref('')
const customValueSelector = ref('')

// Initialize state when modal opens or props change
import { watch } from 'vue'
watch(() => props.open, (isOpen) => {
  if (isOpen) {
    // Always generate default selectors first
    const defaultKeySelector = generatePairSelector(props.labelElement)
    const defaultValueSelector = generatePairSelector(props.valueElement)

    if (props.initialConfig) {
      // Edit mode: Initialize with existing config
      groupName.value = props.initialConfig.groupName
      outputFormat.value = props.initialConfig.outputFormat
      
      const config = props.initialConfig.config
      keyType.value = config.key_type
      valueType.value = config.value_type
      keyAttribute.value = config.key_attribute || ''
      valueAttribute.value = config.value_attribute || ''
      transform.value = (config.transform as any) || 'none'
      
      // Use existing selectors if available, otherwise default
      customKeySelector.value = config.key_selector || defaultKeySelector
      customValueSelector.value = config.value_selector || defaultValueSelector
    } else {
      // Add mode: Reset to defaults
      keyType.value = 'text'
      valueType.value = 'text'
      keyAttribute.value = ''
      valueAttribute.value = ''
      transform.value = 'trim'
      
      // Initialize with generated selectors
      customKeySelector.value = defaultKeySelector
      customValueSelector.value = defaultValueSelector
    }
  }
}, { immediate: true })

// Computed
// We use the custom selectors for preview and saving
const keySelector = computed(() => customKeySelector.value)
const valueSelector = computed(() => customValueSelector.value)

const keyPreview = computed(() => {
  try {
    // Try to find element by custom selector first
    const els = document.querySelectorAll(keySelector.value)
    // If selector matches the original element or we find something, use it
    // But for preview in modal, we might want to stick to the passed element if selector matches it
    // However, if user changes selector, we should try to find what it matches
    
    let el = props.labelElement
    if (els.length > 0) {
      el = els[0] as HTMLElement
    }
    
    if (keyType.value === 'text') {
      return el.textContent?.trim() || 'Empty'
    } else if (keyType.value === 'attr' && keyAttribute.value) {
      return el.getAttribute(keyAttribute.value) || 'No attribute'
    } else if (keyType.value === 'html') {
      const html = el.innerHTML
      return html.length > 50 ? html.substring(0, 50) + '...' : html
    }
    return 'No data'
  } catch {
    return 'Error'
  }
})

const valuePreview = computed(() => {
  try {
    const els = document.querySelectorAll(valueSelector.value)
    let el = props.valueElement
    if (els.length > 0) {
      el = els[0] as HTMLElement
    }
    
    if (valueType.value === 'text') {
      return el.textContent?.trim() || 'Empty'
    } else if (valueType.value === 'attr' && valueAttribute.value) {
      return el.getAttribute(valueAttribute.value) || 'No attribute'
    } else if (valueType.value === 'html') {
      const html = el.innerHTML
      return html.length > 50 ? html.substring(0, 50) + '...' : html
    }
    return 'No data'
  } catch {
    return 'Error'
  }
})

function save(addAnother: boolean = false) {
  if (!groupName.value.trim()) {
    toast.error('Group name cannot be empty')
    return
  }
  
  if (!customKeySelector.value.trim()) {
    toast.error('Key selector cannot be empty')
    return
  }
  
  if (!customValueSelector.value.trim()) {
    toast.error('Value selector cannot be empty')
    return
  }
  
  if (keyType.value === 'attr' && !keyAttribute.value.trim()) {
    toast.error('Please enter a key attribute name')
    return
  }
  
  if (valueType.value === 'attr' && !valueAttribute.value.trim()) {
    toast.error('Please enter a value attribute name')
    return
  }
  
  const config: KeyValuePairConfig = {
    key_selector: customKeySelector.value,
    value_selector: customValueSelector.value,
    key_type: keyType.value,
    value_type: valueType.value,
    key_attribute: keyType.value === 'attr' ? keyAttribute.value : undefined,
    value_attribute: valueType.value === 'attr' ? valueAttribute.value : undefined,
    transform: transform.value === 'none' ? '' : transform.value
  }
  
  emit('save', {
    groupName: groupName.value,
    config,
    outputFormat: outputFormat.value
  })
  
  if (addAnother) {
    emit('addAnother')
  }
}

const isOpen = computed({
  get: () => props.open,
  set: (val) => {
    emit('update:open', val)
    if (!val) {
      emit('cancel')
    }
  }
})
</script>

<template>
  <Popover v-model:open="isOpen">
    <!-- Positioned anchor element at click position -->
    <PopoverTrigger as-child>
      <div :style="anchorStyle" />
    </PopoverTrigger>
    
    <PopoverContent 
      class="w-[420px] p-4 z-[1000001]"
      :side="'bottom'"
      :align="'start'"
    >
      <div class="space-y-3">
        <!-- Header -->
        <div class="flex items-center justify-between pb-2 border-b">
          <h4 class="font-semibold text-sm">Configure Pair</h4>
        </div>
        
        <div class="space-y-3 text-sm">
          <!-- Preview -->
          <div class="rounded border bg-blue-50 p-3 max-h-32 overflow-y-auto">
            <div class="flex items-center gap-2 mb-2">
              <span class="text-xl">🔑</span>
              <code class="text-xs font-mono text-blue-900 flex-1 break-words">{{ keyPreview }}</code>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-xl">🔓</span>
              <code class="text-xs font-mono text-green-900 flex-1 break-words">{{ valuePreview }}</code>
            </div>
          </div>
          
          <!-- Group Name -->
          <div class="space-y-1">
            <Label for="group-name" class="text-xs">Group Name</Label>
            <Input 
              id="group-name" 
              v-model="groupName" 
              placeholder="e.g., attributes, specifications" 
              class="h-8 text-sm" 
            />
          </div>
          
          <!-- Selectors (Manual Edit) -->
          <div class="space-y-2">
            <div class="space-y-1">
              <Label for="key-selector" class="text-xs">Key Selector</Label>
              <Input 
                id="key-selector" 
                v-model="customKeySelector" 
                class="h-8 text-xs font-mono" 
              />
            </div>
            <div class="space-y-1">
              <Label for="value-selector" class="text-xs">Value Selector</Label>
              <Input 
                id="value-selector" 
                v-model="customValueSelector" 
                class="h-8 text-xs font-mono" 
              />
            </div>
          </div>
          
          <!-- Key Type & Attribute -->
          <div class="grid gap-2" :class="keyType === 'attr' ? 'grid-cols-2' : 'grid-cols-1'">
            <div class="space-y-1">
              <Label class="text-xs">Key Type</Label>
              <Select v-model="keyType">
                <SelectTrigger class="h-8 text-sm">
                  <SelectValue placeholder="Select type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="text">Text</SelectItem>
                  <SelectItem value="attr">Attribute</SelectItem>
                  <SelectItem value="html">HTML</SelectItem>
                </SelectContent>
              </Select>
            </div>
            
            <div v-if="keyType === 'attr'" class="space-y-1">
              <Label for="key-attr" class="text-xs">Key Attribute</Label>
              <Input 
                id="key-attr" 
                v-model="keyAttribute" 
                placeholder="src, href" 
                class="h-8 text-sm" 
              />
            </div>
          </div>
          
          <!-- Value Type & Attribute -->
          <div class="grid gap-2" :class="valueType === 'attr' ? 'grid-cols-2' : 'grid-cols-1'">
            <div class="space-y-1">
              <Label class="text-xs">Value Type</Label>
              <Select v-model="valueType">
                <SelectTrigger class="h-8 text-sm">
                  <SelectValue placeholder="Select type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="text">Text</SelectItem>
                  <SelectItem value="attr">Attribute</SelectItem>
                  <SelectItem value="html">HTML</SelectItem>
                </SelectContent>
              </Select>
            </div>
            
            <div v-if="valueType === 'attr'" class="space-y-1">
              <Label for="value-attr" class="text-xs">Value Attribute</Label>
              <Input 
                id="value-attr" 
                v-model="valueAttribute" 
                placeholder="src, href" 
                class="h-8 text-sm" 
              />
            </div>
          </div>
          
          <!-- Transform & Output Format -->
          <div class="grid grid-cols-2 gap-2">
            <div class="space-y-1">
              <Label class="text-xs">Transform</Label>
              <Select v-model="transform">
                <SelectTrigger class="h-8 text-sm">
                  <SelectValue placeholder="None" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">None</SelectItem>
                  <SelectItem value="trim">Trim</SelectItem>
                  <SelectItem value="lowercase">Lowercase</SelectItem>
                  <SelectItem value="uppercase">Uppercase</SelectItem>
                </SelectContent>
              </Select>
            </div>
            
            <div class="space-y-1">
              <Label class="text-xs">Output Format</Label>
              <Select v-model="outputFormat">
                <SelectTrigger class="h-8 text-sm">
                  <SelectValue placeholder="Array" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="array">Array</SelectItem>
                  <SelectItem value="object">Object</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex justify-between gap-2 pt-2 border-t">
          <Button variant="outline" size="sm" @click="isOpen = false" class="h-7 text-xs">
            Cancel
          </Button>
          <div class="flex gap-2">
            <Button size="sm" @click="save(true)" variant="secondary" class="h-7 text-xs">
              Add & Continue
            </Button>
            <Button size="sm" @click="save(false)" class="h-7 text-xs">
              Add Pair
            </Button>
          </div>
        </div>
      </div>
    </PopoverContent>
  </Popover>
</template>
