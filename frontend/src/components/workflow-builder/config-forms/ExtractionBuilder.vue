<script setup lang="ts">
import { ref, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Trash2, Plus, ChevronRight, ChevronDown, GripVertical } from 'lucide-vue-next'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

interface ExtractionField {
  selector: string
  type: 'text' | 'html' | 'attribute' | 'list' | 'nested'
  attribute?: string
  transform?: string
  default?: string
  fields?: Record<string, ExtractionField> // For nested/list types
  item?: Record<string, ExtractionField> // For list type (legacy/alternative structure)
}

interface Props {
  modelValue: Record<string, ExtractionField>
}

interface Emits {
  (e: 'update:modelValue', value: Record<string, ExtractionField>): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const localFields = ref<Record<string, ExtractionField>>({})

// Initialize local state
watch(() => props.modelValue, (newVal) => {
  if (JSON.stringify(newVal) !== JSON.stringify(localFields.value)) {
    localFields.value = JSON.parse(JSON.stringify(newVal || {}))
  }
}, { immediate: true, deep: true })

// Emit changes
watch(localFields, (newVal) => {
  emit('update:modelValue', newVal)
}, { deep: true })

const expandedFields = ref<Set<string>>(new Set())

function toggleExpand(key: string) {
  if (expandedFields.value.has(key)) {
    expandedFields.value.delete(key)
  } else {
    expandedFields.value.add(key)
  }
}

function addField() {
  const key = `field_${Object.keys(localFields.value).length + 1}`
  localFields.value[key] = {
    selector: '',
    type: 'text'
  }
  expandedFields.value.add(key)
}

function removeField(key: string) {
  delete localFields.value[key]
}

function updateFieldKey(oldKey: string, newKey: string) {
  if (oldKey === newKey) return
  if (localFields.value[newKey]) return // Key exists

  const field = localFields.value[oldKey]
  delete localFields.value[oldKey]
  localFields.value[newKey] = field
  
  if (expandedFields.value.has(oldKey)) {
    expandedFields.value.delete(oldKey)
    expandedFields.value.add(newKey)
  }
}

// Recursive component for nested fields
const NestedFields = {
  name: 'NestedFields',
  props: ['fields', 'level'],
  emits: ['update'],
  setup(props: any, { emit }: any) {
    return () => null // Placeholder, actual implementation below in template
  }
}
</script>

<template>
  <div class="space-y-4">
    <div v-if="Object.keys(localFields).length === 0" class="text-center p-8 border-2 border-dashed rounded-lg text-muted-foreground">
      <p class="mb-2">No extraction fields configured</p>
      <Button variant="outline" size="sm" @click="addField">
        <Plus class="w-4 h-4 mr-2" />
        Add First Field
      </Button>
    </div>

    <div v-else class="space-y-3">
      <div 
        v-for="(field, key) in localFields" 
        :key="key"
        class="border rounded-lg bg-card transition-all"
        :class="{ 'ring-1 ring-primary/20': expandedFields.has(key as string) }"
      >
        <!-- Field Header -->
        <div class="flex items-center gap-2 p-3">
          <Button 
            variant="ghost" 
            size="icon" 
            class="h-6 w-6 shrink-0"
            @click="toggleExpand(key as string)"
          >
            <ChevronDown v-if="expandedFields.has(key as string)" class="w-4 h-4" />
            <ChevronRight v-else class="w-4 h-4" />
          </Button>

          <div class="flex-1 grid grid-cols-12 gap-2 items-center">
            <!-- Key Input -->
            <div class="col-span-4">
              <Input 
                :model-value="key"
                @update:model-value="val => updateFieldKey(key as string, val as string)"
                class="h-8 font-mono text-xs"
                placeholder="field_name"
              />
            </div>

            <!-- Selector Input (Quick Access) -->
            <div class="col-span-5">
              <Input 
                v-model="field.selector"
                class="h-8 text-xs"
                placeholder="CSS Selector (e.g. .price)"
              />
            </div>

            <!-- Type Badge -->
            <div class="col-span-3 flex justify-end">
              <Badge variant="secondary" class="text-xs font-mono">
                {{ field.type }}
              </Badge>
            </div>
          </div>

          <Button 
            variant="ghost" 
            size="icon" 
            class="h-8 w-8 text-muted-foreground hover:text-destructive shrink-0"
            @click="removeField(key as string)"
          >
            <Trash2 class="w-4 h-4" />
          </Button>
        </div>

        <!-- Expanded Config -->
        <div v-if="expandedFields.has(key as string)" class="p-3 pt-0 border-t bg-muted/10">
          <div class="grid grid-cols-2 gap-4 py-3">
            <div class="space-y-1">
              <Label class="text-xs">Type</Label>
              <Select v-model="field.type">
                <SelectTrigger class="h-8">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="text">Text Content</SelectItem>
                  <SelectItem value="html">Inner HTML</SelectItem>
                  <SelectItem value="attribute">Attribute</SelectItem>
                  <SelectItem value="list">List of Items</SelectItem>
                  <SelectItem value="nested">Nested Object</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="space-y-1">
              <Label class="text-xs">Transform</Label>
              <Select v-model="field.transform">
                <SelectTrigger class="h-8">
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

            <div v-if="field.type === 'attribute'" class="col-span-2 space-y-1">
              <Label class="text-xs">Attribute Name</Label>
              <Input v-model="field.attribute" placeholder="e.g. href, src, data-id" class="h-8" />
            </div>
          </div>

          <!-- Nested Fields / List Item Config -->
          <div v-if="field.type === 'list' || field.type === 'nested'" class="pl-4 border-l-2 border-primary/20 mt-2">
            <div class="text-xs font-medium text-muted-foreground mb-2">
              {{ field.type === 'list' ? 'List Item Fields' : 'Nested Fields' }}
            </div>
            
            <!-- Recursive usage for nested structures -->
            <!-- Note: For simplicity in this iteration, we'll handle one level of nesting directly 
                 or use a recursive component if we were using a full SFC structure. 
                 Since we are inside the component, we can use a self-reference if registered, 
                 but for now let's use a simplified approach for the immediate children. -->
            
            <ExtractionBuilder 
              v-if="field.fields || field.item"
              :model-value="(field.fields || field.item) as Record<string, ExtractionField>"
              @update:model-value="(val) => {
                if (field.type === 'list') field.item = val
                else field.fields = val
              }"
            />
            <div v-else class="text-center py-2">
              <Button variant="outline" size="sm" @click="() => {
                if (field.type === 'list') field.item = {}
                else field.fields = {}
              }">
                Configure {{ field.type === 'list' ? 'Items' : 'Fields' }}
              </Button>
            </div>
          </div>
        </div>
      </div>

      <Button variant="outline" size="sm" class="w-full border-dashed" @click="addField">
        <Plus class="w-4 h-4 mr-2" />
        Add Field
      </Button>
    </div>
  </div>
</template>
