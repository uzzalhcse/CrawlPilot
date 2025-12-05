<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Copy, Trash2, ChevronDown, ChevronUp } from 'lucide-vue-next'
import FieldInput from './FieldInput.vue'
import SelectInput from './SelectInput.vue'
import IndependentArrayManager from './IndependentArrayManager.vue'
import FieldActionsManager from './FieldActionsManager.vue'

interface ParamField {
  key: string
  label: string
  type: string
  required?: boolean
  placeholder?: string
  description?: string
  options?: Array<{label: string; value: string}>
  defaultValue?: any
}

interface Props {
  fieldName: string
  fieldData: Record<string, any>
  schema: ParamField[]
  collapsed?: boolean
  index: number
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:fieldData': [data: Record<string, any>]
  'update:fieldName': [oldName: string, newName: string]
  'delete': []
  'duplicate': []
  'toggle-collapse': []
}>()

// Local editable field name to prevent jumping while typing
const localFieldName = ref(props.fieldName)

// Watch for external field name changes (from parent)
watch(() => props.fieldName, (newName: string) => {
  localFieldName.value = newName
})

// Determine which fields to show based on current field data
function shouldShowField(field: ParamField): boolean {
  // Show attribute field only when type is 'attr'
  if (field.key === 'attribute') {
    return props.fieldData.type === 'attr'
  }
  
  // Show limit field only when multiple is true
  if (field.key === 'limit') {
    return props.fieldData.multiple === true
  }
  
  // Show nested fields only when multiple is true and no extractions
  if (field.key === 'fields') {
    return props.fieldData.multiple === true && !props.fieldData.extractions
  }
  
  // Show extractions field when it exists or neither selector nor multiple are set
  if (field.key === 'extractions') {
    return props.fieldData.extractions || (!props.fieldData.selector && !props.fieldData.multiple)
  }
  
  // Hide selector, type, multiple if extractions is being used
  if (['selector', 'type', 'multiple'].includes(field.key)) {
    return !props.fieldData.extractions
  }
  
  // Show transform only when type is set
  if (field.key === 'transform') {
    return props.fieldData.type && props.fieldData.type !== ''
  }
  
  return true
}

function updateField(key: string, value: any) {
  emit('update:fieldData', { ...props.fieldData, [key]: value })
}

function handleFieldNameBlur() {
  const newName = localFieldName.value.trim()
  if (newName && newName !== props.fieldName) {
    emit('update:fieldName', props.fieldName, newName)
  } else if (!newName) {
    // Reset to original if empty
    localFieldName.value = props.fieldName
  }
}

const badges = computed(() => {
  const result = []
  
  if (props.fieldData.multiple) {
    result.push({ label: '📋 Array', color: 'purple' })
  }
  
  if (props.fieldData.multiple && props.fieldData.fields) {
    result.push({ label: '🔗 Nested', color: 'blue' })
  }
  
  if (props.fieldData.extractions) {
    const count = Array.isArray(props.fieldData.extractions) 
      ? props.fieldData.extractions.length 
      : 0
    result.push({ label: `🔗 K-V Pairs (${count})`, color: 'green' })
  }
  
  // Show actions badge if field has pre-extraction actions
  if (props.fieldData.actions) {
    let count = 0
    try {
      const actions = typeof props.fieldData.actions === 'string' 
        ? JSON.parse(props.fieldData.actions) 
        : props.fieldData.actions
      count = Array.isArray(actions) ? actions.length : 0
    } catch { count = 0 }
    if (count > 0) {
      result.push({ label: `⚡ ${count} Actions`, color: 'amber' })
    }
  }
  
  return result
})
</script>

