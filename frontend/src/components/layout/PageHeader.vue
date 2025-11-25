<script setup lang="ts">
import { HelpCircle } from 'lucide-vue-next'

interface Props {
  title: string
  description?: string
  showHelpIcon?: boolean
}

defineProps<Props>()
</script>

<template>
  <div class="border-b border-border bg-background px-4 md:px-6 py-4">
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 mb-3">
      <div class="flex items-center gap-2">
        <h1 class="text-xl sm:text-2xl font-bold">{{ title }}</h1>
        <HelpCircle v-if="showHelpIcon" class="w-4 h-4 text-muted-foreground cursor-pointer" />
      </div>
      <div class="flex flex-wrap items-center gap-2 w-full sm:w-auto">
        <slot name="actions" />
      </div>
    </div>
    <p v-if="description" class="text-sm text-muted-foreground">{{ description }}</p>
  </div>
</template>
