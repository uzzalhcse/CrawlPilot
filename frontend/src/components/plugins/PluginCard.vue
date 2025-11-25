<template>
  <Card class="group border-0 bg-[#1a1d21] hover:bg-[#222529] transition-all duration-200 cursor-pointer h-full flex flex-col overflow-hidden">
    <CardContent class="p-4 flex flex-col h-full">
      <!-- Header -->
      <div class="flex items-start gap-3 mb-3">
        <!-- Icon -->
        <div class="w-10 h-10 rounded-lg bg-[#2d3136] flex items-center justify-center text-xl shrink-0">
          {{ getPhaseIcon(plugin.phase_type) }}
        </div>
        
        <!-- Title & Subtitle -->
        <div class="min-w-0 flex-1">
          <h3 class="text-[15px] font-semibold text-white leading-tight mb-0.5 truncate group-hover:text-blue-400 transition-colors">
            {{ plugin.name }}
          </h3>
          <p class="text-xs text-gray-400 truncate">
            {{ plugin.slug }}
          </p>
        </div>
      </div>

      <!-- Description -->
      <p class="text-sm text-gray-400 line-clamp-3 mb-4 flex-1 leading-relaxed">
        {{ plugin.description }}
      </p>

      <!-- Footer Stats -->
      <div class="flex items-center gap-4 text-xs text-gray-500 mt-auto pt-3 border-t border-gray-800/50">
        <!-- Author -->
        <div class="flex items-center gap-1.5">
          <div class="w-4 h-4 rounded-full bg-gray-700 flex items-center justify-center text-[8px] font-bold text-gray-300">
            {{ plugin.author_name.charAt(0).toUpperCase() }}
          </div>
          <span class="truncate max-w-[80px]">{{ plugin.author_name }}</span>
        </div>

        <!-- Downloads/Users -->
        <div class="flex items-center gap-1">
          <Users class="w-3.5 h-3.5" />
          <span>{{ formatNumber(plugin.total_downloads) }}</span>
        </div>

        <!-- Rating -->
        <div class="flex items-center gap-1">
          <Star class="w-3.5 h-3.5 fill-gray-500" />
          <span>{{ plugin.average_rating.toFixed(1) }}</span>
        </div>
      </div>
    </CardContent>
  </Card>
</template>

<script setup lang="ts">
import { Card, CardContent } from '@/components/ui/card'
import { Users, Star } from 'lucide-vue-next'
import type { Plugin } from '@/types'

defineProps<{
  plugin: Plugin
}>()

defineEmits<{
  // (e: 'click'): void // Removed to allow native click event
}>()

const formatNumber = (num: number): string => {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num.toString()
}

const getPhaseIcon = (phaseType: string): string => {
  const icons: Record<string, string> = {
    discovery: '🔍',
    extraction: '📦',
    processing: '⚙️',
    custom: '🔧'
  }
  return icons[phaseType] || '📄'
}
</script>
