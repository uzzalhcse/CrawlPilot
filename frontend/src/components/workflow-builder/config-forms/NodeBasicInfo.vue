<script setup lang="ts">
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface Props {
  label: string
  nodeType: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:label': [value: string]
}>()
</script>

<template>
  <div class="space-y-4">
    <!-- Node Label -->
    <div class="space-y-2">
      <Label for="node-label">Node Label</Label>
      <Input
        id="node-label"
        :model-value="label"
        @update:model-value="(val) => emit('update:label', String(val))"
        placeholder="Enter node label"
        class="text-base"
      />
    </div>

    <!-- Node Type (Read-only) -->
    <div class="space-y-2">
      <Label>Node Type</Label>
      <div class="p-2.5 bg-muted border border-border rounded-md text-sm font-mono">
        {{ nodeType }}
      </div>
    </div>
  </div>
</template>
