<script setup lang="ts">
import { useRoute } from 'vue-router'

const route = useRoute()
const workflowId = route.params.id
</script>

<template>
  <div class="space-y-6">
    <div>
      <h2 class="text-3xl font-bold tracking-tight">Workflow Details</h2>
      <p class="text-muted-foreground">ID: {{ workflowId }}</p>
    </div>

    <div class="rounded-lg border bg-card p-6 shadow-sm">
      <p class="text-muted-foreground">Workflow details - API implementation coming soon...</p>
    </div>
  </div>
</template>
