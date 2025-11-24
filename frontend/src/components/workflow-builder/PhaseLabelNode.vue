<script setup lang="ts">
defineProps<{
  data: {
    label: string
    index: number
    phaseWidth?: number
  }
}>()
</script>

<template>
  <div class="px-6 py-3 rounded-lg bg-gradient-to-r from-primary/10 to-primary/5 border-2 border-primary/20 font-bold text-xl text-foreground shadow-lg backdrop-blur-sm whitespace-nowrap min-w-[300px] text-center">
    <div class="flex items-center justify-center gap-3">
      <span class="text-primary text-sm font-semibold uppercase tracking-wider">Phase {{ data.index + 1 }}</span>
      <span class="w-1.5 h-1.5 rounded-full bg-primary/40"></span>
      <span class="text-lg font-bold">{{ data.label }}</span>
    </div>
  </div>
</template>
