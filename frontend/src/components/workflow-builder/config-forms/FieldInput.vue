<script setup lang="ts">
import { computed } from 'vue'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'

interface Props {
  type: 'text' | 'number' | 'textarea' | 'boolean'
  modelValue: any
  placeholder?: string
  rows?: number
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  rows: 4,
  disabled: false
})

const emit = defineEmits<{
  'update:modelValue': [value: any]
}>()

const displayValue = computed(() => {
  if (props.type === 'textarea' && typeof props.modelValue === 'object') {
    return JSON.stringify(props.modelValue, null, 2)
  }
  return props.modelValue ?? ''
})

function handleTextareaBlur(event: Event) {
  const value = (event.target as HTMLTextAreaElement).value
  try {
    const parsed = JSON.parse(value)
    emit('update:modelValue', parsed)
  } catch {
    emit('update:modelValue', value)
  }
}
</script>

<template>
  <!-- Text Input -->
  <Input
    v-if="type === 'text'"
    :model-value="modelValue"
    @update:model-value="emit('update:modelValue', $event)"
    :placeholder="placeholder"
    :disabled="disabled"
  />
  
  <!-- Number Input -->
  <Input
    v-else-if="type === 'number'"
    type="number"
    :model-value="modelValue"
    @update:model-value="emit('update:modelValue', Number($event))"
    :disabled="disabled"
  />
  
  <!-- Textarea (supports JSON objects) -->
  <Textarea
    v-else-if="type === 'textarea'"
    :model-value="displayValue"
    @blur="handleTextareaBlur"
    :placeholder="placeholder"
    :rows="rows"
    :disabled="disabled"
    class="font-mono text-sm"
  />
  
  <!-- Boolean Switch -->
  <Switch
    v-else-if="type === 'boolean'"
    :checked="!!modelValue"
    @update:checked="emit('update:modelValue', $event)"
    :disabled="disabled"
  />
</template>
