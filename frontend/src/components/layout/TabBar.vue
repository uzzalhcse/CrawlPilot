<script setup lang="ts">
interface Tab {
  id: string
  label: string
}

interface Props {
  tabs: Tab[]
  modelValue: string
}

interface Emits {
  (e: 'update:modelValue', value: string): void
}

defineProps<Props>()
const emit = defineEmits<Emits>()

const handleTabClick = (tabId: string) => {
  emit('update:modelValue', tabId)
}
</script>

<template>
  <div class="border-b border-border bg-background px-4 md:px-6 overflow-x-auto">
    <div class="flex gap-4 md:gap-6 min-w-max">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        @click="handleTabClick(tab.id)"
        :class="[
          'py-3 text-sm font-medium border-b-2 transition-colors whitespace-nowrap',
          modelValue === tab.id 
            ? 'border-primary text-foreground' 
            : 'border-transparent text-muted-foreground hover:text-foreground'
        ]"
      >
        {{ tab.label }}
      </button>
    </div>
  </div>
</template>
