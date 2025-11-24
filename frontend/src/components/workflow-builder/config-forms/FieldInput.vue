<script setup lang="ts">
import { computed } from 'vue'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'

interface Props {
  modelValue: any
  type?: 'text' | 'number' | 'textarea' | 'boolean'
  placeholder?: string
  label?: string
}

const props = withDefaults(defineProps<Props>(), {
  type: 'text',
  placeholder: ''
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: any): void
}>()

const localValue = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const jsonValue = computed({
  get: () => {
    if (typeof props.modelValue === 'object') {
      return JSON.stringify(props.modelValue, null, 2)
    }
    return props.modelValue
  },
  set: (val) => {
    try {
      const parsed = JSON.parse(val)
      emit('update:modelValue', parsed)
    } catch (e) {
      // If invalid JSON, just emit the string
      emit('update:modelValue', val)
    }
  }
})
</script>

<template>
  <div class="w-full">
    <div v-if="type === 'boolean'" class="flex items-center space-x-2">
      <Switch 
        :checked="localValue"
        @update:checked="localValue = $event"
      />
      <Label v-if="label" class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
        {{ label }}
      </Label>
    </div>

    <div v-else-if="type === 'textarea'" class="space-y-2">
      <Label v-if="label">{{ label }}</Label>
      <Textarea
        v-model="jsonValue"
        :placeholder="placeholder"
        class="font-mono text-xs min-h-[100px]"
      />
    </div>

    <div v-else class="space-y-2">
      <Label v-if="label">{{ label }}</Label>
      <Input
        v-if="type === 'number'"
        type="number"
        v-model.number="localValue"
        :placeholder="placeholder"
      />
      <Input
        v-else
        type="text"
        v-model="localValue"
        :placeholder="placeholder"
      />
    </div>
  </div>
</template>