<template>
  <div 
    data-field-card
    class="relative border-2 rounded-lg bg-card shadow-sm hover:shadow-md transition-all"
    :class="collapsed ? 'border-border' : 'border-primary/20'"
  >
    <!-- Header -->
    <div 
      class="flex items-center justify-between p-3 cursor-pointer hover:bg-muted/50 transition-colors"
      @click="emit('toggle-collapse')"
    >
      <div class="flex items-center gap-2 flex-1 min-w-0">
        <!-- Index Badge -->
        <div class="flex items-center justify-center w-6 h-6 rounded-full bg-primary/10 text-primary text-xs font-semibold shrink-0">
          {{ index + 1 }}
        </div>
        
        <!-- Field Name Display (non-editable in header for simplicity) -->
        <div class="font-semibold text-sm truncate" :title="fieldName">
          {{ fieldName }}
        </div>
        
        <!-- Badges -->
        <span 
          v-for="badge in badges" 
          :key="badge.label"
          class="text-xs px-1.5 py-0.5 rounded font-medium border shrink-0"
          :class="{
            'bg-purple-100 text-purple-700 border-purple-300': badge.color === 'purple',
            'bg-blue-100 text-blue-700 border-blue-300': badge.color === 'blue',
            'bg-green-100 text-green-700 border-green-300': badge.color === 'green',
            'bg-amber-100 text-amber-700 border-amber-300': badge.color === 'amber'
          }"
        >
          {{ badge.label }}
        </span>
        
        <!-- Selector Preview -->
        <div v-if="fieldData.selector" class="text-xs text-muted-foreground truncate">
          <span class="font-mono bg-muted px-1.5 py-0.5 rounded">
            {{ fieldData.selector }}
          </span>
        </div>
      </div>
      
      <!-- Actions -->
      <div class="flex items-center gap-1 shrink-0">
        <Button
          type="button"
          size="sm"
          variant="ghost"
          class="h-8 w-8 p-0 hover:bg-primary/10"
          @click.stop="emit('duplicate')"
          title="Duplicate field"
        >
          <Copy class="h-3.5 w-3.5" />
        </Button>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          class="h-8 w-8 p-0 hover:bg-destructive/10 hover:text-destructive"
          @click.stop="emit('delete')"
          title="Delete field"
        >
          <Trash2 class="h-3.5 w-3.5" />
        </Button>
        <div class="h-4 w-4 text-muted-foreground shrink-0">
          <ChevronDown v-if="!collapsed" class="h-4 w-4 transition-transform" />
          <ChevronUp v-else class="h-4 w-4 transition-transform" />
        </div>
      </div>
    </div>
    
    <!-- Body (Collapsible) -->
    <div 
      v-show="!collapsed"
      class="p-4 pt-3 border-t border-border space-y-4"
    >
      <!-- Field Name (Editable, prominent) -->
      <div class="space-y-2">
        <Label for="field-name-input" class="text-sm font-semibold">
          Field Name <span class="text-destructive">*</span>
        </Label>
        <Input
          id="field-name-input"
          v-model="localFieldName"
          @blur="handleFieldNameBlur"
          @keydown.enter="($event.target as HTMLInputElement).blur()"
          placeholder="e.g., title, price, images"
        />
        <p class="text-xs text-muted-foreground">Unique identifier for this extracted field</p>
      </div>

      <!-- Main Extraction Settings (Grid Layout) -->
      <div class="grid grid-cols-2 gap-3">
        <!-- Selector -->
        <div 
          v-for="field in schema" 
          :key="field.key"
          v-show="shouldShowField(field) && field.key === 'selector'"
          class="col-span-2 space-y-1.5"
        >
          <Label :for="`field-${fieldName}-${field.key}`" class="text-sm">
            {{ field.label }}
            <span v-if="field.required" class="text-red-500">*</span>
          </Label>
          <FieldInput
            :type="field.type as any"
            :model-value="fieldData[field.key]"
            :placeholder="field.placeholder"
            @update:model-value="updateField(field.key, $event)"
          />
          <p v-if="field.description" class="text-xs text-muted-foreground">
            {{ field.description }}
          </p>
        </div>

        <!-- Type & Multiple (Side by Side) -->
        <div 
          v-for="field in schema" 
          :key="field.key"
          v-show="shouldShowField(field) && field.key === 'type'"
          class="space-y-1.5"
        >
          <Label :for="`field-${fieldName}-${field.key}`" class="text-sm">
            {{ field.label }}
            <span v-if="field.required" class="text-red-500">*</span>
          </Label>
          <SelectInput
            v-if="field.type === 'select' && field.options"
            :model-value="fieldData[field.key] || field.defaultValue"
            :options="field.options"
            @update:model-value="updateField(field.key, $event)"
          />
          <FieldInput
            v-else
            :type="field.type as any"
            :model-value="fieldData[field.key]"
            :placeholder="field.placeholder"
            @update:model-value="updateField(field.key, $event)"
          />
        </div>

        <div 
          v-for="field in schema" 
          :key="field.key"
          v-show="shouldShowField(field) && field.key === 'multiple'"
          class="space-y-1.5 flex flex-col justify-end"
        >
          <div class="flex items-center space-x-2 h-10">
            <FieldInput
              type="boolean"
              :model-value="fieldData[field.key] ?? field.defaultValue ?? false"
              @update:model-value="updateField(field.key, $event)"
            />
            <Label class="font-normal text-sm">
              {{ field.label }}
            </Label>
          </div>
          <p v-if="field.description" class="text-xs text-muted-foreground">
            {{ field.description }}
          </p>
        </div>

        <!-- Attribute (when type=attr, full width) -->
        <div 
          v-for="field in schema" 
          :key="field.key"
          v-show="shouldShowField(field) && field.key === 'attribute'"
          class="col-span-2 space-y-1.5"
        >
          <Label :for="`field-${fieldName}-${field.key}`" class="text-sm">
            {{ field.label }}
            <span v-if="field.required" class="text-red-500">*</span>
          </Label>
          <FieldInput
            :type="field.type as any"
            :model-value="fieldData[field.key]"
            :placeholder="field.placeholder"
            @update:model-value="updateField(field.key, $event)"
          />
          <p v-if="field.description" class="text-xs text-muted-foreground">
            {{ field.description }}
          </p>
        </div>

        <!-- Transform & Limit (Side by Side) -->
        <div 
          v-for="field in schema" 
          :key="field.key"
          v-show="shouldShowField(field) && field.key === 'transform'"
          class="space-y-1.5"
        >
          <Label :for="`field-${fieldName}-${field.key}`" class="text-sm">
            {{ field.label }}
          </Label>
          <SelectInput
            v-if="field.type === 'select' && field.options"
            :model-value="fieldData[field.key] || field.defaultValue"
            :options="field.options"
            @update:model-value="updateField(field.key, $event)"
          />
          <FieldInput
            v-else
            :type="field.type as any"
            :model-value="fieldData[field.key]"
            :placeholder="field.placeholder"
            @update:model-value="updateField(field.key, $event)"
          />
        </div>

        <div 
          v-for="field in schema" 
          :key="field.key"
          v-show="shouldShowField(field) && field.key === 'limit'"
          class="space-y-1.5"
        >
          <Label :for="`field-${fieldName}-${field.key}`" class="text-sm">
            {{ field.label }}
          </Label>
          <FieldInput
            :type="field.type as any"
            :model-value="fieldData[field.key]"
            :placeholder="field.placeholder"
            @update:model-value="updateField(field.key, $event)"
          />
          <p v-if="field.description" class="text-xs text-muted-foreground">
            {{ field.description }}
          </p>
        </div>

        <!-- Default Value (Full width) -->
        <div 
          v-for="field in schema" 
          :key="field.key"
          v-show="shouldShowField(field) && field.key === 'default_value'"
          class="col-span-2 space-y-1.5"
        >
          <Label :for="`field-${fieldName}-${field.key}`" class="text-sm">
            {{ field.label }}
          </Label>
          <FieldInput
            :type="field.type as any"
            :model-value="fieldData[field.key]"
            :placeholder="field.placeholder"
            @update:model-value="updateField(field.key, $event)"
          />
          <p v-if="field.description" class="text-xs text-muted-foreground">
            {{ field.description }}
          </p>
        </div>

        <!-- Nested Fields (Full width) -->
        <div 
          v-for="field in schema" 
          :key="field.key"
          v-show="shouldShowField(field) && field.key === 'fields'"
          class="col-span-2 space-y-1.5"
        >
          <Label :for="`field-${fieldName}-${field.key}`" class="text-sm">
            {{ field.label }}
          </Label>
          <FieldInput
            type="textarea"
            :model-value="fieldData[field.key]"
            :placeholder="field.placeholder"
            :rows="3"
            @update:model-value="updateField(field.key, $event)"
          />
          <p v-if="field.description" class="text-xs text-muted-foreground">
            {{ field.description }}
          </p>
        </div>
      </div>

      <!-- Independent Array Manager for extractions (Full width, separate section) -->
      <div 
        v-for="field in schema" 
        :key="field.key"
        v-show="shouldShowField(field) && field.key === 'extractions'"
        class="pt-3 border-t border-border"
      >
        <div class="mb-2">
          <Label class="text-sm font-medium">{{ field.label }}</Label>
          <p v-if="field.description" class="text-xs text-muted-foreground mt-1">
            {{ field.description }}
          </p>
        </div>
        <IndependentArrayManager
          :model-value="fieldData[field.key] || []"
          @update:model-value="updateField(field.key, $event)"
        />
      </div>

      <!-- Pre-Extraction Actions (Visual Editor) -->
      <div class="pt-3 border-t border-border">
        <FieldActionsManager
          :model-value="fieldData['actions']"
          @update:model-value="updateField('actions', $event)"
        />
      </div>
    </div>
  </div>
</template>
