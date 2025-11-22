<script setup lang="ts">
import { cn } from '@/lib/utils'

const props = defineProps<{ class?: string }>()
</script>

<template>
  <div :class="cn('relative overflow-y-auto overflow-x-hidden crawlify-scrollbar', props.class)">
    <slot />
  </div>
</template>
