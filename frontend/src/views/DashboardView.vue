<script setup lang="ts">
import { ref } from 'vue'

// Placeholder for dashboard
const stats = ref([
  { label: 'Total Workflows', value: '12', change: '+2 from last week' },
  { label: 'Active Executions', value: '3', change: 'Running now' },
  { label: 'Items Extracted', value: '1,234', change: '+180 today' },
  { label: 'Success Rate', value: '98%', change: '+2% this week' }
])
</script>

<template>
  <div class="space-y-6">
    <div>
      <h2 class="text-3xl font-bold tracking-tight">Dashboard</h2>
      <p class="text-muted-foreground">
        Welcome to Crawlify - Your web crawling automation platform
      </p>
    </div>

    <!-- Stats Grid -->
    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      <div
        v-for="stat in stats"
        :key="stat.label"
        class="rounded-lg border bg-card p-6 shadow-sm"
      >
        <div class="flex flex-col space-y-2">
          <span class="text-sm font-medium text-muted-foreground">{{ stat.label }}</span>
          <div class="flex items-baseline gap-2">
            <span class="text-3xl font-bold">{{ stat.value }}</span>
          </div>
          <p class="text-xs text-muted-foreground">{{ stat.change }}</p>
        </div>
      </div>
    </div>

    <!-- Recent Activity Placeholder -->
    <div class="rounded-lg border bg-card p-6 shadow-sm">
      <h3 class="mb-4 text-lg font-semibold">Recent Activity</h3>
      <p class="text-muted-foreground">API implementation coming soon...</p>
    </div>
  </div>
</template>
