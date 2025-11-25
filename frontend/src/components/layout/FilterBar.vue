<script setup lang="ts">
import { Input } from '@/components/ui/input'
import { Search } from 'lucide-vue-next'

interface Props {
  searchPlaceholder?: string
  searchValue?: string
}

interface Emits {
  (e: 'update:searchValue', value: string): void
}

defineProps<Props>()
const emit = defineEmits<Emits>()

const handleSearchInput = (event: Event) => {
  const target = event.target as HTMLInputElement
  emit('update:searchValue', target.value)
}
</script>

<template>
  <div class="border-b border-border bg-background px-4 md:px-6 py-3">
    <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
      <div class="relative flex-1 max-w-full sm:max-w-xs">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <Input
          :value="searchValue"
          @input="handleSearchInput"
          :placeholder="searchPlaceholder || 'Search...'"
          class="pl-9 h-9"
        />
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <slot name="filters" />
      </div>
    </div>
  </div>
</template>
