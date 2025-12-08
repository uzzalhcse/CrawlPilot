<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { generateSingleSelector } from '../utils/selector'
import type { Field, FieldConfig } from '../types'
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
import { inject } from 'vue'
import { toast } from 'vue-sonner'

const props = defineProps<{
  open: boolean
  mode: 'add' | 'edit'
  element?: HTMLElement | null
  field?: Field | null
  clickPosition?: { x: number, y: number }
}>()

const emit = defineEmits<{
  (e: 'save', data: { name: string, config: FieldConfig }): void
  (e: 'cancel'): void
  (e: 'update:open', value: boolean): void
}>()

// Inject existing fields to check for duplicates
const existingFields = inject<any>('fields', { value: [] })

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
const name = ref('')
const selector = ref('')
const type = ref<'text' | 'attr' | 'html'>('text')
const attribute = ref('')
const multiple = ref<'single' | 'multiple'>('single')
const transform = ref<'trim' | 'lowercase' | 'uppercase' | 'none'>('none')

// Initialize state based on mode
function initialize() {
  if (props.mode === 'add' && props.element) {
    name.value = ''
    selector.value = generateSingleSelector(props.element) // Use single selector
    type.value = 'text'
    attribute.value = ''
    multiple.value = 'single'
    transform.value = 'none'
  } else if (props.mode === 'edit' && props.field && props.field.fieldType === 'single') {
    name.value = props.field.name
    selector.value = props.field.config.selector
    type.value = props.field.config.type
    attribute.value = props.field.config.attribute || ''
    multiple.value = props.field.config.multiple ? 'multiple' : 'single'
    transform.value = props.field.config.transform || 'none'
  }
}

// Watch for open prop to initialize
watch(() => props.open, (isOpen) => {
  if (isOpen) {
    initialize()
    highlightMatchingElements()
  } else {
    clearPreviewHighlights()
  }
})

// Highlighting Logic
let highlightedElements: HTMLElement[] = []

function highlightMatchingElements() {
  if (!props.open) return

  clearPreviewHighlights()
  
  try {
    if (!selector.value) return
    
    const elements = document.querySelectorAll(selector.value)
    elements.forEach((el) => {
      const htmlEl = el as HTMLElement
      htmlEl.classList.add('selectflow-highlight-preview')
      highlightedElements.push(htmlEl)
    })
  } catch {
    // Invalid selector, ignore
  }
}

function clearPreviewHighlights() {
  highlightedElements.forEach(el => {
    el.classList.remove('selectflow-highlight-preview')
  })
  highlightedElements = []
}

// Computed
const matchCount = computed(() => {
  try {
    return selector.value ? document.querySelectorAll(selector.value).length : 0
  } catch {
    return 0
  }
})

const previewValue = computed(() => {
  try {
    if (!selector.value) return 'No selector'
    
    const elements = document.querySelectorAll(selector.value)
    if (elements.length === 0) return 'No elements found'
    
    const extractValue = (el: Element) => {
      let val = ''
      if (type.value === 'text') {
        val = el.textContent?.trim() || ''
      } else if (type.value === 'attr' && attribute.value) {
        val = el.getAttribute(attribute.value) || ''
      } else if (type.value === 'html') {
        val = el.innerHTML
      }
      
      if (transform.value === 'trim') val = val.trim()
      if (transform.value === 'lowercase') val = val.toLowerCase()
      if (transform.value === 'uppercase') val = val.toUpperCase()
      
      if (type.value === 'html' && val.length > 100) {
        return val.substring(0, 100) + '...'
      }
      
      return val
    }
    
    if (multiple.value === 'multiple') {
      const values = Array.from(elements).slice(0, 3).map(extractValue)
      return values.length > 0 ? values.join(', ') + (elements.length > 3 ? '...' : '') : 'No data'
    } else {
      return extractValue(elements[0]) || 'Empty'
    }
  } catch (e) {
    return 'Invalid selector'
  }
})

// Watchers
watch(selector, highlightMatchingElements)
watch(type, highlightMatchingElements)

