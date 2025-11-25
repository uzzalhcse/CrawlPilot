<script setup lang="ts">
interface Stat {
  label: string
  value: number | string
  color?: string
}

interface Props {
  stats: Stat[]
}

defineProps<Props>()
</script>

<template>
  <div class="border-b border-border bg-background px-4 md:px-6 py-4">
    <div class="grid gap-4 md:gap-8" :style="{ gridTemplateColumns: `repeat(${stats.length}, minmax(0, 1fr))` }">
      <div v-for="(stat, index) in stats" :key="index">
        <div class="text-xs text-muted-foreground mb-1">{{ stat.label }}</div>
        <div class="text-xl md:text-2xl font-semibold" :class="stat.color">{{ stat.value }}</div>
      </div>
    </div>
  </div>
</template>
