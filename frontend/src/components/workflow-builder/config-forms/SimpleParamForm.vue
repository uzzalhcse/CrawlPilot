<script setup lang="ts">
import { Label } from '@/components/ui/label'
import FieldInput from './FieldInput.vue'
import SelectInput from './SelectInput.vue'

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
  params: Record<string, any>
  schema: ParamField[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:params': [key: string, value: any]
}>()
</script>

<template>
  <div class="space-y-4">
    <div 
      v-for="field in schema" 
      :key="field.key"
      class="space-y-2"
    >
      <Label :for="`param-${field.key}`">
        {{ field.label }}
        <span v-if="field.required" class="text-red-500">*</span>
      </Label>
      
      <!-- Select Input -->
      <SelectInput
        v-if="field.type === 'select' && field.options"
        :id="`param-${field.key}`"
        :model-value="params[field.key] || field.defaultValue"
        :options="field.options"
        :placeholder="field.placeholder"
        @update:model-value="emit('update:params', field.key, $event)"
      />
      
      <!-- Boolean as separate row for better UX -->
      <div v-else-if="field.type === 'boolean'" class="flex items-center space-x-2">
        <FieldInput
          type="boolean"
          :model-value="params[field.key] ?? field.defaultValue ?? false"
          @update:model-value="emit('update:params', field.key, $event)"
        />
        <Label class="font-normal text-sm text-muted-foreground">
          {{ field.description }}
        </Label>
      </div>
      
      <!-- Other Inputs -->
      <FieldInput
        v-else
        :id="`param-${field.key}`"
        :type="field.type as any"
        :model-value="params[field.key]"
        :placeholder="field.placeholder"
        @update:model-value="emit('update:params', field.key, $event)"
      />
      
      <!-- Description (not for boolean as it's already inline) -->
      <p v-if="field.description && field.type !== 'boolean'" class="text-xs text-muted-foreground">
        {{field.description }}
      </p>
    </div>
  </div>
</template>