function save() {
  console.log('[FieldModal] Save button clicked')
  
  if (!name.value.trim()) {
    toast.error('Field name cannot be empty')
    return
  }
  
  const isDuplicate = existingFields.value.some((f: any) => {
    if (props.mode === 'edit' && props.field) {
      return f.name === name.value && f.id !== props.field.id
    }
    return f.name === name.value
  })
  
  if (isDuplicate) {
    toast.error(`Field name "${name.value}" already exists! Please choose a different name.`)
    return
  }
  
  if (type.value === 'attr' && !attribute.value.trim()) {
    toast.error('Please enter an attribute name (e.g., src, href, alt)')
    return
  }
  
  console.log('[FieldModal] Multiple value:', multiple.value)
  console.log('[FieldModal] Emitting save event')
  
  emit('save', {
    name: name.value,
    config: {
      selector: selector.value,
      type: type.value,
      attribute: type.value === 'attr' ? attribute.value : undefined,
      multiple: multiple.value === 'multiple',
      transform: transform.value === 'none' ? '' : transform.value
    }
  })
}

const isOpen = computed({
  get: () => props.open,
  set: (val) => {
    console.log('[FieldModal] Popover open state changing to:', val)
    emit('update:open', val)
    if (!val) {
      console.log('[FieldModal] Emitting cancel event')
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
      class="w-[380px] p-4 z-[1000001]"
      :side="'bottom'"
      :align="'start'"
    >
      <div class="space-y-3">
        <!-- Header -->
        <div class="flex items-center justify-between pb-2 border-b">
          <h4 class="font-semibold text-sm">{{ mode === 'add' ? 'Add Field' : 'Edit Field' }}</h4>
        </div>
        
        <div class="space-y-3 text-sm">
          <!-- Field Name -->
          <div class="space-y-1">
            <Label for="name" class="text-xs">Name</Label>
            <Input id="name" v-model="name" placeholder="e.g., product_name" class="h-8 text-sm" />
          </div>
          
          <!-- Type & Attribute in row when attr -->
          <div class="grid gap-2" :class="type === 'attr' ? 'grid-cols-2' : 'grid-cols-1'">
            <div class="space-y-1">
              <Label class="text-xs">Type</Label>
              <Select v-model="type">
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
            
            <div v-if="type === 'attr'" class="space-y-1">
              <Label for="attribute" class="text-xs">Attribute</Label>
              <Input id="attribute" v-model="attribute" placeholder="src, href" class="h-8 text-sm" />
            </div>
          </div>
          
          <!-- Transform & Multiple in row -->
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
              <Label class="text-xs">Extract</Label>
              <Select v-model="multiple">
                <SelectTrigger class="h-8 text-sm">
                  <SelectValue placeholder="Single" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="single">Single</SelectItem>
                  <SelectItem value="multiple">Multiple</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          
          <!-- CSS Selector -->
          <div class="space-y-1">
            <Label for="selector" class="text-xs">Selector</Label>
            <Input id="selector" v-model="selector" placeholder=".product-title" class="h-8 font-mono text-xs" />
          </div>
          
          <!-- Preview -->
          <div class="rounded border bg-muted/50 p-2">
            <div class="flex justify-between items-center mb-1">
              <span class="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">Preview</span>
              <span class="text-[10px] text-primary font-medium">{{ matchCount }} {{ matchCount === 1 ? 'match' : 'matches' }}</span>
            </div>
            <code class="block text-[10px] font-mono text-muted-foreground break-all max-h-8 overflow-y-auto leading-tight">{{ previewValue }}</code>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex justify-end gap-2 pt-2 border-t">
          <Button variant="outline" size="sm" @click="isOpen = false" class="h-7 text-xs">Cancel</Button>
          <Button size="sm" @click="save" class="h-7 text-xs">{{ mode === 'add' ? 'Add' : 'Save' }}</Button>
        </div>
      </div>
    </PopoverContent>
  </Popover>
</template>
